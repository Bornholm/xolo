package port

import (
	"context"
	"time"

	"github.com/bornholm/xolo/internal/core/model"
)

type AlertStore interface {
	CreateAlert(ctx context.Context, alert model.Alert) error
	UpdateAlert(ctx context.Context, alert model.Alert) error
	DeleteAlert(ctx context.Context, id model.AlertID) error
	GetAlertByID(ctx context.Context, id model.AlertID) (model.Alert, error)
	ListAlerts(ctx context.Context, orgID model.OrgID) ([]model.Alert, error)
	// ListEnabledAlerts returns every enabled alert across all orgs, for the
	// periodic evaluator.
	ListEnabledAlerts(ctx context.Context) ([]model.Alert, error)
	// UpdateAlertState persists only the evaluation state of an alert, without
	// touching its configuration or updated_at.
	UpdateAlertState(ctx context.Context, id model.AlertID, state model.AlertState, pendingSince *time.Time, lastEvaluatedAt *time.Time) error
}

type AlertIncidentStore interface {
	CreateIncident(ctx context.Context, incident model.AlertIncident) error
	ResolveIncident(ctx context.Context, id model.AlertIncidentID, resolvedAt time.Time) error
	UpdateIncidentPeak(ctx context.Context, id model.AlertIncidentID, peak float64) error
	// GetOpenIncident returns the currently firing (unresolved) incident for an
	// alert, or ErrNotFound when none is open.
	GetOpenIncident(ctx context.Context, alertID model.AlertID) (model.AlertIncident, error)
	ListIncidents(ctx context.Context, filter IncidentFilter) ([]model.AlertIncident, error)
}

type IncidentFilter struct {
	OrgID   *model.OrgID
	AlertID *model.AlertID
	Status  *string
	Limit   *int
	Offset  *int
}
