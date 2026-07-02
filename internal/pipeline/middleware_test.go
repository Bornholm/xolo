package pipeline_test

import (
	"context"
	"strings"
	"testing"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/pipeline"
	"github.com/bornholm/xolo/internal/pipeline/pipelinetest"
	"github.com/pkg/errors"
)

// A passthrough model node resolves the caller's requested (real) model.
func TestMiddleware_PassthroughRealModel(t *testing.T) {
	resolver := pipelinetest.NewModelResolver().
		WithResponse("org/gpt4", "hi from gpt4")

	graph := pipelinetest.NewGraph().
		Generator("gen").
		ModelPassthrough("mdl").
		Sink("sink").
		Edge("gen", "request", "mdl", "request").
		Edge("mdl", "response", "sink", "response").
		Build()

	h := pipelinetest.New(pipelinetest.WithModelResolver(resolver))

	exec, err := h.Engine().RunForward(context.Background(), graph,
		pipelinetest.NewExecutionContext(pipelinetest.WithTargetModel("org/gpt4")))
	if err != nil {
		t.Fatalf("RunForward failed: %v", err)
	}
	if exec.ResolvedModel != "org/gpt4" {
		t.Errorf("expected org/gpt4, got %q", exec.ResolvedModel)
	}
}

// A passthrough model node resolves a target that is itself a virtual model,
// recursing into its pipeline.
func TestMiddleware_PassthroughVirtualModel(t *testing.T) {
	resolver := pipelinetest.NewModelResolver().
		WithResponse("org/claude", "hi from claude")

	innerGraph := pipelinetest.NewGraph().
		Generator("gen").
		ModelWithProxy("mdl", "org/claude").
		Sink("sink").
		Edge("gen", "request", "mdl", "request").
		Edge("mdl", "response", "sink", "response").
		Build()

	innerVM := model.NewVirtualModel("test-org", "inner", "inner vm")
	innerVM.SetGraph(innerGraph)

	graph := pipelinetest.NewGraph().
		Generator("gen").
		ModelPassthrough("mdl").
		Sink("sink").
		Edge("gen", "request", "mdl", "request").
		Edge("mdl", "response", "sink", "response").
		Build()

	h := pipelinetest.New(
		pipelinetest.WithModelResolver(resolver),
		pipelinetest.WithVirtualModelStore(pipelinetest.NewVirtualModelStore(innerVM)),
	)

	exec, err := h.Engine().RunForward(context.Background(), graph,
		pipelinetest.NewExecutionContext(pipelinetest.WithTargetModel("test-org/inner")))
	if err != nil {
		t.Fatalf("RunForward failed: %v", err)
	}
	if exec.ResolvedModel != "org/claude" {
		t.Errorf("expected org/claude (via VM), got %q", exec.ResolvedModel)
	}
}

// Two chained middlewares: the outer passthrough runs the inner middleware,
// whose own passthrough resolves the real target model.
func TestMiddleware_Chaining(t *testing.T) {
	resolver := pipelinetest.NewModelResolver().
		WithResponse("org/gpt4", "hi")

	// inner middleware uses the default seeded passthrough graph.
	innerMW := model.NewMiddleware("test-org", "inner", "")

	outerGraph := pipelinetest.NewGraph().
		Generator("gen").
		ModelPassthrough("mdl").
		Sink("sink").
		Edge("gen", "request", "mdl", "request").
		Edge("mdl", "response", "sink", "response").
		Build()

	h := pipelinetest.New(pipelinetest.WithModelResolver(resolver))

	ec := pipelinetest.NewExecutionContext(
		pipelinetest.WithTargetModel("org/gpt4"),
		pipelinetest.WithPendingMiddlewares(innerMW),
	)

	exec, err := h.Engine().RunForward(context.Background(), outerGraph, ec)
	if err != nil {
		t.Fatalf("RunForward failed: %v", err)
	}
	if exec.ResolvedModel != "org/gpt4" {
		t.Errorf("expected org/gpt4 through the chain, got %q", exec.ResolvedModel)
	}
}

