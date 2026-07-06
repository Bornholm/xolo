package gorm

import (
	"context"
	"time"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/ncruces/go-sqlite3"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type EventSettings struct {
	OrgID     string `gorm:"primaryKey;autoIncrement:false"`
	MaxEvents *int
	UpdatedAt time.Time
}

// GetMaxEvents implements port.EventSettingsStore.
func (s *Store) GetMaxEvents(ctx context.Context, orgID model.OrgID) (*int, error) {
	var settings EventSettings
	err := s.withRetry(ctx, false, func(ctx context.Context, db *gorm.DB) error {
		if err := db.First(&settings, "org_id = ?", string(orgID)).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.WithStack(port.ErrNotFound)
			}
			return errors.WithStack(err)
		}
		return nil
	}, sqlite3.BUSY, sqlite3.LOCKED)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return settings.MaxEvents, nil
}

// SetMaxEvents implements port.EventSettingsStore.
func (s *Store) SetMaxEvents(ctx context.Context, orgID model.OrgID, maxEvents *int) error {
	return s.withRetry(ctx, true, func(ctx context.Context, db *gorm.DB) error {
		settings := EventSettings{
			OrgID:     string(orgID),
			MaxEvents: maxEvents,
			UpdatedAt: time.Now(),
		}
		return errors.WithStack(db.Save(&settings).Error)
	}, sqlite3.BUSY, sqlite3.LOCKED)
}

var _ port.EventSettingsStore = &Store{}
