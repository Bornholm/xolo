package proxy

import (
	"context"
	"strings"
	"testing"

	"github.com/bornholm/genai/llm"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/pipeline"
)

// rewriteExecutor is a test NodeExecutor whose Backward pass replaces a
// placeholder in the response content, simulating the pseudonymizer plugin.
type rewriteExecutor struct {
	from, to string
}

func (e *rewriteExecutor) Forward(_ context.Context, _ model.PipelineNode, _ map[string]interface{}, _ pipeline.ExecutionContext) (*pipeline.ForwardResult, error) {
	return &pipeline.ForwardResult{}, nil
}

func (e *rewriteExecutor) Backward(_ context.Context, _ model.PipelineNode, _ []byte, responseContent string, _ *pipeline.TokensUsed, _ bool) (*pipeline.BackwardResult, error) {
	return &pipeline.BackwardResult{ModifiedResponseContent: strings.ReplaceAll(responseContent, e.from, e.to)}, nil
}

func newTestForwardExecution() *pipeline.ForwardExecution {
	return &pipeline.ForwardExecution{
		ExecutedNodes: []pipeline.ExecutedNode{
			{Node: model.PipelineNode{ID: "rewrite", Type: "rewrite"}},
		},
	}
}

func newTestEngine(from, to string) *pipeline.Engine {
	registry := pipeline.NewRegistry()
	registry.Register("rewrite", &rewriteExecutor{from: from, to: to})
	return pipeline.NewEngine(registry)
}

// multiChunkClient streams the given content split across several delta chunks.
type multiChunkClient struct {
	parts []string
}

func (c *multiChunkClient) ChatCompletion(_ context.Context, _ ...llm.ChatCompletionOptionFunc) (llm.ChatCompletionResponse, error) {
	return nil, nil
}

func (c *multiChunkClient) ChatCompletionStream(_ context.Context, _ ...llm.ChatCompletionOptionFunc) (<-chan llm.StreamChunk, error) {
	ch := make(chan llm.StreamChunk, len(c.parts)+1)
	go func() {
		defer close(ch)
		for _, p := range c.parts {
			ch <- llm.NewStreamChunk(llm.NewStreamDelta(llm.RoleAssistant, p))
		}
		ch <- llm.NewCompleteStreamChunk(llm.NewChatCompletionUsage(0, 0, 0))
	}()
	return ch, nil
}

func (c *multiChunkClient) Embeddings(_ context.Context, _ []string, _ ...llm.EmbeddingsOptionFunc) (llm.EmbeddingsResponse, error) {
	return nil, nil
}

var _ llm.Client = (*multiChunkClient)(nil)

func collectContent(t *testing.T, ch <-chan llm.StreamChunk) string {
	t.Helper()
	var b strings.Builder
	for chunk := range ch {
		if d := chunk.Delta(); d != nil {
			b.WriteString(d.Content())
		}
	}
	return b.String()
}

func TestPipelineWrappedClient_ChatCompletionStream_Deanonymize(t *testing.T) {
	inner := &multiChunkClient{parts: []string{"Bonjour [PERSON_1], votre email est ", "[EMAIL_1]."}}
	engine := newTestEngine("[EMAIL_1]", "wpetit@cadoles.com")
	wrapped := NewPipelineWrappedClient(inner, engine, newTestForwardExecution(), pipeline.ExecutionContext{})

	ch, err := wrapped.ChatCompletionStream(context.Background())
	if err != nil {
		t.Fatalf("ChatCompletionStream() error = %v", err)
	}

	got := collectContent(t, ch)
	want := "Bonjour [PERSON_1], votre email est wpetit@cadoles.com."
	if got != want {
		t.Errorf("content = %q, want %q", got, want)
	}
}

func TestPipelineWrappedClient_ChatCompletionStream_Unmodified(t *testing.T) {
	inner := &multiChunkClient{parts: []string{"Bonjour ", "tout le monde."}}
	engine := newTestEngine("[EMAIL_1]", "wpetit@cadoles.com")
	wrapped := NewPipelineWrappedClient(inner, engine, newTestForwardExecution(), pipeline.ExecutionContext{})

	ch, err := wrapped.ChatCompletionStream(context.Background())
	if err != nil {
		t.Fatalf("ChatCompletionStream() error = %v", err)
	}

	got := collectContent(t, ch)
	want := "Bonjour tout le monde."
	if got != want {
		t.Errorf("content = %q, want %q", got, want)
	}
}
