package port

import (
	"context"

	"github.com/xolo-gateway/xolo/internal/core/model"
)

// EventSettingsStore persists per-org event retention overrides. A nil value
// means the org uses the global default.
type EventSettingsStore interface {
	GetMaxEvents(ctx context.Context, orgID model.OrgID) (*int, error)
	SetMaxEvents(ctx context.Context, orgID model.OrgID, maxEvents *int) error
}
