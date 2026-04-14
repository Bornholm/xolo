package main

import (
	"strings"
	"testing"
)

func TestExtractTextForComplexity_IncludesToolResults(t *testing.T) {
	messagesJSON := `[
		{"role":"system","content":"You are helpful."},
		{"role":"user","content":"What is the weather in Paris?"},
		{"role":"assistant","tool_calls":[{"function":{"name":"get_weather","arguments":"{\"location\":\"Paris\"}"}}]},
		{"role":"tool","tool_call_id":"call_1","content":"It is 18°C and sunny."}
	]`

	text := extractTextForComplexity(messagesJSON)

	if !strings.Contains(text, "You are helpful.") {
		t.Error("should contain system message")
	}
	if !strings.Contains(text, "What is the weather in Paris?") {
		t.Error("should contain user message")
	}
	if !strings.Contains(text, `"location":"Paris"`) {
		t.Error("should contain tool_call arguments")
	}
	if !strings.Contains(text, "It is 18°C and sunny.") {
		t.Error("should contain tool result")
	}
}

func TestExtractText_ExcludesToolResults(t *testing.T) {
	messagesJSON := `[
		{"role":"system","content":"You are helpful."},
		{"role":"user","content":"What is the weather in Paris?"},
		{"role":"assistant","tool_calls":[{"function":{"name":"get_weather","arguments":"{\"location\":\"Paris\"}"}}]},
		{"role":"tool","tool_call_id":"call_1","content":"It is 18°C and sunny."}
	]`

	text := extractText(messagesJSON)

	if !strings.Contains(text, "You are helpful.") {
		t.Error("should contain system message")
	}
	if !strings.Contains(text, "What is the weather in Paris?") {
		t.Error("should contain user message")
	}
	if strings.Contains(text, "It is 18°C and sunny.") {
		t.Error("should NOT contain tool result")
	}
	if strings.Contains(text, `"location":"Paris"`) {
		t.Error("should NOT contain tool_call arguments")
	}
}

func TestTopCategory_UnaffectedByToolResults(t *testing.T) {
	withoutTools := `[
		{"role":"user","content":"Write a Python function to sort a list."}
	]`
	withTools := `[
		{"role":"user","content":"Write a Python function to sort a list."},
		{"role":"tool","tool_call_id":"call_1","content":"[{\"temperature\":20},{\"wind\":\"5km/h\"}]"}
	]`

	cat1 := topCategory(withoutTools)
	cat2 := topCategory(withTools)

	if cat1 != cat2 {
		t.Errorf("category should be stable with/without tool results: got %q vs %q", cat1, cat2)
	}
}
