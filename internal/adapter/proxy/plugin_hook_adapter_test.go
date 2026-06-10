package proxy

import (
	"context"
	"testing"

	"github.com/bornholm/genai/llm"
	genaiProxy "github.com/bornholm/genai/proxy"
	"github.com/bornholm/xolo/internal/pipeline"
)

func TestApplyModifiedMessages_OpenAI(t *testing.T) {
	ec := pipeline.ExecutionContext{
		MessagesJSON: `[{"role":"user","content":"hi"}]`,
	}
	forwardExec := &pipeline.ForwardExecution{
		FinalMessagesJSON: `[{"role":"system","content":"injected"},{"role":"user","content":"hi"}]`,
	}
	req := &genaiProxy.ProxyRequest{Type: genaiProxy.RequestTypeChatCompletion}

	applyModifiedMessages(context.Background(), req, ec, forwardExec)

	if len(req.ChatOptions) != 1 {
		t.Fatalf("expected ChatOptions to be appended, got %d", len(req.ChatOptions))
	}

	opts := llm.NewChatCompletionOptions(req.ChatOptions...)
	if len(opts.Messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(opts.Messages))
	}
	if opts.Messages[0].Role() != llm.RoleSystem {
		t.Errorf("first message role = %q, want system", opts.Messages[0].Role())
	}
	if opts.Messages[0].Content() != "injected" {
		t.Errorf("first message content = %q, want injected", opts.Messages[0].Content())
	}
}

func TestApplyModifiedMessages_Anthropic(t *testing.T) {
	ec := pipeline.ExecutionContext{
		MessagesJSON: `[{"role":"user","content":"hi"}]`,
	}
	forwardExec := &pipeline.ForwardExecution{
		FinalMessagesJSON: `[{"role":"system","content":"injected"},{"role":"user","content":"hi"}]`,
	}
	req := &genaiProxy.ProxyRequest{Type: genaiProxy.RequestTypeMessage}

	applyModifiedMessages(context.Background(), req, ec, forwardExec)

	opts := llm.NewChatCompletionOptions(req.ChatOptions...)
	if len(opts.Messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(opts.Messages))
	}
	if opts.Messages[0].Role() != llm.RoleSystem {
		t.Errorf("first message role = %q, want system", opts.Messages[0].Role())
	}
}

func TestApplyModifiedMessages_NoChange(t *testing.T) {
	ec := pipeline.ExecutionContext{
		MessagesJSON: `[{"role":"user","content":"hi"}]`,
	}
	forwardExec := &pipeline.ForwardExecution{
		FinalMessagesJSON: ec.MessagesJSON,
	}
	req := &genaiProxy.ProxyRequest{Type: genaiProxy.RequestTypeChatCompletion}

	applyModifiedMessages(context.Background(), req, ec, forwardExec)

	if len(req.ChatOptions) != 0 {
		t.Errorf("expected no ChatOptions appended, got %d", len(req.ChatOptions))
	}
}

func TestApplyModifiedMessages_ConversionError(t *testing.T) {
	ec := pipeline.ExecutionContext{
		MessagesJSON: `[{"role":"user","content":"hi"}]`,
	}
	forwardExec := &pipeline.ForwardExecution{
		FinalMessagesJSON: `not json`,
	}
	req := &genaiProxy.ProxyRequest{Type: genaiProxy.RequestTypeChatCompletion}

	applyModifiedMessages(context.Background(), req, ec, forwardExec)

	if len(req.ChatOptions) != 0 {
		t.Errorf("expected no ChatOptions appended on conversion error, got %d", len(req.ChatOptions))
	}
}
