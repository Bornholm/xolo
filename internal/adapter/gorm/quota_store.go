package gorm

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/ncruces/go-sqlite3"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SetQuota implements port.QuotaStore.
func (s *Store) SetQuota(ctx context.Context, quota model.Quota) error {
	return s.withRetry(ctx, true, func(ctx context.Context, db *gorm.DB) error {
		return errors.WithStack(db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "scope"}, {Name: "scope_id"}},
			UpdateAll: true,
		}).Create(fromQuota(quota)).Error)
	}, sqlite3.BUSY, sqlite3.LOCKED)
}

// GetQuota implements port.QuotaStore.
func (s *Store) GetQuota(ctx context.Context, scope model.QuotaScope, scopeID string) (model.Quota, error) {
	var q Quota
	err := s.withRetry(ctx, false, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Where("scope = ? AND scope_id = ?", string(scope), scopeID).First(&q).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.WithStack(port.ErrNotFound)
			}
			return errors.WithStack(err)
		}
		return nil
	}, sqlite3.BUSY, sqlite3.LOCKED)
	if err != nil {
		return nil, err
	}
	return &wrappedQuota{&q}, nil
}

// ResolveEffectiveQuota implements port.QuotaStore.
// Takes the minimum non-nil budget at each period across user and org quotas.
func (s *Store) ResolveEffectiveQuota(ctx context.Context, userID model.UserID, orgID model.OrgID) (*model.EffectiveQuota, error) {
	effective := &model.EffectiveQuota{}

	userQuota, err := s.GetQuota(ctx, model.QuotaScopeUser, string(userID))
	if err != nil && !errors.Is(err, port.ErrNotFound) {
		return nil, errors.WithStack(err)
	}

	orgQuota, err := s.GetQuota(ctx, model.QuotaScopeOrg, string(orgID))
	if err != nil && !errors.Is(err, port.ErrNotFound) {
		return nil, errors.WithStack(err)
	}

	// Currency: org quota takes precedence; fall back to user quota, then default.
	switch {
	case orgQuota != nil:
		effective.Currency = orgQuota.Currency()
	case userQuota != nil:
		effective.Currency = userQuota.Currency()
	default:
		effective.Currency = model.DefaultCurrency
	}

	// Merge budgets from both levels, taking the stricter (minimum) non-nil value.
	// Currency labels are intentionally ignored here: since quotas are always saved
	// using the org's currency, budget values are in the same unit regardless of the
	// currency field stored on each record (which may be stale from an earlier org
	// currency setting).
	effective.DailyBudget = minPtr(quotaDaily(userQuota), quotaDaily(orgQuota))
	effective.MonthlyBudget = minPtr(quotaMonthly(userQuota), quotaMonthly(orgQuota))
	effective.YearlyBudget = minPtr(quotaYearly(userQuota), quotaYearly(orgQuota))

	return effective, nil
}

func quotaDaily(q model.Quota) *int64 {
	if q == nil {
		return nil
	}
	return q.DailyBudget()
}

func quotaMonthly(q model.Quota) *int64 {
	if q == nil {
		return nil
	}
	return q.MonthlyBudget()
}

func quotaYearly(q model.Quota) *int64 {
	if q == nil {
		return nil
	}
	return q.YearlyBudget()
}

// minPtr returns the minimum of two *int64. nil means "no limit".
// If both are nil → nil (no limit).
// If one is nil → the non-nil one (a concrete limit is stricter than none).
func minPtr(a, b *int64) *int64 {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	if *a < *b {
		return a
	}
	return b
}

var _ port.QuotaStore = &Store{}
