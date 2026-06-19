package main

import (
	"encoding/json"
	"fmt"
)

// configSchemaJSON est retourné dans PluginDescriptor.ConfigSchema.
// La valeur d'authentification n'apparaît pas ici : elle est stockée via
// SetSecret, jamais dans la config visible du nœud.
const configSchemaJSON = `{
  "type": "object",
  "required": ["endpoint"],
  "properties": {
    "endpoint": {
      "type": "string",
      "title": "URL du serveur MCP",
      "description": "Endpoint Streamable HTTP, ex: https://mcp.example.com/mcp"
    },
    "authHeaderName": {
      "type": "string",
      "title": "Nom de l'en-tête d'authentification",
      "description": "ex: Authorization"
    },
    "toolFilter": {
      "type": "array",
      "title": "Tools autorisés (vide = tous)",
      "items": { "type": "string" }
    },
    "timeoutSeconds": {
      "type": "number",
      "title": "Timeout (secondes)"
    },
    "maxConsecutiveToolCalls": {
      "type": "number",
      "title": "Appels d'outils consécutifs max.",
      "description": "Au-delà, le LLM est forcé de répondre sans appeler d'outil supplémentaire. Défaut: 2."
    }
  }
}`

// Config représente la configuration non-sensible du nœud (PluginNodeData.Config).
type Config struct {
	Endpoint                string   `json:"endpoint"`
	AuthHeaderName          string   `json:"authHeaderName"`
	ToolFilter              []string `json:"toolFilter,omitempty"`
	TimeoutSeconds          int      `json:"timeoutSeconds,omitempty"`
	MaxConsecutiveToolCalls int      `json:"maxConsecutiveToolCalls,omitempty"`
}

// parseConfig désérialise configJSON en Config.
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
