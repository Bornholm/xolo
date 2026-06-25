package context

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/rbac"
)

const keyPermissionResolver contextKey = "permission_resolver"

// PermissionResolverFunc resolves the effective permission set of the current
// user within the given organization.
type PermissionResolverFunc func(ctx context.Context, orgID model.OrgID) (rbac.PermissionSet, error)

func PermissionResolver(ctx context.Context) PermissionResolverFunc {
	resolver, ok := ctx.Value(keyPermissionResolver).(PermissionResolverFunc)
	if !ok {
		return nil
	}
	return resolver
}

func SetPermissionResolver(ctx context.Context, resolver PermissionResolverFunc) context.Context {
	return context.WithValue(ctx, keyPermissionResolver, resolver)
}

// ResolvePermissions resolves the current user's permission set for orgID using
// the resolver installed in the context. If no resolver is present (e.g. an
// unauthenticated request), an empty set is returned.
func ResolvePermissions(ctx context.Context, orgID model.OrgID) (rbac.PermissionSet, error) {
	resolver := PermissionResolver(ctx)
	if resolver == nil {
		return rbac.PermissionSet{}, nil
	}
	return resolver(ctx, orgID)
}
