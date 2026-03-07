package port

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
)

type QuotaStore interface {
	SetQuota(ctx context.Context, quota model.Quota) error
	GetQuota(ctx context.Context, scope model.QuotaScope, scopeID string) (model.Quota, error)
	// ResolveEffectiveQuota merges user and org quotas, taking the minimum non-nil value at each period.
	ResolveEffectiveQuota(ctx context.Context, userID model.UserID, orgID model.OrgID) (*model.EffectiveQuota, error)
}
