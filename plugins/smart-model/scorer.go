package main

import (
	"encoding/json"
	"math"
	"strings"

	"github.com/bornholm/xolo/internal/estimator"
	"github.com/bornholm/xolo/pkg/pluginsdk/proto"
	"github.com/bornholm/xolo/plugins/smart-model/complexity"
	"github.com/bornholm/xolo/plugins/smart-model/complexity/data"
)

// InputVars holds all computed fuzzy input variable values (all in [0,1]).
type InputVars struct {
	// Complexity is the composite complexity score of the request (0=trivial, 1=very complex).
	Complexity float64 `json:"complexity"`
	// Category is the top NaiveBayes category (informational only, not a fuzzy variable).
	Category string `json:"category"`
	// ComplexityLabel is the human-readable complexity level (trivial/simple/moderate/complex/very_complex).
	ComplexityLabel string `json:"complexity_label"`
	// EstimatedInputTokens is the estimated input token count.
	EstimatedInputTokens int `json:"estimated_input_tokens"`
	// EstimatedOutputTokens is the estimated output token count (derived from complexity).
	EstimatedOutputTokens int `json:"estimated_output_tokens"`
	// EnergyKWh is the estimated energy consumption in kWh for a reference model.
	EnergyKWh float64 `json:"energy_kwh"`
	// EnergyCost is a normalised [0,1] energy score for the request.
	EnergyCost float64 `json:"energy_cost"`
	// BudgetPressure is the ratio of quota consumed vs total (0=plenty of budget, 1=exhausted).
	BudgetPressure float64 `json:"budget_pressure"`
	// HasVision is true when the request contains image inputs.
	HasVision bool `json:"has_vision"`
	// HasReasoning is true when the request explicitly enables extended reasoning/thinking.
	HasReasoning bool `json:"has_reasoning"`
}

// ScoreRequest computes fuzzy input variables from the incoming request.
func ScoreRequest(messagesJSON string, bodyJSON string, quota *proto.QuotaInfo, cfg Config) InputVars {
	vars := InputVars{}

	// ── Complexity & category ────────────────────────────────────────────────
	fullText := extractTextForComplexity(messagesJSON)
	analysis := complexity.AnalyzeDefault(fullText)
	vars.Complexity = analysis.Composite
	vars.ComplexityLabel = analysis.Level
	vars.Category = topCategory(messagesJSON)

	// ── Token estimation ─────────────────────────────────────────────────────
	vars.EstimatedInputTokens = analysis.Stats.TokenCount
	// Output estimate: input × (0.2 + complexity × 1.3), capped at 4096
	outputRatio := 0.2 + analysis.Composite*1.3
	vars.EstimatedOutputTokens = int(math.Min(float64(vars.EstimatedInputTokens)*outputRatio, 4096))
	if vars.EstimatedOutputTokens < 1 {
		vars.EstimatedOutputTokens = 64
	}

	// ── Energy estimation (70B model on MajorCloud as baseline reference) ────
	req := estimator.InferenceRequest{
		InputTokens:  vars.EstimatedInputTokens,
		OutputTokens: vars.EstimatedOutputTokens,
	}
	est := estimator.NewCloudEstimator(estimator.TierMajorCloud)
	energyRange := est.EstimateFromParams(70.0, req, 0, 0) // 70B baseline
	vars.EnergyKWh = energyRange.Mid.TotalKWh

	// Normalise energy cost: sigmoid centred at 0.001 kWh (typical simple request).
	// 0.0001 kWh → ~0.12, 0.001 kWh → ~0.5, 0.01 kWh → ~0.88
	vars.EnergyCost = sigmoid(math.Log10(vars.EnergyKWh+1e-8)+8, 1.5)

	// ── Budget pressure ───────────────────────────────────────────────────────
	vars.BudgetPressure = computeBudgetPressure(quota)

	// ── Vision detection ─────────────────────────────────────────────────────
	vars.HasVision = hasImageContent(messagesJSON)

	// ── Reasoning detection ───────────────────────────────────────────────────
	vars.HasReasoning = hasReasoningRequest(bodyJSON)

	return vars
}

// sigmoid returns 1/(1+e^(-k*x)).
func sigmoid(x, k float64) float64 {
	return 1 / (1 + math.Exp(-k*x))
}

// computeBudgetPressure returns the maximum quota consumption ratio across all periods.
// Returns 0 if no quota is defined.
func computeBudgetPressure(quota *proto.QuotaInfo) float64 {
	if quota == nil {
		return 0
	}
	var maxPressure float64
	if quota.DailyTotal > 0 {
		p := 1 - quota.DailyRemaining/quota.DailyTotal
		maxPressure = math.Max(maxPressure, p)
	}
	if quota.MonthlyTotal > 0 {
		p := 1 - quota.MonthlyRemaining/quota.MonthlyTotal
		maxPressure = math.Max(maxPressure, p)
	}
	if quota.YearlyTotal > 0 {
		p := 1 - quota.YearlyRemaining/quota.YearlyTotal
		maxPressure = math.Max(maxPressure, p)
	}
	return math.Max(0, math.Min(1, maxPressure))
}

// extractTextForComplexity concatène les textes des messages system, user,
// tool (résultats), et les arguments des tool_calls assistant.
// Utilisé pour l'analyse de complexité et l'estimation de tokens.
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

// extractText concatenates user/system message text for category classification.
// Intentionally excludes tool results to preserve classification quality.
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

// topCategory returns the best NaiveBayes category for the request text.
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

// hasReasoningRequest returns true when the request body explicitly enables extended
// reasoning/thinking, as indicated by common API parameters across providers:
//   - "thinking": {"type": "enabled"} (Anthropic)
//   - "enable_thinking": true (some Mistral/OpenAI-compat variants)
//   - "reasoning_effort": non-empty (OpenAI o-series)
func hasReasoningRequest(bodyJSON string) bool {
	if bodyJSON == "" {
		return false
	}
	var body struct {
		Thinking       *struct{ Type string `json:"type"` } `json:"thinking"`
		EnableThinking *bool                                `json:"enable_thinking"`
		ReasoningEffort string                              `json:"reasoning_effort"`
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
	if body.ReasoningEffort != "" {
		return true
	}
	return false
}

// hasImageContent returns true if any message part is an image_url or image type.
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
