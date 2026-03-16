package gorm

import (
	"context"
	"time"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/ncruces/go-sqlite3"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PluginActivationRecord is the GORM model for plugin_activations table.
type PluginActivationRecord struct {
	OrgID      string `gorm:"primaryKey;autoIncrement:false;column:org_id"`
	PluginName string `gorm:"primaryKey;autoIncrement:false;column:plugin_name"`
	Enabled    int    `gorm:"column:enabled;default:1"`
	Required   int    `gorm:"column:required;default:0"`
	Order      int    `gorm:"column:order_index;default:0"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (PluginActivationRecord) TableName() string { return "plugin_activations" }

func toPluginActivation(r *PluginActivationRecord) *model.PluginActivation {
	return &model.PluginActivation{
		OrgID:      model.OrgID(r.OrgID),
		PluginName: r.PluginName,
		Enabled:    r.Enabled != 0,
		Required:   r.Required != 0,
		Order:      r.Order,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
}

func fromPluginActivation(a *model.PluginActivation) *PluginActivationRecord {
	return &PluginActivationRecord{
		OrgID:      string(a.OrgID),
		PluginName: a.PluginName,
		Enabled:    boolToInt(a.Enabled),
		Required:   boolToInt(a.Required),
		Order:      a.Order,
	}
}

// GetActivation implements port.PluginActivationStore.
func (s *Store) GetActivation(ctx context.Context, orgID model.OrgID, pluginName string) (*model.PluginActivation, error) {
	var rec PluginActivationRecord
	err := s.withRetry(ctx, false, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Where("org_id = ? AND plugin_name = ?", string(orgID), pluginName).First(&rec).Error; err != nil {
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
	return toPluginActivation(&rec), nil
}

// ListActivations implements port.PluginActivationStore.
// Returns activations sorted by order_index ASC.
func (s *Store) ListActivations(ctx context.Context, orgID model.OrgID) ([]*model.PluginActivation, error) {
	var recs []*PluginActivationRecord
	err := s.withRetry(ctx, false, func(ctx context.Context, db *gorm.DB) error {
		return errors.WithStack(
			db.Where("org_id = ? AND enabled = 1", string(orgID)).
				Order("order_index ASC").
				Find(&recs).Error,
		)
	}, sqlite3.BUSY, sqlite3.LOCKED)
	if err != nil {
		return nil, err
	}
	result := make([]*model.PluginActivation, 0, len(recs))
	for _, r := range recs {
		result = append(result, toPluginActivation(r))
	}
	return result, nil
}

// SaveActivation implements port.PluginActivationStore (upsert on org_id+plugin_name).
func (s *Store) SaveActivation(ctx context.Context, a *model.PluginActivation) error {
	return s.withRetry(ctx, true, func(ctx context.Context, db *gorm.DB) error {
		rec := fromPluginActivation(a)
		return errors.WithStack(db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "org_id"}, {Name: "plugin_name"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"enabled":     rec.Enabled,
				"required":    rec.Required,
				"order_index": rec.Order,
			}),
		}).Create(rec).Error)
	}, sqlite3.BUSY, sqlite3.LOCKED)
}

// DeleteActivation implements port.PluginActivationStore.
func (s *Store) DeleteActivation(ctx context.Context, orgID model.OrgID, pluginName string) error {
	return s.withRetry(ctx, true, func(ctx context.Context, db *gorm.DB) error {
		result := db.Delete(&PluginActivationRecord{}, "org_id = ? AND plugin_name = ?", string(orgID), pluginName)
		if result.Error != nil {
			return errors.WithStack(result.Error)
		}
		return nil
	}, sqlite3.BUSY, sqlite3.LOCKED)
}

var _ port.PluginActivationStore = &Store{}
