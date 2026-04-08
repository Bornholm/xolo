package proxy

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/bornholm/genai/llm"
	genaiProxy "github.com/bornholm/genai/proxy"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
	"github.com/pkg/errors"
)

// PluginHookAdapter bridges the plugin system into the genai proxy hook chain.
// It implements PreRequestHook, PostResponseHook, and ModelListerHook (which
// embeds ModelResolverHook).
type PluginHookAdapter struct {
	clients           map[string]proto.XoloPluginClient
	descriptors       map[string]*proto.PluginDescriptor
	activationStore   port.PluginActivationStore
	configStore       port.PluginConfigStore
	userStore         tokenFinder
	providerStore     port.ProviderStore
	virtualModelStore port.VirtualModelStore
	quotaResolver     quotaResolver
	usageStore        port.UsageStore
}

// NewPluginHookAdapter creates a PluginHookAdapter wired to the given plugin clients and stores.
func NewPluginHookAdapter(
	clients map[string]proto.XoloPluginClient,
	descriptors map[string]*proto.PluginDescriptor,
	activationStore port.PluginActivationStore,
	configStore port.PluginConfigStore,
	userStore tokenFinder,
	providerStore port.ProviderStore,
	virtualModelStore port.VirtualModelStore,
	quotaResolver quotaResolver,
	usageStore port.UsageStore,
) *PluginHookAdapter {
	return &PluginHookAdapter{
		clients:           clients,
		descriptors:       descriptors,
		activationStore:   activationStore,
		configStore:       configStore,
		userStore:         userStore,
		providerStore:     providerStore,
		virtualModelStore: virtualModelStore,
		quotaResolver:     quotaResolver,
		usageStore:        usageStore,
	}
}

