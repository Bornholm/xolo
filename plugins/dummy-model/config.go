package main

import "encoding/json"

// Config is the persisted configuration for the dummy-model plugin (per org).
type Config struct {
	// TriggerModels lists the virtual model names that activate this plugin.
	// If empty, the plugin is inactive (no model intercepted).
	TriggerModels []string `json:"trigger_models,omitempty"`
}

// ParseConfig deserialises a JSON config string. Returns an empty Config on empty input.
func ParseConfig(raw string) (Config, error) {
	var cfg Config
	if raw == "" || raw == "{}" {
		return cfg, nil
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// isTriggerModel returns true when name matches one of the configured trigger models.
// Returns false when the list is empty (plugin inactive).
func (c Config) isTriggerModel(name string) bool {
	for _, t := range c.TriggerModels {
		if t == name {
			return true
		}
	}
	return false
}
