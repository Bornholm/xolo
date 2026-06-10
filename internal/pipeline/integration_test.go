package pipeline_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/bornholm/genai/llm"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/bornholm/xolo/internal/pipeline"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
	"google.golang.org/grpc"
)

// ─────────────────────────────────────────
// Test infrastructure
// ─────────────────────────────────────────

// FakeLLMClient implements llm.Client returning fixed content.
type FakeLLMClient struct {
	response string
}

func (c *FakeLLMClient) ChatCompletion(_ context.Context, _ ...llm.ChatCompletionOptionFunc) (llm.ChatCompletionResponse, error) {
	return &fakeChatResponse{content: c.response}, nil
}
func (c *FakeLLMClient) ChatCompletionStream(_ context.Context, _ ...llm.ChatCompletionOptionFunc) (<-chan llm.StreamChunk, error) {
	ch := make(chan llm.StreamChunk)
	close(ch)
	return ch, nil
}
func (c *FakeLLMClient) Embeddings(_ context.Context, _ []string, _ ...llm.EmbeddingsOptionFunc) (llm.EmbeddingsResponse, error) {
	return nil, nil
}

type fakeChatResponse struct{ content string }

func (r *fakeChatResponse) Message() llm.Message           { return &fakeMessage{r.content} }
func (r *fakeChatResponse) ToolCalls() []llm.ToolCall      { return nil }
func (r *fakeChatResponse) Usage() llm.ChatCompletionUsage { return nil }

type fakeMessage struct{ content string }

func (m *fakeMessage) Role() llm.Role               { return llm.RoleAssistant }
func (m *fakeMessage) Content() string              { return m.content }
func (m *fakeMessage) Attachments() []llm.Attachment { return nil }

// FakeModelResolver implements pipeline.ModelResolver returning a FakeLLMClient.
type FakeModelResolver struct {
	clients map[string]*FakeLLMClient
}

func (r *FakeModelResolver) ResolveRealModel(_ context.Context, _ model.OrgID, proxyName string) (llm.Client, string, model.LLMModelID, error) {
	if c, ok := r.clients[proxyName]; ok {
		return c, proxyName, "", nil
	}
	return &FakeLLMClient{response: "default"}, proxyName, "", nil
}

// FakeVirtualModelStore returns ErrNotFound for all lookups.
type FakeVirtualModelStore struct{}

func (s *FakeVirtualModelStore) CreateVirtualModel(_ context.Context, _ model.VirtualModel) error {
	return nil
}
func (s *FakeVirtualModelStore) GetVirtualModelByID(_ context.Context, _ model.VirtualModelID) (model.VirtualModel, error) {
	return nil, port.ErrNotFound
}
func (s *FakeVirtualModelStore) GetVirtualModelByName(_ context.Context, _ model.OrgID, _ string) (model.VirtualModel, error) {
	return nil, port.ErrNotFound
}
func (s *FakeVirtualModelStore) ListVirtualModels(_ context.Context, _ model.OrgID) ([]model.VirtualModel, error) {
	return nil, nil
}
func (s *FakeVirtualModelStore) SaveVirtualModel(_ context.Context, _ model.VirtualModel) error {
	return nil
}
func (s *FakeVirtualModelStore) DeleteVirtualModel(_ context.Context, _ model.VirtualModelID) error {
	return nil
}

// FakeXoloPluginClient implements proto.XoloPluginClient for tests.
type FakeXoloPluginClient struct {
	preRequestFn   func(ctx context.Context, in *proto.PreRequestInput) (*proto.PreRequestOutput, error)
	postResponseFn func(ctx context.Context, in *proto.PostResponseInput) (*proto.PostResponseOutput, error)
	resolveModelFn func(ctx context.Context, in *proto.ResolveModelInput) (*proto.ResolveModelOutput, error)
}

