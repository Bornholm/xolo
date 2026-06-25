package gorm_test

import (
	"context"
	"testing"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/rbac"
)

func TestRoleStore_BuiltinRolesAndResolution(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	org := model.NewOrganization("acme", "Acme", "")
	if err := store.CreateOrg(ctx, org); err != nil {
		t.Fatalf("CreateOrg: %v", err)
	}

	// EnsureBuiltinRoles is idempotent.
	for range 2 {
		if err := store.EnsureBuiltinRoles(ctx, org.ID()); err != nil {
			t.Fatalf("EnsureBuiltinRoles: %v", err)
		}
	}

	roles, err := store.ListOrgRoles(ctx, org.ID())
	if err != nil {
		t.Fatalf("ListOrgRoles: %v", err)
	}
	if len(roles) != 3 {
		t.Fatalf("expected 3 builtin roles, got %d", len(roles))
	}

	var ownerID, memberID model.RoleID
	for _, r := range roles {
		switch r.BuiltinKind() {
		case model.BuiltinKindOwner:
			ownerID = r.ID()
		case model.BuiltinKindMember:
			memberID = r.ID()
		}
	}
	if ownerID == "" || memberID == "" {
		t.Fatal("missing builtin owner/member role")
	}

	// Member with a member role: usage permission but no admin permission.
	user := model.NewUser("test", "u1", "u1@example.com", "U1", true, "user")
	user.SetPreferences(model.NewUserPreferences())
	if err := store.SaveUser(ctx, user); err != nil {
		t.Fatalf("SaveUser: %v", err)
	}
	userID := user.ID()
	membership := model.NewMembership(userID, org.ID())
	if err := store.AddMember(ctx, membership); err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	if err := store.SetMembershipRoles(ctx, membership.ID(), []model.RoleID{memberID}); err != nil {
		t.Fatalf("SetMembershipRoles: %v", err)
	}

	perms, err := store.ResolveEffectivePermissions(ctx, userID, org.ID())
	if err != nil {
		t.Fatalf("ResolveEffectivePermissions: %v", err)
	}
	if perms.IsOwner() {
		t.Error("member should not be owner")
	}
	if !perms.Has(rbac.PermModelUseOrg) {
		t.Error("member should have model:use:org")
	}
	if perms.Has(rbac.PermMembersWrite) {
		t.Error("member should not have members:write")
	}

	// Assigning the owner role grants the owner bypass.
	if err := store.SetMembershipRoles(ctx, membership.ID(), []model.RoleID{ownerID}); err != nil {
		t.Fatalf("SetMembershipRoles owner: %v", err)
	}
	perms, err = store.ResolveEffectivePermissions(ctx, userID, org.ID())
	if err != nil {
		t.Fatalf("ResolveEffectivePermissions owner: %v", err)
	}
	if !perms.IsOwner() {
		t.Error("expected owner bypass")
	}
}

func TestRoleStore_CustomRoleAndModelGrant(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	org := model.NewOrganization("acme", "Acme", "")
	if err := store.CreateOrg(ctx, org); err != nil {
		t.Fatalf("CreateOrg: %v", err)
	}

	role := model.NewRole(org.ID(), "Lecteur", "Accès lecture")
	role.SetPermissions([]string{string(rbac.PermUsageRead), string(rbac.PermMembersWrite)})
	role.SetModelGrants([]model.ModelGrant{{ModelID: "m1", Kind: rbac.ModelKindLLM}})
	if err := store.CreateRole(ctx, role); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	got, err := store.GetRoleByID(ctx, role.ID())
	if err != nil {
		t.Fatalf("GetRoleByID: %v", err)
	}
	if len(got.Permissions()) != 2 {
		t.Errorf("expected 2 permissions, got %d", len(got.Permissions()))
	}
	if len(got.ModelGrants()) != 1 {
		t.Errorf("expected 1 model grant, got %d", len(got.ModelGrants()))
	}

	// SaveRole fully replaces permissions and grants.
	updated := model.UpdateRole(got, model.WithRolePermissions([]string{string(rbac.PermUsageRead)}), model.WithRoleModelGrants(nil))
	if err := store.SaveRole(ctx, updated); err != nil {
		t.Fatalf("SaveRole: %v", err)
	}
	got, err = store.GetRoleByID(ctx, role.ID())
	if err != nil {
		t.Fatalf("GetRoleByID after save: %v", err)
	}
	if len(got.Permissions()) != 1 || len(got.ModelGrants()) != 0 {
		t.Errorf("expected 1 perm / 0 grants after save, got %d / %d", len(got.Permissions()), len(got.ModelGrants()))
	}

	// Custom role is deletable.
	if err := store.DeleteRole(ctx, role.ID()); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}
}

func TestRoleStore_BuiltinRoleNotDeletable(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	org := model.NewOrganization("acme", "Acme", "")
	if err := store.CreateOrg(ctx, org); err != nil {
		t.Fatalf("CreateOrg: %v", err)
	}
	if err := store.EnsureBuiltinRoles(ctx, org.ID()); err != nil {
		t.Fatalf("EnsureBuiltinRoles: %v", err)
	}
	roles, err := store.ListOrgRoles(ctx, org.ID())
	if err != nil {
		t.Fatalf("ListOrgRoles: %v", err)
	}
	if err := store.DeleteRole(ctx, roles[0].ID()); err == nil {
		t.Fatal("expected error deleting builtin role")
	}
}
