package api

import (
	"context"
	"slices"

	"github.com/bornholm/xolo/internal/core/model"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	"github.com/bornholm/xolo/internal/http/middleware/authz"
)

// hasOrgAdminAccess reports whether the user in ctx may manage org-scoped
// virtual models for orgID: a global admin, or a member with the org:admin
// or org:owner role. Mirrors webui/org.Handler.hasOrgAdminRole, which gates
// the equivalent admin UI routes.
func (h *Handler) hasOrgAdminAccess(ctx context.Context, orgID model.OrgID) (bool, error) {
	user := httpCtx.User(ctx)
	if user == nil {
		return false, nil
	}
	if slices.Contains(user.Roles(), authz.RoleAdmin) {
		return true, nil
	}

	membership, err := h.orgStore.GetUserOrgMembership(ctx, user.ID(), orgID)
	if err != nil {
		return false, nil // not a member
	}

	return membership.Role() == model.RoleOrgAdmin || membership.Role() == model.RoleOrgOwner, nil
}