func (c *FakeXoloPluginClient) Describe(_ context.Context, _ *proto.DescribeRequest, _ ...grpc.CallOption) (*proto.PluginDescriptor, error) {
	return &proto.PluginDescriptor{}, nil
}
func (c *FakeXoloPluginClient) Initialize(_ context.Context, _ *proto.InitializeRequest, _ ...grpc.CallOption) (*proto.InitializeResponse, error) {
	return &proto.InitializeResponse{}, nil
}
func (c *FakeXoloPluginClient) PreRequest(ctx context.Context, in *proto.PreRequestInput, _ ...grpc.CallOption) (*proto.PreRequestOutput, error) {
	if c.preRequestFn != nil {
		return c.preRequestFn(ctx, in)
	}
	return &proto.PreRequestOutput{Allowed: true}, nil
}
func (c *FakeXoloPluginClient) PostResponse(ctx context.Context, in *proto.PostResponseInput, _ ...grpc.CallOption) (*proto.PostResponseOutput, error) {
	if c.postResponseFn != nil {
		return c.postResponseFn(ctx, in)
	}
	return &proto.PostResponseOutput{}, nil
}
func (c *FakeXoloPluginClient) ResolveModel(ctx context.Context, in *proto.ResolveModelInput, _ ...grpc.CallOption) (*proto.ResolveModelOutput, error) {
	if c.resolveModelFn != nil {
		return c.resolveModelFn(ctx, in)
	}
	return &proto.ResolveModelOutput{}, nil
}
func (c *FakeXoloPluginClient) ListModels(_ context.Context, _ *proto.ListModelsInput, _ ...grpc.CallOption) (*proto.ListModelsOutput, error) {
	return &proto.ListModelsOutput{}, nil
}

var _ proto.XoloPluginClient = (*FakeXoloPluginClient)(nil)

// staticPluginProvider implements pipeline.PluginProvider using static maps (for tests).
type staticPluginProvider struct {
	clients map[string]proto.XoloPluginClient
	descs   map[string]*proto.PluginDescriptor
}

func (p *staticPluginProvider) GetOrRestart(_ context.Context, name string) (proto.XoloPluginClient, *proto.PluginDescriptor, bool) {
	c, ok := p.clients[name]
	if !ok {
		return nil, nil, false
	}
	return c, p.descs[name], true
}

// buildEngine creates a pipeline engine with the standard executors.
func buildEngine(
	resolver pipeline.ModelResolver,
	vmStore port.VirtualModelStore,
	clients map[string]proto.XoloPluginClient,
	descs map[string]*proto.PluginDescriptor,
) *pipeline.Engine {
	reg := pipeline.NewRegistry()
	eng := pipeline.NewEngine(reg)
	reg.Register(model.NodeTypeGenerator, pipeline.NewGeneratorExecutor())
	reg.Register(model.NodeTypeSink, pipeline.NewSinkExecutor())
	reg.Register(model.NodeTypePlugin, pipeline.NewPluginExecutor(&staticPluginProvider{clients: clients, descs: descs}))
	reg.Register(model.NodeTypeModel, pipeline.NewModelExecutor(resolver, vmStore, eng))
	return eng
}

func buildEC() pipeline.ExecutionContext {
	return pipeline.ExecutionContext{
		OrgID:      "test-org",
		VisitedVMs: map[model.VirtualModelID]struct{}{},
	}
}

// ─────────────────────────────────────────
// Test 1: smart-model pipeline reconstitution
// ─────────────────────────────────────────

