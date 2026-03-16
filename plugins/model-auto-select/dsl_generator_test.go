package main

import (
	"strings"
	"testing"

	"github.com/bornholm/go-fuzzy/dsl"
)

func TestGenerateDSL_SingleMapping(t *testing.T) {
	mappings := []SimpleMapping{
		{Signal: "code_ratio", Tag: "coding"},
	}
	result := GenerateDSL(mappings)

	if !strings.Contains(result, "DEFINE code_ratio") {
		t.Errorf("missing DEFINE code_ratio in:\n%s", result)
	}
	if !strings.Contains(result, "IF code_ratio IS high THEN tag_coding IS preferred") {
		t.Errorf("missing rule in:\n%s", result)
	}
	if _, err := dsl.ParseRulesAndVariables(result); err != nil {
		t.Errorf("generated DSL is invalid: %v\nDSL:\n%s", err, result)
	}
}

func TestGenerateDSL_BudgetMapping(t *testing.T) {
	mappings := []SimpleMapping{
		{Signal: "budget_preference", Tag: "cheap"},
	}
	result := GenerateDSL(mappings)

	if !strings.Contains(result, "DEFINE budget_preference") {
		t.Errorf("missing DEFINE budget_preference in:\n%s", result)
	}
	if !strings.Contains(result, "IF budget_preference IS low THEN tag_cheap IS preferred") {
		t.Errorf("missing budget rule in:\n%s", result)
	}
	if _, err := dsl.ParseRulesAndVariables(result); err != nil {
		t.Errorf("generated DSL is invalid: %v", err)
	}
}

func TestGenerateDSL_MultipleSignals_NoDuplicateDefine(t *testing.T) {
	mappings := []SimpleMapping{
		{Signal: "code_ratio", Tag: "coding"},
		{Signal: "reasoning_ratio", Tag: "reasoning"},
	}
	result := GenerateDSL(mappings)

	defineCount := strings.Count(result, "DEFINE code_ratio")
	if defineCount != 1 {
		t.Errorf("expected 1 DEFINE code_ratio, got %d", defineCount)
	}
	if _, err := dsl.ParseRulesAndVariables(result); err != nil {
		t.Errorf("generated DSL is invalid: %v", err)
	}
}

func TestGenerateDSL_Empty_ReturnsEmpty(t *testing.T) {
	result := GenerateDSL([]SimpleMapping{})
	if strings.TrimSpace(result) != "" {
		t.Errorf("expected empty string for empty mappings, got %q", result)
	}
}
