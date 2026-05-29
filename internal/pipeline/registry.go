package pipeline

import "github.com/bornholm/xolo/internal/core/model"

// Registry maps node types to their executors.
type Registry struct {
	executors map[model.PipelineNodeType]NodeExecutor
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{executors: make(map[model.PipelineNodeType]NodeExecutor)}
}

// Register adds or replaces the executor for a node type.
func (r *Registry) Register(t model.PipelineNodeType, e NodeExecutor) {
	r.executors[t] = e
}

// Get returns the executor for a node type.
func (r *Registry) Get(t model.PipelineNodeType) (NodeExecutor, bool) {
	e, ok := r.executors[t]
	return e, ok
}
