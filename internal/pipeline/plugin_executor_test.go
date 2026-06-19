package pipeline_test

import (
	"context"
	"testing"

	"github.com/bornholm/genai/llm"
	"github.com/bornholm/xolo/internal/pipeline/pipelinetest"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
)

// TestPipeline_ToolProviderPlugin wires Generator -> Plugin(TOOL_PROVIDER) ->
// Model -> Sink and verifies that the resolved client (a) calls the
// plugin's CallTool RPC to resolve a matching tool_call, and (b) loops back
// to the underlying model client until a final answer is produced.
func TestPipeline_ToolProviderPlugin(t *testing.T) {
	var callToolInvocations []*proto.CallToolInput

	pluginClient := &pipelinetest.PluginClient{
		ListToolsFunc: func(_ context.Context, _ *proto.ListToolsInput) (*proto.ListToolsOutput, error) {
			return &proto.ListToolsOutput{
				Tools: []*proto.ToolDescriptor{
					{Name: "search", Description: "search the web"},
				},
			}, nil
		},
		CallToolFunc: func(_ context.Context, in *proto.CallToolInput) (*proto.CallToolOutput, error) {
			callToolInvocations = append(callToolInvocations, in)
			return &proto.CallToolOutput{ResultText: "42"}, nil
		},
	}

	plugins := pipelinetest.NewPluginProvider().
		Register("tool-bridge", pipelinetest.ToolProviderDescriptor("tool-bridge"), pluginClient)

	toolCall := llm.NewToolCall("call-1", "search", `{"q":"life, the universe and everything"}`)
	modelClient := &scriptedClient{responses: []*scriptedResponse{
		{toolCalls: []llm.ToolCall{toolCall}},
		{content: "the answer is 42"},
	}}
	resolver := pipelinetest.NewModelResolver().WithModel("org/gpt", modelClient)

	graph := pipelinetest.NewGraph().
		Generator("gen").
		Plugin("tools", "tool-bridge").
		ModelWithProxy("mdl", "org/gpt").
		Sink("sink").
		Edge("gen", "request", "tools", "request").
		Edge("tools", "request", "mdl", "request").
		Edge("mdl", "response", "sink", "response").
		Build()

	h := pipelinetest.New(
		pipelinetest.WithPlugins(plugins),
		pipelinetest.WithModelResolver(resolver),
	)

	exec, err := h.Engine().RunForward(context.Background(), graph, pipelinetest.NewExecutionContext())
	if err != nil {
		t.Fatalf("RunForward failed: %v", err)
	}
	if exec.ResolvedClient == nil {
		t.Fatal("expected a resolved client, got nil")
	}

	resp, err := exec.ResolvedClient.ChatCompletion(context.Background(), llm.WithMessages(llm.NewMessage(llm.RoleUser, "what is the answer?")))
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}

	if resp.Message().Content() != "the answer is 42" {
		t.Errorf("expected final answer, got %q", resp.Message().Content())
	}
	if len(callToolInvocations) != 1 {
		t.Fatalf("expected exactly 1 CallTool invocation, got %d", len(callToolInvocations))
	}
	if callToolInvocations[0].Name != "search" {
		t.Errorf("expected CallTool for %q, got %q", "search", callToolInvocations[0].Name)
	}
	if len(modelClient.calls) != 2 {
		t.Errorf("expected 2 model calls (tool round + final), got %d", len(modelClient.calls))
	}
}
