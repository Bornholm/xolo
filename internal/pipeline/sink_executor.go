package pipeline

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
)

// SinkExecutor handles NodeTypeSink.
// The sink node accepts a "response" input and terminates the forward chain.
// It never produces an LLM client — the model node upstream does that.
type SinkExecutor struct{}

func NewSinkExecutor() *SinkExecutor { return &SinkExecutor{} }

func (e *SinkExecutor) Forward(_ context.Context, _ model.PipelineNode, inputs map[string]interface{}, _ ExecutionContext) (*ForwardResult, error) {
	// Nothing to do: the model node upstream already set ResolvedClient.
	return &ForwardResult{
		OutputValues: inputs, // pass through
	}, nil
}

func (e *SinkExecutor) Backward(ctx context.Context, node model.PipelineNode, state []byte, responseContent string, tokens *TokensUsed, hadError bool) (*BackwardResult, error) {
	return noopBackward(ctx, node, state, responseContent, tokens, hadError)
}
