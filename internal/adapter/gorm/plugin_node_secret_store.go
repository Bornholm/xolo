package gorm

import (
	"context"

	"github.com/bornholm/xolo/internal/core/port"
	"github.com/ncruces/go-sqlite3"
	"github.com/pkg/errors"
	"github.com/rs/xid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GetSecret implements port.SecretStore.
func (s *Store) GetSecret(ctx context.Context, orgID, pluginName, nodeID, key string) (string, bool, error) {
	var secret PluginNodeSecret
	err := s.withRetry(ctx, false, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Where("node_id = ? AND key = ?", nodeID, key).First(&secret).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return errors.WithStack(err)
		}
		return nil
	}, sqlite3.BUSY, sqlite3.LOCKED)
	if err != nil {
		return "", false, err
	}
	if secret.ID == "" {
		return "", false, nil
	}
	return secret.ValueEncrypted, true, nil
}

// SetSecret implements port.SecretStore.
func (s *Store) SetSecret(ctx context.Context, orgID, pluginName, nodeID, key, value string) error {
	return s.withRetry(ctx, true, func(ctx context.Context, db *gorm.DB) error {
		secret := &PluginNodeSecret{
			ID:             xid.New().String(),
			OrgID:          orgID,
			PluginName:     pluginName,
			NodeID:         nodeID,
			Key:            key,
			ValueEncrypted: value,
		}
		return errors.WithStack(db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "node_id"}, {Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value_encrypted", "org_id", "plugin_name", "updated_at"}),
		}).Create(secret).Error)
	}, sqlite3.BUSY, sqlite3.LOCKED)
}

// DeleteSecret implements port.SecretStore.
func (s *Store) DeleteSecret(ctx context.Context, orgID, pluginName, nodeID, key string) error {
	return s.withRetry(ctx, true, func(ctx context.Context, db *gorm.DB) error {
		return errors.WithStack(db.Where("node_id = ? AND key = ?", nodeID, key).Delete(&PluginNodeSecret{}).Error)
	}, sqlite3.BUSY, sqlite3.LOCKED)
}

// DeleteAllForNode implements port.SecretStore.
func (s *Store) DeleteAllForNode(ctx context.Context, nodeID string) error {
	return s.withRetry(ctx, true, func(ctx context.Context, db *gorm.DB) error {
		return errors.WithStack(db.Where("node_id = ?", nodeID).Delete(&PluginNodeSecret{}).Error)
	}, sqlite3.BUSY, sqlite3.LOCKED)
}

var _ port.SecretStore = &Store{}
