package proxy

import (
	"bytes"
	"context"
	"log/slog"

	"github.com/bornholm/genai/llm"
	"github.com/bornholm/xolo/internal/pipeline"
)

// PipelineWrappedClient wraps an llm.Client to run the pipeline's backward pass
// (post-response processing) after each LLM call.
type PipelineWrappedClient struct {
	inner       llm.Client
	engine      *pipeline.Engine
	forwardExec *pipeline.ForwardExecution
	ec          pipeline.ExecutionContext
}

// NewPipelineWrappedClient creates a PipelineWrappedClient.
func NewPipelineWrappedClient(
	inner llm.Client,
	engine *pipeline.Engine,
	forwardExec *pipeline.ForwardExecution,
	ec pipeline.ExecutionContext,
) *PipelineWrappedClient {
	return &PipelineWrappedClient{inner: inner, engine: engine, forwardExec: forwardExec, ec: ec}
}

// ChatCompletion calls the inner client and runs the backward pass on the result.
func (c *PipelineWrappedClient) ChatCompletion(ctx context.Context, funcs ...llm.ChatCompletionOptionFunc) (llm.ChatCompletionResponse, error) {
	resp, err := c.inner.ChatCompletion(ctx, funcs...)
	if err != nil {
		c.runBackward(ctx, "", nil, true)
		return resp, err
	}

	content := ""
	if msg := resp.Message(); msg != nil {
		content = msg.Content()
	}
	tokens := extractResponseTokens(resp)

	modified, backErr := c.engine.RunBackward(ctx, c.forwardExec, content, tokens, false)
	if backErr != nil {
		slog.WarnContext(ctx, "pipeline backward pass failed", slog.Any("error", backErr))
		return resp, nil
	}

	if modified != content {
		return &wrappedChatCompletionResponse{inner: resp, modifiedContent: modified}, nil
	}
	return resp, nil
}

// ChatCompletionStream buffers the full streaming response, runs the backward
// pass, then re-streams the (possibly modified) content.
func (c *PipelineWrappedClient) ChatCompletionStream(ctx context.Context, funcs ...llm.ChatCompletionOptionFunc) (<-chan llm.StreamChunk, error) {
	sourceCh, err := c.inner.ChatCompletionStream(ctx, funcs...)
	if err != nil {
		return nil, err
	}

	outCh := make(chan llm.StreamChunk, 8)
	go func() {
		defer close(outCh)

		var chunks []llm.StreamChunk
		var buf bytes.Buffer
		var lastTokens *pipeline.TokensUsed

		for chunk := range sourceCh {
			chunks = append(chunks, chunk)
			if d := chunk.Delta(); d != nil {
				buf.WriteString(d.Content())
			}
			if u := chunk.Usage(); u != nil {
				lastTokens = &pipeline.TokensUsed{
					Prompt:     u.PromptTokens(),
					Completion: u.CompletionTokens(),
				}
			}
		}

		content := buf.String()
		_, backErr := c.engine.RunBackward(ctx, c.forwardExec, content, lastTokens, false)
		if backErr != nil {
			slog.WarnContext(ctx, "pipeline backward pass (stream) failed", slog.Any("error", backErr))
		}
		// Re-emit original chunks unchanged (stream modification is complex, deferred to v2).
		for _, ch := range chunks {
			outCh <- ch
		}
	}()

	return outCh, nil
}

// Embeddings is passed through unchanged.
func (c *PipelineWrappedClient) Embeddings(ctx context.Context, inputs []string, funcs ...llm.EmbeddingsOptionFunc) (llm.EmbeddingsResponse, error) {
	return c.inner.Embeddings(ctx, inputs, funcs...)
}

func (c *PipelineWrappedClient) runBackward(ctx context.Context, content string, tokens *pipeline.TokensUsed, hadError bool) {
	if _, err := c.engine.RunBackward(ctx, c.forwardExec, content, tokens, hadError); err != nil {
		slog.WarnContext(ctx, "pipeline backward pass failed", slog.Any("error", err))
	}
}

func extractResponseTokens(resp llm.ChatCompletionResponse) *pipeline.TokensUsed {
	if resp == nil {
		return nil
	}
	u := resp.Usage()
	if u == nil {
		return nil
	}
	return &pipeline.TokensUsed{Prompt: u.PromptTokens(), Completion: u.CompletionTokens()}
}

// wrappedChatCompletionResponse replaces the message content while keeping
// everything else from the original response.
type wrappedChatCompletionResponse struct {
	inner           llm.ChatCompletionResponse
	modifiedContent string
}

func (r *wrappedChatCompletionResponse) Message() llm.Message {
	return &modifiedMessage{original: r.inner.Message(), content: r.modifiedContent}
}

func (r *wrappedChatCompletionResponse) ToolCalls() []llm.ToolCall { return r.inner.ToolCalls() }
func (r *wrappedChatCompletionResponse) Usage() llm.ChatCompletionUsage { return r.inner.Usage() }

// modifiedMessage replaces Content() while delegating everything else.
type modifiedMessage struct {
	original llm.Message
	content  string
}

func (m *modifiedMessage) Role() llm.Role         { return m.original.Role() }
func (m *modifiedMessage) Content() string        { return m.content }
func (m *modifiedMessage) Attachments() []llm.Attachment {
	if a, ok := m.original.(interface{ Attachments() []llm.Attachment }); ok {
		return a.Attachments()
	}
	return nil
}

var _ llm.Client = (*PipelineWrappedClient)(nil)
