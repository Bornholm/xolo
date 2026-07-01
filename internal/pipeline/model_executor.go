package pipeline

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/bornholm/genai/llm"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/pkg/errors"
)

// ModelResolver resolves a qualified proxy name (e.g. "org/gpt-4o") to an
// llm.Client. This interface is implemented by OrgModelRouter.
type ModelResolver interface {
	ResolveRealModel(ctx context.Context, orgID model.OrgID, proxyName string) (llm.Client, string, model.LLMModelID, error)
}

// ModelExecutor handles NodeTypeModel.
// It reads the model proxy name either from the "model_name" input port
// (connected at runtime) or from the static ProxyName in its config, then
// resolves it via the ModelResolver.
//
// If the resolved name corresponds to a VirtualModel or personal virtual model,
// it recurses into that model's pipeline (with cycle detection).
type ModelExecutor struct {
	resolver          ModelResolver
	virtualModelStore port.VirtualModelStore
	engine            *Engine // for recursive VM resolution
}

// NewModelExecutor creates a ModelExecutor.
func NewModelExecutor(resolver ModelResolver, virtualModelStore port.VirtualModelStore, engine *Engine) *ModelExecutor {
	return &ModelExecutor{
		resolver:          resolver,
		virtualModelStore: virtualModelStore,
		engine:            engine,
	}
}

func (e *ModelExecutor) Forward(ctx context.Context, node model.PipelineNode, inputs map[string]interface{}, ec ExecutionContext) (*ForwardResult, error) {
	proxyName, err := resolveModelName(node, inputs)
	if err != nil {
		return nil, errors.Wrap(err, "model node: could not determine model name")
	}

	orgID := model.OrgID(ec.OrgID)

	// Personal virtual model (~/name) → recurse using the user's personal store.
	if strings.HasPrefix(proxyName, "~/") && ec.PersonalVMStore != nil && ec.UserID != "" {
		localName := proxyName[2:]
		pvm, pvmErr := ec.PersonalVMStore.GetPersonalVirtualModelByName(ctx, model.UserID(ec.UserID), localName)
		if pvmErr == nil && pvm != nil {
			// Use a namespaced key to avoid collision with org VM IDs.
			pvmKey := model.VirtualModelID("~:" + string(pvm.ID()))
			if _, alreadyVisited := ec.VisitedVMs[pvmKey]; alreadyVisited {
				return nil, errors.Errorf("personal virtual model cycle detected: %s", proxyName)
			}
			if pvm.Graph() == nil {
				return nil, errors.Errorf("personal virtual model %q has no pipeline configured", proxyName)
			}
			childEC := ec
			childEC.VisitedVMs = copyVisitedVMs(ec.VisitedVMs)
			childEC.VisitedVMs[pvmKey] = struct{}{}

			sub, err := e.engine.RunForward(ctx, pvm.Graph(), childEC)
			if err != nil {
				return nil, errors.Wrapf(err, "personal virtual model %q pipeline failed", proxyName)
			}
			return &ForwardResult{
				ResolvedClient:  sub.ResolvedClient,
				ResolvedModel:   sub.ResolvedModel,
				ResolvedModelID: sub.ResolvedModelID,
				OutputValues:    map[string]interface{}{"response": ""},
			}, nil
		}
	}

	// Org virtual model → recurse.
	if e.virtualModelStore != nil {
		vm, vmErr := e.virtualModelStore.GetVirtualModelByName(ctx, orgID, localModelName(proxyName))
		if vmErr == nil && vm != nil {
			vmID := vm.ID()
			if _, alreadyVisited := ec.VisitedVMs[vmID]; alreadyVisited {
				return nil, errors.Errorf("virtual model cycle detected: %s", proxyName)
			}
			if vm.Graph() == nil {
				return nil, errors.Errorf("virtual model %q has no pipeline configured", proxyName)
			}
			// Clone visited set to avoid mutations affecting sibling branches.
			childEC := ec
			childEC.VisitedVMs = copyVisitedVMs(ec.VisitedVMs)
			childEC.VisitedVMs[vmID] = struct{}{}

			sub, err := e.engine.RunForward(ctx, vm.Graph(), childEC)
			if err != nil {
				return nil, errors.Wrapf(err, "virtual model %q pipeline failed", proxyName)
			}
			return &ForwardResult{
				ResolvedClient:  sub.ResolvedClient,
				ResolvedModel:   sub.ResolvedModel,
				ResolvedModelID: sub.ResolvedModelID,
				OutputValues:    map[string]interface{}{"response": ""},
			}, nil
		}
	}

	// Real model: delegate to the ModelResolver.
	client, realModel, modelID, err := e.resolver.ResolveRealModel(ctx, orgID, proxyName)
	if err != nil {
		return nil, errors.Wrapf(err, "model executor: could not resolve %q", proxyName)
	}

	return &ForwardResult{
		ResolvedClient:  client,
		ResolvedModel:   realModel,
		ResolvedModelID: modelID,
		OutputValues:    map[string]interface{}{"response": ""},
	}, nil
}

func (e *ModelExecutor) Backward(ctx context.Context, node model.PipelineNode, state []byte, responseContent string, tokens *TokensUsed, hadError bool) (*BackwardResult, error) {
	return noopBackward(ctx, node, state, responseContent, tokens, hadError)
}

// resolveModelName returns the proxy model name from the "model_name" input
// port (when connected) or the static ProxyName in the node config.
func resolveModelName(node model.PipelineNode, inputs map[string]interface{}) (string, error) {
	if v, ok := inputs["model_name"]; ok {
		if s, ok := v.(string); ok && s != "" {
			return s, nil
		}
	}
	if node.Data != nil {
		var d model.ModelNodeData
		if err := json.Unmarshal(node.Data, &d); err == nil && d.ProxyName != "" {
			return d.ProxyName, nil
		}
	}
	return "", errors.New("no model_name input connected and no static proxyName configured")
}

// localModelName strips the "org-slug/" prefix if present.
func localModelName(name string) string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '/' {
			return name[i+1:]
		}
	}
	return name
}

func copyVisitedVMs(m map[model.VirtualModelID]struct{}) map[model.VirtualModelID]struct{} {
	out := make(map[model.VirtualModelID]struct{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
