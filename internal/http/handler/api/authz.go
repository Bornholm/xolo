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

// hasPermission reports whether the principal in ctx holds the given permission
// within orgID. A global admin and a principal with the builtin owner role
// bypass the check. Mirrors webui/org.Handler.hasPermission.
//
// Resolution goes through the context resolver, which knows how to resolve both
// members (via their membership) and applications (via the roles assigned to
// the application itself).
func (h *Handler) hasPermission(ctx context.Context, orgID model.OrgID, perm rbac.Permission) (bool, error) {
	user := httpCtx.User(ctx)
	if user == nil {
		return false, nil
	}
	// The shadow user backing an application never inherits the platform admin
	// bypass: its roles are an artefact of token authentication.
	if user.Provider() != model.ApplicationProvider && slices.Contains(user.Roles(), authz.RoleAdmin) {
		return true, nil
	}

	set, err := httpCtx.ResolvePermissions(ctx, orgID)
	if err != nil {
		return false, errors.WithStack(err)
	}

	return set.IsOwner() || set.Has(perm), nil
}
