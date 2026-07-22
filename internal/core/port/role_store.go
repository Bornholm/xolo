package port

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/rbac"
)

type RoleStore interface {
	// Roles
	CreateRole(ctx context.Context, role model.Role) error
	GetRoleByID(ctx context.Context, id model.RoleID) (model.Role, error)
	ListOrgRoles(ctx context.Context, orgID model.OrgID) ([]model.Role, error)
	// SaveRole upserts the role and fully replaces its permissions and model grants.
	SaveRole(ctx context.Context, role model.Role) error
	// DeleteRole removes a role. It must refuse to delete builtin roles.
	DeleteRole(ctx context.Context, id model.RoleID) error

	// Membership assignments
	SetMembershipRoles(ctx context.Context, membershipID model.MembershipID, roleIDs []model.RoleID) error
	ListMembershipRoles(ctx context.Context, membershipID model.MembershipID) ([]model.Role, error)

	// Application assignments. An application is an org principal in its own
	// right: it holds roles directly rather than through a membership.
	SetApplicationRoles(ctx context.Context, appID model.ApplicationID, roleIDs []model.RoleID) error
	ListApplicationRoles(ctx context.Context, appID model.ApplicationID) ([]model.Role, error)

	// Builtin roles & resolution
	EnsureBuiltinRoles(ctx context.Context, orgID model.OrgID) error
	ResolveEffectivePermissions(ctx context.Context, userID model.UserID, orgID model.OrgID) (rbac.PermissionSet, error)
	ResolveApplicationPermissions(ctx context.Context, appID model.ApplicationID, orgID model.OrgID) (rbac.PermissionSet, error)
}
