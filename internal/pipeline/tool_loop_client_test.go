package pipeline_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bornholm/genai/llm"
	"github.com/bornholm/xolo/internal/pipeline"
)

// scriptedClient replays a fixed sequence of responses, one per
// ChatCompletion call, and records the messages it was called with.
type scriptedClient struct {
	responses   []*scriptedResponse
	calls       [][]llm.Message
	toolChoices []llm.ToolChoice
	streamErr   error
}

type scriptedResponse struct {
	content   string
	toolCalls []llm.ToolCall
}

func (c *scriptedClient) ChatCompletion(_ context.Context, funcs ...llm.ChatCompletionOptionFunc) (llm.ChatCompletionResponse, error) {
	opts := llm.NewChatCompletionOptions(funcs...)
	c.calls = append(c.calls, opts.Messages)
	c.toolChoices = append(c.toolChoices, opts.ToolChoice)

	idx := len(c.calls) - 1
	if idx >= len(c.responses) {
		return nil, errors.New("scriptedClient: no more scripted responses")
	}
	r := c.responses[idx]
	return llm.NewChatCompletionResponse(llm.NewMessage(llm.RoleAssistant, r.content), nil, r.toolCalls...), nil
}

func (c *scriptedClient) ChatCompletionStream(ctx context.Context, funcs ...llm.ChatCompletionOptionFunc) (<-chan llm.StreamChunk, error) {
	if c.streamErr != nil {
		return nil, c.streamErr
	}
	resp, err := c.ChatCompletion(ctx, funcs...)
	if err != nil {
		return nil, err
	}

	ch := make(chan llm.StreamChunk, 4)
	var deltas []llm.ToolCallDelta
	for i, tc := range resp.ToolCalls() {
		deltas = append(deltas, llm.NewToolCallDelta(i, tc.ID(), tc.Name(), toString(tc.Parameters())))
	}
	ch <- llm.NewStreamChunk(llm.NewStreamDelta(llm.RoleAssistant, resp.Message().Content(), deltas...))
	close(ch)
	return ch, nil
}