// A middleware whose plugin rejects the request aborts the whole chain.
func TestMiddleware_Rejection(t *testing.T) {
	plugins := pipelinetest.NewPluginProvider().
		Register("time-restriction", pipelinetest.PreRequestDescriptor("time-restriction"),
			pipelinetest.RejectPreRequest("hors des créneaux autorisés"))

	resolver := pipelinetest.NewModelResolver().
		WithResponse("org/gpt4", "hi")

	graph := pipelinetest.NewGraph().
		Generator("gen").
		Plugin("gate", "time-restriction").
		ModelPassthrough("mdl").
		Sink("sink").
		Edge("gen", "request", "gate", "request").
		Edge("gate", "request", "mdl", "request").
		Edge("mdl", "response", "sink", "response").
		Build()

	h := pipelinetest.New(
		pipelinetest.WithPlugins(plugins),
		pipelinetest.WithModelResolver(resolver),
	)

	_, err := h.Engine().RunForward(context.Background(), graph,
		pipelinetest.NewExecutionContext(pipelinetest.WithTargetModel("org/gpt4")))
	if err == nil {
		t.Fatal("expected rejection error, got nil")
	}
	var rejErr *pipeline.RejectionError
	if !errors.As(err, &rejErr) {
		t.Fatalf("expected RejectionError, got %T: %v", err, err)
	}
	if !strings.Contains(rejErr.Reason, "créneaux") {
		t.Errorf("unexpected rejection reason: %q", rejErr.Reason)
	}
}

// In a middleware chain (ForcePassthrough), even a model node with a fixed proxy
// name resolves the requested model instead of the fixed one.
func TestMiddleware_ForcePassthroughOverridesFixedModel(t *testing.T) {
	resolver := pipelinetest.NewModelResolver().
		WithResponse("org/requested", "ok").
		WithResponse("org/fixed", "should not be used")

	graph := pipelinetest.NewGraph().
		Generator("gen").
		ModelWithProxy("mdl", "org/fixed").
		Sink("sink").
		Edge("gen", "request", "mdl", "request").
		Edge("mdl", "response", "sink", "response").
		Build()

	h := pipelinetest.New(pipelinetest.WithModelResolver(resolver))

	ec := pipelinetest.NewExecutionContext(pipelinetest.WithTargetModel("org/requested"))
	ec.ForcePassthrough = true

	exec, err := h.Engine().RunForward(context.Background(), graph, ec)
	if err != nil {
		t.Fatalf("RunForward failed: %v", err)
	}
	if exec.ResolvedModel != "org/requested" {
		t.Errorf("expected forced passthrough to org/requested, got %q", exec.ResolvedModel)
	}
}

// A middleware wrapping a virtual model must not swallow the message
// modifications performed *inside* that VM's pipeline (e.g. pseudonymization):
// the modified messages have to reach the top-level FinalMessagesJSON so they
// are sent to the provider. Regression test for nested-pipeline forward state.
func TestMiddleware_PreservesWrappedVMMessageModifications(t *testing.T) {
	const modified = `[{"role":"system","content":"anon"},{"role":"user","content":"REDACTED"}]`

	plugins := pipelinetest.NewPluginProvider().
		Register("pseudonymizer", pipelinetest.PreRequestDescriptor("pseudonymizer"),
			pipelinetest.ModifiedMessages(modified))

	resolver := pipelinetest.NewModelResolver().WithResponse("org/gpt4", "ok")

	// Virtual model "anon" pseudonymizes then forwards to a real model.
	innerGraph := pipelinetest.NewGraph().
		Generator("gen").
		Plugin("pseudo", "pseudonymizer").
		ModelWithProxy("mdl", "org/gpt4").
		Sink("sink").
		Edge("gen", "request", "pseudo", "request").
		Edge("pseudo", "request", "mdl", "request").
		Edge("mdl", "response", "sink", "response").
		Build()

	anonVM := model.NewVirtualModel("test-org", "anon", "")
	anonVM.SetGraph(innerGraph)

	// Middleware chain: a passthrough graph wrapping the requested model.
	mwGraph := pipelinetest.NewGraph().
		Generator("gen").
		ModelPassthrough("mdl").
		Sink("sink").
		Edge("gen", "request", "mdl", "request").
		Edge("mdl", "response", "sink", "response").
		Build()

	h := pipelinetest.New(
		pipelinetest.WithPlugins(plugins),
		pipelinetest.WithModelResolver(resolver),
		pipelinetest.WithVirtualModelStore(pipelinetest.NewVirtualModelStore(anonVM)),
	)

	ec := pipelinetest.NewExecutionContext(
		pipelinetest.WithTargetModel("test-org/anon"),
		pipelinetest.WithMessagesJSON(`[{"role":"user","content":"jean.dupont@example.com"}]`),
	)
	ec.ForcePassthrough = true

	exec, err := h.Engine().RunForward(context.Background(), mwGraph, ec)
	if err != nil {
		t.Fatalf("RunForward failed: %v", err)
	}
	if exec.FinalMessagesJSON != modified {
		t.Errorf("wrapped VM message modification lost: FinalMessagesJSON = %q, want %q", exec.FinalMessagesJSON, modified)
	}
}

