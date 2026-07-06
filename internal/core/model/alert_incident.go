package model

import (
	"time"

	"github.com/rs/xid"
)

type AlertIncidentID string

func NewAlertIncidentID() AlertIncidentID {
	return AlertIncidentID(xid.New().String())
}

const (
	IncidentStatusFiring   = "firing"
	IncidentStatusResolved = "resolved"
)

// AlertIncident records one firing episode of an alert. It is stored durably and
// the events that contributed to it are pinned so they survive ring-buffer
// eviction.
type AlertIncident interface {
	WithID[AlertIncidentID]

	AlertID() AlertID
	OrgID() OrgID
	Status() string
	StartedAt() time.Time
	ResolvedAt() *time.Time
	PeakValue() float64
}

type BaseAlertIncident struct {
	id         AlertIncidentID
	alertID    AlertID
	orgID      OrgID
	status     string
	startedAt  time.Time
	resolvedAt *time.Time
	peakValue  float64
}

func (i *BaseAlertIncident) ID() AlertIncidentID  { return i.id }
func (i *BaseAlertIncident) AlertID() AlertID      { return i.alertID }
func (i *BaseAlertIncident) OrgID() OrgID          { return i.orgID }
func (i *BaseAlertIncident) Status() string        { return i.status }
func (i *BaseAlertIncident) StartedAt() time.Time  { return i.startedAt }
func (i *BaseAlertIncident) ResolvedAt() *time.Time { return i.resolvedAt }
func (i *BaseAlertIncident) PeakValue() float64    { return i.peakValue }

func (i *BaseAlertIncident) SetPeakValue(v float64) { i.peakValue = v }
func (i *BaseAlertIncident) SetStatus(s string)     { i.status = s }
func (i *BaseAlertIncident) SetResolvedAt(t *time.Time) { i.resolvedAt = t }

var _ AlertIncident = &BaseAlertIncident{}

// NewAlertIncident creates a new firing incident for an alert.
func NewAlertIncident(alertID AlertID, orgID OrgID, peakValue float64) *BaseAlertIncident {
	return &BaseAlertIncident{
		id:        NewAlertIncidentID(),
		alertID:   alertID,
		orgID:     orgID,
		status:    IncidentStatusFiring,
		startedAt: time.Now(),
		peakValue: peakValue,
	}
}
