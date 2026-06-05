package proxy

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bornholm/genai/llm"
	genaiProxy "github.com/bornholm/genai/proxy"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	"github.com/bornholm/xolo/internal/pipeline"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
	"github.com/pkg/errors"
)

const metaPipelineExecution = "pipeline.execution"

// PipelineHookAdapter bridges the pipeline engine into the genai proxy hook chain.
// It implements PreRequestHook (runs the pipeline forward pass, handles rejection)
// and ModelListerHook (returns the pre-resolved client to the proxy chain).
type PipelineHookAdapter struct {
	engine            *pipeline.Engine
	virtualModelStore port.VirtualModelStore
	personalVMStore   port.PersonalVirtualModelStore
	orgStore          port.OrgStore
	providerStore     port.ProviderStore
}

// NewPipelineHookAdapter creates a PipelineHookAdapter and wires the pipeline engine.
func NewPipelineHookAdapter(
	pluginProvider pipeline.PluginProvider,
	virtualModelStore port.VirtualModelStore,
	personalVMStore port.PersonalVirtualModelStore,
	providerStore port.ProviderStore,
	orgStore port.OrgStore,
	orgModelRouter *OrgModelRouter,
) *PipelineHookAdapter {
	reg := pipeline.NewRegistry()
	eng := pipeline.NewEngine(reg)

	reg.Register(model.NodeTypeGenerator, pipeline.NewGeneratorExecutor())
	reg.Register(model.NodeTypeSink, pipeline.NewSinkExecutor())
	reg.Register(model.NodeTypeValue, pipeline.NewValueExecutor())
	reg.Register(model.NodeTypePlugin, pipeline.NewPluginExecutor(pluginProvider))
	// ModelExecutor needs the engine for recursive VirtualModel resolution.
	reg.Register(model.NodeTypeModel, pipeline.NewModelExecutor(orgModelRouter, virtualModelStore, eng))

	return &PipelineHookAdapter{
		engine:            eng,
		virtualModelStore: virtualModelStore,
		personalVMStore:   personalVMStore,
		orgStore:          orgStore,
		providerStore:     providerStore,
	}
}

func (a *PipelineHookAdapter) Name() string  { return "pipeline" }
func (a *PipelineHookAdapter) Priority() int { return 3 }

