package pipeline

import (
	"encoding/json"

	"github.com/bornholm/xolo/internal/core/model"
)

// ValueContext accumulates the output values produced by executed nodes.
// Keys are (nodeID, portName) pairs. Values are of the port's declared type.
type ValueContext struct {
	values map[string]map[string]interface{} // nodeID → portName → value
}

func newValueContext() *ValueContext {
	return &ValueContext{values: make(map[string]map[string]interface{})}
}

// Set stores a value produced by nodeID on portName.
func (vc *ValueContext) Set(nodeID, portName string, value interface{}) {
	if vc.values[nodeID] == nil {
		vc.values[nodeID] = make(map[string]interface{})
	}
	vc.values[nodeID][portName] = value
}

// Get retrieves the value produced by nodeID on portName.
func (vc *ValueContext) Get(nodeID, portName string) (interface{}, bool) {
	m, ok := vc.values[nodeID]
	if !ok {
		return nil, false
	}
	v, ok := m[portName]
	return v, ok
}

// ResolveInputs builds the inputs map for targetNodeID by following incoming
// edges in the graph. The result maps targetPortName → value.
func (vc *ValueContext) ResolveInputs(graph *model.PipelineGraph, targetNodeID string) map[string]interface{} {
	inputs := make(map[string]interface{})
	for _, edge := range graph.Edges {
		if edge.Target != targetNodeID {
			continue
		}
		val, ok := vc.Get(edge.Source, edge.SourcePort)
		if ok {
			inputs[edge.TargetPort] = val
		}
	}
	return inputs
}

// InputsJSON serialises the inputs map to a JSON string for passing to plugins.
func InputsJSON(inputs map[string]interface{}) string {
	if len(inputs) == 0 {
		return "{}"
	}
	b, err := json.Marshal(inputs)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// ParseOutputsJSON deserialises a plugin's outputs_json into a map.
func ParseOutputsJSON(s string) map[string]interface{} {
	if s == "" || s == "{}" {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil
	}
	return m
}
