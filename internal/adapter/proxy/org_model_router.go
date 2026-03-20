package proxy

import (
	"context"
	"log/slog"
	"reflect"
	"strings"

	"github.com/bornholm/genai/llm"
	"github.com/bornholm/genai/llm/provider"
	llmratelimit "github.com/bornholm/genai/llm/ratelimit"
	llmretry "github.com/bornholm/genai/llm/retry"
	"github.com/bornholm/genai/llm/tokenlimit"
	genaiProxy "github.com/bornholm/genai/proxy"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/bornholm/xolo/internal/crypto"
	"github.com/pkg/errors"

	_ "github.com/bornholm/genai/llm/provider/mistral"
	_ "github.com/bornholm/genai/llm/provider/openai"
	_ "github.com/bornholm/genai/llm/provider/openrouter"
)

// OrgModelRouter resolves the requested model name to an LLM client.
// Clients must use the qualified format "<org-slug>/<model-name>" to avoid
// ambiguity between models belonging to different organizations or having the
// same local name within an organization.
type OrgModelRouter struct {
	providerStore port.ProviderStore
	orgStore      port.OrgStore
	secretKey     string
}

func NewOrgModelRouter(providerStore port.ProviderStore, orgStore port.OrgStore, secretKey string) *OrgModelRouter {
	return &OrgModelRouter{
		providerStore: providerStore,
		orgStore:      orgStore,
		secretKey:     secretKey,
	}
}

func (r *OrgModelRouter) Name() string  { return "xolo.org-model-router" }
func (r *OrgModelRouter) Priority() int { return 10 }

// ResolveModel implements proxy.ModelResolverHook.
// The request model field must be in the format "<org-slug>/<model-name>".
func (r *OrgModelRouter) ResolveModel(ctx context.Context, req *genaiProxy.ProxyRequest) (llm.Client, string, error) {
	PopulateMetaFromContext(ctx, req.Metadata)

	tokenOrgID := OrgIDFromMeta(req.Metadata)
	if tokenOrgID == "" {
		return nil, "", errors.New("no org ID in request context; use an API key scoped to an organization")
	}

	orgSlug, proxyName, err := parseQualifiedModelName(req.Model)
	if err != nil {
		return nil, "", errors.Errorf("invalid model name %q: expected format \"<org-slug>/<model-name>\"", req.Model)
	}

	org, err := r.orgStore.GetOrgBySlug(ctx, orgSlug)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, "", errors.Errorf("model '%s' not available in your organization", req.Model)
		}
		return nil, "", errors.WithStack(err)
	}

	// The token must be scoped to the same organization as the requested model.
	if org.ID() != tokenOrgID {
		return nil, "", errors.Errorf("model '%s' not available in your organization", req.Model)
	}

	llmModel, err := r.providerStore.GetLLMModelByProxyName(ctx, org.ID(), proxyName)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, "", errors.Errorf("model '%s' not available in your organization", req.Model)
		}
		return nil, "", errors.WithStack(err)
	}

	// Store model ID in metadata for UsageTracker
	req.Metadata[MetaModelID] = string(llmModel.ID())

	p, err := r.providerStore.GetProviderByID(ctx, llmModel.ProviderID())
	if err != nil {
		return nil, "", errors.WithStack(err)
	}

	if !p.Active() {
		return nil, "", errors.Errorf("provider '%s' is not active", p.Name())
	}

	// Decrypt the API key
	decryptedKey, err := crypto.Decrypt(r.secretKey, p.APIKey())
	if err != nil {
		slog.ErrorContext(ctx, "could not decrypt provider API key", slog.Any("error", err))
		return nil, "", errors.New("provider configuration error")
	}

	client, err := provider.Create(ctx,
		withDynamicChatCompletion(provider.Name(p.Type()), p.BaseURL(), decryptedKey, llmModel.RealModel()),
	)
	if err != nil {
		return nil, "", errors.Wrapf(err, "could not create LLM client for provider '%s'", p.Name())
	}

	// Token limit (innermost — applied first, closest to the provider).
	// Always pass WithChatCompletionLimit explicitly to avoid the 500k TPM
	// silent default from tokenlimit.NewClient called bare.
	if cfg := llmModel.TokenLimitConfig(); cfg != nil && cfg.Enabled {
		client = tokenlimit.NewClient(client,
			tokenlimit.WithChatCompletionLimit(cfg.MaxTokens, cfg.Interval),
		)
	}

	// Rate limit wraps token limit: token-bucket (minInterval between requests, burst = MaxBurst).
	if cfg := p.RateLimitConfig(); cfg != nil && cfg.Enabled {
		client = llmratelimit.NewClient(client, cfg.Interval, cfg.MaxBurst)
	}

	// Retry wraps everything (outermost): each retry attempt goes through
	// rate-limit and token-limit, preventing burst hammering during retries.
	if cfg := p.RetryConfig(); cfg != nil && cfg.Enabled {
		client = llmretry.NewClient(client, cfg.Delay, cfg.MaxAttempts)
	}

	return client, llmModel.RealModel(), nil
}

// ListModels implements proxy.ModelListerHook.
// Returns models in the qualified "<org-slug>/<model-name>" format.
func (r *OrgModelRouter) ListModels(ctx context.Context) ([]genaiProxy.ModelInfo, error) {
	orgID := OrgIDFromContext(ctx)
	if orgID == "" {
		return nil, nil
	}

	org, err := r.orgStore.GetOrgByID(ctx, model.OrgID(orgID))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	models, err := r.providerStore.ListEnabledLLMModels(ctx, model.OrgID(orgID))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	infos := make([]genaiProxy.ModelInfo, 0, len(models))
	for _, m := range models {
		infos = append(infos, genaiProxy.ModelInfo{
			ID:      org.Slug() + "/" + m.ProxyName(),
			OwnedBy: "xolo",
		})
	}
	return infos, nil
}

// parseQualifiedModelName splits "org-slug/model-name" into its two parts.
func parseQualifiedModelName(name string) (orgSlug, proxyName string, err error) {
	idx := strings.IndexByte(name, '/')
	if idx <= 0 || idx == len(name)-1 {
		return "", "", errors.New("expected format \"<org-slug>/<model-name>\"")
	}
	return name[:idx], name[idx+1:], nil
}

var _ genaiProxy.ModelResolverHook = &OrgModelRouter{}
var _ genaiProxy.ModelListerHook = &OrgModelRouter{}

// withDynamicChatCompletion crée une OptionFunc pour un provider identifié à runtime.
// Tous les providers enregistrés embarquent provider.CommonOptions, on utilise
// la réflexion pour définir BaseURL, APIKey et Model sans type connu à la compilation.
func withDynamicChatCompletion(name provider.Name, baseURL, apiKey, model string) provider.OptionFunc {
	return func(o *provider.Options) error {
		opts := provider.NewChatCompletionProviderOptions(name)
		if opts == nil {
			return errors.Errorf("unknown provider %q", name)
		}
		v := reflect.ValueOf(opts).Elem()
		if common := v.FieldByName("CommonOptions"); common.IsValid() {
			common.FieldByName("BaseURL").SetString(baseURL)
			common.FieldByName("APIKey").SetString(apiKey)
			common.FieldByName("Model").SetString(model)
		}
		o.ChatCompletion = &provider.ResolvedClientOptions{
			Provider: name,
			Specific: opts,
		}
		return nil
	}
}
