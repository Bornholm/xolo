package main

import (
	"fmt"

	fuzzy "github.com/bornholm/go-fuzzy"
	"github.com/bornholm/go-fuzzy/dsl"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
)

// Score evaluates each model and returns the winning proxy_name.
// Returns "" if no eligible model.
func Score(
	cfg Config,
	signals map[string]float64,
	available []*proto.ModelInfo,
	estimatedTokens int64,
) (string, error) {
	if len(cfg.Models) == 0 || len(available) == 0 {
		return "", nil
	}

	// Build values: signals + budget_preference.
	values := fuzzy.Values{"budget_preference": cfg.BudgetPreference}
	for k, v := range signals {
		values[k] = v
	}

	// Parse DSL rules and variables.
	parseResult, err := dsl.ParseRulesAndVariables(cfg.Rules)
	if err != nil {
		return "", fmt.Errorf("parse fuzzy rules: %w", err)
	}

	// Create engine and register variables + rules.
	engine := fuzzy.NewEngine(fuzzy.Centroid(1000)).
		Variables(parseResult.Variables...).
		Rules(parseResult.Rules...)

	// Run fuzzy inference.
	results, err := engine.Infer(values)
	if err != nil {
		return "", fmt.Errorf("fuzzy inference: %w", err)
	}

	// Build index of available models by proxy_name.
	availableMap := make(map[string]*proto.ModelInfo, len(available))
	for _, m := range available {
		availableMap[m.ProxyName] = m
	}

	type candidate struct {
		proxyName string
		score     float64
		cost      float64
	}

	var best *candidate

	for _, entry := range cfg.Models {
		info, ok := availableMap[entry.ProxyName]
		if !ok {
			continue // not in available list
		}
		if info.TokenLimit > 0 && info.TokenLimit < estimatedTokens {
			continue // context window too small
		}

		var score float64
		for _, tag := range entry.Tags {
			if result, ok := results.Best("tag_" + tag); ok {
				score += result.TruthDegree()
			}
		}

		c := &candidate{
			proxyName: entry.ProxyName,
			score:     score,
			cost:      info.PromptCostPer_1KTokens,
		}

		if best == nil || c.score > best.score || (c.score == best.score && c.cost < best.cost) {
			best = c
		}
	}

	if best == nil {
		return "", nil
	}

	return best.proxyName, nil
}
