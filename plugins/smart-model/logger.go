package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"time"

	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
)

// decisionLogEntry is a single decision log record (JSON-lines format).
type decisionLogEntry struct {
	Timestamp          string     `json:"ts"`
	OrgID              string     `json:"org_id"`
	UserID             string     `json:"user_id"`
	RequestedModel     string     `json:"requested_model"`
	SelectedModel      string     `json:"selected_model"`
	Category           string     `json:"category"`
	Complexity         float64    `json:"complexity"`
	BudgetPressure     float64    `json:"budget_pressure"`
	EnergyCost         float64    `json:"energy_cost"`
	EnergyKWh          float64    `json:"energy_kwh"`
	EstInputTokens     int        `json:"estimated_input_tokens"`
	EstOutputTokens    int        `json:"estimated_output_tokens"`
	PowerLevelInferred float64    `json:"power_level_inferred"`
	HasVision          bool       `json:"has_vision"`
	FeasibleModels     []string   `json:"feasible_models,omitempty"`
}

// writeDecisionLog appends a JSON-lines entry to the configured log file.
func writeDecisionLog(
	ctx context.Context,
	logPath string,
	in *proto.ResolveModelInput,
	vars InputVars,
	desiredPL float64,
	selected string,
) {
	if logPath == "" {
		logPath = "smart-model.jsonl"
	}

	// Collect feasible model names for diagnostics.
	var feasible []string
	for _, m := range in.AvailableModels {
		if !m.IsVirtual {
			feasible = append(feasible, m.ProxyName)
		}
	}

	entry := decisionLogEntry{
		Timestamp:          time.Now().UTC().Format(time.RFC3339),
		OrgID:              in.GetCtx().GetOrgId(),
		UserID:             in.GetCtx().GetUserId(),
		RequestedModel:     in.RequestedModel,
		SelectedModel:      selected,
		Category:           vars.Category,
		Complexity:         vars.Complexity,
		BudgetPressure:     vars.BudgetPressure,
		EnergyCost:         vars.EnergyCost,
		EnergyKWh:          vars.EnergyKWh,
		EstInputTokens:     vars.EstimatedInputTokens,
		EstOutputTokens:    vars.EstimatedOutputTokens,
		PowerLevelInferred: desiredPL,
		HasVision:          vars.HasVision,
		FeasibleModels:     feasible,
	}

	line, err := json.Marshal(entry)
	if err != nil {
		slog.WarnContext(ctx, "smart-model: failed to marshal decision log entry", slog.Any("error", err))
		return
	}

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.WarnContext(ctx, "smart-model: failed to open log file",
			slog.String("path", logPath),
			slog.Any("error", err),
		)
		return
	}
	defer f.Close()

	if _, err := f.Write(append(line, '\n')); err != nil {
		slog.WarnContext(ctx, "smart-model: failed to write decision log", slog.Any("error", err))
	}
}