// hasCapability reports whether the named plugin declared the given capability in its descriptor.
func (a *PluginHookAdapter) hasCapability(pluginName string, cap proto.PluginDescriptor_Capability) bool {
	desc, ok := a.descriptors[pluginName]
	if !ok {
		return false
	}
	for _, c := range desc.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

func (a *PluginHookAdapter) Name() string  { return "xolo.plugin-hook-adapter" }
func (a *PluginHookAdapter) Priority() int { return 3 }

// userGetter is an optional extension of tokenFinder that allows fetching a user by ID.
// port.UserStore satisfies this interface.
type userGetter interface {
	GetUserByID(ctx context.Context, userID model.UserID) (model.User, error)
}

// buildRequestContext loads org and user config for a plugin and assembles a RequestContext.
func (a *PluginHookAdapter) buildRequestContext(ctx context.Context, orgID model.OrgID, userID string, tokenID string, pluginName string) (*proto.RequestContext, error) {
	reqCtx := &proto.RequestContext{
		OrgId:   string(orgID),
		UserId:  userID,
		TokenId: tokenID,
	}

	orgCfg, err := a.configStore.GetConfig(ctx, orgID, pluginName, model.PluginConfigScopeOrg, string(orgID))
	if err != nil && !errors.Is(err, port.ErrNotFound) {
		return nil, errors.WithStack(err)
	}
	if orgCfg != nil {
		reqCtx.ConfigJson = orgCfg.ConfigJSON
	}

	if userID != "" {
		userCfg, err := a.configStore.GetConfig(ctx, orgID, pluginName, model.PluginConfigScopeUser, userID)
		if err != nil && !errors.Is(err, port.ErrNotFound) {
			return nil, errors.WithStack(err)
		}
		if userCfg != nil {
			reqCtx.UserConfigJson = userCfg.ConfigJSON
		}

		if ug, ok := a.userStore.(userGetter); ok {
			if u, err := ug.GetUserByID(ctx, model.UserID(userID)); err == nil {
				reqCtx.DisplayName = u.DisplayName()
			}
		}
	}

	return reqCtx, nil
}

// PreRequest implements proxy.PreRequestHook.
func (a *PluginHookAdapter) PreRequest(ctx context.Context, req *genaiProxy.ProxyRequest) (*genaiProxy.HookResult, error) {
	populateMetaFromHeader(ctx, a.userStore, req)

	orgID := OrgIDFromMeta(req.Metadata)
	if orgID == "" {
		return nil, nil
	}

	activations, err := a.activationStore.ListActivations(ctx, orgID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	for _, act := range activations {
		if !act.Enabled {
			continue
		}
		if !a.hasCapability(act.PluginName, proto.PluginDescriptor_PRE_REQUEST) {
			continue
		}
		client, ok := a.clients[act.PluginName]
		if !ok {
			continue
		}

		tokenID := AuthTokenIDFromMeta(req.Metadata)
		reqCtx, err := a.buildRequestContext(ctx, orgID, req.UserID, tokenID, act.PluginName)
		if err != nil {
			slog.WarnContext(ctx, "plugin_hook_adapter: failed to build request context",
				slog.String("plugin", act.PluginName),
				slog.Any("error", err),
			)
			if act.Required {
				return &genaiProxy.HookResult{Response: serviceUnavailableResponse(act.PluginName)}, nil
			}
			continue
		}

		var messagesJSON string
		if len(req.Body) > 0 {
			var bodyMessages struct {
				Messages json.RawMessage `json:"messages"`
			}
			if err := json.Unmarshal(req.Body, &bodyMessages); err == nil && len(bodyMessages.Messages) > 0 {
				messagesJSON = string(bodyMessages.Messages)
			}
		}

		out, err := client.PreRequest(ctx, &proto.PreRequestInput{
			Ctx:          reqCtx,
			Model:        req.Model,
			MessagesJson: messagesJSON,
		})
		if err != nil {
			slog.WarnContext(ctx, "plugin_hook_adapter: PreRequest gRPC error",
				slog.String("plugin", act.PluginName),
				slog.Any("error", err),
			)
			if act.Required {
				return &genaiProxy.HookResult{Response: serviceUnavailableResponse(act.PluginName)}, nil
			}
			continue
		}

		if out.ResponseJson != "" {
			var body map[string]any
			if err := json.Unmarshal([]byte(out.ResponseJson), &body); err == nil {
				return &genaiProxy.HookResult{
					Response: &genaiProxy.ProxyResponse{
						StatusCode: 200,
						Body:       body,
					},
				}, nil
			}
			slog.WarnContext(ctx, "plugin_hook_adapter: failed to parse response_json from plugin",
				slog.String("plugin", act.PluginName),
			)
		}

		if !out.Allowed {
			return &genaiProxy.HookResult{Response: forbiddenResponse(out.RejectionReason)}, nil
		}

		if out.ModifiedMessagesJson != "" {
			if err := applyModifiedMessages(req, out.ModifiedMessagesJson); err != nil {
				slog.WarnContext(ctx, "plugin_hook_adapter: failed to apply modified messages",
					slog.String("plugin", act.PluginName),
					slog.Any("error", err),
				)
				if act.Required {
					return &genaiProxy.HookResult{Response: serviceUnavailableResponse(act.PluginName)}, nil
				}
				continue
			}
			slog.DebugContext(ctx, "plugin_hook_adapter: applied modified messages",
				slog.String("plugin", act.PluginName),
			)
		}
	}

	return nil, nil
}

// PostResponse implements proxy.PostResponseHook.
func (a *PluginHookAdapter) PostResponse(ctx context.Context, req *genaiProxy.ProxyRequest, res *genaiProxy.ProxyResponse) (*genaiProxy.HookResult, error) {
	orgID := OrgIDFromMeta(req.Metadata)
	if orgID == "" {
		orgID = model.OrgID(OrgIDFromContext(ctx))
	}
	if orgID == "" {
		return nil, nil
	}

	activations, err := a.activationStore.ListActivations(ctx, orgID)
	if err != nil {
		slog.WarnContext(ctx, "plugin_hook_adapter: failed to list activations in PostResponse",
			slog.Any("error", err),
		)
		// PostResponse is best-effort; a store failure must not block the already-completed response.
		return nil, nil
	}

	var promptTokens, completionTokens int64
	hadError := res.StatusCode >= 400
	if res.TokensUsed != nil {
		promptTokens = int64(res.TokensUsed.PromptTokens)
		completionTokens = int64(res.TokensUsed.CompletionTokens)
	}

	for _, act := range activations {
		if !act.Enabled {
			continue
		}
		if !a.hasCapability(act.PluginName, proto.PluginDescriptor_POST_RESPONSE) {
			continue
		}
		client, ok := a.clients[act.PluginName]
		if !ok {
			continue
		}

		tokenID := AuthTokenIDFromMeta(req.Metadata)
		reqCtx, err := a.buildRequestContext(ctx, orgID, req.UserID, tokenID, act.PluginName)
		if err != nil {
			slog.WarnContext(ctx, "plugin_hook_adapter: failed to build request context for PostResponse",
				slog.String("plugin", act.PluginName),
				slog.Any("error", err),
			)
			continue
		}

		_, err = client.PostResponse(ctx, &proto.PostResponseInput{
			Ctx:              reqCtx,
			Model:            req.Model,
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			HadError:         hadError,
		})
		if err != nil {
			slog.WarnContext(ctx, "plugin_hook_adapter: PostResponse gRPC error",
				slog.String("plugin", act.PluginName),
				slog.Any("error", err),
			)
		}
	}

	return nil, nil
}

// ResolveModel implements proxy.ModelResolverHook.
func (a *PluginHookAdapter) ResolveModel(ctx context.Context, req *genaiProxy.ProxyRequest) (llm.Client, string, error) {
	orgID := OrgIDFromMeta(req.Metadata)
	if orgID == "" {
		return nil, "", genaiProxy.ErrModelNotFound
	}

	activations, err := a.activationStore.ListActivations(ctx, orgID)
	if err != nil {
		return nil, "", errors.WithStack(err)
	}

	// Build available models list once for all plugins.
	models, err := a.providerStore.ListEnabledLLMModels(ctx, orgID)
	if err != nil {
		slog.WarnContext(ctx, "plugin_hook_adapter: failed to list models for ResolveModel",
			slog.Any("error", err),
		)
		models = nil
	}
	protoModels := make([]*proto.ModelInfo, 0, len(models))
	for _, m := range models {
		caps := m.Capabilities()
		protoModels = append(protoModels, &proto.ModelInfo{
			ProxyName:                  m.ProxyName(),
			RealModel:                  m.RealModel(),
			ProviderId:                 string(m.ProviderID()),
			PromptCostPer_1KTokens:     float64(m.PromptCostPer1KTokens()),
			CompletionCostPer_1KTokens: float64(m.CompletionCostPer1KTokens()),
			TokenLimit:                 m.ContextWindow(),
			IsVirtual:                  m.IsVirtual(),
			ContextLength:              m.ContextWindow(),
			SupportsVision:             caps.Vision,
			SupportsReasoning:          caps.Reasoning,
			ActiveParamsBillions:       float32(m.ActiveParams()) / 1e9,
		})
	}

	// Get virtual models for this org.
	var virtualModels []model.VirtualModel
	if a.virtualModelStore != nil {
		virtualModels, _ = a.virtualModelStore.ListVirtualModels(ctx, orgID)
	}
	protoVirtualModels := make([]*proto.VirtualModelInfo, 0, len(virtualModels))
	for _, vm := range virtualModels {
		protoVirtualModels = append(protoVirtualModels, &proto.VirtualModelInfo{
			Id:          string(vm.ID()),
			Name:        vm.Name(),
			OrgId:       string(vm.OrgID()),
			Description: vm.Description(),
		})
	}

	// Compute quota info for the requesting user/org.
	protoQuota := a.buildQuotaInfo(ctx, model.UserID(req.UserID), orgID)

	// Extract messages JSON from raw request body (chat completions only).
	var messagesJSON string
	if len(req.Body) > 0 {
		var bodyMessages struct {
			Messages json.RawMessage `json:"messages"`
		}
		if err := json.Unmarshal(req.Body, &bodyMessages); err == nil && len(bodyMessages.Messages) > 0 {
			messagesJSON = string(bodyMessages.Messages)
		}
	}

	for _, act := range activations {
		if !act.Enabled {
			continue
		}
		if !a.hasCapability(act.PluginName, proto.PluginDescriptor_RESOLVE_MODEL) {
			continue
		}
		client, ok := a.clients[act.PluginName]
		if !ok {
			continue
		}

		tokenID := AuthTokenIDFromMeta(req.Metadata)
		reqCtx, err := a.buildRequestContext(ctx, orgID, req.UserID, tokenID, act.PluginName)
		if err != nil {
			slog.WarnContext(ctx, "plugin_hook_adapter: failed to build request context for ResolveModel",
				slog.String("plugin", act.PluginName),
				slog.Any("error", err),
			)
			continue
		}

		out, err := client.ResolveModel(ctx, &proto.ResolveModelInput{
			Ctx:             reqCtx,
			RequestedModel:  req.Model,
			AvailableModels: protoModels,
			MessagesJson:    messagesJSON,
			VirtualModels:   protoVirtualModels,
			Quota:           protoQuota,
			BodyJson:        string(req.Body),
		})
		if err != nil {
			slog.WarnContext(ctx, "plugin_hook_adapter: ResolveModel gRPC error",
				slog.String("plugin", act.PluginName),
				slog.Any("error", err),
			)
			continue
		}

		if out.ResponseContent != "" {
			req.Metadata[MetaOriginalModel] = req.Model
			req.Metadata[MetaResolvedModel] = req.Model
			return NewDummyLLMClient(out.ResponseContent, req.Model), req.Model, nil
		}

		if out.ResolvedProxyName != "" {
			originalModel := req.Model
			// Qualify the resolved name with the org slug if the original request
			// used the qualified format ("org-slug/model-name") and the resolved
			// name is local (no "/"). OrgModelRouter requires the qualified format.
			resolved := out.ResolvedProxyName
			if idx := strings.IndexByte(req.Model, '/'); idx > 0 && !strings.Contains(resolved, "/") {
				resolved = req.Model[:idx] + "/" + resolved
			}
			req.Model = resolved
			req.Metadata[MetaOriginalModel] = originalModel
			req.Metadata[MetaResolvedModel] = resolved
			return nil, "", genaiProxy.ErrModelNotFound
		}
	}

	// No plugin resolved the model - store original as both
	req.Metadata[MetaOriginalModel] = req.Model
	req.Metadata[MetaResolvedModel] = req.Model

	return nil, "", genaiProxy.ErrModelNotFound
}

// ListModels implements proxy.ModelListerHook.
func (a *PluginHookAdapter) ListModels(ctx context.Context) ([]genaiProxy.ModelInfo, error) {
	orgIDStr := OrgIDFromContext(ctx)
	if orgIDStr == "" {
		return nil, nil
	}
	orgID := model.OrgID(orgIDStr)

	activations, err := a.activationStore.ListActivations(ctx, orgID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var result []genaiProxy.ModelInfo

	for _, act := range activations {
		if !act.Enabled {
			continue
		}
		if !a.hasCapability(act.PluginName, proto.PluginDescriptor_LIST_MODELS) {
			continue
		}
		client, ok := a.clients[act.PluginName]
		if !ok {
			continue
		}

		reqCtx := &proto.RequestContext{
			OrgId: string(orgID),
		}

		out, err := client.ListModels(ctx, &proto.ListModelsInput{
			Ctx: reqCtx,
		})
		if err != nil {
			slog.WarnContext(ctx, "plugin_hook_adapter: ListModels gRPC error",
				slog.String("plugin", act.PluginName),
				slog.Any("error", err),
			)
			continue
		}

		for _, name := range out.AdditionalProxyNames {
			result = append(result, genaiProxy.ModelInfo{ID: name})
		}
	}

	return result, nil
}

// buildQuotaInfo computes remaining budget for the user/org and returns a QuotaInfo proto.
// Returns nil if quota stores are not configured.
func (a *PluginHookAdapter) buildQuotaInfo(ctx context.Context, userID model.UserID, orgID model.OrgID) *proto.QuotaInfo {
	if a.quotaResolver == nil || a.usageStore == nil {
		return nil
	}
	effectiveQuota, err := a.quotaResolver.ResolveEffectiveQuota(ctx, userID, orgID)
	if err != nil {
		slog.WarnContext(ctx, "plugin_hook_adapter: failed to resolve effective quota for QuotaInfo",
			slog.Any("error", err),
		)
		return nil
	}

	now := time.Now()
	info := &proto.QuotaInfo{}

	if effectiveQuota.DailyBudget != nil {
		total := float64(*effectiveQuota.DailyBudget)
		spent, err := a.usageStore.SumCostSince(ctx, userID, orgID, startOfDay(now))
		if err == nil {
			info.DailyTotal = total
			info.DailyRemaining = max(0, total-float64(spent))
		}
	}
	if effectiveQuota.MonthlyBudget != nil {
		total := float64(*effectiveQuota.MonthlyBudget)
		spent, err := a.usageStore.SumCostSince(ctx, userID, orgID, startOfMonth(now))
		if err == nil {
			info.MonthlyTotal = total
			info.MonthlyRemaining = max(0, total-float64(spent))
		}
	}
	if effectiveQuota.YearlyBudget != nil {
		total := float64(*effectiveQuota.YearlyBudget)
		spent, err := a.usageStore.SumCostSince(ctx, userID, orgID, startOfYear(now))
		if err == nil {
			info.YearlyTotal = total
			info.YearlyRemaining = max(0, total-float64(spent))
		}
	}

	return info
}

func forbiddenResponse(reason string) *genaiProxy.ProxyResponse {
	msg := "Request blocked by plugin"
	if reason != "" {
		msg = reason
	}
	return &genaiProxy.ProxyResponse{
		StatusCode: 403,
		Body: map[string]any{
			"error": map[string]any{
				"message": msg,
				"type":    "permission_error",
				"code":    "plugin_blocked",
			},
		},
	}
}

func serviceUnavailableResponse(pluginName string) *genaiProxy.ProxyResponse {
	return &genaiProxy.ProxyResponse{
		StatusCode: 503,
		Body: map[string]any{
			"error": map[string]any{
				"message": "Plugin " + pluginName + " is temporarily unavailable",
				"type":    "service_unavailable",
				"code":    "plugin_unavailable",
			},
		},
	}
}

// applyModifiedMessages replaces the messages in the request body with the
// modified messages provided by the plugin. It updates req.Body in place.
func applyModifiedMessages(req *genaiProxy.ProxyRequest, modifiedMessagesJSON string) error {
	if len(req.Body) == 0 {
		return nil
	}

	var body map[string]any
	if err := json.Unmarshal(req.Body, &body); err != nil {
		return errors.Wrap(err, "unmarshal request body")
	}

	body["messages"] = json.RawMessage(modifiedMessagesJSON)

	updatedBody, err := json.Marshal(body)
	if err != nil {
		return errors.Wrap(err, "marshal updated body")
	}

	req.Body = updatedBody
	return nil
}

// Compile-time interface assertions.
var _ genaiProxy.PreRequestHook = &PluginHookAdapter{}
var _ genaiProxy.PostResponseHook = &PluginHookAdapter{}
var _ genaiProxy.ModelListerHook = &PluginHookAdapter{}
