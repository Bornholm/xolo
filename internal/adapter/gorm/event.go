package gorm

import (
	"time"

	"github.com/bornholm/xolo/internal/core/eventql"
	"github.com/bornholm/xolo/internal/core/model"
)

type Event struct {
	ID         string    `gorm:"primaryKey;autoIncrement:false"`
	CreatedAt  time.Time `gorm:"index:idx_events_org_created,priority:2"`
	OrgID      string    `gorm:"index:idx_events_org_created,priority:1;index:idx_events_org_type,priority:1;index:idx_events_org_user,priority:1"`
	UserID     string    `gorm:"index:idx_events_org_user,priority:2"`
	Source     string    `gorm:"index"`
	Type       string    `gorm:"index:idx_events_org_type,priority:2"`
	Severity   string
	Message    string
	Attributes JSONColumn[map[string]string] `gorm:"type:text"`
	Pinned     bool   `gorm:"index"`
	IncidentID string `gorm:"index"`
}

type wrappedEvent struct {
	e *Event
}

func (w *wrappedEvent) ID() model.EventID     { return model.EventID(w.e.ID) }
func (w *wrappedEvent) OrgID() model.OrgID    { return model.OrgID(w.e.OrgID) }
func (w *wrappedEvent) UserID() model.UserID  { return model.UserID(w.e.UserID) }
func (w *wrappedEvent) Source() string        { return w.e.Source }
func (w *wrappedEvent) Type() string          { return w.e.Type }
func (w *wrappedEvent) Severity() model.EventSeverity {
	return model.EventSeverity(w.e.Severity)
}
func (w *wrappedEvent) Message() string { return w.e.Message }
func (w *wrappedEvent) Attributes() map[string]string {
	if w.e.Attributes.Val == nil {
		return map[string]string{}
	}
	return *w.e.Attributes.Val
}
func (w *wrappedEvent) Pinned() bool                    { return w.e.Pinned }
func (w *wrappedEvent) IncidentID() model.AlertIncidentID {
	return model.AlertIncidentID(w.e.IncidentID)
}
func (w *wrappedEvent) CreatedAt() time.Time { return w.e.CreatedAt }

var _ model.Event = &wrappedEvent{}

func fromEvent(e model.Event) *Event {
	attrs := e.Attributes()
	return &Event{
		ID:         string(e.ID()),
		CreatedAt:  e.CreatedAt(),
		OrgID:      string(e.OrgID()),
		UserID:     string(e.UserID()),
		Source:     e.Source(),
		Type:       e.Type(),
		Severity:   string(e.Severity()),
		Message:    e.Message(),
		Attributes: JSONColumn[map[string]string]{Val: &attrs},
		Pinned:     e.Pinned(),
		IncidentID: string(e.IncidentID()),
	}
}

// eventFields projects a stored event into the eventql evaluation struct.
func eventFields(e *Event) eventql.Fields {
	attrs := map[string]string{}
	if e.Attributes.Val != nil {
		attrs = *e.Attributes.Val
	}
	return eventql.Fields{
		Type:       e.Type,
		Source:     e.Source,
		Severity:   e.Severity,
		Org:        e.OrgID,
		User:       e.UserID,
		Message:    e.Message,
		Attributes: attrs,
	}
}

// eventLabelColumn maps an eventql label name to its promoted column.
func eventLabelColumn(name string) (string, bool) {
	switch name {
	case eventql.LabelType:
		return "type", true
	case eventql.LabelSource:
		return "source", true
	case eventql.LabelSeverity:
		return "severity", true
	case eventql.LabelOrg:
		return "org_id", true
	case eventql.LabelUser:
		return "user_id", true
	}
	return "", false
}
