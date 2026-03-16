package model

import "time"

// PluginActivation records whether a plugin is enabled for an org.
type PluginActivation struct {
	OrgID      OrgID
	PluginName string
	Enabled    bool
	Required   bool
	Order      int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// PluginConfig stores the JSON configuration for a plugin in a given scope.
type PluginConfig struct {
	OrgID      OrgID
	PluginName string
	Scope      PluginConfigScope
	ScopeID    string
	ConfigJSON string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type PluginConfigScope string

const (
	PluginConfigScopeOrg  PluginConfigScope = "org"
	PluginConfigScopeUser PluginConfigScope = "user"
)
