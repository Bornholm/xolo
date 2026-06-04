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

				err := tx.AutoMigrate(
					// User store
					&User{}, &AuthToken{}, &UserRole{}, &UserPreferences{},
					// Org store
					&Organization{}, &Membership{}, &Application{},
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