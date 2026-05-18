package gorm

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func createGetDatabase(db *gorm.DB, models ...any) func(ctx context.Context) (*gorm.DB, error) {
	var (
		migrateOnce sync.Once
		migrateErr  error
	)

	return func(ctx context.Context) (*gorm.DB, error) {
		migrateOnce.Do(func() {
			db.Exec("DROP INDEX IF EXISTS `idx_users_email`")
			if err := db.AutoMigrate(models...); err != nil {
				migrateErr = errors.WithStack(err)
				return
			}
		})
		if migrateErr != nil {
			return nil, errors.WithStack(migrateErr)
		}

		return db, nil
	}
}
