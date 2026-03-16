package main

import (
	"testing"
)

const validConfigJSON = `{
	"virtual_model": "auto",
	"budget_preference": 5,
	"signals": [
		{"name": "code_ratio", "type": "keyword_ratio", "patterns": ["` + "`" + `", "def "]}
	],
	"rules": "DEFINE code_ratio ( TERM high LINEAR (0.5, 1.0) ); DEFINE tag_coding ( TERM high LINEAR (0.5, 1.0) ); IF code_ratio IS high THEN tag_coding IS high;",
	"models": [{"proxy_name": "gpt-4o", "tags": ["coding"]}]
}`

func TestParseConfig_Valid(t *testing.T) {
	cfg, err := parseConfig(validConfigJSON)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.VirtualModel != "auto" {
		t.Errorf("expected virtual_model=auto, got %q", cfg.VirtualModel)
	}
	if cfg.BudgetPreference != 5 {
		t.Errorf("expected budget_preference=5, got %f", cfg.BudgetPreference)
	}
	if len(cfg.Signals) != 1 || cfg.Signals[0].Name != "code_ratio" {
		t.Errorf("expected 1 signal named code_ratio, got %+v", cfg.Signals)
	}
	if len(cfg.Models) != 1 || cfg.Models[0].ProxyName != "gpt-4o" {
		t.Errorf("expected 1 model gpt-4o, got %+v", cfg.Models)
	}
}

func TestParseConfig_EmptyString(t *testing.T) {
	cfg, err := parseConfig("")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.VirtualModel != "auto" {
		t.Errorf("expected default virtual_model=auto, got %q", cfg.VirtualModel)
	}
}

func TestParseConfig_InvalidDSL(t *testing.T) {
	bad := `{"rules": "BADDSLINVALIDSTUFF!!!"}`
	_, err := parseConfig(bad)
	if err == nil {
		t.Fatal("expected error for invalid DSL, got nil")
	}
}

func TestParseConfig_InvalidJSON(t *testing.T) {
	_, err := parseConfig("{invalid json")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestParseConfig_DefaultVirtualModel(t *testing.T) {
	cfg, err := parseConfig(`{"budget_preference": 7}`)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.VirtualModel != "auto" {
		t.Errorf("expected default virtual_model=auto, got %q", cfg.VirtualModel)
	}
}