func (c *scriptedClient) Embeddings(_ context.Context, _ []string, _ ...llm.EmbeddingsOptionFunc) (llm.EmbeddingsResponse, error) {
	return nil, nil
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

var _ llm.Client = (*scriptedClient)(nil)

func newEchoTool(name string) llm.Tool {
	return llm.NewFuncTool(name, "echoes its input", nil, func(_ context.Context, params map[string]any) (llm.ToolResult, error) {
		return llm.NewToolResult("tool result for " + name), nil
	})
}

func newFailingTool(name string, err error) llm.Tool {
	return llm.NewFuncTool(name, "always fails", nil, func(_ context.Context, params map[string]any) (llm.ToolResult, error) {
		return nil, err
	})
}

func TestToolLoopClient_NoToolCalls_ReturnsResponseAsIs(t *testing.T) {
	inner := &scriptedClient{responses: []*scriptedResponse{{content: "hello"}}}
	c := pipeline.NewToolLoopClient(inner, []llm.Tool{newEchoTool("search")}, 0, 0)

	resp, err := c.ChatCompletion(context.Background(), llm.WithMessages(llm.NewMessage(llm.RoleUser, "hi")))
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if resp.Message().Content() != "hello" {
		t.Errorf("expected %q, got %q", "hello", resp.Message().Content())
	}
	if len(inner.calls) != 1 {
		t.Errorf("expected exactly 1 inner call, got %d", len(inner.calls))
	}
}

func TestToolLoopClient_ResolvesKnownToolCall_ThenReturnsFinal(t *testing.T) {
	tc := llm.NewToolCall("call-1", "search", `{"q":"xolo"}`)
	inner := &scriptedClient{responses: []*scriptedResponse{
		{toolCalls: []llm.ToolCall{tc}},
		{content: "final answer"},
	}}
	c := pipeline.NewToolLoopClient(inner, []llm.Tool{newEchoTool("search")}, 0, 0)

	resp, err := c.ChatCompletion(context.Background(), llm.WithMessages(llm.NewMessage(llm.RoleUser, "hi")))
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if resp.Message().Content() != "final answer" {
		t.Errorf("expected final answer, got %q", resp.Message().Content())
	}
	if len(inner.calls) != 2 {
		t.Fatalf("expected 2 inner calls (tool round + final), got %d", len(inner.calls))
	}
	// Second call's messages must include the tool_calls message and its result.
	secondCallMessages := inner.calls[1]
	foundToolResult := false
	for _, m := range secondCallMessages {
		if m.Role() == llm.RoleTool {
			foundToolResult = true
		}
	}
	if !foundToolResult {
		t.Error("expected a tool result message to be appended before the second call")
	}
}

func TestToolLoopClient_UnknownToolCall_PassesThroughUnresolved(t *testing.T) {
	tc := llm.NewToolCall("call-1", "not-a-known-tool", `{}`)
	inner := &scriptedClient{responses: []*scriptedResponse{
		{toolCalls: []llm.ToolCall{tc}},
	}}
	c := pipeline.NewToolLoopClient(inner, []llm.Tool{newEchoTool("search")}, 0, 0)

	resp, err := c.ChatCompletion(context.Background(), llm.WithMessages(llm.NewMessage(llm.RoleUser, "hi")))
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if len(resp.ToolCalls()) != 1 {
		t.Fatalf("expected the unresolved tool call to be passed through, got %d", len(resp.ToolCalls()))
	}
	if len(inner.calls) != 1 {
		t.Errorf("expected exactly 1 inner call (no loop on unknown tool), got %d", len(inner.calls))
	}
}

func TestToolLoopClient_MaxIterationsExceeded_ReturnsError(t *testing.T) {
	tc := llm.NewToolCall("call-1", "search", `{}`)
	// Always returns the same tool call, never resolving.
	responses := make([]*scriptedResponse, 5)
	for i := range responses {
		responses[i] = &scriptedResponse{toolCalls: []llm.ToolCall{tc}}
	}
	inner := &scriptedClient{responses: responses}
	c := pipeline.NewToolLoopClient(inner, []llm.Tool{newEchoTool("search")}, 3, 10)

	_, err := c.ChatCompletion(context.Background(), llm.WithMessages(llm.NewMessage(llm.RoleUser, "hi")))
	if err == nil {
		t.Fatal("expected an error after exceeding max iterations")
	}
}

func TestToolLoopClient_ChatCompletionStream_BuffersToolTurnAndReplaysFinal(t *testing.T) {
	tc := llm.NewToolCall("call-1", "search", `{}`)
	inner := &scriptedClient{responses: []*scriptedResponse{
		{toolCalls: []llm.ToolCall{tc}},
		{content: "final streamed answer"},
	}}
	c := pipeline.NewToolLoopClient(inner, []llm.Tool{newEchoTool("search")}, 0, 0)

	ch, err := c.ChatCompletionStream(context.Background(), llm.WithMessages(llm.NewMessage(llm.RoleUser, "hi")))
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}

	var contents []string
	for chunk := range ch {
		if d := chunk.Delta(); d != nil {
			contents = append(contents, d.Content())
		}
	}

	if len(contents) != 1 || contents[0] != "final streamed answer" {
		t.Errorf("expected only the final turn to be replayed, got %v", contents)
	}
}

// Resilience: a failing tool call (e.g. a rate-limited or unreachable MCP
// server) must not abort the whole chat completion with an error — it
// should be surfaced to the LLM as a tool result so a final answer can
// still be produced.
func TestToolLoopClient_FailingToolCall_DoesNotAbortCompletion(t *testing.T) {
	tc := llm.NewToolCall("call-1", "search", `{}`)
	inner := &scriptedClient{responses: []*scriptedResponse{
		{toolCalls: []llm.ToolCall{tc}},
		{content: "sorry, the search tool is unavailable"},
	}}
	c := pipeline.NewToolLoopClient(inner, []llm.Tool{newFailingTool("search", errors.New("rate limited"))}, 0, 0)

	resp, err := c.ChatCompletion(context.Background(), llm.WithMessages(llm.NewMessage(llm.RoleUser, "hi")))
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if resp.Message().Content() != "sorry, the search tool is unavailable" {
		t.Errorf("expected the LLM's final answer despite the tool failure, got %q", resp.Message().Content())
	}
	if len(inner.calls) != 2 {
		t.Fatalf("expected the loop to continue after the tool failure, got %d calls", len(inner.calls))
	}

	secondCallMessages := inner.calls[1]
	foundErrorMention := false
	for _, m := range secondCallMessages {
		if m.Role() == llm.RoleTool && m.Content() != "" {
			foundErrorMention = true
		}
	}
	if !foundErrorMention {
		t.Error("expected a tool result message describing the failure")
	}
}

