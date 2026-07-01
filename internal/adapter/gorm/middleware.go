package gorm

import (
	"encoding/json"
	"time"

	"github.com/bornholm/xolo/internal/core/model"
)

type Middleware struct {
	ID          string `gorm:"primaryKey;autoIncrement:false"`
	OrgID       string `gorm:"uniqueIndex:idx_mw_org_name;index;not null"`
	Name        string `gorm:"uniqueIndex:idx_mw_org_name;not null"`
	Description string
	Enabled     bool `gorm:"default:true"`
	Priority    int  `gorm:"default:0"`
	// AppliesToAll wraps every model of the org when true, ignoring TargetsJSON.
	AppliesToAll bool
	// TargetsJSON stores []model.ModelRef as JSON.
	TargetsJSON string `gorm:"type:text;default:'[]'"`
	// GraphJSON stores the PipelineGraph as JSON. Empty string or "{}" means no graph.
	GraphJSON string `gorm:"type:text;default:'{}'"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type wrappedMiddleware struct {
	m *Middleware
}

func (w *wrappedMiddleware) ID() model.MiddlewareID { return model.MiddlewareID(w.m.ID) }
func (w *wrappedMiddleware) EntityID() string       { return w.m.ID }
func (w *wrappedMiddleware) OrgID() model.OrgID     { return model.OrgID(w.m.OrgID) }
func (w *wrappedMiddleware) Name() string           { return w.m.Name }
func (w *wrappedMiddleware) Description() string    { return w.m.Description }
func (w *wrappedMiddleware) Enabled() bool          { return w.m.Enabled }
func (w *wrappedMiddleware) Priority() int          { return w.m.Priority }
func (w *wrappedMiddleware) AppliesToAll() bool     { return w.m.AppliesToAll }
func (w *wrappedMiddleware) CreatedAt() time.Time   { return w.m.CreatedAt }
func (w *wrappedMiddleware) UpdatedAt() time.Time   { return w.m.UpdatedAt }

func (w *wrappedMiddleware) Targets() []model.ModelRef {
	if w.m.TargetsJSON == "" {
		return nil
	}
	var refs []model.ModelRef
	if err := json.Unmarshal([]byte(w.m.TargetsJSON), &refs); err != nil {
		return nil
	}
	return refs
}

func (w *wrappedMiddleware) Graph() *model.PipelineGraph {
	if w.m.GraphJSON == "" || w.m.GraphJSON == "{}" {
		return nil
	}
	var g model.PipelineGraph
	if err := json.Unmarshal([]byte(w.m.GraphJSON), &g); err != nil {
		return nil
	}
	if len(g.Nodes) == 0 {
		return nil
	}
	return &g
}

// Setters — needed by the generic pipeline handlers and the webui form.
func (w *wrappedMiddleware) SetName(v string)        { w.m.Name = v }
func (w *wrappedMiddleware) SetDescription(v string) { w.m.Description = v }
func (w *wrappedMiddleware) SetEnabled(v bool)       { w.m.Enabled = v }
func (w *wrappedMiddleware) SetPriority(v int)       { w.m.Priority = v }
func (w *wrappedMiddleware) SetAppliesToAll(v bool)  { w.m.AppliesToAll = v }

func (w *wrappedMiddleware) SetTargets(v []model.ModelRef) {
	if len(v) == 0 {
		w.m.TargetsJSON = "[]"
		return
	}
	if data, err := json.Marshal(v); err == nil {
		w.m.TargetsJSON = string(data)
	}
}

func (w *wrappedMiddleware) SetGraph(v *model.PipelineGraph) {
	if v == nil {
		w.m.GraphJSON = "{}"
		return
	}
	if data, err := json.Marshal(v); err == nil {
		w.m.GraphJSON = string(data)
	}
}

func (w *wrappedMiddleware) SetUpdatedAt(v time.Time) { w.m.UpdatedAt = v }

var (
	_ model.Middleware     = &wrappedMiddleware{}
	_ model.PipelineEntity = &wrappedMiddleware{}
)

func fromMiddleware(mw model.Middleware) *Middleware {
	graphJSON := "{}"
	if g := mw.Graph(); g != nil {
		if data, err := json.Marshal(g); err == nil {
			graphJSON = string(data)
		}
	}
	targetsJSON := "[]"
	if refs := mw.Targets(); len(refs) > 0 {
		if data, err := json.Marshal(refs); err == nil {
			targetsJSON = string(data)
		}
	}
	return &Middleware{
		ID:           string(mw.ID()),
		OrgID:        string(mw.OrgID()),
		Name:         mw.Name(),
		Description:  mw.Description(),
		Enabled:      mw.Enabled(),
		Priority:     mw.Priority(),
		AppliesToAll: mw.AppliesToAll(),
		TargetsJSON:  targetsJSON,
		GraphJSON:    graphJSON,
		CreatedAt:    mw.CreatedAt(),
		UpdatedAt:    mw.UpdatedAt(),
	}
}