func TestPipeline_SmartModel(t *testing.T) {
	requestEvalDesc := &proto.PluginDescriptor{
		Name:         "request-evaluator",
		Capabilities: []proto.PluginDescriptor_Capability{proto.PluginDescriptor_PRE_REQUEST},
	}
	requestEvalClient := &FakeXoloPluginClient{
		preRequestFn: func(_ context.Context, in *proto.PreRequestInput) (*proto.PreRequestOutput, error) {
			outputs := map[string]interface{}{
				"complexity":      0.8,
				"category":        "code",
				"budget_pressure": 0.1,
			}
			b, _ := json.Marshal(outputs)
			return &proto.PreRequestOutput{Allowed: true, OutputsJson: string(b)}, nil
		},
	}

	fuzzyEvalDesc := &proto.PluginDescriptor{
		Name:         "fuzzy-evaluator",
		Capabilities: []proto.PluginDescriptor_Capability{proto.PluginDescriptor_PRE_REQUEST},
	}
	fuzzyEvalClient := &FakeXoloPluginClient{
		preRequestFn: func(_ context.Context, in *proto.PreRequestInput) (*proto.PreRequestOutput, error) {
			var inputs map[string]interface{}
			json.Unmarshal([]byte(in.InputsJson), &inputs)
			complexity, _ := inputs["complexity"].(float64)
			outputs := map[string]interface{}{"power_level": complexity}
			b, _ := json.Marshal(outputs)
			return &proto.PreRequestOutput{Allowed: true, OutputsJson: string(b)}, nil
		},
	}

	// Fake script-processor: reads power_level from inputs, applies threshold
	tengoRouterDesc := &proto.PluginDescriptor{
		Name:         "script-processor",
		Capabilities: []proto.PluginDescriptor_Capability{proto.PluginDescriptor_PRE_REQUEST},
	}
	tengoRouterClient := &FakeXoloPluginClient{
		preRequestFn: func(_ context.Context, in *proto.PreRequestInput) (*proto.PreRequestOutput, error) {
			var inputs map[string]interface{}
			json.Unmarshal([]byte(in.InputsJson), &inputs)
			powerLevel, _ := inputs["power_level"].(float64)
			modelName := "org/gpt4-mini"
			if powerLevel > 0.7 {
				modelName = "org/claude"
			}
			outputs := map[string]interface{}{"model_name": modelName}
			b, _ := json.Marshal(outputs)
			return &proto.PreRequestOutput{Allowed: true, OutputsJson: string(b)}, nil
		},
	}

	clients := map[string]proto.XoloPluginClient{
		"request-evaluator": requestEvalClient,
		"fuzzy-evaluator":   fuzzyEvalClient,
		"script-processor":      tengoRouterClient,
	}
	descs := map[string]*proto.PluginDescriptor{
		"request-evaluator": requestEvalDesc,
		"fuzzy-evaluator":   fuzzyEvalDesc,
		"script-processor":      tengoRouterDesc,
	}

	resolver := &FakeModelResolver{clients: map[string]*FakeLLMClient{
		"org/claude":    {response: "Hello from claude"},
		"org/gpt4-mini": {response: "Hello from gpt4-mini"},
	}}

	eng := buildEngine(resolver, &FakeVirtualModelStore{}, clients, descs)

	graph := &model.PipelineGraph{
		Nodes: []model.PipelineNode{
			{ID: "gen", Type: model.NodeTypeGenerator},
			{ID: "req-eval", Type: model.NodeTypePlugin, Data: mustJSON(model.PluginNodeData{PluginName: "request-evaluator"})},
			{ID: "fuzzy", Type: model.NodeTypePlugin, Data: mustJSON(model.PluginNodeData{PluginName: "fuzzy-evaluator"})},
			{ID: "router", Type: model.NodeTypePlugin, Data: mustJSON(model.PluginNodeData{PluginName: "script-processor"})},
			{ID: "mdl", Type: model.NodeTypeModel},
			{ID: "sink", Type: model.NodeTypeSink},
		},
		Edges: []model.PipelineEdge{
			{ID: "e1", Source: "gen", SourcePort: "request", Target: "req-eval", TargetPort: "request"},
			{ID: "e2", Source: "req-eval", SourcePort: "complexity", Target: "fuzzy", TargetPort: "complexity"},
			{ID: "e3", Source: "fuzzy", SourcePort: "power_level", Target: "router", TargetPort: "power_level"},
			{ID: "e4", Source: "router", SourcePort: "model_name", Target: "mdl", TargetPort: "model_name"},
			{ID: "e5", Source: "gen", SourcePort: "request", Target: "mdl", TargetPort: "request"},
			{ID: "e6", Source: "mdl", SourcePort: "response", Target: "sink", TargetPort: "response"},
		},
	}

	exec, err := eng.RunForward(context.Background(), graph, buildEC())
	if err != nil {
		t.Fatalf("RunForward failed: %v", err)
	}
	if exec.ResolvedClient == nil {
		t.Fatal("expected a resolved client, got nil")
	}
	// complexity=0.8 → power_level=0.8 > 0.7 → org/claude
	if exec.ResolvedModel != "org/claude" {
		t.Errorf("expected org/claude, got %q", exec.ResolvedModel)
	}
}

// ─────────────────────────────────────────
// Test 2: backward pass (pseudonymisation mock)
// ─────────────────────────────────────────

