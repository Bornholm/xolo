package gorm

import (
	"time"

	"github.com/bornholm/xolo/internal/core/model"
)

type Alert struct {
	ID              string `gorm:"primaryKey;autoIncrement:false"`
	OrgID           string `gorm:"index"`
	OwnerID         string `gorm:"index"`
	Scope           string `gorm:"index;default:org"`
	Name            string
	Description     string
	Query           string
	Aggregation     string
	WindowSeconds   int64
	Comparator      string
	Threshold       float64
	ForSeconds      int64
	Enabled         bool `gorm:"index"`
	State           string
	PendingSince    *time.Time
	LastEvaluatedAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type wrappedAlert struct {
	a *Alert
}

func (w *wrappedAlert) ID() model.AlertID      { return model.AlertID(w.a.ID) }
func (w *wrappedAlert) OrgID() model.OrgID      { return model.OrgID(w.a.OrgID) }
func (w *wrappedAlert) OwnerID() model.UserID   { return model.UserID(w.a.OwnerID) }
func (w *wrappedAlert) Scope() model.AlertScope {
	if w.a.Scope == "" {
		return model.AlertScopeOrg
	}
	return model.AlertScope(w.a.Scope)
}
func (w *wrappedAlert) Name() string           { return w.a.Name }
func (w *wrappedAlert) Description() string     { return w.a.Description }
func (w *wrappedAlert) Query() string          { return w.a.Query }
func (w *wrappedAlert) Aggregation() model.AlertAggregation {
	return model.AlertAggregation(w.a.Aggregation)
}
func (w *wrappedAlert) Window() time.Duration { return time.Duration(w.a.WindowSeconds) * time.Second }
func (w *wrappedAlert) Comparator() model.AlertComparator {
	return model.AlertComparator(w.a.Comparator)
}
func (w *wrappedAlert) Threshold() float64        { return w.a.Threshold }
func (w *wrappedAlert) For() time.Duration        { return time.Duration(w.a.ForSeconds) * time.Second }
func (w *wrappedAlert) Enabled() bool            { return w.a.Enabled }
func (w *wrappedAlert) State() model.AlertState   { return model.AlertState(w.a.State) }
func (w *wrappedAlert) PendingSince() *time.Time  { return w.a.PendingSince }
func (w *wrappedAlert) LastEvaluatedAt() *time.Time { return w.a.LastEvaluatedAt }
func (w *wrappedAlert) CreatedAt() time.Time      { return w.a.CreatedAt }
func (w *wrappedAlert) UpdatedAt() time.Time      { return w.a.UpdatedAt }

var _ model.Alert = &wrappedAlert{}

func fromAlert(a model.Alert) *Alert {
	return &Alert{
		ID:              string(a.ID()),
		OrgID:           string(a.OrgID()),
		OwnerID:         string(a.OwnerID()),
		Scope:           string(a.Scope()),
		Name:            a.Name(),
		Description:     a.Description(),
		Query:           a.Query(),
		Aggregation:     string(a.Aggregation()),
		WindowSeconds:   int64(a.Window() / time.Second),
		Comparator:      string(a.Comparator()),
		Threshold:       a.Threshold(),
		ForSeconds:      int64(a.For() / time.Second),
		Enabled:         a.Enabled(),
		State:           string(a.State()),
		PendingSince:    a.PendingSince(),
		LastEvaluatedAt: a.LastEvaluatedAt(),
		CreatedAt:       a.CreatedAt(),
		UpdatedAt:       a.UpdatedAt(),
	}
}

type AlertIncident struct {
	ID         string    `gorm:"primaryKey;autoIncrement:false"`
	AlertID    string    `gorm:"index"`
	OrgID      string    `gorm:"index"`
	Status     string    `gorm:"index"`
	StartedAt  time.Time
	ResolvedAt *time.Time
	PeakValue  float64
}

type wrappedAlertIncident struct {
	i *AlertIncident
}

func (w *wrappedAlertIncident) ID() model.AlertIncidentID {
	return model.AlertIncidentID(w.i.ID)
}
func (w *wrappedAlertIncident) AlertID() model.AlertID { return model.AlertID(w.i.AlertID) }
func (w *wrappedAlertIncident) OrgID() model.OrgID      { return model.OrgID(w.i.OrgID) }
func (w *wrappedAlertIncident) Status() string          { return w.i.Status }
func (w *wrappedAlertIncident) StartedAt() time.Time    { return w.i.StartedAt }
func (w *wrappedAlertIncident) ResolvedAt() *time.Time  { return w.i.ResolvedAt }
func (w *wrappedAlertIncident) PeakValue() float64      { return w.i.PeakValue }

var _ model.AlertIncident = &wrappedAlertIncident{}

func fromAlertIncident(i model.AlertIncident) *AlertIncident {
	return &AlertIncident{
		ID:         string(i.ID()),
		AlertID:    string(i.AlertID()),
		OrgID:      string(i.OrgID()),
		Status:     i.Status(),
		StartedAt:  i.StartedAt(),
		ResolvedAt: i.ResolvedAt(),
		PeakValue:  i.PeakValue(),
	}
}
