package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ExtractSignals analyses messages_json and returns a map signal_name → value (0–1).
// Empty messages_json → all signals at 0 (no division by zero).
func ExtractSignals(messagesJSON string, patterns []SignalPattern) (map[string]float64, error) {
	result := make(map[string]float64, len(patterns))
	for _, p := range patterns {
		result[p.Name] = 0
	}

	if messagesJSON == "" {
		return result, nil
	}

	var messages []struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(messagesJSON), &messages); err != nil {
		return nil, fmt.Errorf("parse messages_json: %w", err)
	}

	if len(messages) == 0 {
		return result, nil
	}

	for _, p := range patterns {
		switch p.Type {
		case "keyword_ratio":
			matchCount := 0
			for _, msg := range messages {
				lower := strings.ToLower(msg.Content)
				for _, pat := range p.Patterns {
					if strings.Contains(lower, strings.ToLower(pat)) {
						matchCount++
						break // count each message at most once per signal
					}
				}
			}
			result[p.Name] = float64(matchCount) / float64(len(messages))

		case "keyword_count":
			total := 0
			for _, msg := range messages {
				lower := strings.ToLower(msg.Content)
				for _, pat := range p.Patterns {
					total += strings.Count(lower, strings.ToLower(pat))
				}
			}
			val := float64(total) / 100.0
			if val > 1.0 {
				val = 1.0
			}
			result[p.Name] = val
		}
	}

	return result, nil
}