func TestPipeline_BackwardPass(t *testing.T) {
	pseudoDesc := &proto.PluginDescriptor{
		Name: "pseudonymizer",
		Capabilities: []proto.PluginDescriptor_Capability{
			proto.PluginDescriptor_PRE_REQUEST,
			proto.PluginDescriptor_POST_RESPONSE,
		},
	}
	const mapping = `{"John":"Person_A"}`
	pseudoClient := &FakeXoloPluginClient{
		preRequestFn: func(_ context.Context, _ *proto.PreRequestInput) (*proto.PreRequestOutput, error) {
			return &proto.PreRequestOutput{Allowed: true, NodeState: []byte(mapping)}, nil
		},
		postResponseFn: func(_ context.Context, in *proto.PostResponseInput) (*proto.PostResponseOutput, error) {
			modified := strings.ReplaceAll(in.ResponseContent, "Person_A", "John")
			return &proto.PostResponseOutput{ModifiedResponseContent: modified}, nil
		},
	}

	clients := map[string]proto.XoloPluginClient{"pseudonymizer": pseudoClient}
	descs := map[string]*proto.PluginDescriptor{"pseudonymizer": pseudoDesc}

	resolver := &FakeModelResolver{clients: map[string]*FakeLLMClient{
		"org/gpt4": {response: "Person_A est arrivé"},
	}}

	eng := buildEngine(resolver, &FakeVirtualModelStore{}, clients, descs)

	graph := &model.PipelineGraph{
		Nodes: []model.PipelineNode{
			{ID: "gen", Type: model.NodeTypeGenerator},
			{ID: "pseudo", Type: model.NodeTypePlugin, Data: mustJSON(model.PluginNodeData{PluginName: "pseudonymizer"})},
			{ID: "mdl", Type: model.NodeTypeModel, Data: mustJSON(model.ModelNodeData{ProxyName: "org/gpt4"})},
			{ID: "sink", Type: model.NodeTypeSink},
		},
		Edges: []model.PipelineEdge{
			{ID: "e1", Source: "gen", SourcePort: "request", Target: "pseudo", TargetPort: "request"},
			{ID: "e2", Source: "pseudo", SourcePort: "request", Target: "mdl", TargetPort: "request"},
			{ID: "e3", Source: "mdl", SourcePort: "response", Target: "sink", TargetPort: "response"},
		},
	}

	exec, err := eng.RunForward(context.Background(), graph, buildEC())
	if err != nil {
		t.Fatalf("RunForward failed: %v", err)
	}

	final, err := eng.RunBackward(context.Background(), exec, "Person_A est arrivé", nil, false)
	if err != nil {
		t.Fatalf("RunBackward failed: %v", err)
	}
	if final != "John est arrivé" {
		t.Errorf("expected 'John est arrivé', got %q", final)
	}
}

// ─────────────────────────────────────────
// Test 3: graph validation
// ─────────────────────────────────────────

func TestPipeline_GraphValidation(t *testing.T) {
	eng := buildEngine(&FakeModelResolver{}, &FakeVirtualModelStore{}, nil, nil)

	t.Run("no_generator", func(t *testing.T) {
		graph := &model.PipelineGraph{
			Nodes: []model.PipelineNode{{ID: "sink", Type: model.NodeTypeSink}},
		}
		_, err := eng.RunForward(context.Background(), graph, buildEC())
		if err == nil {
			t.Fatal("expected error for graph without generator")
		}
	})

	t.Run("no_sink", func(t *testing.T) {
		graph := &model.PipelineGraph{
			Nodes: []model.PipelineNode{{ID: "gen", Type: model.NodeTypeGenerator}},
		}
		_, err := eng.RunForward(context.Background(), graph, buildEC())
		if err == nil {
			t.Fatal("expected error for graph without sink")
		}
	})

	t.Run("cycle", func(t *testing.T) {
		graph := &model.PipelineGraph{
			Nodes: []model.PipelineNode{
				{ID: "gen", Type: model.NodeTypeGenerator},
				{ID: "a", Type: model.NodeTypeSink},
				{ID: "b", Type: model.NodeTypeSink},
				{ID: "sink", Type: model.NodeTypeSink},
			},
			Edges: []model.PipelineEdge{
				{ID: "e1", Source: "a", SourcePort: "out", Target: "b", TargetPort: "in"},
				{ID: "e2", Source: "b", SourcePort: "out", Target: "a", TargetPort: "in"},
			},
		}
		_, err := eng.RunForward(context.Background(), graph, buildEC())
		if err == nil {
			t.Fatal("expected error for cyclic graph")
		}
	})
}

// ─────────────────────────────────────────
// Test 4: chaining VirtualModels
// ─────────────────────────────────────────

