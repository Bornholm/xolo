package api

import (
	"context"
	"slices"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/rbac"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	"github.com/bornholm/xolo/internal/http/middleware/authz"
	"github.com/pkg/errors"
)

// hasPermission reports whether the user in ctx holds the given permission
// within orgID. A global admin and a member with the builtin owner role bypass
// the check. Mirrors webui/org.Handler.hasPermission.
func (h *Handler) hasPermission(ctx context.Context, orgID model.OrgID, perm rbac.Permission) (bool, error) {
	user := httpCtx.User(ctx)
	if user == nil {
		return false, nil
	}
	if slices.Contains(user.Roles(), authz.RoleAdmin) {
		return true, nil
	}

	set, err := h.roleStore.ResolveEffectivePermissions(ctx, user.ID(), orgID)
	if err != nil {
		return false, errors.WithStack(err)
	}

	return set.IsOwner() || set.Has(perm), nil
}