// A middleware wrapping a virtual model must also run that VM's backward
// (post-response) pass — e.g. de-pseudonymization of the response. Regression
// test for nested-pipeline backward state (spliced ExecutedNodes).
func TestMiddleware_PreservesWrappedVMBackwardPass(t *testing.T) {
	pseudo := pipelinetest.JSONPreRequestWithState(func(_ map[string]any) (map[string]any, []byte) {
		return nil, []byte("state")
	})
	pseudo.PostResponseFunc = pipelinetest.PostResponseRewrite(func(_ []byte, content string) string {
		return strings.ReplaceAll(content, "Person_A", "John")
	})

	plugins := pipelinetest.NewPluginProvider().
		Register("pseudonymizer", pipelinetest.PrePostDescriptor("pseudonymizer"), pseudo)

	resolver := pipelinetest.NewModelResolver().WithResponse("org/gpt4", "Person_A est arrivé")

	innerGraph := pipelinetest.NewGraph().
		Generator("gen").
		Plugin("pseudo", "pseudonymizer").
		ModelWithProxy("mdl", "org/gpt4").
		Sink("sink").
		Edge("gen", "request", "pseudo", "request").
		Edge("pseudo", "request", "mdl", "request").
		Edge("mdl", "response", "sink", "response").
		Build()

	anonVM := model.NewVirtualModel("test-org", "anon", "")
	anonVM.SetGraph(innerGraph)

	mwGraph := pipelinetest.NewGraph().
		Generator("gen").
		ModelPassthrough("mdl").
		Sink("sink").
		Edge("gen", "request", "mdl", "request").
		Edge("mdl", "response", "sink", "response").
		Build()

	h := pipelinetest.New(
		pipelinetest.WithPlugins(plugins),
		pipelinetest.WithModelResolver(resolver),
		pipelinetest.WithVirtualModelStore(pipelinetest.NewVirtualModelStore(anonVM)),
	)

	ec := pipelinetest.NewExecutionContext(pipelinetest.WithTargetModel("test-org/anon"))
	ec.ForcePassthrough = true

	result, err := h.Run(context.Background(), mwGraph, ec)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.FinalContent != "John est arrivé" {
		t.Errorf("wrapped VM backward pass did not run: FinalContent = %q, want %q", result.FinalContent, "John est arrivé")
	}
}

// Without a target model, a passthrough node errors out clearly.
func TestMiddleware_PassthroughNoTarget(t *testing.T) {
	graph := pipelinetest.NewGraph().
		Generator("gen").
		ModelPassthrough("mdl").
		Sink("sink").
		Edge("gen", "request", "mdl", "request").
		Edge("mdl", "response", "sink", "response").
		Build()

	h := pipelinetest.New()

	_, err := h.Engine().RunForward(context.Background(), graph, pipelinetest.NewExecutionContext())
	if err == nil {
		t.Fatal("expected error when no target model is set")
	}
}