func TestPipeline_ChainedVirtualModels(t *testing.T) {
	resolver := &FakeModelResolver{clients: map[string]*FakeLLMClient{
		"org/claude": {response: "chained response"},
	}}

	innerGraph := &model.PipelineGraph{
		Nodes: []model.PipelineNode{
			{ID: "gen", Type: model.NodeTypeGenerator},
			{ID: "mdl", Type: model.NodeTypeModel, Data: mustJSON(model.ModelNodeData{ProxyName: "org/claude"})},
			{ID: "sink", Type: model.NodeTypeSink},
		},
		Edges: []model.PipelineEdge{
			{ID: "e1", Source: "gen", SourcePort: "request", Target: "mdl", TargetPort: "request"},
			{ID: "e2", Source: "mdl", SourcePort: "response", Target: "sink", TargetPort: "response"},
		},
	}

	innerVM := model.NewVirtualModel("test-org", "inner", "inner vm")
	innerVM.SetGraph(innerGraph)

	vmStore := &fixedVirtualModelStore{vm: innerVM}
	eng := buildEngine(resolver, vmStore, nil, nil)

	outerGraph := &model.PipelineGraph{
		Nodes: []model.PipelineNode{
			{ID: "gen", Type: model.NodeTypeGenerator},
			{ID: "mdl", Type: model.NodeTypeModel, Data: mustJSON(model.ModelNodeData{ProxyName: "test-org/inner"})},
			{ID: "sink", Type: model.NodeTypeSink},
		},
		Edges: []model.PipelineEdge{
			{ID: "e1", Source: "gen", SourcePort: "request", Target: "mdl", TargetPort: "request"},
			{ID: "e2", Source: "mdl", SourcePort: "response", Target: "sink", TargetPort: "response"},
		},
	}

	ec := pipeline.ExecutionContext{OrgID: "test-org", VisitedVMs: map[model.VirtualModelID]struct{}{}}
	exec, err := eng.RunForward(context.Background(), outerGraph, ec)
	if err != nil {
		t.Fatalf("RunForward failed: %v", err)
	}
	if exec.ResolvedClient == nil {
		t.Fatal("expected resolved client from chained VM")
	}
}

// ─────────────────────────────────────────
// Test: FinalMessagesJSON propagation from a plugin's modified_messages_json
// ─────────────────────────────────────────

func TestPipeline_FinalMessagesJSON(t *testing.T) {
	const modifiedMessages = `[{"role":"system","content":"injected"},{"role":"user","content":"hi"}]`

	rewriterDesc := &proto.PluginDescriptor{
		Name:         "message-rewriter",
		Capabilities: []proto.PluginDescriptor_Capability{proto.PluginDescriptor_PRE_REQUEST},
	}
	rewriterClient := &FakeXoloPluginClient{
		preRequestFn: func(_ context.Context, _ *proto.PreRequestInput) (*proto.PreRequestOutput, error) {
			return &proto.PreRequestOutput{Allowed: true, ModifiedMessagesJson: modifiedMessages}, nil
		},
	}

	clients := map[string]proto.XoloPluginClient{"message-rewriter": rewriterClient}
	descs := map[string]*proto.PluginDescriptor{"message-rewriter": rewriterDesc}

	resolver := &FakeModelResolver{clients: map[string]*FakeLLMClient{
		"org/claude": {response: "ok"},
	}}

	eng := buildEngine(resolver, &FakeVirtualModelStore{}, clients, descs)

	graph := &model.PipelineGraph{
		Nodes: []model.PipelineNode{
			{ID: "gen", Type: model.NodeTypeGenerator},
			{ID: "rewriter", Type: model.NodeTypePlugin, Data: mustJSON(model.PluginNodeData{PluginName: "message-rewriter"})},
			{ID: "mdl", Type: model.NodeTypeModel, Data: mustJSON(model.ModelNodeData{ProxyName: "org/claude"})},
			{ID: "sink", Type: model.NodeTypeSink},
		},
		Edges: []model.PipelineEdge{
			{ID: "e1", Source: "gen", SourcePort: "request", Target: "rewriter", TargetPort: "request"},
			{ID: "e2", Source: "gen", SourcePort: "request", Target: "mdl", TargetPort: "request"},
			{ID: "e3", Source: "mdl", SourcePort: "response", Target: "sink", TargetPort: "response"},
		},
	}

	ec := buildEC()
	ec.MessagesJSON = `[{"role":"user","content":"<system-reminder>foo</system-reminder>hi"}]`

	exec, err := eng.RunForward(context.Background(), graph, ec)
	if err != nil {
		t.Fatalf("RunForward failed: %v", err)
	}
	if exec.FinalMessagesJSON != modifiedMessages {
		t.Errorf("FinalMessagesJSON = %q, want %q", exec.FinalMessagesJSON, modifiedMessages)
	}
}

