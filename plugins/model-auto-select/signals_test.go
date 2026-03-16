package main

import (
	"encoding/json"
	"testing"
)

// msgJSON serialises a list of messages in OpenAI format (content string).
func msgJSON(contents ...string) string {
	type msg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	messages := make([]msg, len(contents))
	for i, c := range contents {
		messages[i] = msg{Role: "user", Content: c}
	}
	b, _ := json.Marshal(messages)
	return string(b)
}

var codePatterns = []SignalPattern{
	{Name: "code_ratio", Type: "keyword_ratio", Patterns: []string{"```", "def ", "function "}},
}

var reasoningPatterns = []SignalPattern{
	{Name: "reasoning_ratio", Type: "keyword_ratio", Patterns: []string{"analyze", "why", "explain"}},
}

func TestExtractSignals_CodeHeavy(t *testing.T) {
	j := msgJSON("```python\ndef foo(): pass\n```", "```js\nfunction bar() {}\n```")
	signals, err := ExtractSignals(j, codePatterns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if signals["code_ratio"] < 0.9 {
		t.Errorf("expected code_ratio ≈ 1, got %f", signals["code_ratio"])
	}
}

func TestExtractSignals_NoCode(t *testing.T) {
	j := msgJSON("Hello, how are you?", "What is the weather like today?")
	signals, err := ExtractSignals(j, codePatterns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if signals["code_ratio"] != 0 {
		t.Errorf("expected code_ratio=0, got %f", signals["code_ratio"])
	}
}

func TestExtractSignals_ReasoningKeywords(t *testing.T) {
	j := msgJSON("Can you analyze this text?", "Why does this work?", "Please explain the concept.")
	signals, err := ExtractSignals(j, reasoningPatterns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if signals["reasoning_ratio"] < 0.9 {
		t.Errorf("expected reasoning_ratio ≈ 1, got %f", signals["reasoning_ratio"])
	}
}

func TestExtractSignals_MultiplePatterns(t *testing.T) {
	j := msgJSON(
		"```python\ndef foo(): pass\n```",
		"Why does this work? Please analyze.",
	)
	all := []SignalPattern{
		{Name: "code_ratio", Type: "keyword_ratio", Patterns: []string{"```", "def "}},
		{Name: "reasoning_ratio", Type: "keyword_ratio", Patterns: []string{"why", "analyze"}},
	}
	signals, err := ExtractSignals(j, all)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if signals["code_ratio"] < 0.4 {
		t.Errorf("expected code_ratio > 0, got %f", signals["code_ratio"])
	}
	if signals["reasoning_ratio"] < 0.4 {
		t.Errorf("expected reasoning_ratio > 0, got %f", signals["reasoning_ratio"])
	}
}

func TestExtractSignals_EmptyMessages(t *testing.T) {
	signals, err := ExtractSignals("", codePatterns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if signals["code_ratio"] != 0 {
		t.Errorf("expected code_ratio=0 for empty input, got %f", signals["code_ratio"])
	}
}
