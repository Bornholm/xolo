package pipeline

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
)

// GeneratorExecutor handles NodeTypeGenerator.
// The generator node has no inputs; its "request" output is seeded by the
// engine before execution begins (via ValueContext.Set).
type GeneratorExecutor struct{}

func NewGeneratorExecutor() *GeneratorExecutor { return &GeneratorExecutor{} }

func (e *GeneratorExecutor) Forward(_ context.Context, _ model.PipelineNode, inputs map[string]interface{}, ec ExecutionContext) (*ForwardResult, error) {
	// The request value was already seeded by the engine; just expose it as output.
	return &ForwardResult{
		OutputValues: map[string]interface{}{
			"request": ec.RequestJSON,
		},
	}, nil
}

func (e *GeneratorExecutor) Backward(ctx context.Context, node model.PipelineNode, state []byte, responseContent string, tokens *TokensUsed, hadError bool) (*BackwardResult, error) {
	return noopBackward(ctx, node, state, responseContent, tokens, hadError)
}
