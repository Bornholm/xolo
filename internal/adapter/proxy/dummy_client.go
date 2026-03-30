package proxy

import (
	"context"
	"fmt"

	"github.com/bornholm/genai/llm"
	"github.com/pkg/errors"
)

// DummyLLMClient is an llm.Client that returns a fixed text response without
// calling any real LLM provider. It supports both streaming and non-streaming
// transparently, making it suitable for test/dummy proxy hooks.
type DummyLLMClient struct {
	content string
	model   string
}

// NewDummyLLMClient creates a DummyLLMClient that always replies with content.
func NewDummyLLMClient(content, model string) *DummyLLMClient {
	return &DummyLLMClient{content: content, model: model}
}

// ChatCompletion implements llm.ChatCompletionClient.
func (c *DummyLLMClient) ChatCompletion(_ context.Context, _ ...llm.ChatCompletionOptionFunc) (llm.ChatCompletionResponse, error) {
	msg := llm.NewMessage(llm.RoleAssistant, c.content)
	usage := llm.NewChatCompletionUsage(0, 0, 0)
	return llm.NewChatCompletionResponse(msg, usage), nil
}

// ChatCompletionStream implements llm.ChatCompletionStreamingClient.
// It sends the full content as a single delta chunk, then a completion chunk.
func (c *DummyLLMClient) ChatCompletionStream(_ context.Context, _ ...llm.ChatCompletionOptionFunc) (<-chan llm.StreamChunk, error) {
	ch := make(chan llm.StreamChunk, 2)
	go func() {
		defer close(ch)
		delta := llm.NewStreamDelta(llm.RoleAssistant, c.content)
		ch <- llm.NewStreamChunk(delta)
		ch <- llm.NewCompleteStreamChunk(llm.NewChatCompletionUsage(0, 0, 0))
	}()
	return ch, nil
}

// Embeddings implements llm.EmbeddingsClient. Dummy clients do not support embeddings.
func (c *DummyLLMClient) Embeddings(_ context.Context, _ []string, _ ...llm.EmbeddingsOptionFunc) (llm.EmbeddingsResponse, error) {
	return nil, errors.New(fmt.Sprintf("dummy-model (%s): embeddings not supported", c.model))
}

// Compile-time interface assertions.
var _ llm.Client = &DummyLLMClient{}
