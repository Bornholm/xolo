package service

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/pkg/errors"
)

// OrgProvider is a narrow interface used by QuotaService.
// It is satisfied by port.OrgStore (and any fake with just these two methods).
type OrgProvider interface {
	GetOrgByID(ctx context.Context, id model.OrgID) (model.Organization, error)
	ListOrgMembers(ctx context.Context, orgID model.OrgID, opts port.ListOrgMembersOptions) ([]model.Membership, int64, error)
}

type QuotaService struct {
	quotaStore port.QuotaStore
	orgStore   OrgProvider
}

func NewQuotaService(quotaStore port.QuotaStore, orgStore OrgProvider) *QuotaService {
	return &QuotaService{quotaStore: quotaStore, orgStore: orgStore}
}

func (s *QuotaService) ResolveEffectiveQuota(
	ctx context.Context,
	userID model.UserID,
	orgID model.OrgID,
) (*model.EffectiveQuota, error) {
	userQuota, err := s.quotaStore.GetQuota(ctx, model.QuotaScopeUser, string(userID))
	if err != nil && !errors.Is(err, port.ErrNotFound) {
		return nil, errors.WithStack(err)
	}
	if errors.Is(err, port.ErrNotFound) {
		userQuota = nil
	}

	orgQuota, err := s.quotaStore.GetQuota(ctx, model.QuotaScopeOrg, string(orgID))
	if err != nil && !errors.Is(err, port.ErrNotFound) {
		return nil, errors.WithStack(err)
	}
	if errors.Is(err, port.ErrNotFound) {
		orgQuota = nil
	}

	org, err := s.orgStore.GetOrgByID(ctx, orgID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	effective := &model.EffectiveQuota{}

	// A quota record with all-nil budgets is treated as "no personal quota".
	userHasPersonalQuota := userQuota != nil &&
		(userQuota.DailyBudget() != nil || userQuota.MonthlyBudget() != nil || userQuota.YearlyBudget() != nil)

	// Currency: org quota > user quota > default.
	switch {
	case orgQuota != nil:
		effective.Currency = orgQuota.Currency()
	case userHasPersonalQuota:
		effective.Currency = userQuota.Currency()
	default:
		effective.Currency = model.DefaultCurrency
	}

	// Sharing: distribute org quota equally only when user has no personal quota.
	if !userHasPersonalQuota && org.ShareQuotaEqually() && orgQuota != nil {
		members, _, err := s.orgStore.ListOrgMembers(ctx, orgID, port.ListOrgMembersOptions{})
		if err != nil {
			return nil, errors.WithStack(err)
		}
		n := int64(len(members))
		if n > 0 {
			// Synthesise a per-user quota by integer division (floor truncation).
			effective.DailyBudget = divPtr(orgQuota.DailyBudget(), n)
			effective.MonthlyBudget = divPtr(orgQuota.MonthlyBudget(), n)
			effective.YearlyBudget = divPtr(orgQuota.YearlyBudget(), n)
			return effective, nil
		}
		// n == 0: unlimited (theoretically impossible if requester is a member).
		return effective, nil
	}

	// Default: min-merge of user and org quotas.
	effective.DailyBudget = minPtrSvc(ptrOf(userQuota, func(q model.Quota) *int64 { return q.DailyBudget() }),
		ptrOf(orgQuota, func(q model.Quota) *int64 { return q.DailyBudget() }))
	effective.MonthlyBudget = minPtrSvc(ptrOf(userQuota, func(q model.Quota) *int64 { return q.MonthlyBudget() }),
		ptrOf(orgQuota, func(q model.Quota) *int64 { return q.MonthlyBudget() }))
	effective.YearlyBudget = minPtrSvc(ptrOf(userQuota, func(q model.Quota) *int64 { return q.YearlyBudget() }),
		ptrOf(orgQuota, func(q model.Quota) *int64 { return q.YearlyBudget() }))

	return effective, nil
}

func divPtr(v *int64, n int64) *int64 {
	if v == nil {
		return nil
	}
	r := *v / n
	return &r
}

func ptrOf(q model.Quota, f func(model.Quota) *int64) *int64 {
	if q == nil {
		return nil
	}
	return f(q)
}

func minPtrSvc(a, b *int64) *int64 {
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
