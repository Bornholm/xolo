package gorm

import (
	"testing"

	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/gormlite"
	"github.com/bornholm/xolo/internal/core/model"
	gormpkg "gorm.io/gorm"
)

// TestMigrateLegacyMembershipRoles exercises the upgrade path on an existing
// database: legacy memberships carrying a single Role string must be converted
// into builtin role assignments.
func TestMigrateLegacyMembershipRoles(t *testing.T) {
	db, err := gormpkg.Open(gormlite.Open(":memory:"), &gormpkg.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	// Legacy schema: organizations + memberships with a "role" column.
	if err := db.AutoMigrate(&Organization{}); err != nil {
		t.Fatalf("migrate org: %v", err)
	}
	type legacyMembershipTable struct {
		ID     string `gorm:"primaryKey"`
		UserID string
		OrgID  string
		Role   string
	}
	if err := db.Table("memberships").AutoMigrate(&legacyMembershipTable{}); err != nil {
		t.Fatalf("migrate legacy memberships: %v", err)
	}

	org := fromOrganization(model.NewOrganization("acme", "Acme", ""))
	if err := db.Create(org).Error; err != nil {
		t.Fatalf("create org: %v", err)
	}

	rows := []legacyMembershipTable{
		{ID: "m-owner", UserID: "u1", OrgID: org.ID, Role: model.RoleOrgOwner},
		{ID: "m-admin", UserID: "u2", OrgID: org.ID, Role: model.RoleOrgAdmin},
		{ID: "m-member", UserID: "u3", OrgID: org.ID, Role: model.RoleMember},
	}
	for _, r := range rows {
		if err := db.Table("memberships").Create(&r).Error; err != nil {
			t.Fatalf("create legacy membership: %v", err)
		}
	}

	// Run the migration body (idempotently, twice). gormigrate runs migrations
	// outside a transaction, so PRAGMA foreign_keys=off takes effect — mirror that.
	runMigration := func() error {
		if err := db.Exec("PRAGMA foreign_keys=off").Error; err != nil {
			return err
		}
		if err := db.SetupJoinTable(&Membership{}, "Roles", &MembershipRole{}); err != nil {
			return err
		}
		if err := db.AutoMigrate(&Role{}, &RolePermission{}, &RoleModel{}, &MembershipRole{}); err != nil {
			return err
		}
		if err := migrateLegacyMembershipRoles(db); err != nil {
			return err
		}
		if db.Migrator().HasColumn(&Membership{}, "role") {
			if err := db.Migrator().DropColumn(&Membership{}, "role"); err != nil {
				return err
			}
		}
		return db.Exec("PRAGMA foreign_keys=on").Error
	}
	for range 2 {
		if err := runMigration(); err != nil {
			t.Fatalf("migration: %v", err)
		}
	}

	// Builtin roles created.
	var roleCount int64
	db.Model(&Role{}).Where("org_id = ?", org.ID).Count(&roleCount)
	if roleCount != 3 {
		t.Fatalf("expected 3 builtin roles, got %d", roleCount)
	}

	// Each legacy membership got exactly one role assignment.
	var assignCount int64
	db.Model(&MembershipRole{}).Count(&assignCount)
	if assignCount != 3 {
		t.Fatalf("expected 3 membership role assignments, got %d", assignCount)
	}

	// The owner membership maps to the owner builtin role.
	var ownerRoleID string
	db.Model(&Role{}).Where("org_id = ? AND builtin_kind = ?", org.ID, model.BuiltinKindOwner).Pluck("id", &ownerRoleID)
	var got int64
	db.Model(&MembershipRole{}).Where("membership_id = ? AND role_id = ?", "m-owner", ownerRoleID).Count(&got)
	if got != 1 {
		t.Fatalf("owner membership not mapped to owner role")
	}
}
