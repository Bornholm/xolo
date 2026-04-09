package proxy

import (
	"context"
	"log/slog"

	genaiProxy "github.com/bornholm/genai/proxy"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/bornholm/xolo/internal/core/service"
	"github.com/bornholm/xolo/internal/metrics"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

// XoloUsageTracker is a PostResponseHook that records one UsageRecord per
// successful proxy call with cost frozen from the model's pricing, converted
// to the organization's base currency.
type XoloUsageTracker struct {
	usageStore          port.UsageStore
	providerStore       port.ProviderStore
	orgStore            port.OrgStore
	exchangeRateService *service.ExchangeRateService
}

func NewXoloUsageTracker(
	usageStore port.UsageStore,
	providerStore port.ProviderStore,
	orgStore port.OrgStore,
	exchangeRateService *service.ExchangeRateService,
) *XoloUsageTracker {
	return &XoloUsageTracker{
		usageStore:          usageStore,
		providerStore:       providerStore,
		orgStore:            orgStore,
		exchangeRateService: exchangeRateService,
	}
}

func (t *XoloUsageTracker) Name() string  { return "xolo.usage-tracker" }
func (t *XoloUsageTracker) Priority() int { return 100 }

// PostResponse implements proxy.PostResponseHook.
func (t *XoloUsageTracker) PostResponse(ctx context.Context, req *genaiProxy.ProxyRequest, res *genaiProxy.ProxyResponse) (*genaiProxy.HookResult, error) {
	if res.TokensUsed == nil || req.UserID == "" {
		return nil, nil
	}

	PopulateMetaFromContext(ctx, req.Metadata)

	orgID := OrgIDFromMeta(req.Metadata)
	if orgID == "" {
		return nil, nil
	}

	modelID := ModelIDFromMeta(req.Metadata)
	if modelID == "" {
		return nil, nil
	}

	// Get original and resolved model names from metadata.
	originalModel := req.Model
	resolvedModel := req.Model
	if v, ok := req.Metadata[MetaOriginalModel].(string); ok && v != "" {
		originalModel = v
	}
	if v, ok := req.Metadata[MetaResolvedModel].(string); ok && v != "" {
		resolvedModel = v
	}

	authTokenID := AuthTokenIDFromMeta(req.Metadata)
	applicationID := ApplicationIDFromMeta(req.Metadata)

	llmModel, err := t.providerStore.GetLLMModelByID(ctx, modelID)
	if err != nil {
		slog.ErrorContext(ctx, "usage tracker: could not load LLM model", slog.Any("error", err), slog.String("modelID", string(modelID)))
		return nil, nil
	}

	p, err := t.providerStore.GetProviderByID(ctx, llmModel.ProviderID())
	if err != nil {
		slog.ErrorContext(ctx, "usage tracker: could not load provider", slog.Any("error", err), slog.String("providerID", string(llmModel.ProviderID())))
		return nil, nil
	}

	promptTokens := res.TokensUsed.PromptTokens
	completionTokens := res.TokensUsed.CompletionTokens

	metrics.ChatCompletionRequests.With(prometheus.Labels{
		metrics.LabelOrg: string(orgID),
	}).Inc()

	metrics.CompletionTokens.With(prometheus.Labels{
		metrics.LabelOrg: string(orgID),
	}).Add(float64(completionTokens))

	metrics.PromptTokens.With(prometheus.Labels{
		metrics.LabelOrg: string(orgID),
	}).Add(float64(promptTokens))

	providerCost := (int64(promptTokens) * llmModel.PromptCostPer1KTokens() / 1000) +
		(int64(completionTokens) * llmModel.CompletionCostPer1KTokens() / 1000)

	providerCurrency := p.Currency()
	recordCost := providerCost
	recordCurrency := providerCurrency

	// Convert to org's base currency
	org, err := t.orgStore.GetOrgByID(ctx, orgID)
	if err != nil {
		slog.WarnContext(ctx, "usage tracker: could not load org, using provider currency", slog.Any("error", errors.WithStack(err)), slog.String("orgID", string(orgID)))
	} else {
		orgCurrency := org.Currency()
		if orgCurrency == "" {
			orgCurrency = model.DefaultCurrency
		}
		converted, convertErr := t.exchangeRateService.Convert(ctx, providerCost, providerCurrency, orgCurrency)
		if convertErr != nil {
			slog.WarnContext(ctx, "usage tracker: currency conversion failed, using provider currency", slog.Any("error", convertErr))
		} else {
			recordCost = converted
			recordCurrency = orgCurrency
		}
	}

	record := model.NewUsageRecord(
		model.UserID(req.UserID),
		applicationID,
		orgID,
		llmModel.ProviderID(),
		modelID,
		originalModel,
		authTokenID,
		promptTokens,
		completionTokens,
		recordCost,
		recordCurrency,
		resolvedModel,
	)

	if err := t.usageStore.RecordUsage(ctx, record); err != nil {
		slog.ErrorContext(ctx, "usage tracker: could not record usage", slog.Any("error", errors.WithStack(err)))
	}

	return nil, nil
}

var _ genaiProxy.PostResponseHook = &XoloUsageTracker{}