// Regression test: llm.NewChatCompletionOptions defaults ToolChoice to
// ToolChoiceNone, which silently prevents providers from ever using the
// tools we inject unless we explicitly force it to "auto".
func TestToolLoopClient_ForcesToolChoiceAuto_WhenToolsPresent(t *testing.T) {
	inner := &scriptedClient{responses: []*scriptedResponse{{content: "hello"}}}
	c := pipeline.NewToolLoopClient(inner, []llm.Tool{newEchoTool("search")}, 0, 0)

	_, err := c.ChatCompletion(context.Background(), llm.WithMessages(llm.NewMessage(llm.RoleUser, "hi")))
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if len(inner.toolChoices) != 1 || inner.toolChoices[0] != llm.ToolChoiceAuto {
		t.Errorf("expected ToolChoiceAuto to be forced, got %v", inner.toolChoices)
	}
}

// After maxConsecutiveToolCalls rounds, the next call must force
// ToolChoiceNone instead of erroring out, and its response — whatever it
// is — must be returned as final without looping further, even if the
// model still tries to call a tool despite ToolChoiceNone.
func TestToolLoopClient_ForcesToolChoiceNone_AfterMaxConsecutiveToolCalls(t *testing.T) {
	tc := llm.NewToolCall("call-1", "search", `{}`)
	// Always returns a tool call — would loop forever without the soft cap.
	responses := make([]*scriptedResponse, 5)
	for i := range responses {
		responses[i] = &scriptedResponse{content: "final despite tool_calls", toolCalls: []llm.ToolCall{tc}}
	}
	inner := &scriptedClient{responses: responses}
	c := pipeline.NewToolLoopClient(inner, []llm.Tool{newEchoTool("search")}, 0, 2)

	resp, err := c.ChatCompletion(context.Background(), llm.WithMessages(llm.NewMessage(llm.RoleUser, "hi")))
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if resp.Message().Content() != "final despite tool_calls" {
		t.Errorf("expected the forced-final response to be returned, got %q", resp.Message().Content())
	}
	// 2 consecutive tool rounds (i=0,1) + 1 forced-final round (i=2) = 3 calls.
	if len(inner.calls) != 3 {
		t.Fatalf("expected exactly 3 inner calls, got %d", len(inner.calls))
	}
	if inner.toolChoices[0] != llm.ToolChoiceAuto || inner.toolChoices[1] != llm.ToolChoiceAuto {
		t.Errorf("expected the first 2 calls to use ToolChoiceAuto, got %v", inner.toolChoices[:2])
	}
	if inner.toolChoices[2] != llm.ToolChoiceNone {
		t.Errorf("expected the 3rd call to force ToolChoiceNone, got %v", inner.toolChoices[2])
	}
}

func TestToolLoopClient_DefaultMaxConsecutiveToolCalls_IsTwo(t *testing.T) {
	tc := llm.NewToolCall("call-1", "search", `{}`)
	responses := make([]*scriptedResponse, 5)
	for i := range responses {
		responses[i] = &scriptedResponse{content: "forced final", toolCalls: []llm.ToolCall{tc}}
	}
	inner := &scriptedClient{responses: responses}
	c := pipeline.NewToolLoopClient(inner, []llm.Tool{newEchoTool("search")}, 0, 0)

	if _, err := c.ChatCompletion(context.Background(), llm.WithMessages(llm.NewMessage(llm.RoleUser, "hi"))); err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if len(inner.calls) != 3 {
		t.Errorf("expected the default of 2 consecutive tool calls before forcing a final answer (3 calls total), got %d", len(inner.calls))
	}
}
