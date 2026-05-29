package main

import (
	"context"
	"encoding/json"
	"math"
	"strings"

	"github.com/bornholm/xolo/internal/estimator"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
	"github.com/bornholm/xolo/plugins/request-evaluator/complexity"
	"github.com/bornholm/xolo/plugins/request-evaluator/complexity/data"
)

const PluginName = "request-evaluator"
const PluginVersion = "0.1.0"

// Plugin implements the request-evaluator gRPC plugin.
// It analyses the request and emits typed outputs on its output ports:
//   complexity, category, energy_cost, budget_pressure,
//   estimated_input_tokens, estimated_output_tokens, has_vision, has_reasoning
type Plugin struct {
	proto.UnimplementedXoloPluginServer
}

func (p *Plugin) Describe(_ context.Context, _ *proto.DescribeRequest) (*proto.PluginDescriptor, error) {
	return &proto.PluginDescriptor{
		Name:         PluginName,
		Version:      PluginVersion,
		Description:  "Analyse la requête et produit des métriques typées (complexité, catégorie, énergie, budget) sur ses ports de sortie.",
		Capabilities: []proto.PluginDescriptor_Capability{proto.PluginDescriptor_PRE_REQUEST},
		InputPorts: []*proto.PortDescriptor{
			{Name: "request", PortType: "request", Required: true},
		},
		OutputPorts: []*proto.PortDescriptor{
			{Name: "complexity", PortType: "number"},
			{Name: "category", PortType: "string"},
			{Name: "energy_cost", PortType: "number"},
			{Name: "budget_pressure", PortType: "number"},
			{Name: "estimated_input_tokens", PortType: "number"},
			{Name: "estimated_output_tokens", PortType: "number"},
			{Name: "has_vision", PortType: "boolean"},
			{Name: "has_reasoning", PortType: "boolean"},
		},
	}, nil
}

// PreRequest analyses the request and stores results in outputs_json.
// Messages are passed through unchanged.
func (p *Plugin) PreRequest(_ context.Context, in *proto.PreRequestInput) (*proto.PreRequestOutput, error) {
	vars := scoreRequest(in.MessagesJson, in.Model, in.GetCtx().GetConfigJson(), in.GetCtx().GetConfigJson())

	outputs := map[string]interface{}{
		"complexity":              vars.Complexity,
		"category":                vars.Category,
		"energy_cost":             vars.EnergyCost,
		"budget_pressure":         vars.BudgetPressure,
		"estimated_input_tokens":  vars.EstimatedInputTokens,
		"estimated_output_tokens": vars.EstimatedOutputTokens,
		"has_vision":              vars.HasVision,
		"has_reasoning":           vars.HasReasoning,
	}
	b, _ := json.Marshal(outputs)

	return &proto.PreRequestOutput{
		Allowed:     true,
		OutputsJson: string(b),
	}, nil
}

// ─── Scoring logic (adapted from smart-model/scorer.go) ──────────────────────

type inputVars struct {
	Complexity              float64 `json:"complexity"`
	Category                string  `json:"category"`
	EstimatedInputTokens    int     `json:"estimated_input_tokens"`
	EstimatedOutputTokens   int     `json:"estimated_output_tokens"`
	EnergyKWh               float64 `json:"energy_kwh"`
	EnergyCost              float64 `json:"energy_cost"`
	BudgetPressure          float64 `json:"budget_pressure"`
	HasVision               bool    `json:"has_vision"`
	HasReasoning            bool    `json:"has_reasoning"`
}

func scoreRequest(messagesJSON, bodyJSON, _, quotaJSON string) inputVars {
	vars := inputVars{}

	fullText := extractTextForComplexity(messagesJSON)
	analysis := complexity.AnalyzeDefault(fullText)
	vars.Complexity = analysis.Composite
	vars.Category = topCategory(messagesJSON)

	vars.EstimatedInputTokens = analysis.Stats.TokenCount
	outputRatio := 0.2 + analysis.Composite*1.3
	vars.EstimatedOutputTokens = int(math.Min(float64(vars.EstimatedInputTokens)*outputRatio, 4096))
	if vars.EstimatedOutputTokens < 1 {
		vars.EstimatedOutputTokens = 64
	}

	req := estimator.InferenceRequest{
		InputTokens:  vars.EstimatedInputTokens,
		OutputTokens: vars.EstimatedOutputTokens,
	}
	est := estimator.NewCloudEstimator(estimator.TierMajorCloud)
	energyRange := est.EstimateFromParams(70.0, req, 0, 0)
	vars.EnergyKWh = energyRange.Mid.TotalKWh
	vars.EnergyCost = sigmoid(math.Log10(vars.EnergyKWh+1e-8)+8, 1.5)

	vars.BudgetPressure = computeBudgetPressure(quotaJSON)
	vars.HasVision = hasImageContent(messagesJSON)
	vars.HasReasoning = hasReasoningRequest(bodyJSON)

	return vars
}

