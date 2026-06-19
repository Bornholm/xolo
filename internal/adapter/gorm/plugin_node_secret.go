package gorm

import "time"

// PluginNodeSecret persists an encrypted key/value secret scoped to a single
// plugin node instance within a pipeline graph.
type PluginNodeSecret struct {
	ID             string `gorm:"primaryKey;autoIncrement:false"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	OrgID          string `gorm:"index;not null"`
	PluginName     string `gorm:"index;not null"`
	NodeID         string `gorm:"uniqueIndex:idx_plugin_node_secret_node_key;not null"`
	Key            string `gorm:"uniqueIndex:idx_plugin_node_secret_node_key;not null"`
	ValueEncrypted string `gorm:"not null"` // AES-GCM encrypted hex
}
