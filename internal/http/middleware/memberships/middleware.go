package memberships

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"sync"

	"github.com/bornholm/go-x/slogx"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/bornholm/xolo/internal/core/rbac"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	"github.com/bornholm/xolo/internal/http/middleware/authz"
)

// Middleware fetches the current user's org memberships and stores them in the
// request context so that templates can render role-aware navigation items
// without each handler needing to fetch them individually. It also installs a
// memoized permission resolver used by RBAC permission checks.
func Middleware(orgStore port.OrgStore, roleStore port.RoleStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			user := httpCtx.User(ctx)

			if user != nil {
				memberships, err := orgStore.GetUserMemberships(ctx, user.ID())
				if err != nil {
					slog.ErrorContext(ctx, "memberships middleware: could not fetch memberships", slogx.Error(err))
				} else {
					ctx = httpCtx.SetMemberships(ctx, memberships)
				}

				ctx = httpCtx.SetPermissionResolver(ctx, newPermissionResolver(roleStore, user))
				r = r.WithContext(ctx)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// newPermissionResolver builds a per-request, memoized resolver of effective
// permissions for the given user. A global admin always resolves to an owner
// permission set, bypassing org-level role resolution.
//
// Applications authenticate through a shadow user whose provider is
// [model.ApplicationProvider] and whose subject is the ApplicationID. They are
// org principals without a membership, so their permissions are resolved from
// the roles assigned to the application itself.
func newPermissionResolver(roleStore port.RoleStore, user model.User) httpCtx.PermissionResolverFunc {
	isApplication := user.Provider() == model.ApplicationProvider
	// An application must never inherit the global admin bypass: its shadow user
	// is a platform-level artefact, not an operator account.
	isGlobalAdmin := !isApplication && slices.Contains(user.Roles(), authz.RoleAdmin)
	appID := model.ApplicationID(user.Subject())

	var mu sync.Mutex
	cache := map[model.OrgID]rbac.PermissionSet{}

	return func(ctx context.Context, orgID model.OrgID) (rbac.PermissionSet, error) {
		if isGlobalAdmin {
			return rbac.OwnerPermissionSet(), nil
		}

		mu.Lock()
		defer mu.Unlock()

		if set, ok := cache[orgID]; ok {
			return set, nil
		}

		var (
			set rbac.PermissionSet
			err error
		)
		if isApplication {
			set, err = roleStore.ResolveApplicationPermissions(ctx, appID, orgID)
		} else {
			set, err = roleStore.ResolveEffectivePermissions(ctx, user.ID(), orgID)
		}
		if err != nil {
			return rbac.PermissionSet{}, err
		}
		cache[orgID] = set
		return set, nil
	}
}
