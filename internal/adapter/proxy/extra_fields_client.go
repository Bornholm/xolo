package proxy

import (
	"context"

	"github.com/bornholm/genai/llm"
)

// extraFieldsClient wraps an llm.Client to inject a fixed set of provider-specific
// extra body fields into every chat-completion request. It is used to apply a
// model's configured ExtraBody (e.g. MiniMax's "reasoning_split") to all calls
// routed to that model.
type extraFieldsClient struct {
	inner  llm.Client
	fields map[string]any
}

// newExtraFieldsClient returns a client that appends llm.WithExtraFields(fields)
// to every chat-completion call. When fields is empty the inner client is
// returned unchanged so no wrapper overhead is added.
func newExtraFieldsClient(inner llm.Client, fields map[string]any) llm.Client {
	if len(fields) == 0 {
		return inner
	}
	return &extraFieldsClient{inner: inner, fields: fields}
}

func (c *extraFieldsClient) withExtra(funcs []llm.ChatCompletionOptionFunc) []llm.ChatCompletionOptionFunc {
	out := make([]llm.ChatCompletionOptionFunc, 0, len(funcs)+1)
	out = append(out, funcs...)
	out = append(out, llm.WithExtraFields(c.fields))
	return out
}

func (c *extraFieldsClient) ChatCompletion(ctx context.Context, funcs ...llm.ChatCompletionOptionFunc) (llm.ChatCompletionResponse, error) {
	return c.inner.ChatCompletion(ctx, c.withExtra(funcs)...)
}

func (c *extraFieldsClient) ChatCompletionStream(ctx context.Context, funcs ...llm.ChatCompletionOptionFunc) (<-chan llm.StreamChunk, error) {
	return c.inner.ChatCompletionStream(ctx, c.withExtra(funcs)...)
}

func (c *extraFieldsClient) Embeddings(ctx context.Context, inputs []string, funcs ...llm.EmbeddingsOptionFunc) (llm.EmbeddingsResponse, error) {
	return c.inner.Embeddings(ctx, inputs, funcs...)
}

var _ llm.Client = (*extraFieldsClient)(nil)
