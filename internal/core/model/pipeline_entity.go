package model

import "time"

// PipelineEntity is the common contract shared by any organization-owned entity
// that carries an editable pipeline graph (VirtualModel, Middleware).
//
// It lets the generic pipeline-graph HTTP handlers and secret cleanup operate
// uniformly over both kinds of entities without duplicating serialization,
// validation or persistence glue.
type PipelineEntity interface {
	// EntityID returns the stable string identifier of the entity.
	EntityID() string
	OrgID() OrgID
	Name() string
	Description() string
	Graph() *PipelineGraph
	CreatedAt() time.Time
	UpdatedAt() time.Time

	SetDescription(string)
	SetGraph(*PipelineGraph)
	SetUpdatedAt(time.Time)
}
