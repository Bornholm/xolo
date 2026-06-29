package gorm

import (
	"context"
	"sync"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func createGetDatabase(db *gorm.DB) func(ctx context.Context) (*gorm.DB, error) {
	var (
		migrateOnce sync.Once
		migrateErr  error
	)

	return func(ctx context.Context) (*gorm.DB, error) {
		migrateOnce.Do(func() {
			m := gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
				{
					// Add GraphJSON column to virtual_models and drop plugin tables.
					ID: "202602010001",
					Migrate: func(tx *gorm.DB) error {
						if err := tx.Migrator().DropTable("plugin_activations", "plugin_configs"); err != nil {
							// Ignore if tables don't exist (fresh install).
							_ = err
						}
						return tx.AutoMigrate(&VirtualModel{})
					},
					Rollback: func(tx *gorm.DB) error {
						return tx.Migrator().DropColumn(&VirtualModel{}, "graph_json")
					},
				},
				{
					// Add user_virtual_models table for personal virtual models.
					ID: "202506040001",
					Migrate: func(tx *gorm.DB) error {
						return tx.AutoMigrate(&PersonalVirtualModel{})
					},
					Rollback: func(tx *gorm.DB) error {
						return tx.Migrator().DropTable("personal_virtual_models")
					},
				},
				{
					// Add cached token columns for prompt caching support.
					ID: "202606080001",
					Migrate: func(tx *gorm.DB) error {
						return tx.AutoMigrate(&LLMModel{}, &UsageRecord{})
					},
					Rollback: func(tx *gorm.DB) error {
						if err := tx.Migrator().DropColumn(&LLMModel{}, "cached_prompt_cost_per1_k_tokens"); err != nil {
							return err
						}
						return tx.Migrator().DropColumn(&UsageRecord{}, "cached_tokens")
					},
				},
				{
					// Add plugin_node_secrets table backing the GetSecret/SetSecret
					// host service RPCs (per-node-instance encrypted key/value store).
					ID: "202606180001",
					Migrate: func(tx *gorm.DB) error {
						return tx.AutoMigrate(&PluginNodeSecret{})
					},
					Rollback: func(tx *gorm.DB) error {
						return tx.Migrator().DropTable("plugin_node_secrets")
					},
				},
				{
					// Introduce org-scoped RBAC: roles, role permissions, role model
					// grants and the membership<->role join table. Migrate the legacy
					// single Membership.Role to builtin role assignments, then drop the
					// deprecated column.
					ID: "202606250001",
					Migrate: func(tx *gorm.DB) error {
						// Disable foreign keys while rebuilding tables: gormlite/SQLite
						// recreates tables via INSERT...SELECT into a temp table, which
						// would otherwise fail FK checks on legacy rows.
						if err := tx.Exec("PRAGMA foreign_keys=off").Error; err != nil {
							return errors.WithStack(err)
						}
						if err := tx.SetupJoinTable(&Membership{}, "Roles", &MembershipRole{}); err != nil {
							return errors.WithStack(err)
						}
						if err := tx.AutoMigrate(&Role{}, &RolePermission{}, &RoleModel{}, &MembershipRole{}); err != nil {
							return errors.WithStack(err)
						}
						if err := migrateLegacyMembershipRoles(tx); err != nil {
							return errors.WithStack(err)
						}
						if tx.Migrator().HasColumn(&Membership{}, "role") {
							if err := tx.Migrator().DropColumn(&Membership{}, "role"); err != nil {
								return errors.WithStack(err)
							}
						}
						if err := tx.Exec("PRAGMA foreign_keys=on").Error; err != nil {
							return errors.WithStack(err)
						}
						return nil
					},
					Rollback: func(tx *gorm.DB) error {
						return tx.Migrator().DropTable("membership_roles", "role_models", "role_permissions", "roles")
					},
				},
				{
					// Add cost_source column to usage_records to track whether the
					// cost was reported by the provider or computed from the tariff.
					ID: "202606290001",
					Migrate: func(tx *gorm.DB) error {
						return tx.AutoMigrate(&UsageRecord{})
					},
					Rollback: func(tx *gorm.DB) error {
						return tx.Migrator().DropColumn(&UsageRecord{}, "cost_source")
					},
				},
			})

			m.InitSchema(func(tx *gorm.DB) error {
				// Disable foreign keys during schema migration to avoid
				// FK constraint failures when copying data between tables
				// during AutoMigrate (SQLite issue with INSERT...SELECT)
				if err := tx.Exec("PRAGMA foreign_keys=off").Error; err != nil {
					return errors.WithStack(err)
				}

				// Drop the deprecated index if exists (used in old migration)
				tx.Exec("DROP INDEX IF EXISTS `idx_users_email`")

				if err := tx.SetupJoinTable(&Membership{}, "Roles", &MembershipRole{}); err != nil {
					return errors.WithStack(err)
				}

				err := tx.AutoMigrate(
					// User store
					&User{}, &AuthToken{}, &UserRole{}, &UserPreferences{},
					// Org store
					&Organization{}, &Membership{}, &Application{},
					// RBAC store
					&Role{}, &RolePermission{}, &RoleModel{}, &MembershipRole{},
					// Provider store
					&Provider{}, &LLMModel{},
					// Virtual model store
					&VirtualModel{},
					// Personal virtual model store
					&PersonalVirtualModel{},
					// Quota store
					&Quota{},
					// Usage store
					&UsageRecord{},
					// Invite store
					&InviteToken{},
					// Exchange rate cache
					&ExchangeRate{},
					// Plugin node secrets
					&PluginNodeSecret{},
				)
				if err != nil {
					return errors.WithStack(err)
				}

				// Re-enable foreign keys after schema migration
				if err := tx.Exec("PRAGMA foreign_keys=on").Error; err != nil {
					return errors.WithStack(err)
				}

				return nil
			})

			migrateErr = m.Migrate()
		})

		if migrateErr != nil {
			return nil, errors.WithStack(migrateErr)
		}

		return db, nil
	}
}