package component

import (
	"context"
	"net/url"

	"github.com/a-h/templ"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/http/middleware/authz"
	httpURL "github.com/bornholm/xolo/internal/http/url"
	"github.com/pkg/errors"
)

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

// OrgAdminMemberships returns the user's memberships where they hold an admin or owner role.
func OrgAdminMemberships(ctx context.Context) []model.Membership {
	all := httpCtx.Memberships(ctx)
	var result []model.Membership
	for _, m := range all {
		if m.Role() == model.RoleOrgAdmin || m.Role() == model.RoleOrgOwner {
			result = append(result, m)
		}
	}
	return result
}