// PreRequest runs the full pipeline forward pass for virtual models.
// Rejection results in a 403 response. A successful execution stores the
// ForwardExecution in req.Metadata for ResolveModel to pick up.
func (a *PipelineHookAdapter) PreRequest(ctx context.Context, req *genaiProxy.ProxyRequest) (*genaiProxy.HookResult, error) {
	PopulateMetaFromContext(ctx, req.Metadata)

	org, vm, lookupErr := a.lookupVirtualModel(ctx, req.Model)
	if lookupErr != nil || vm == nil {
		// Not a virtual model — pass through to OrgModelRouter.
		slog.DebugContext(ctx, "pipeline: not a virtual model, delegating to OrgModelRouter",
			slog.String("model", req.Model))
		return nil, nil
	}

	slog.DebugContext(ctx, "pipeline: virtual model found",
		slog.String("model", req.Model),
		slog.String("vmID", string(vm.ID())))

	if vm.Graph() == nil {
		slog.WarnContext(ctx, "pipeline: virtual model has no pipeline configured — returning error",
			slog.String("model", req.Model))
		return &genaiProxy.HookResult{
			Response: &genaiProxy.ProxyResponse{
				StatusCode: http.StatusUnprocessableEntity,
				Body: map[string]any{"error": map[string]any{
					"type":    "invalid_request_error",
					"message": "Virtual model \"" + req.Model + "\" has no pipeline configured.",
					"code":    "pipeline_not_configured",
				}},
			},
		}, nil
	}

	ec := a.buildEC(ctx, req, org, vm)
	forwardExec, err := a.engine.RunForward(ctx, vm.Graph(), ec)
	if err != nil {
		var rejErr *pipeline.RejectionError
		if errors.As(err, &rejErr) {
			slog.InfoContext(ctx, "pipeline: request rejected by node",
				slog.String("model", req.Model),
				slog.String("reason", rejErr.Reason))
			return &genaiProxy.HookResult{
				Response: &genaiProxy.ProxyResponse{
					StatusCode: http.StatusForbidden,
					Body:       map[string]any{"error": rejErr.Error()},
				},
			}, nil
		}
		// All pipeline errors are surfaced as API responses — never as Go errors,
		// because the genai proxy's RunOnError swallows errors and continues silently.
		slog.ErrorContext(ctx, "pipeline: forward pass failed",
			slog.String("model", req.Model),
			slog.Any("error", err))
		return &genaiProxy.HookResult{
			Response: &genaiProxy.ProxyResponse{
				StatusCode: http.StatusInternalServerError,
				Body: map[string]any{"error": map[string]any{
					"type":    "server_error",
					"message": "Pipeline execution failed for \"" + req.Model + "\": " + err.Error(),
					"code":    "pipeline_error",
				}},
			},
		}, nil
	}

	if forwardExec.ResolvedClient == nil {
		slog.ErrorContext(ctx, "pipeline: forward pass completed but no LLM client resolved — pipeline has no terminal node",
			slog.String("model", req.Model))
		return &genaiProxy.HookResult{
			Response: &genaiProxy.ProxyResponse{
				StatusCode: http.StatusUnprocessableEntity,
				Body: map[string]any{"error": map[string]any{
					"type":    "invalid_request_error",
					"message": "Pipeline for model \"" + req.Model + "\" has no terminal node (LLM model or dummy-model).",
					"code":    "pipeline_no_terminal",
				}},
			},
		}, nil
	}

	slog.DebugContext(ctx, "pipeline: forward pass succeeded",
		slog.String("model", req.Model),
		slog.String("resolvedModel", forwardExec.ResolvedModel))

	// Store the execution result for ResolveModel.
	req.Metadata[metaPipelineExecution] = forwardExec
	return nil, nil
}

// ResolveModel returns the pre-resolved llm.Client from the pipeline execution.
// If no pipeline execution was stored (non-virtual model), it returns ErrModelNotFound
// so the OrgModelRouter can handle it.
func (a *PipelineHookAdapter) ResolveModel(ctx context.Context, req *genaiProxy.ProxyRequest) (llm.Client, string, error) {
	execAny, ok := req.Metadata[metaPipelineExecution]
	if !ok {
		return nil, "", genaiProxy.ErrModelNotFound
	}

	forwardExec, ok := execAny.(*pipeline.ForwardExecution)
	if !ok || forwardExec == nil || forwardExec.ResolvedClient == nil {
		return nil, "", genaiProxy.ErrModelNotFound
	}

	// Retrieve ec from metadata if possible (stored by PreRequest via closue capture).
	ec, _ := req.Metadata[metaPipelineExecution+".ec"].(pipeline.ExecutionContext)

	// Populate metadata for UsageTracker and QuotaEnforcer.
	if forwardExec.ResolvedModelID != "" {
		req.Metadata[MetaModelID] = string(forwardExec.ResolvedModelID)
	}
	req.Metadata[MetaOriginalModel] = req.Model
	req.Metadata[MetaResolvedModel] = forwardExec.ResolvedModel

	client := NewPipelineWrappedClient(forwardExec.ResolvedClient, a.engine, forwardExec, ec)
	return client, forwardExec.ResolvedModel, nil
}

