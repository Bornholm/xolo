package pipeline

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/bornholm/xolo/internal/core/model"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
	"github.com/pkg/errors"
)

// PluginExecutor handles NodeTypePlugin nodes by delegating to the gRPC plugin.
type PluginExecutor struct {
	clients     map[string]proto.XoloPluginClient
	descriptors map[string]*proto.PluginDescriptor
}

// NewPluginExecutor creates a PluginExecutor backed by the loaded plugin clients.
func NewPluginExecutor(
	clients map[string]proto.XoloPluginClient,
	descriptors map[string]*proto.PluginDescriptor,
) *PluginExecutor {
	return &PluginExecutor{clients: clients, descriptors: descriptors}
}

func (e *PluginExecutor) Forward(ctx context.Context, node model.PipelineNode, inputs map[string]interface{}, ec ExecutionContext) (*ForwardResult, error) {
	data, err := parsePluginNodeData(node)
	if err != nil {
		return nil, errors.Wrap(err, "invalid plugin node data")
	}

	client, ok := e.clients[data.PluginName]
	if !ok {
		return nil, errors.Errorf("plugin %q is not loaded", data.PluginName)
	}
	desc := e.descriptors[data.PluginName]

	configJSON := "{}"
	if data.Config != nil {
		configJSON = string(data.Config)
	}

	reqCtx := &proto.RequestContext{
		OrgId:       ec.OrgID,
		UserId:      ec.UserID,
		TokenId:     ec.TokenID,
		DisplayName: ec.DisplayName,
		ConfigJson:  configJSON,
	}

	inputsJSON := InputsJSON(inputs)

	// Dispatch based on capability.
	if hasCapability(desc, proto.PluginDescriptor_PRE_REQUEST) {
		return e.forwardPreRequest(ctx, client, reqCtx, node, inputs, inputsJSON, ec)
	}
	if hasCapability(desc, proto.PluginDescriptor_RESOLVE_MODEL) {
		return e.forwardResolveModel(ctx, client, reqCtx, node, inputs, inputsJSON, ec)
	}

	// No relevant capability: pass through.
	return &ForwardResult{OutputValues: inputs}, nil
}

func (e *PluginExecutor) forwardPreRequest(
	ctx context.Context,
	client proto.XoloPluginClient,
	reqCtx *proto.RequestContext,
	node model.PipelineNode,
	inputs map[string]interface{},
	inputsJSON string,
	ec ExecutionContext,
) (*ForwardResult, error) {
	messagesJSON, _ := inputs["messages_json"].(string)
	if messagesJSON == "" {
		messagesJSON = ec.MessagesJSON
	}

	out, err := client.PreRequest(ctx, &proto.PreRequestInput{
		Ctx:          reqCtx,
		Model:        ec.RequestJSON,
		MessagesJson: messagesJSON,
		InputsJson:   inputsJSON,
	})
	if err != nil {
		return nil, errors.Wrap(err, "plugin PreRequest failed")
	}

	if !out.Allowed {
		return &ForwardResult{Rejected: true, RejectionReason: out.RejectionReason}, nil
	}

	result := &ForwardResult{
		NodeState:    out.NodeState,
		OutputValues: ParseOutputsJSON(out.OutputsJson),
	}
	if result.OutputValues == nil {
		result.OutputValues = make(map[string]interface{})
	}

	// Pass the (possibly modified) request through.
	if out.ModifiedMessagesJson != "" {
		result.OutputValues["messages_json"] = out.ModifiedMessagesJson
	}

	return result, nil
}

func (e *PluginExecutor) forwardResolveModel(
	ctx context.Context,
	client proto.XoloPluginClient,
	reqCtx *proto.RequestContext,
	node model.PipelineNode,
	inputs map[string]interface{},
	inputsJSON string,
	ec ExecutionContext,
) (*ForwardResult, error) {
	messagesJSON, _ := inputs["messages_json"].(string)
	if messagesJSON == "" {
		messagesJSON = ec.MessagesJSON
	}

	out, err := client.ResolveModel(ctx, &proto.ResolveModelInput{
		Ctx:             reqCtx,
		RequestedModel:  modelFromInputs(inputs),
		AvailableModels: ec.ProtoModels,
		MessagesJson:    messagesJSON,
		VirtualModels:   ec.ProtoVMs,
		Quota:           ec.ProtoQuota,
		BodyJson:        ec.BodyJSON,
	})
	if err != nil {
		return nil, errors.Wrap(err, "plugin ResolveModel failed")
	}

	// Short-circuit: plugin returns a forged response (e.g. dummy-model).
	if out.ResponseContent != "" {
		return &ForwardResult{
			ResolvedClient: newDummyClient(out.ResponseContent),
			ResolvedModel:  "dummy",
			OutputValues:   map[string]interface{}{"response": out.ResponseContent},
		}, nil
	}

	result := &ForwardResult{
		OutputValues: make(map[string]interface{}),
	}
	if out.ResolvedProxyName != "" {
		result.OutputValues["model_name"] = out.ResolvedProxyName
	}
	return result, nil
}

func (e *PluginExecutor) Backward(ctx context.Context, node model.PipelineNode, state []byte, responseContent string, tokens *TokensUsed, hadError bool) (*BackwardResult, error) {
	data, err := parsePluginNodeData(node)
	if err != nil {
		return &BackwardResult{}, nil
	}

	client, ok := e.clients[data.PluginName]
	if !ok {
		return &BackwardResult{}, nil
	}
	desc := e.descriptors[data.PluginName]

	if !hasCapability(desc, proto.PluginDescriptor_POST_RESPONSE) {
		return &BackwardResult{}, nil
	}

	var prompt, completion int64
	if tokens != nil {
		prompt = tokens.Prompt
		completion = tokens.Completion
	}

	out, err := client.PostResponse(ctx, &proto.PostResponseInput{
		Model:           "",
		PromptTokens:    prompt,
		CompletionTokens: completion,
		HadError:        hadError,
		ResponseContent: responseContent,
		NodeState:       state,
	})
	if err != nil {
		slog.WarnContext(ctx, "plugin PostResponse failed",
			slog.String("plugin", data.PluginName),
			slog.Any("error", err))
		return &BackwardResult{}, nil
	}

	return &BackwardResult{ModifiedResponseContent: out.ModifiedResponseContent}, nil
}

func parsePluginNodeData(node model.PipelineNode) (*model.PluginNodeData, error) {
	if node.Data == nil {
		return nil, errors.New("plugin node has no data")
	}
	var d model.PluginNodeData
	if err := json.Unmarshal(node.Data, &d); err != nil {
		return nil, errors.Wrap(err, "unmarshal plugin node data")
	}
	if d.PluginName == "" {
		return nil, errors.New("plugin node data has no pluginName")
	}
	return &d, nil
}

func hasCapability(desc *proto.PluginDescriptor, cap proto.PluginDescriptor_Capability) bool {
	if desc == nil {
		return false
	}
	for _, c := range desc.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

func modelFromInputs(inputs map[string]interface{}) string {
	if v, ok := inputs["model_name"]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
