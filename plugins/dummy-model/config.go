package main

import "encoding/json"

const defaultResponseTemplate = "**[dummy-model — réponse de test]**\n\n" +
	"- **Utilisateur** : {{.User}}\n" +
	"- **Dernier message** : {{.LastMessage}}\n\n" +
	"_Cette réponse a été produite par le plugin dummy-model à des fins de test, sans appel à un LLM réel._"

// Config is the per-node configuration for the dummy-model plugin.
type Config struct {
	// ResponseTemplate is the Markdown text returned as the forged LLM response.
	// Supports simple placeholders: {{.User}}, {{.LastMessage}}.
	// Defaults to defaultResponseTemplate when empty.
	ResponseTemplate string `json:"response_template,omitempty"`
}

// ParseConfig deserialises a JSON config string.
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

func (c Config) template() string {
	if c.ResponseTemplate != "" {
		return c.ResponseTemplate
	}
	return defaultResponseTemplate
}