func sigmoid(x, k float64) float64 { return 1 / (1 + math.Exp(-k*x)) }

func computeBudgetPressure(quotaJSON string) float64 {
	if quotaJSON == "" || quotaJSON == "{}" {
		return 0
	}
	var q struct {
		DailyTotal      float64 `json:"daily_total"`
		DailyRemaining  float64 `json:"daily_remaining"`
		MonthlyTotal    float64 `json:"monthly_total"`
		MonthlyRemaining float64 `json:"monthly_remaining"`
		YearlyTotal     float64 `json:"yearly_total"`
		YearlyRemaining float64 `json:"yearly_remaining"`
	}
	if err := json.Unmarshal([]byte(quotaJSON), &q); err != nil {
		return 0
	}
	var maxPressure float64
	if q.DailyTotal > 0 {
		maxPressure = math.Max(maxPressure, 1-q.DailyRemaining/q.DailyTotal)
	}
	if q.MonthlyTotal > 0 {
		maxPressure = math.Max(maxPressure, 1-q.MonthlyRemaining/q.MonthlyTotal)
	}
	if q.YearlyTotal > 0 {
		maxPressure = math.Max(maxPressure, 1-q.YearlyRemaining/q.YearlyTotal)
	}
	return math.Max(0, math.Min(1, maxPressure))
}

func extractTextForComplexity(messagesJSON string) string {
	if messagesJSON == "" {
		return ""
	}
	var messages []struct {
		Role      string          `json:"role"`
		Content   json.RawMessage `json:"content"`
		ToolCalls []struct {
			Function struct {
				Arguments string `json:"arguments"`
			} `json:"function"`
		} `json:"tool_calls"`
	}
	if err := json.Unmarshal([]byte(messagesJSON), &messages); err != nil {
		return messagesJSON
	}
	var sb strings.Builder
	for _, m := range messages {
		switch m.Role {
		case "system", "user", "tool":
			var s string
			if err := json.Unmarshal(m.Content, &s); err == nil {
				sb.WriteString(s)
				sb.WriteByte(' ')
				continue
			}
			var parts []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			if err := json.Unmarshal(m.Content, &parts); err == nil {
				for _, p := range parts {
					if p.Type == "text" {
						sb.WriteString(p.Text)
						sb.WriteByte(' ')
					}
				}
			}
		case "assistant":
			for _, tc := range m.ToolCalls {
				if tc.Function.Arguments != "" {
					sb.WriteString(tc.Function.Arguments)
					sb.WriteByte(' ')
				}
			}
		}
	}
	return sb.String()
}

func extractText(messagesJSON string) string {
	if messagesJSON == "" {
		return ""
	}
	var messages []struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal([]byte(messagesJSON), &messages); err != nil {
		return messagesJSON
	}
	var sb strings.Builder
	for _, m := range messages {
		if m.Role != "system" && m.Role != "user" {
			continue
		}
		var s string
		if err := json.Unmarshal(m.Content, &s); err == nil {
			sb.WriteString(s)
			sb.WriteByte(' ')
			continue
		}
		var parts []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal(m.Content, &parts); err == nil {
			for _, p := range parts {
				if p.Type == "text" {
					sb.WriteString(p.Text)
					sb.WriteByte(' ')
				}
			}
		}
	}
	return sb.String()
}

func topCategory(messagesJSON string) string {
	nb, err := complexity.LoadModel(data.RawModel)
	if err != nil {
		return "unknown"
	}
	text := extractText(messagesJSON)
	if text == "" {
		return "unknown"
	}
	pred := nb.Predict(text)
	return pred.Class
}

func hasReasoningRequest(bodyJSON string) bool {
	if bodyJSON == "" {
		return false
	}
	var body struct {
		Thinking        *struct{ Type string `json:"type"` } `json:"thinking"`
		EnableThinking  *bool                                `json:"enable_thinking"`
		ReasoningEffort string                               `json:"reasoning_effort"`
	}
	if err := json.Unmarshal([]byte(bodyJSON), &body); err != nil {
		return false
	}
	if body.Thinking != nil && body.Thinking.Type == "enabled" {
		return true
	}
	if body.EnableThinking != nil && *body.EnableThinking {
		return true
	}
	return body.ReasoningEffort != ""
}

func hasImageContent(messagesJSON string) bool {
	if messagesJSON == "" {
		return false
	}
	var messages []struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal([]byte(messagesJSON), &messages); err != nil {
		return false
	}
	for _, m := range messages {
		var parts []struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(m.Content, &parts); err == nil {
			for _, p := range parts {
				if p.Type == "image_url" || p.Type == "image" {
					return true
				}
			}
		}
	}
	return false
}
