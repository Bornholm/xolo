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

type PluginConfigRecord struct {
	OrgID      string `gorm:"primaryKey;autoIncrement:false;column:org_id"`
	PluginName string `gorm:"primaryKey;autoIncrement:false;column:plugin_name"`
	Scope      string `gorm:"primaryKey;autoIncrement:false;column:scope"`
	ScopeID    string `gorm:"primaryKey;autoIncrement:false;column:scope_id"`
	ConfigJSON string `gorm:"column:config_json;type:text"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (PluginConfigRecord) TableName() string { return "plugin_configs" }

func toPluginConfig(r *PluginConfigRecord) *model.PluginConfig {
	return &model.PluginConfig{
		OrgID:      model.OrgID(r.OrgID),
		PluginName: r.PluginName,
		Scope:      model.PluginConfigScope(r.Scope),
		ScopeID:    r.ScopeID,
		ConfigJSON: r.ConfigJSON,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
}

func fromPluginConfig(c *model.PluginConfig) *PluginConfigRecord {
	return &PluginConfigRecord{
		OrgID:      string(c.OrgID),
		PluginName: c.PluginName,
		Scope:      string(c.Scope),
		ScopeID:    c.ScopeID,
		ConfigJSON: c.ConfigJSON,
	}
}

// GetConfig implements port.PluginConfigStore.
func (s *Store) GetConfig(ctx context.Context, orgID model.OrgID, pluginName string, scope model.PluginConfigScope, scopeID string) (*model.PluginConfig, error) {
	var rec PluginConfigRecord
	err := s.withRetry(ctx, false, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Where(
			"org_id = ? AND plugin_name = ? AND scope = ? AND scope_id = ?",
			string(orgID), pluginName, string(scope), scopeID,
		).First(&rec).Error; err != nil {
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
	return toPluginConfig(&rec), nil
}

// SaveConfig implements port.PluginConfigStore (upsert on org_id+plugin_name+scope+scope_id).
func (s *Store) SaveConfig(ctx context.Context, cfg *model.PluginConfig) error {
	return s.withRetry(ctx, true, func(ctx context.Context, db *gorm.DB) error {
		rec := fromPluginConfig(cfg)
		return errors.WithStack(db.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "org_id"}, {Name: "plugin_name"}, {Name: "scope"}, {Name: "scope_id"},
			},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"config_json": rec.ConfigJSON,
			}),
		}).Create(rec).Error)
	}, sqlite3.BUSY, sqlite3.LOCKED)
}

// DeleteConfig implements port.PluginConfigStore.
func (s *Store) DeleteConfig(ctx context.Context, orgID model.OrgID, pluginName string, scope model.PluginConfigScope, scopeID string) error {
	return s.withRetry(ctx, true, func(ctx context.Context, db *gorm.DB) error {
		result := db.Delete(&PluginConfigRecord{},
			"org_id = ? AND plugin_name = ? AND scope = ? AND scope_id = ?",
			string(orgID), pluginName, string(scope), scopeID,
		)
		if result.Error != nil {
			return errors.WithStack(result.Error)
		}
		return nil
	}, sqlite3.BUSY, sqlite3.LOCKED)
}

var _ port.PluginConfigStore = &Store{}
