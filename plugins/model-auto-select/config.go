package main

import (
	"encoding/json"
	"fmt"

	"github.com/bornholm/go-fuzzy/dsl"
)

const configSchemaJSON = `{
  "type": "object",
  "properties": {
    "virtual_model": {
      "type": "string",
      "title": "Nom du modèle virtuel",
      "description": "Le plugin s'active uniquement si requested_model == virtual_model",
      "default": "auto"
    },
    "budget_preference": {
      "type": "number",
      "title": "Préférence budgétaire",
      "description": "0 = économie maximale, 10 = qualité maximale",
      "minimum": 0,
      "maximum": 10,
      "default": 5
    },
    "signals": {
      "type": "array",
      "title": "Détecteurs de signaux",
      "items": {
        "type": "object",
        "required": ["name", "type", "patterns"],
        "properties": {
          "name":     { "type": "string" },
          "type":     { "type": "string", "enum": ["keyword_ratio", "keyword_count"] },
          "patterns": { "type": "array", "items": { "type": "string" } }
        }
      }
    },
    "rules": {
      "type": "string",
      "title": "Règles floues (DSL go-fuzzy)"
    },
    "models": {
      "type": "array",
      "title": "Modèles avec tags",
      "items": {
        "type": "object",
        "required": ["proxy_name", "tags"],
        "properties": {
          "proxy_name": { "type": "string" },
          "tags":       { "type": "array", "items": { "type": "string" } }
        }
      }
    }
  }
}`

type Config struct {
	VirtualModel     string          `json:"virtual_model"`
	BudgetPreference float64         `json:"budget_preference"`
	Signals          []SignalPattern `json:"signals"`
	Rules            string          `json:"rules"`
	Models           []ModelEntry    `json:"models"`
	// UI metadata — ignored by the inference engine
	UIMode         string          `json:"ui_mode"`         // "simple" | "advanced"; default "simple"
	SimpleMappings []SimpleMapping `json:"simple_mappings"` // associations for simple mode
}

type SimpleMapping struct {
	Signal string `json:"signal"` // signal name OR "budget_preference"
	Tag    string `json:"tag"`    // tag without "tag_" prefix
}

type SignalPattern struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Patterns []string `json:"patterns"`
}

type ModelEntry struct {
	ProxyName string   `json:"proxy_name"`
	Tags      []string `json:"tags"`
}

// parseConfig deserializes configJSON into Config.
// Empty string → Config{VirtualModel: "auto"}, nil.
// Invalid DSL → error.
func parseConfig(configJSON string) (Config, error) {
	if configJSON == "" {
		return Config{VirtualModel: "auto"}, nil
	}
	var cfg Config
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return Config{}, fmt.Errorf("parse plugin config: %w", err)
	}
	if cfg.VirtualModel == "" {
		cfg.VirtualModel = "auto"
	}
	if cfg.UIMode == "" {
		cfg.UIMode = "simple"
	}
	if cfg.Rules != "" {
		if _, err := dsl.ParseRulesAndVariables(cfg.Rules); err != nil {
			return Config{}, fmt.Errorf("invalid fuzzy rules: %w", err)
		}
	}
	return cfg, nil
}