// ListModels lists available virtual models for the org and the current user's personal VMs.
func (a *PipelineHookAdapter) ListModels(ctx context.Context) ([]genaiProxy.ModelInfo, error) {
	var infos []genaiProxy.ModelInfo

	// Org virtual models
	orgID := OrgIDFromContext(ctx)
	if orgID != "" {
		org, err := a.orgStore.GetOrgByID(ctx, model.OrgID(orgID))
		if err == nil {
			vms, err := a.virtualModelStore.ListVirtualModels(ctx, model.OrgID(orgID))
			if err != nil {
				return nil, errors.WithStack(err)
			}
			for _, vm := range vms {
				infos = append(infos, genaiProxy.ModelInfo{
					ID:      org.Slug() + "/" + vm.Name(),
					OwnedBy: "xolo",
				})
			}
		}
	}

	// Personal virtual models
	if a.personalVMStore != nil {
		if u := httpCtx.User(ctx); u != nil {
			pvms, err := a.personalVMStore.ListPersonalVirtualModels(ctx, u.ID())
			if err != nil {
				slog.WarnContext(ctx, "pipeline: could not list personal virtual models", slog.Any("error", err))
			} else {
				for _, pvm := range pvms {
					infos = append(infos, genaiProxy.ModelInfo{
						ID:      "~/" + pvm.Name(),
						OwnedBy: "xolo",
					})
				}
			}
		}
	}

	return infos, nil
}

// lookupVirtualModel resolves the org and virtual model from a qualified model name.
// For personal virtual models ("~/name"), org is nil and the VM is user-scoped.
func (a *PipelineHookAdapter) lookupVirtualModel(ctx context.Context, modelName string) (model.Organization, model.VirtualModel, error) {
	orgSlug, localName, ok := splitQualifiedName(modelName)
	if !ok {
		return nil, nil, nil
	}

	// Personal virtual model: "~/model-name"
	if orgSlug == "~" {
		if a.personalVMStore == nil {
			return nil, nil, nil
		}
		u := httpCtx.User(ctx)
		if u == nil {
			return nil, nil, nil
		}
		pvm, err := a.personalVMStore.GetPersonalVirtualModelByName(ctx, u.ID(), localName)
		if err != nil {
			return nil, nil, nil //nolint:nilerr
		}
		return nil, &personalVMAdapter{pvm: pvm}, nil
	}

	// Org virtual model
	org, err := a.orgStore.GetOrgBySlug(ctx, orgSlug)
	if err != nil {
		return nil, nil, nil //nolint:nilerr
	}

	vm, err := a.virtualModelStore.GetVirtualModelByName(ctx, org.ID(), localName)
	if err != nil {
		return nil, nil, nil //nolint:nilerr
	}

	return org, vm, nil
}

// personalVMAdapter wraps PersonalVirtualModel as VirtualModel so the pipeline engine
// can handle it without knowing about the personal VM type.
type personalVMAdapter struct {
	pvm model.PersonalVirtualModel
}

func (a *personalVMAdapter) ID() model.VirtualModelID    { return model.VirtualModelID(a.pvm.ID()) }
func (a *personalVMAdapter) OrgID() model.OrgID          { return "" }
func (a *personalVMAdapter) Name() string                 { return a.pvm.Name() }
func (a *personalVMAdapter) Description() string          { return a.pvm.Description() }
func (a *personalVMAdapter) Graph() *model.PipelineGraph  { return a.pvm.Graph() }
func (a *personalVMAdapter) CreatedAt() time.Time         { return a.pvm.CreatedAt() }
func (a *personalVMAdapter) UpdatedAt() time.Time         { return a.pvm.UpdatedAt() }

var _ model.VirtualModel = &personalVMAdapter{}

