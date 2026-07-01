package gorm

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/ncruces/go-sqlite3"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Store) CreateMiddleware(ctx context.Context, mw model.Middleware) error {
	return s.withRetry(ctx, true, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Create(fromMiddleware(mw)).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return errors.WithStack(port.ErrAlreadyExists)
			}
			return errors.WithStack(err)
		}
		return nil
	}, sqlite3.BUSY, sqlite3.LOCKED)
}

func (s *Store) GetMiddlewareByID(ctx context.Context, id model.MiddlewareID) (model.Middleware, error) {
	var mw Middleware
	err := s.withRetry(ctx, false, func(ctx context.Context, db *gorm.DB) error {
		if err := db.First(&mw, "id = ?", string(id)).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.WithStack(port.ErrNotFound)
			}
			return errors.WithStack(err)
		}
		return nil
	}, sqlite3.BUSY, sqlite3.LOCKED)
	if err != nil {
		return nil, err
	}
	return &wrappedMiddleware{&mw}, nil
}

func (s *Store) ListMiddlewares(ctx context.Context, orgID model.OrgID) ([]model.Middleware, error) {
	return s.queryMiddlewares(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("org_id = ?", string(orgID)).Order("priority ASC").Order("name ASC")
	})
}

func (s *Store) ListEnabledMiddlewares(ctx context.Context, orgID model.OrgID) ([]model.Middleware, error) {
	return s.queryMiddlewares(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("org_id = ? AND enabled = ?", string(orgID), true).Order("priority ASC").Order("name ASC")
	})
}

func (s *Store) queryMiddlewares(ctx context.Context, scope func(*gorm.DB) *gorm.DB) ([]model.Middleware, error) {
	var mws []*Middleware
	err := s.withRetry(ctx, false, func(ctx context.Context, db *gorm.DB) error {
		return errors.WithStack(scope(db).Find(&mws).Error)
	}, sqlite3.BUSY, sqlite3.LOCKED)
	if err != nil {
		return nil, err
	}
	result := make([]model.Middleware, 0, len(mws))
	for _, mw := range mws {
		result = append(result, &wrappedMiddleware{mw})
	}
	return result, nil
}

func (s *Store) SaveMiddleware(ctx context.Context, mw model.Middleware) error {
	return s.withRetry(ctx, true, func(ctx context.Context, db *gorm.DB) error {
		return errors.WithStack(db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			UpdateAll: true,
		}).Create(fromMiddleware(mw)).Error)
	}, sqlite3.BUSY, sqlite3.LOCKED)
}

func (s *Store) DeleteMiddleware(ctx context.Context, id model.MiddlewareID) error {
	return s.withRetry(ctx, true, func(ctx context.Context, db *gorm.DB) error {
		result := db.Delete(&Middleware{}, "id = ?", string(id))
		if result.Error != nil {
			return errors.WithStack(result.Error)
		}
		if result.RowsAffected == 0 {
			return errors.WithStack(port.ErrNotFound)
		}
		return nil
	}, sqlite3.BUSY, sqlite3.LOCKED)
}

var _ port.MiddlewareStore = &Store{}
