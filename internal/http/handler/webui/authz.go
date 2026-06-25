package webui

import (
	"context"
	"slices"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/rbac"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	"github.com/bornholm/xolo/internal/http/middleware/authz"
	"github.com/pkg/errors"
)

// hasPermissionInAnyOrg returns an AssertFunc that allows access if the user
// holds perm in at least one of their org memberships (or is a global admin).
func (h *Handler) hasPermissionInAnyOrg(perm rbac.Permission) authz.AssertFunc {
	return func(ctx context.Context, user model.User) (bool, error) {
		if user == nil {
			return false, nil
		}
		if slices.Contains(user.Roles(), authz.RoleAdmin) {
			return true, nil
		}
		for _, m := range httpCtx.Memberships(ctx) {
			set, err := h.roleStore.ResolveEffectivePermissions(ctx, user.ID(), m.OrgID())
			if err != nil {
				return false, errors.WithStack(err)
			}
			if set.IsOwner() || set.Has(perm) {
				return true, nil
			}
		}
		return false, nil
	}
}

// canAccessModelsPage returns an AssertFunc that allows access to /models if
// the user has PermModelUseOrg, PermModelUseVirtual, or any per-model grant in
// at least one of their org memberships.
func (h *Handler) canAccessModelsPage() authz.AssertFunc {
	return func(ctx context.Context, user model.User) (bool, error) {
		if user == nil {
			return false, nil
		}
		if slices.Contains(user.Roles(), authz.RoleAdmin) {
			return true, nil
		}
		for _, m := range httpCtx.Memberships(ctx) {
			set, err := h.roleStore.ResolveEffectivePermissions(ctx, user.ID(), m.OrgID())
			if err != nil {
				return false, errors.WithStack(err)
			}
			if set.IsOwner() ||
				set.Has(rbac.PermModelUseOrg) ||
				set.Has(rbac.PermModelUseVirtual) ||
				set.HasAnyGrant(rbac.ModelKindLLM) ||
				set.HasAnyGrant(rbac.ModelKindVirtual) {
				return true, nil
			}
		}
		return false, nil
	}
}
