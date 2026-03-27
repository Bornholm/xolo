package main

import (
	"encoding/json"
	"math"
)

// Config is the persisted configuration for the smart-model plugin (per org).
type Config struct {
	// Rules is the fuzzy DSL string (variables + rules).
	Rules string `json:"rules"`
	// Models holds per-model overrides. Models not listed are enabled by default.
	Models []ModelConfig `json:"models,omitempty"`
	// TriggerModels is the list of virtual model names that activate the plugin.
	// If empty, the plugin handles all virtual model requests.
	TriggerModels []string `json:"trigger_models,omitempty"`
	// EnergySensitivity is the global energy minimisation weight (0=ignore energy, 1=maximise frugality).
	EnergySensitivity float64 `json:"energy_sensitivity"`
	// LogEnabled activates decision logging.
	LogEnabled bool `json:"log_enabled"`
	// LogPath is the path of the decision log file.
	LogPath string `json:"log_path"`
}

// ModelConfig holds optional overrides for a specific model.
type ModelConfig struct {
	// ProxyName is the model's proxy name as returned by the provider store.
	ProxyName string `json:"proxy_name"`
	// Enabled controls whether the model is a candidate for automatic selection (default true).
	Enabled bool `json:"enabled"`
	// PowerLevelOverride, when non-nil, overrides the auto-computed power level.
	PowerLevelOverride *float64 `json:"power_level_override,omitempty"`
	// Categories lists the request categories for which this model is preferred.
	// When non-empty the scorer can apply a bonus to steer selection toward this model.
	Categories []string `json:"categories,omitempty"`
}

// DefaultConfig returns the default configuration with built-in frugality rules.
func DefaultConfig() Config {
	return Config{
		Rules:             DefaultRulesDSL,
		EnergySensitivity: 0.6,
		LogEnabled:        false,
		LogPath:           "smart-model.jsonl",
	}
}

// ParseConfig deserialises a JSON config string. Missing fields fall back to defaults.
func ParseConfig(raw string) (Config, error) {
	cfg := DefaultConfig()
	if raw == "" || raw == "{}" {
		return cfg, nil
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// isTriggerSelected returns true when name is in triggerModels (or triggerModels is empty = all selected).
func isTriggerSelected(name string, triggerModels []string) bool {
	if len(triggerModels) == 0 {
		return false // empty = all virtual models, nothing pre-checked
	}
	for _, t := range triggerModels {
		if t == name {
			return true
		}
	}
	return false
}

// modelOverride returns the ModelConfig for the given proxy name, or a default (enabled=true, no override).
func (c Config) modelOverride(proxyName string) ModelConfig {
	for _, m := range c.Models {
		if m.ProxyName == proxyName {
			return m
		}
	}
	return ModelConfig{ProxyName: proxyName, Enabled: true}
}

// powerLevel returns the effective power level for a model, given its active params.
// activeParamsBillions == 0 means unknown (falls back to 0.5).
func (c Config) powerLevel(proxyName string, activeParamsBillions float32) float64 {
	override := c.modelOverride(proxyName)
	if override.PowerLevelOverride != nil {
		return *override.PowerLevelOverride
	}
	return computePowerLevel(activeParamsBillions)
}

// computePowerLevel maps active parameters (billions) to a [0,1] power level.
// Formula: clamp(log2(params) / log2(500), 0, 1)
// Examples: 7B→0.30, 13B→0.40, 70B→0.67, 175B→0.80, 400B→0.95
func computePowerLevel(paramsBillions float32) float64 {
	if paramsBillions <= 0 {
		// Unknown — use heuristic or default midpoint.
		return 0.5
	}
	pl := math.Log2(float64(paramsBillions)) / math.Log2(500)
	return math.Max(0, math.Min(1, pl))
}
