package pipeline

import (
	"context"
	"fmt"

	"github.com/bornholm/genai/llm"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/pkg/errors"
)

// Engine executes pipeline graphs.
type Engine struct {
	registry *Registry
}

// NewEngine creates an Engine backed by the given registry.
func NewEngine(registry *Registry) *Engine {
	return &Engine{registry: registry}
}

// ForwardExecution is the result of running the forward pass of a pipeline.
type ForwardExecution struct {
	// ResolvedClient is the llm.Client returned by the terminal model node.
	ResolvedClient llm.Client
	// ResolvedModel is the real model name to forward to the provider.
	ResolvedModel string
	// ResolvedModelID is the internal database ID of the resolved LLM model.
	ResolvedModelID model.LLMModelID
	// ExecutedNodes is the ordered list of nodes that ran (for backward pass).
	ExecutedNodes []ExecutedNode
}

// ExecutedNode pairs a node with the opaque state returned by its Forward call.
type ExecutedNode struct {
	Node      model.PipelineNode
	NodeState []byte
}

// RunForward executes the graph in topological order (forward pass).
// It returns after the first terminal (model) node resolves an LLM client.
func (e *Engine) RunForward(ctx context.Context, graph *model.PipelineGraph, ec ExecutionContext) (*ForwardExecution, error) {
	if err := validateGraph(graph); err != nil {
		return nil, errors.Wrap(err, "invalid pipeline graph")
	}

	order, err := topoSort(graph)
	if err != nil {
		return nil, errors.Wrap(err, "pipeline graph has a cycle")
	}

	vc := newValueContext()
	var executed []ExecutedNode

	// Seed the generator node's output.
	for _, node := range graph.Nodes {
		if node.Type == model.NodeTypeGenerator {
			vc.Set(node.ID, "request", ec.RequestJSON)
		}
	}

	for _, nodeID := range order {
		node := nodeByID(graph, nodeID)
		if node == nil {
			continue
		}

		exec, ok := e.registry.Get(node.Type)
		if !ok {
			return nil, fmt.Errorf("no executor registered for node type %q (node %s)", node.Type, node.ID)
		}

		inputs := vc.ResolveInputs(graph, node.ID)
		result, err := exec.Forward(ctx, *node, inputs, ec)
		if err != nil {
			return nil, errors.Wrapf(err, "node %s (%s) forward failed", node.ID, node.Type)
		}

		if result.Rejected {
			return nil, &RejectionError{Reason: result.RejectionReason}
		}

		executed = append(executed, ExecutedNode{Node: *node, NodeState: result.NodeState})

		// Store output values in the value context.
		for port, val := range result.OutputValues {
			vc.Set(node.ID, port, val)
		}

		// Terminal: model node resolved the LLM client.
		if result.ResolvedClient != nil {
			return &ForwardExecution{
				ResolvedClient:  result.ResolvedClient,
				ResolvedModel:   result.ResolvedModel,
				ResolvedModelID: result.ResolvedModelID,
				ExecutedNodes:   executed,
			}, nil
		}
	}

	return nil, errors.New("pipeline graph has no terminal model node")
}

// RunBackward executes nodes in reverse order (post-response pass).
// It returns the (potentially modified) response content.
func (e *Engine) RunBackward(
	ctx context.Context,
	exec *ForwardExecution,
	responseContent string,
	tokens *TokensUsed,
	hadError bool,
) (string, error) {
	current := responseContent
	// Iterate in reverse order.
	for i := len(exec.ExecutedNodes) - 1; i >= 0; i-- {
		en := exec.ExecutedNodes[i]
		ex, ok := e.registry.Get(en.Node.Type)
		if !ok {
			continue
		}
		result, err := ex.Backward(ctx, en.Node, en.NodeState, current, tokens, hadError)
		if err != nil {
			// Non-fatal: log and continue.
			continue
		}
		if result.ModifiedResponseContent != "" {
			current = result.ModifiedResponseContent
		}
	}
	return current, nil
}

// RejectionError is returned when a pipeline node blocks the request.
type RejectionError struct {
	Reason string
}

func (e *RejectionError) Error() string {
	if e.Reason != "" {
		return "request rejected by pipeline: " + e.Reason
	}
	return "request rejected by pipeline"
}

// validateGraph checks graph structure: generator + sink present, sink connected.
func validateGraph(g *model.PipelineGraph) error {
	hasGenerator := false
	var sinkIDs []string
	for _, n := range g.Nodes {
		switch n.Type {
		case model.NodeTypeGenerator:
			hasGenerator = true
		case model.NodeTypeSink:
			sinkIDs = append(sinkIDs, n.ID)
		}
	}
	if !hasGenerator {
		return errors.New("pipeline must have a generator node")
	}
	if len(sinkIDs) == 0 {
		return errors.New("pipeline must have a sink node")
	}
	// Every sink must have at least one incoming edge.
	for _, sinkID := range sinkIDs {
		connected := false
		for _, e := range g.Edges {
			if e.Target == sinkID {
				connected = true
				break
			}
		}
		if !connected {
			return errors.New("sink node has no incoming edge: connect a terminal node's response port to the sink")
		}
	}
	return nil
}

// topoSort returns nodes in topological order using Kahn's algorithm.
func topoSort(g *model.PipelineGraph) ([]string, error) {
	inDegree := make(map[string]int, len(g.Nodes))
	adj := make(map[string][]string, len(g.Nodes))
	for _, n := range g.Nodes {
		inDegree[n.ID] = 0
		adj[n.ID] = nil
	}
	for _, e := range g.Edges {
		adj[e.Source] = append(adj[e.Source], e.Target)
		inDegree[e.Target]++
	}

	queue := make([]string, 0, len(g.Nodes))
	for _, n := range g.Nodes {
		if inDegree[n.ID] == 0 {
			queue = append(queue, n.ID)
		}
	}

	var order []string
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		order = append(order, cur)
		for _, next := range adj[cur] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	if len(order) != len(g.Nodes) {
		return nil, errors.New("cycle detected in pipeline graph")
	}
	return order, nil
}

func nodeByID(g *model.PipelineGraph, id string) *model.PipelineNode {
	for i := range g.Nodes {
		if g.Nodes[i].ID == id {
			return &g.Nodes[i]
		}
	}
	return nil
}
