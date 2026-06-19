package pipeline

import (
	"context"

	"github.com/bornholm/genai/llm"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
)

// ExecutionContext carries per-request metadata through the pipeline engine.
type ExecutionContext struct {
	OrgID       string
	UserID      string
	TokenID     string
	DisplayName string
	// RequestJSON is the raw JSON of the LLM request body.
	RequestJSON string
	// MessagesJSON is the JSON array of messages extracted from RequestJSON.
	MessagesJSON string
	// BodyJSON is the full request body JSON.
	BodyJSON string
	// ProtoModels is the list of available real models for the org.
	ProtoModels []*proto.ModelInfo
	// ProtoVMs is the list of virtual models visible to the org.
	ProtoVMs []*proto.VirtualModelInfo
	// ProtoQuota contains the remaining quota for the user/org.
	ProtoQuota *proto.QuotaInfo
	// VisitedVMs tracks VirtualModelIDs already resolved to detect cycles.
	VisitedVMs map[model.VirtualModelID]struct{}
	// PersonalVMStore is used by ModelExecutor to resolve personal virtual models (~/name).
	PersonalVMStore port.PersonalVirtualModelStore
}

// ForwardResult is the output of a node's Forward execution.
type ForwardResult struct {
	// OutputValues are the typed values produced on output ports.
	OutputValues map[string]interface{}
	// NodeState is an opaque byte blob the pipeline engine stores and passes
	// back to the same node in the backward (post-response) pass.
	NodeState []byte
	// ResolvedClient is set by terminal (model) nodes to the llm.Client that
	// should handle the actual LLM call.
	ResolvedClient llm.Client
	// ResolvedModel is the real model name forwarded to the provider.
	ResolvedModel string
	// ResolvedModelID is the internal database ID of the resolved LLM model.
	ResolvedModelID model.LLMModelID
	// Rejected is true when a plugin node blocks the request.
	Rejected        bool
	RejectionReason string
	// Tools are additional llm.Tool definitions contributed by this node
	// (e.g. a TOOL_PROVIDER plugin), to be made available to the LLM call.
	Tools []llm.Tool
	// ClientDecorator, when set, wraps the eventually-resolved llm.Client
	// (e.g. to intercept and resolve tool calls server-side before they
	// reach the API client). Applied by the engine once the terminal model
	// node has resolved a client.
	ClientDecorator func(llm.Client) llm.Client
}

// BackwardResult is the output of a node's Backward execution.
type BackwardResult struct {
	// ModifiedResponseContent, when non-empty, replaces the response sent to the client.
	ModifiedResponseContent string
}

// TokensUsed holds token counts reported by the LLM.
type TokensUsed struct {
	Prompt     int64
	Completion int64
}

// NodeExecutor is implemented by each node type.
type NodeExecutor interface {
	// Forward is called during the request phase (before the LLM call).
	Forward(ctx context.Context, node model.PipelineNode, inputs map[string]interface{}, ec ExecutionContext) (*ForwardResult, error)
	// Backward is called during the response phase (after the LLM call) in
	// reverse order. state is the NodeState returned by Forward for the same
	// node in the same execution.
	Backward(ctx context.Context, node model.PipelineNode, state []byte, responseContent string, tokens *TokensUsed, hadError bool) (*BackwardResult, error)
}

// noopBackward is a helper that returns an empty BackwardResult without error.
func noopBackward(_ context.Context, _ model.PipelineNode, _ []byte, _ string, _ *TokensUsed, _ bool) (*BackwardResult, error) {
	return &BackwardResult{}, nil
}
