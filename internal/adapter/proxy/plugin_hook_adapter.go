package proxy

import (
	"context"
	"encoding/json"
	"log/slog"

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
	clients         map[string]proto.XoloPluginClient
	descriptors     map[string]*proto.PluginDescriptor
	activationStore port.PluginActivationStore
	configStore     port.PluginConfigStore
	userStore       tokenFinder
	providerStore   port.ProviderStore
}

// NewPluginHookAdapter creates a PluginHookAdapter wired to the given plugin clients and stores.
func NewPluginHookAdapter(
	clients map[string]proto.XoloPluginClient,
	descriptors map[string]*proto.PluginDescriptor,
	activationStore port.PluginActivationStore,
	configStore port.PluginConfigStore,
	userStore tokenFinder,
	providerStore port.ProviderStore,
) *PluginHookAdapter {
	return &PluginHookAdapter{
		clients:         clients,
		descriptors:     descriptors,
		activationStore: activationStore,
		configStore:     configStore,
		userStore:       userStore,
		providerStore:   providerStore,
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

		out, err := client.PreRequest(ctx, &proto.PreRequestInput{
			Ctx:   reqCtx,
			Model: req.Model,
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

		if !out.Allowed {
			return &genaiProxy.HookResult{Response: forbiddenResponse(out.RejectionReason)}, nil
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
		protoModels = append(protoModels, &proto.ModelInfo{
			ProxyName:                  m.ProxyName(),
			RealModel:                  m.RealModel(),
			ProviderId:                 string(m.ProviderID()),
			PromptCostPer_1KTokens:     float64(m.PromptCostPer1KTokens()),
			CompletionCostPer_1KTokens: float64(m.CompletionCostPer1KTokens()),
			TokenLimit:                 m.ContextWindow(),
		})
	}

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
		})
		if err != nil {
			slog.WarnContext(ctx, "plugin_hook_adapter: ResolveModel gRPC error",
				slog.String("plugin", act.PluginName),
				slog.Any("error", err),
			)
			continue
		}

		if out.ResolvedProxyName != "" {
			req.Model = out.ResolvedProxyName
			return nil, "", genaiProxy.ErrModelNotFound
		}
	}

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

// Compile-time interface assertions.
var _ genaiProxy.PreRequestHook = &PluginHookAdapter{}
var _ genaiProxy.PostResponseHook = &PluginHookAdapter{}
var _ genaiProxy.ModelListerHook = &PluginHookAdapter{}
