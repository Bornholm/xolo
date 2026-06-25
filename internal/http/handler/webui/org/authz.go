package org

import (
	"context"
	"slices"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/rbac"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	"github.com/bornholm/xolo/internal/http/middleware/authz"
	"github.com/pkg/errors"
)

// hasPermission returns an authz.AssertFunc that checks whether the current
// user holds the given permission within the org identified by orgSlug. A
// global admin and a member with the builtin owner role bypass the check.
func (h *Handler) hasPermission(orgSlug string, perm rbac.Permission) authz.AssertFunc {
	return func(ctx context.Context, user model.User) (bool, error) {
		if user == nil {
			return false, nil
		}
		// Global admin has access to everything.
		if slices.Contains(user.Roles(), authz.RoleAdmin) {
			return true, nil
		}

		org, err := h.orgStore.GetOrgBySlug(ctx, orgSlug)
		if err != nil {
			return false, errors.WithStack(err)
		}

		set, err := h.roleStore.ResolveEffectivePermissions(ctx, user.ID(), org.ID())
		if err != nil {
			return false, errors.WithStack(err)
		}

		return set.IsOwner() || set.Has(perm), nil
	}
}

// hasOrgMembership returns an authz.AssertFunc checking whether the user is any member of the org.
func (h *Handler) hasOrgMembership(orgSlug string) authz.AssertFunc {
	return func(ctx context.Context, user model.User) (bool, error) {
		if user == nil {
			return false, nil
		}
		if slices.Contains(user.Roles(), authz.RoleAdmin) {
			return true, nil
		}

		org, err := h.orgStore.GetOrgBySlug(ctx, orgSlug)
		if err != nil {
			return false, errors.WithStack(err)
		}

		return h.orgStore.IsMember(ctx, user.ID(), org.ID())
	}
}

// orgFromSlug resolves the org from the request path slug.
func (h *Handler) orgFromSlug(ctx context.Context, orgSlug string) (model.Organization, error) {
	return h.orgStore.GetOrgBySlug(ctx, orgSlug)
}

// currentUser retrieves the current user from context.
func currentUser(ctx context.Context) model.User {
	return httpCtx.User(ctx)
}
