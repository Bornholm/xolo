package main

import "encoding/json"

type Config struct {
	SystemPrompt string `json:"system_prompt"`
	Append       bool   `json:"append"`
}

func parseConfig(raw string) Config {
	cfg := Config{}
	if raw != "" && raw != "{}" {
		json.Unmarshal([]byte(raw), &cfg) //nolint:errcheck
	}
	return cfg
}