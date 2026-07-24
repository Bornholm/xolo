package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Config struct {
	CacheDir         string   `json:"cache_dir"`
	ManifestURL      string   `json:"manifest_url"`
	Offline          bool     `json:"offline"`
	Model            string   `json:"model"`
	Language         string   `json:"language"`
	Threshold        float64  `json:"threshold"`
	LabelsToBlock    []string `json:"labels_to_block"`
	FailureMode      string   `json:"failure_mode"`
	RejectionMessage string   `json:"rejection_message"`
}

const configSchemaJSON = `{
  "type": "object",
  "properties": {
    "model":          { "type": "string", "enum": ["safety-fr","auto"], "default": "safety-fr", "title": "Modèle", "description": "Nom du modèle ou 'auto' pour détection automatique de la langue." },
    "language":       { "type": "string", "enum": ["fr","en","auto"], "default": "fr", "title": "Langue", "description": "Langue de fallback quand 'auto' est sélectionné." },
    "threshold":      { "type": "number", "minimum": 0, "maximum": 1, "default": 0.5, "title": "Seuil de confiance", "description": "Score minimum (0.0–1.0) pour déclencher un blocage." },
    "labels_to_block":{ "type": "array",  "items": {"type": "string"}, "default": ["insecure"], "title": "Labels bloquants", "description": "Liste des labels considérés comme sensibles." },
    "failure_mode":   { "type": "string", "enum": ["allow","block"], "default": "allow", "title": "Mode dégradé", "description": "Comportement si le modèle ne peut pas être chargé." },
    "rejection_message": { "type": "string", "title": "Message de rejet personnalisé", "description": "Message renvoyé au client en cas de blocage. Placeholders : {label}, {score}, {threshold}. Laisser vide pour le défaut localisé." },
    "cache_dir":      { "type": "string", "title": "Répertoire de cache", "description": "Répertoire local pour stocker les modèles téléchargés." },
    "manifest_url":   { "type": "string", "title": "URL du manifest", "description": "URL du manifest des modèles disponibles." },
    "offline":        { "type": "boolean", "default": false, "title": "Mode hors-ligne", "description": "Désactive les téléchargements réseau." }
  }
}`

func defaultConfig() Config {
	return Config{
		Model:         "safety-fr",
		Language:      "fr",
		Threshold:     0.5,
		LabelsToBlock: []string{"insecure"},
		FailureMode:   "allow",
	}
}

func parseConfig(configJSON string) (Config, error) {
	cfg := defaultConfig()
	if configJSON == "" {
		return cfg, nil
	}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Model == "" {
		cfg.Model = "safety-fr"
	}
	if cfg.Language == "" {
		cfg.Language = "fr"
	}
	if cfg.Threshold == 0 {
		cfg.Threshold = 0.5
	}
	if cfg.FailureMode == "" {
		cfg.FailureMode = "allow"
	}
	if len(cfg.LabelsToBlock) == 0 {
		cfg.LabelsToBlock = []string{"insecure"}
	}
	return cfg, nil
}

func configFromForm(r *http.Request) Config {
	cfg := defaultConfig()
	cfg.Model = r.FormValue("model")
	cfg.Language = r.FormValue("language")
	cfg.FailureMode = r.FormValue("failure_mode")
	cfg.CacheDir = r.FormValue("cache_dir")
	cfg.ManifestURL = r.FormValue("manifest_url")
	cfg.Offline = r.FormValue("offline") == "on"
	cfg.RejectionMessage = strings.TrimSpace(r.FormValue("rejection_message"))

	if t := r.FormValue("threshold"); t != "" {
		var f float64
		if err := json.Unmarshal([]byte(t), &f); err == nil {
			cfg.Threshold = f
		}
	}
	if raw := r.FormValue("labels_to_block"); raw != "" {
		parts := strings.Split(raw, ",")
		cfg.LabelsToBlock = make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				cfg.LabelsToBlock = append(cfg.LabelsToBlock, p)
			}
		}
	}
	return cfg
}