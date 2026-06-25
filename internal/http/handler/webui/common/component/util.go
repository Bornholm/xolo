package component

import (
	"context"
	"net/url"

	"github.com/a-h/templ"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/rbac"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	"github.com/bornholm/xolo/internal/http/middleware/authz"
	httpURL "github.com/bornholm/xolo/internal/http/url"
	"github.com/pkg/errors"
)

// HasPermission reports whether the current user holds perm within orgID.
// Returns true for org owners (IsOwner bypass) and false on resolver errors.
func HasPermission(ctx context.Context, orgID model.OrgID, perm rbac.Permission) bool {
	perms, err := httpCtx.ResolvePermissions(ctx, orgID)
	if err != nil {
		return false
	}
	return perms.IsOwner() || perms.Has(perm)
}

var (
	WithPath        = httpURL.WithPath
	WithoutValues   = httpURL.WithoutValues
	WithValuesReset = httpURL.WithValuesReset
	WithValues      = httpURL.WithValues
)

func IsDesktopApp(ctx context.Context) bool {
	return false
}

func WithUser(username string, password string) httpURL.MutationFunc {
	return func(u *url.URL) {
		u.User = url.UserPassword(username, password)
	}
}

func BaseURL(ctx context.Context, funcs ...httpURL.MutationFunc) templ.SafeURL {
	baseURL := httpCtx.BaseURL(ctx)
	mutated := httpURL.Mutate(baseURL, funcs...)
	return templ.SafeURL(mutated.String())
}

func BaseURLString(ctx context.Context, funcs ...httpURL.MutationFunc) string {
	return string(BaseURL(ctx, funcs...))
}

func CurrentURL(ctx context.Context, funcs ...httpURL.MutationFunc) templ.SafeURL {
	currentURL := clone(httpCtx.CurrentURL(ctx))
	mutated := httpURL.Mutate(currentURL, funcs...)
	return templ.SafeURL(mutated.String())
}

func MatchPath(ctx context.Context, path string) bool {
	currentURL := httpCtx.CurrentURL(ctx)
	return currentURL.Path == path
}

func clone[T any](v *T) *T {
	copy := *v
	return &copy
}

func AssertUser(ctx context.Context, funcs ...authz.AssertFunc) bool {
	user := httpCtx.User(ctx)
	if user == nil {
		return false
	}

	allowed, err := authz.Assert(ctx, user, funcs...)
	if err != nil {
		panic(errors.WithStack(err))
	}

	return allowed
}

var User = httpCtx.User

// WithUserVariant sets the page color scheme variant from the user's preferences,
// matching the behavior of AppLayout. No-op if user is nil or has no preference set.
func WithUserVariant(user model.User) PageOptionFunc {
	return func(opts *PageOptions) {
		if user == nil {
			return
		}
		if darkMode, exists := user.Preferences().DarkMode(); exists {
			if darkMode {
				opts.Variant = VariantDark
			} else {
				opts.Variant = VariantLight
			}
		}
	}
}

// HasPermissionInAnyOrg reports whether the current user holds perm in at least
// one of their org memberships. Useful for cross-org features like personal VMs.
func HasPermissionInAnyOrg(ctx context.Context, perm rbac.Permission) bool {
	for _, m := range httpCtx.Memberships(ctx) {
		if HasPermission(ctx, m.OrgID(), perm) {
			return true
		}
	}
	return false
}

// CanAccessModelsPage reports whether the current user can access the /models
// page, i.e. has PermModelUseOrg, PermModelUseVirtual, or at least one
// model-specific grant in any of their org memberships.
func CanAccessModelsPage(ctx context.Context) bool {
	for _, m := range httpCtx.Memberships(ctx) {
		perms, err := httpCtx.ResolvePermissions(ctx, m.OrgID())
		if err != nil {
			continue
		}
		if perms.IsOwner() ||
			perms.Has(rbac.PermModelUseOrg) ||
			perms.Has(rbac.PermModelUseVirtual) ||
			perms.HasAnyGrant(rbac.ModelKindLLM) ||
			perms.HasAnyGrant(rbac.ModelKindVirtual) {
			return true
		}
	}
	return false
}

// OrgAdminMemberships returns the user's memberships granting access to at
// least one organization administration section.
func OrgAdminMemberships(ctx context.Context) []model.Membership {
	all := httpCtx.Memberships(ctx)
	var result []model.Membership
	for _, m := range all {
		if membershipHasAdminAccess(m) {
			result = append(result, m)
		}
	}
	return result
}

func membershipHasAdminAccess(m model.Membership) bool {
	for _, r := range m.Roles() {
		if r.BuiltinKind() == model.BuiltinKindOwner || r.BuiltinKind() == model.BuiltinKindAdmin {
			return true
		}
		for _, code := range r.Permissions() {
			if rbac.IsAdminPermission(rbac.Permission(code)) {
				return true
			}
		}
	}
	return false
}
