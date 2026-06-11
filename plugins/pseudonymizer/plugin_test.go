package main

import (
	"strings"
	"testing"
)

func TestInjectPlaceholderInstruction_NewSystemMessage(t *testing.T) {
	messages := []map[string]any{
		{"role": "user", "content": "Bonjour [PERSON_1], votre email est [EMAIL_1]."},
	}
	mapping := map[string]string{
		"[PERSON_1]": "William Petit",
		"[EMAIL_1]":  "wpetit@cadoles.com",
	}
	cfg := defaultConfig()

	got := injectPlaceholderInstruction(messages, mapping, cfg)

	if len(got) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(got))
	}
	if role, _ := got[0]["role"].(string); role != "system" {
		t.Fatalf("messages[0].role = %q, want system", role)
	}
	content, _ := got[0]["content"].(string)
	for _, placeholder := range []string{"[PERSON_1]", "[EMAIL_1]"} {
		if !strings.Contains(content, placeholder) {
			t.Errorf("instruction does not mention %q: %q", placeholder, content)
		}
	}
	// Original user message must be untouched.
	if got[1]["content"] != "Bonjour [PERSON_1], votre email est [EMAIL_1]." {
		t.Errorf("user message modified: %v", got[1]["content"])
	}
}

func TestInjectPlaceholderInstruction_PrependsToExistingSystemMessage(t *testing.T) {
	messages := []map[string]any{
		{"role": "system", "content": "Tu es un assistant utile."},
		{"role": "user", "content": "Bonjour [PERSON_1] !"},
	}
	mapping := map[string]string{"[PERSON_1]": "William Petit"}
	cfg := defaultConfig()

	got := injectPlaceholderInstruction(messages, mapping, cfg)

	if len(got) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(got))
	}
	content, _ := got[0]["content"].(string)
	if !strings.Contains(content, "[PERSON_1]") {
		t.Errorf("instruction does not mention [PERSON_1]: %q", content)
	}
	if !strings.Contains(content, "Tu es un assistant utile.") {
		t.Errorf("original system prompt lost: %q", content)
	}
}

func TestInjectPlaceholderInstruction_NoEntities(t *testing.T) {
	messages := []map[string]any{{"role": "user", "content": "Bonjour !"}}
	cfg := defaultConfig()

	got := injectPlaceholderInstruction(messages, nil, cfg)

	if len(got) != 1 {
		t.Fatalf("len(messages) = %d, want 1 (unchanged)", len(got))
	}
}

func TestInjectPlaceholderInstruction_RedactStrategySkipped(t *testing.T) {
	messages := []map[string]any{{"role": "user", "content": "Bonjour ████ !"}}
	mapping := map[string]string{"████████████": "William Petit"}
	cfg := defaultConfig()
	cfg.Strategy = "redact"

	got := injectPlaceholderInstruction(messages, mapping, cfg)

	if len(got) != 1 {
		t.Fatalf("len(messages) = %d, want 1 (unchanged)", len(got))
	}
}

func TestInjectPlaceholderInstruction_Disabled(t *testing.T) {
	messages := []map[string]any{{"role": "user", "content": "Bonjour [PERSON_1] !"}}
	mapping := map[string]string{"[PERSON_1]": "William Petit"}
	cfg := defaultConfig()
	cfg.InjectInstruction = false

	got := injectPlaceholderInstruction(messages, mapping, cfg)

	if len(got) != 1 {
		t.Fatalf("len(messages) = %d, want 1 (unchanged)", len(got))
	}
}
