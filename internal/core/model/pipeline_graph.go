package model

import "encoding/json"

// PortType identifies the type of a pipeline node port.
type PortType string

const (
	PortTypeRequest  PortType = "request"
	PortTypeResponse PortType = "response"
	PortTypeNumber   PortType = "number"
	PortTypeString   PortType = "string"
	PortTypeBoolean  PortType = "boolean"
)

// PortDescriptor describes a named, typed input or output port on a pipeline node.
type PortDescriptor struct {
	Name     string   `json:"name"`
	Type     PortType `json:"type"`
	Required bool     `json:"required,omitempty"`
}

// PipelineNodeType identifies the kind of node in a pipeline graph.
type PipelineNodeType string

const (
	// Built-in node types (no gRPC binary).
	NodeTypeGenerator PipelineNodeType = "generator" // source: outputs request
	NodeTypeSink      PipelineNodeType = "sink"       // sink: inputs response
	NodeTypeModel     PipelineNodeType = "model"      // calls a real LLM
	NodeTypeValue     PipelineNodeType = "value"      // static value emitter

	// gRPC plugin node.
	NodeTypePlugin PipelineNodeType = "plugin"
)

// PipelineGraph is the dataflow graph stored inside a VirtualModel.
type PipelineGraph struct {
	Nodes []PipelineNode `json:"nodes"`
	Edges []PipelineEdge `json:"edges"`
}

// NodeIDs returns the ID of every node in the graph.
func (g *PipelineGraph) NodeIDs() []string {
	if g == nil {
		return nil
	}
	ids := make([]string, len(g.Nodes))
	for i, n := range g.Nodes {
		ids[i] = n.ID
	}
	return ids
}

// PipelineNode is a single node in a pipeline graph.
type PipelineNode struct {
	ID       string           `json:"id"`
	Type     PipelineNodeType `json:"type"`
	Position NodePosition     `json:"position"`
	// Data holds the node-type-specific configuration (JSON).
	Data json.RawMessage `json:"data,omitempty"`
}

// NodePosition is the visual position in the React Flow canvas.
type NodePosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// PipelineEdge connects a source output port to a target input port.
type PipelineEdge struct {
	ID         string `json:"id"`
	Source     string `json:"source"`
	SourcePort string `json:"sourcePort"`
	Target     string `json:"target"`
	TargetPort string `json:"targetPort"`
}

// PluginNodeData is the Data payload for NodeTypePlugin.
type PluginNodeData struct {
	PluginName string          `json:"pluginName"`
	Config     json.RawMessage `json:"config,omitempty"`
}

// ModelNodeData is the Data payload for NodeTypeModel.
// ProxyName is the static model proxy name used when the model_name input port
// is not connected. If the port is connected, the runtime value takes precedence.
//
// Passthrough marks the node as resolving the model that was actually requested
// by the caller (ExecutionContext.TargetModelName) instead of a fixed proxy name.
// It is used by Middleware pipelines, which wrap arbitrary target models
// dynamically. When Passthrough is true, ProxyName is ignored.
type ModelNodeData struct {
	ProxyName   string `json:"proxyName,omitempty"`
	Passthrough bool   `json:"passthrough,omitempty"`
}

// ValueNodeData is the Data payload for NodeTypeValue.
type ValueNodeData struct {
	// PortType determines the output port type and how Value is interpreted.
	PortType string `json:"portType"` // "string" | "number" | "boolean"
	// Value is the raw string representation of the value (parsed at runtime).
	Value string `json:"value"`
}

