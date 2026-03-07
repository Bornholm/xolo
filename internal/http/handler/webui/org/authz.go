package org

import (
	"context"
	"slices"

	"github.com/bornholm/xolo/internal/core/model"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	"github.com/bornholm/xolo/internal/http/middleware/authz"
	"github.com/pkg/errors"
)

// hasOrgAdminRole returns an authz.AssertFunc that checks whether the current
// user has org:admin, org:owner, or global admin role in the given org.
func (h *Handler) hasOrgAdminRole(orgSlug string) authz.AssertFunc {
	return func(ctx context.Context, user model.User) (bool, error) {
		if user == nil {
			return false, nil
		}
		// Global admin has access to everything
		if slices.Contains(user.Roles(), authz.RoleAdmin) {
			return true, nil
		}

		org, err := h.orgStore.GetOrgBySlug(ctx, orgSlug)
		if err != nil {
			return false, errors.WithStack(err)
		}

		membership, err := h.orgStore.GetUserOrgMembership(ctx, user.ID(), org.ID())
		if err != nil {
			return false, nil // not a member
		}

		return membership.Role() == model.RoleOrgAdmin || membership.Role() == model.RoleOrgOwner, nil
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

// orgAdminFromRequest resolves the org from the request path and checks admin permission.
// Returns the org or writes an HTTP error and returns nil.
func (h *Handler) orgFromSlug(ctx context.Context, orgSlug string) (model.Organization, error) {
	return h.orgStore.GetOrgBySlug(ctx, orgSlug)
}

// currentUser retrieves the current user from context.
func currentUser(ctx context.Context) model.User {
	return httpCtx.User(ctx)
}