// buildEC constructs the ExecutionContext for a pipeline run.
// org may be nil for personal virtual models; in that case the org context
// is derived from the token's OrgID.
func (a *PipelineHookAdapter) buildEC(ctx context.Context, req *genaiProxy.ProxyRequest, org model.Organization, vm model.VirtualModel) pipeline.ExecutionContext {
	userID := ""
	displayName := ""
	if u := httpCtx.User(ctx); u != nil {
		userID = string(u.ID())
		displayName = u.DisplayName()
	}

	// For personal VMs, org is nil — fall back to the token's org for model resolution.
	orgID := model.OrgID(OrgIDFromContext(ctx))
	var protoModels []*proto.ModelInfo
	var protoVMs []*proto.VirtualModelInfo
	if org != nil {
		orgID = org.ID()
		protoModels = buildProtoModels(ctx, a.providerStore, orgID)
		protoVMs = buildProtoVMs(ctx, a.virtualModelStore, orgID)
	} else if orgID != "" {
		protoModels = buildProtoModels(ctx, a.providerStore, orgID)
		protoVMs = buildProtoVMs(ctx, a.virtualModelStore, orgID)
	}

	return pipeline.ExecutionContext{
		OrgID:           string(orgID),
		UserID:          userID,
		DisplayName:     displayName,
		TokenID:         AuthTokenIDFromMeta(req.Metadata),
		MessagesJSON:    extractMessagesJSON(req.Body),
		BodyJSON:        string(req.Body),
		ProtoModels:     protoModels,
		ProtoVMs:        protoVMs,
		ProtoQuota:      nil,
		VisitedVMs:      map[model.VirtualModelID]struct{}{vm.ID(): {}},
		PersonalVMStore: a.personalVMStore,
	}
}

// extractMessagesJSON extracts the "messages" JSON array from a chat completions request body.
// Returns "[]" on failure so plugins receive a valid (empty) messages JSON.
func extractMessagesJSON(body []byte) string {
	if len(body) == 0 {
		return "[]"
	}
	var envelope struct {
		Messages json.RawMessage `json:"messages"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil || envelope.Messages == nil {
		return "[]"
	}
	return string(envelope.Messages)
}

// splitQualifiedName splits "org-slug/model-name" → (orgSlug, modelName, true).
func splitQualifiedName(name string) (string, string, bool) {
	idx := strings.IndexByte(name, '/')
	if idx <= 0 || idx == len(name)-1 {
		return "", "", false
	}
	return name[:idx], name[idx+1:], true
}

// buildProtoModels builds the list of available LLM models for the execution context.
func buildProtoModels(ctx context.Context, ps port.ProviderStore, orgID model.OrgID) []*proto.ModelInfo {
	if ps == nil {
		return nil
	}
	models, err := ps.ListEnabledLLMModels(ctx, orgID)
	if err != nil {
		slog.WarnContext(ctx, "could not list models for pipeline context", slog.Any("error", err))
		return nil
	}
	out := make([]*proto.ModelInfo, 0, len(models))
	for _, m := range models {
		caps := m.Capabilities()
		out = append(out, &proto.ModelInfo{
			ProxyName:            m.ProxyName(),
			RealModel:            m.RealModel(),
			ProviderId:           string(m.ProviderID()),
			IsVirtual:            false,
			ContextLength:        m.ContextWindow(),
			SupportsVision:       caps.Vision,
			SupportsReasoning:    caps.Reasoning,
			SupportsEmbeddings:   caps.Embeddings,
			ActiveParamsBillions: float32(m.ActiveParams()) / 1e9,
		})
	}
	return out
}

// buildProtoVMs builds the list of virtual models for the execution context.
func buildProtoVMs(ctx context.Context, vs port.VirtualModelStore, orgID model.OrgID) []*proto.VirtualModelInfo {
	if vs == nil {
		return nil
	}
	vms, err := vs.ListVirtualModels(ctx, orgID)
	if err != nil {
		return nil
	}
	out := make([]*proto.VirtualModelInfo, 0, len(vms))
	for _, vm := range vms {
		out = append(out, &proto.VirtualModelInfo{
			Id:          string(vm.ID()),
			Name:        vm.Name(),
			OrgId:       string(vm.OrgID()),
			Description: vm.Description(),
		})
	}
	return out
}

var (
	_ genaiProxy.PreRequestHook = (*PipelineHookAdapter)(nil)
	_ genaiProxy.ModelListerHook = (*PipelineHookAdapter)(nil)
)
