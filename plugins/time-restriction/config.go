package main

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

// configSchemaJSON est retourné dans PluginDescriptor.ConfigSchema.
const configSchemaJSON = `{
  "type": "object",
  "required": ["timezone", "slots"],
  "properties": {
    "timezone": {
      "type": "string",
      "title": "Fuseau horaire",
      "description": "Identifiant IANA, ex: Europe/Paris, UTC, America/New_York"
    },
    "slots": {
      "type": "array",
      "title": "Créneaux autorisés",
      "items": {
        "type": "object",
        "required": ["days", "start", "end"],
        "properties": {
          "days": {
            "type": "array",
            "title": "Jours de la semaine",
            "items": {
              "type": "string",
              "enum": ["monday","tuesday","wednesday","thursday","friday","saturday","sunday"]
            }
          },
          "start": {
            "type": "string",
            "title": "Heure de début",
            "description": "Format HH:MM (24h, zéro-paddé)"
          },
          "end": {
            "type": "string",
            "title": "Heure de fin",
            "description": "Format HH:MM (24h, zéro-paddé), exclusif. Les créneaux chevauchant minuit ne sont pas supportés."
          }
        }
      }
    }
  }
}`

// Config représente la configuration org-level du plugin.
type Config struct {
	Timezone string `json:"timezone"`
	Slots    []Slot `json:"slots"`
}

// Slot définit un créneau horaire autorisé pour des jours donnés.
// Start est inclusif, End est exclusif. Les créneaux où Start >= End
// ne correspondent à aucune heure (cas minuit non supporté).
type Slot struct {
	Days  []string `json:"days"`  // ex: ["monday","tuesday"]
	Start string   `json:"start"` // "HH:MM" zéro-paddé
	End   string   `json:"end"`   // "HH:MM" zéro-paddé, exclusif
}

// parseConfig désérialise configJSON en Config.
// Retourne Config{} sans erreur si configJSON est vide.
// Retourne une erreur si le JSON est invalide.
func parseConfig(configJSON string) (Config, error) {
	if configJSON == "" {
		return Config{}, nil
	}
	var cfg Config
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return Config{}, fmt.Errorf("parse plugin config: %w", err)
	}
	return cfg, nil
}

// isAllowed retourne true si now tombe dans au moins un créneau de cfg.
// Retourne false si cfg.Slots est vide ou si aucun créneau ne correspond
// à l'heure et au jour courants.
// Retourne une erreur si cfg.Timezone est invalide.
func isAllowed(now time.Time, cfg Config) (bool, error) {
	if len(cfg.Slots) == 0 {
		return false, nil
	}
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return false, fmt.Errorf("load timezone %q: %w", cfg.Timezone, err)
	}
	local := now.In(loc)
	currentDay := strings.ToLower(local.Weekday().String())
	currentHHMM := fmt.Sprintf("%02d:%02d", local.Hour(), local.Minute())
	for _, s := range cfg.Slots {
		if slices.Contains(s.Days, currentDay) &&
			currentHHMM >= s.Start &&
			currentHHMM < s.End {
			return true, nil
		}
	}
	return false, nil
}

