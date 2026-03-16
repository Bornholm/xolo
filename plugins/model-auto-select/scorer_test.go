package main

import (
	"testing"

	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
)

// dslCoding : code_ratio élevé → tag_coding élevé.
const dslCoding = `
	DEFINE code_ratio ( TERM high LINEAR (0.5, 1.0) );
	DEFINE tag_coding ( TERM high LINEAR (0.5, 1.0) );
	IF code_ratio IS high THEN tag_coding IS high;
`

// dslCheap : budget_preference bas → tag_cheap élevé.
const dslCheap = `
	DEFINE budget_preference ( TERM low LINEAR (4, 0) );
	DEFINE tag_cheap ( TERM high LINEAR (0.5, 1.0) );
	IF budget_preference IS low THEN tag_cheap IS high;
`

// model helper
func mi(name string, tokenLimit int64, cost float64) *proto.ModelInfo {
	return &proto.ModelInfo{
		ProxyName:              name,
		TokenLimit:             tokenLimit,
		PromptCostPer_1KTokens: cost,
	}
}

func TestScore_PrefersCoding(t *testing.T) {
	cfg := Config{
		BudgetPreference: 5,
		Rules:            dslCoding,
		Models: []ModelEntry{
			{ProxyName: "coding-model", Tags: []string{"coding"}},
			{ProxyName: "other-model", Tags: []string{}},
		},
	}
	signals := map[string]float64{"code_ratio": 1.0}
	available := []*proto.ModelInfo{
		mi("coding-model", 128000, 1.0),
		mi("other-model", 128000, 1.0),
	}

	got, err := Score(cfg, signals, available, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "coding-model" {
		t.Errorf("expected coding-model, got %q", got)
	}
}

func TestScore_PrefersCheap(t *testing.T) {
	cfg := Config{
		BudgetPreference: 1, // very low budget → economy
		Rules:            dslCheap,
		Models: []ModelEntry{
			{ProxyName: "cheap-model", Tags: []string{"cheap"}},
			{ProxyName: "expensive-model", Tags: []string{}},
		},
	}
	signals := map[string]float64{}
	available := []*proto.ModelInfo{
		mi("cheap-model", 128000, 0.5),
		mi("expensive-model", 128000, 5.0),
	}

	got, err := Score(cfg, signals, available, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "cheap-model" {
		t.Errorf("expected cheap-model, got %q", got)
	}
}

func TestScore_FiltersContextWindow(t *testing.T) {
	cfg := Config{
		BudgetPreference: 5,
		Rules:            dslCoding,
		Models: []ModelEntry{
			{ProxyName: "small-model", Tags: []string{"coding"}},
			{ProxyName: "large-model", Tags: []string{"coding"}},
		},
	}
	signals := map[string]float64{"code_ratio": 1.0}
	available := []*proto.ModelInfo{
		mi("small-model", 1000, 1.0),   // TokenLimit < estimatedTokens → filtered
		mi("large-model", 128000, 1.0), // TokenLimit >= estimatedTokens → kept
	}

	got, err := Score(cfg, signals, available, 5000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "large-model" {
		t.Errorf("expected large-model (small-model filtered out), got %q", got)
	}
}

func TestScore_TieBreakByCost(t *testing.T) {
	cfg := Config{
		BudgetPreference: 5,
		Rules:            dslCoding,
		Models: []ModelEntry{
			{ProxyName: "model-a", Tags: []string{"coding"}},
			{ProxyName: "model-b", Tags: []string{"coding"}},
		},
	}
	signals := map[string]float64{"code_ratio": 1.0}
	available := []*proto.ModelInfo{
		mi("model-a", 128000, 2.0),
		mi("model-b", 128000, 0.5), // cheaper → wins tie
	}

	got, err := Score(cfg, signals, available, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "model-b" {
		t.Errorf("expected model-b (cheapest on tie), got %q", got)
	}
}

func TestScore_NoEligibleModels(t *testing.T) {
	cfg := Config{
		BudgetPreference: 5,
		Rules:            dslCoding,
		Models: []ModelEntry{
			{ProxyName: "tiny-model", Tags: []string{"coding"}},
		},
	}
	signals := map[string]float64{"code_ratio": 1.0}
	available := []*proto.ModelInfo{
		mi("tiny-model", 100, 1.0), // TokenLimit=100 < estimatedTokens=999999 → filtered
	}

	got, err := Score(cfg, signals, available, 999999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string (no eligible model), got %q", got)
	}
}

func TestScore_SkipsUnknownProxyName(t *testing.T) {
	cfg := Config{
		BudgetPreference: 5,
		Rules:            dslCoding,
		Models: []ModelEntry{
			{ProxyName: "unknown-model", Tags: []string{"coding"}}, // not in available
			{ProxyName: "known-model", Tags: []string{"coding"}},
		},
	}
	signals := map[string]float64{"code_ratio": 1.0}
	available := []*proto.ModelInfo{
		mi("known-model", 128000, 1.0),
	}

	got, err := Score(cfg, signals, available, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "known-model" {
		t.Errorf("expected known-model (unknown-model skipped), got %q", got)
	}
}
