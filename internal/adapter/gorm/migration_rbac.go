package gorm

import (
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// migrateLegacyMembershipRoles creates the builtin roles for every existing
// organization and converts the legacy Membership.Role column into membership
// role assignments. It is idempotent and safe to re-run.
func migrateLegacyMembershipRoles(tx *gorm.DB) error {
	// Collect existing organizations.
	var orgIDs []string
	if err := tx.Model(&Organization{}).Pluck("id", &orgIDs).Error; err != nil {
		return errors.WithStack(err)
	}

	// Ensure builtin roles exist for each org and index them by (orgID, kind).
	builtinByOrg := map[string]map[string]string{}
	for _, orgID := range orgIDs {
		kinds := map[string]string{}
		for _, spec := range builtinRoleSpecs() {
			var existing Role
			err := tx.Where("org_id = ? AND builtin_kind = ?", orgID, spec.kind).First(&existing).Error
			if err == nil {
				kinds[spec.kind] = existing.ID
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.WithStack(err)
			}
			role := model.NewBuiltinRole(model.OrgID(orgID), spec.kind, spec.name, spec.description, spec.permissions)
			gormRole := fromRole(role)
			if err := tx.Omit("Org").Create(gormRole).Error; err != nil {
				return errors.WithStack(err)
			}
			kinds[spec.kind] = gormRole.ID
		}
		builtinByOrg[orgID] = kinds
	}

	// If the legacy column is gone (already migrated), nothing more to do.
	if !tx.Migrator().HasColumn(&Membership{}, "role") {
		return nil
	}

	// Read legacy memberships (id, org_id, role) via raw SQL since the Role
	// field no longer exists on the Membership struct.
	type legacyMembership struct {
		ID    string
		OrgID string
		Role  string
	}
	var legacy []legacyMembership
	if err := tx.Raw("SELECT id, org_id, role FROM memberships").Scan(&legacy).Error; err != nil {
		return errors.WithStack(err)
	}

	for _, m := range legacy {
		kinds, ok := builtinByOrg[m.OrgID]
		if !ok {
			continue
		}
		kind := legacyRoleToBuiltinKind(m.Role)
		roleID, ok := kinds[kind]
		if !ok {
			continue
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&MembershipRole{
			MembershipID: m.ID,
			RoleID:       roleID,
		}).Error; err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func legacyRoleToBuiltinKind(role string) string {
	switch role {
	case model.RoleOrgOwner:
		return model.BuiltinKindOwner
	case model.RoleOrgAdmin:
		return model.BuiltinKindAdmin
	default:
		return model.BuiltinKindMember
	}
}
