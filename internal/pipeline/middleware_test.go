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