func TestPipeline_CycleDetected(t *testing.T) {
	vmA := model.NewVirtualModel("test-org", "a", "vm A")
	vmAID := vmA.ID()
	vmStore := &fixedVirtualModelStore{vm: vmA}

	eng := buildEngine(&FakeModelResolver{}, vmStore, nil, nil)

	graph := &model.PipelineGraph{
		Nodes: []model.PipelineNode{
			{ID: "gen", Type: model.NodeTypeGenerator},
			{ID: "mdl", Type: model.NodeTypeModel, Data: mustJSON(model.ModelNodeData{ProxyName: "test-org/a"})},
			{ID: "sink", Type: model.NodeTypeSink},
		},
		Edges: []model.PipelineEdge{
			{ID: "e1", Source: "gen", SourcePort: "request", Target: "mdl", TargetPort: "request"},
			{ID: "e2", Source: "mdl", SourcePort: "response", Target: "sink", TargetPort: "response"},
		},
	}

	// vmA is already in VisitedVMs → cycle
	ec := pipeline.ExecutionContext{
		OrgID:      "test-org",
		VisitedVMs: map[model.VirtualModelID]struct{}{vmAID: {}},
	}
	_, err := eng.RunForward(context.Background(), graph, ec)
	if err == nil {
		t.Fatal("expected cycle detection error")
	}
}

// ─────────────────────────────────────────
// Test 5: VirtualModel without graph
// ─────────────────────────────────────────

func TestPipeline_ModelWithoutGraph(t *testing.T) {
	vmNoGraph := model.NewVirtualModel("test-org", "no-graph", "vm without graph")
	// vmNoGraph.Graph() == nil (no SetGraph called)
	vmStore := &fixedVirtualModelStore{vm: vmNoGraph}
	eng := buildEngine(&FakeModelResolver{}, vmStore, nil, nil)

	graph := &model.PipelineGraph{
		Nodes: []model.PipelineNode{
			{ID: "gen", Type: model.NodeTypeGenerator},
			{ID: "mdl", Type: model.NodeTypeModel, Data: mustJSON(model.ModelNodeData{ProxyName: "test-org/no-graph"})},
			{ID: "sink", Type: model.NodeTypeSink},
		},
		Edges: []model.PipelineEdge{
			{ID: "e1", Source: "gen", SourcePort: "request", Target: "mdl", TargetPort: "request"},
			{ID: "e2", Source: "mdl", SourcePort: "response", Target: "sink", TargetPort: "response"},
		},
	}

	ec := pipeline.ExecutionContext{OrgID: "test-org", VisitedVMs: map[model.VirtualModelID]struct{}{}}
	_, err := eng.RunForward(context.Background(), graph, ec)
	if err == nil {
		t.Fatal("expected error when VM has no pipeline")
	}
}

// ─────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────

func mustJSON(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// fixedVirtualModelStore returns the same VM for any matching name lookup.
type fixedVirtualModelStore struct{ vm model.VirtualModel }

func (s *fixedVirtualModelStore) CreateVirtualModel(_ context.Context, _ model.VirtualModel) error {
	return nil
}
func (s *fixedVirtualModelStore) GetVirtualModelByID(_ context.Context, id model.VirtualModelID) (model.VirtualModel, error) {
	if s.vm != nil && s.vm.ID() == id {
		return s.vm, nil
	}
	return nil, port.ErrNotFound
}
func (s *fixedVirtualModelStore) GetVirtualModelByName(_ context.Context, _ model.OrgID, name string) (model.VirtualModel, error) {
	if s.vm != nil && s.vm.Name() == name {
		return s.vm, nil
	}
	return nil, port.ErrNotFound
}
func (s *fixedVirtualModelStore) ListVirtualModels(_ context.Context, _ model.OrgID) ([]model.VirtualModel, error) {
	if s.vm != nil {
		return []model.VirtualModel{s.vm}, nil
	}
	return nil, nil
}
func (s *fixedVirtualModelStore) SaveVirtualModel(_ context.Context, _ model.VirtualModel) error {
	return nil
}
func (s *fixedVirtualModelStore) DeleteVirtualModel(_ context.Context, _ model.VirtualModelID) error {
	return nil
}
