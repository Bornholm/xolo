package pipeline

import (
	"context"
	"fmt"

	"github.com/bornholm/genai/llm"
	"github.com/pkg/errors"
)

// newDummyClient creates an llm.Client that always returns content as a fixed
// response, without calling any real LLM provider.
func newDummyClient(content string) llm.Client {
	return &dummyClient{content: content}
}

type dummyClient struct{ content string }

func (c *dummyClient) ChatCompletion(_ context.Context, _ ...llm.ChatCompletionOptionFunc) (llm.ChatCompletionResponse, error) {
	msg := llm.NewMessage(llm.RoleAssistant, c.content)
	usage := llm.NewChatCompletionUsage(0, 0, 0)
	return llm.NewChatCompletionResponse(msg, usage), nil
}

func (c *dummyClient) ChatCompletionStream(_ context.Context, _ ...llm.ChatCompletionOptionFunc) (<-chan llm.StreamChunk, error) {
	ch := make(chan llm.StreamChunk, 2)
	go func() {
		defer close(ch)
		delta := llm.NewStreamDelta(llm.RoleAssistant, c.content)
		ch <- llm.NewStreamChunk(delta)
		ch <- llm.NewCompleteStreamChunk(llm.NewChatCompletionUsage(0, 0, 0))
	}()
	return ch, nil
}

func (c *dummyClient) Embeddings(_ context.Context, _ []string, _ ...llm.EmbeddingsOptionFunc) (llm.EmbeddingsResponse, error) {
	return nil, errors.New(fmt.Sprintf("dummy client: embeddings not supported"))
}

var _ llm.Client = &dummyClient{}
