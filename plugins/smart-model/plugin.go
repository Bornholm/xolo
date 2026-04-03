package main

import (
	"context"
	"log/slog"
	"math"
	"strings"

	fuzzy "github.com/bornholm/go-fuzzy"
	"github.com/bornholm/go-fuzzy/dsl"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
)

const PluginName = "smart-model"
const PluginVersion = "0.1.0"

// Plugin implements the smart-model gRPC plugin.
type Plugin struct {
	proto.UnimplementedXoloPluginServer
}

func (p *Plugin) Describe(_ context.Context, _ *proto.DescribeRequest) (*proto.PluginDescriptor, error) {
	return &proto.PluginDescriptor{
		Name:         PluginName,
		Version:      PluginVersion,
		Description:  "Sélection automatique du modèle LLM optimal via logique floue (complexité, énergie, budget).",
		Capabilities: []proto.PluginDescriptor_Capability{proto.PluginDescriptor_RESOLVE_MODEL},
		ConfigSchema: configSchemaJSON,
	}, nil
}

// ResolveModel runs the full smart-model selection pipeline:
//  1. Parse config
//  2. Compute input variables (complexity, energy, budget)
//  3. Run fuzzy inference → desired power_level
//  4. Select the best feasible model
func (p *Plugin) ResolveModel(ctx context.Context, in *proto.ResolveModelInput) (*proto.ResolveModelOutput, error) {
	cfg, err := ParseConfig(in.GetCtx().GetConfigJson())
	if err != nil {
		slog.WarnContext(ctx, "smart-model: failed to parse config, skipping", slog.Any("error", err))
		return &proto.ResolveModelOutput{}, nil
	}

	// If the requested model is not a known virtual model, pass through.
	if !isVirtualModel(in.RequestedModel, in.VirtualModels) {
		return &proto.ResolveModelOutput{}, nil
	}

	// Only handle requests targeting one of the configured trigger models.
	// An empty list means the plugin is disabled (no model selected = inactive).
	if !isTriggerModel(in.RequestedModel, cfg.TriggerModels) {
		return &proto.ResolveModelOutput{}, nil
	}

	// Compute fuzzy input variables.
	vars := ScoreRequest(in.MessagesJson, in.GetBodyJson(), in.GetQuota(), cfg)
	slog.DebugContext(ctx, "smart-model: scored request",
		slog.Float64("complexity", vars.Complexity),
		slog.String("category", vars.Category),
		slog.Float64("energy_cost", vars.EnergyCost),
		slog.Float64("budget_pressure", vars.BudgetPressure),
	)

	// Run fuzzy inference to determine desired power_level.
	desiredPowerLevel, err := runFuzzyInference(cfg.Rules, vars, cfg.EnergySensitivity)
	if err != nil {
		slog.WarnContext(ctx, "smart-model: fuzzy inference failed, skipping", slog.Any("error", err))
		return &proto.ResolveModelOutput{}, nil
	}

	// Select the best candidate model.
	selected := selectModel(in.AvailableModels, desiredPowerLevel, vars, cfg)
	if selected == "" {
		slog.WarnContext(ctx, "smart-model: no suitable model found, skipping")
		return &proto.ResolveModelOutput{}, nil
	}

	slog.InfoContext(ctx, "smart-model: selected model",
		slog.String("selected", selected),
		slog.Float64("desired_power_level", desiredPowerLevel),
		slog.String("category", vars.Category),
	)

	if cfg.LogEnabled {
		writeDecisionLog(ctx, cfg.LogPath, in, vars, desiredPowerLevel, selected)
	}

	return &proto.ResolveModelOutput{ResolvedProxyName: selected}, nil
}

// isVirtualModel returns true when the requested model name is in the virtual models list.
// Smart-model only handles requests that target a virtual model entry point.
// Comparison is done against both the full qualified name ("org/model") and the local name ("model")
// because virtual models are stored with local names but requests arrive in qualified format.
func isVirtualModel(requestedModel string, virtualModels []*proto.VirtualModelInfo) bool {
	localName := localModelName(requestedModel)
	for _, vm := range virtualModels {
		if vm.Name == requestedModel || vm.Name == localName {
			return true
		}
	}
	return false
}

// isTriggerModel returns true when the requested model name is in the configured trigger list.
// Comparison is done against both the full qualified name ("org/model") and the local name ("model")
// because trigger models may be stored with local names by the UI.
func isTriggerModel(requestedModel string, triggerModels []string) bool {
	localName := localModelName(requestedModel)
	for _, name := range triggerModels {
		if name == requestedModel || name == localName {
			return true
		}
	}
	return false
}

// localModelName strips the org prefix from a qualified model name ("org/model" → "model").
// Returns the name unchanged if it contains no "/".
func localModelName(name string) string {
	if idx := strings.LastIndexByte(name, '/'); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

// runFuzzyInference parses the DSL, builds the engine, and returns the defuzzified power_level.
func runFuzzyInference(dslText string, vars InputVars, energySensitivity float64) (float64, error) {
	result, err := dsl.ParseRulesAndVariables(dslText)
	if err != nil {
		return 0.5, err
	}

	engine := fuzzy.NewEngine(fuzzy.Centroid(200))
	engine.Variables(result.Variables...)
	engine.Rules(result.Rules...)

	values := fuzzy.Values{
		"complexity":         vars.Complexity,
		"budget_pressure":    vars.BudgetPressure,
		"energy_sensitivity": energySensitivity,
		"energy_cost":        vars.EnergyCost,
	}

	results, err := engine.Infer(values)
	if err != nil {
		return 0.5, err
	}

	powerLevel, err := engine.Defuzzify("power_level", results)
	if err != nil {
		return 0.5, err
	}

	return math.Max(0, math.Min(1, powerLevel)), nil
}

// selectModel picks the enabled feasible model closest to the desired power_level,
// with a frugality bonus proportional to energy_sensitivity.
func selectModel(models []*proto.ModelInfo, desiredPL float64, vars InputVars, cfg Config) string {
	type candidate struct {
		proxyName  string
		powerLevel float64
		score      float64
	}

	var candidates []candidate

	for _, m := range models {
		if m.IsVirtual {
			continue
		}
		if m.SupportsEmbeddings {
			continue
		}
		override := cfg.modelOverride(m.ProxyName)
		if !override.Enabled {
			continue
		}
		// Context feasibility check.
		if m.ContextLength > 0 {
			needed := int64(vars.EstimatedInputTokens + vars.EstimatedOutputTokens)
			if needed > m.ContextLength {
				continue
			}
		}
		// Vision feasibility check.
		if vars.HasVision && !m.SupportsVision {
			continue
		}
		// Reasoning feasibility check.
		if vars.HasReasoning && !m.SupportsReasoning {
			continue
		}

		pl := cfg.powerLevel(m.ProxyName, m.ActiveParamsBillions)
		distance := math.Abs(pl - desiredPL)

		// Frugality bonus: favour lower-power models proportional to energy_sensitivity.
		// The bonus reduces the score penalty for lighter models.
		frugalityBonus := cfg.EnergySensitivity * (1 - pl) * 0.3

		// Category bonus: when the model has preferred categories configured and one
		// matches the request category, apply a bonus to steer selection toward it.
		categoryBonus := 0.0
		if override.Categories != nil && vars.Category != "" {
			for _, cat := range override.Categories {
				if cat == vars.Category {
					categoryBonus = 0.4
					break
				}
			}
		}

		score := 1 - distance + frugalityBonus + categoryBonus

		candidates = append(candidates, candidate{
			proxyName:  m.ProxyName,
			powerLevel: pl,
			score:      score,
		})
	}

	if len(candidates) == 0 {
		return ""
	}

	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.score > best.score {
			best = c
		}
	}
	return best.proxyName
}

// configSchemaJSON is a minimal JSON schema for UI-driven configuration.
const configSchemaJSON = `{
  "type": "object",
  "properties": {
    "rules":              { "type": "string",  "title": "Règles floues (DSL)" },
    "energy_sensitivity": { "type": "number",  "title": "Sensibilité énergétique", "minimum": 0, "maximum": 1 },
    "log_enabled":        { "type": "boolean", "title": "Activer les logs de décision" },
    "log_path":           { "type": "string",  "title": "Chemin du fichier de log" },
    "models": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "proxy_name":           { "type": "string" },
          "enabled":              { "type": "boolean" },
          "power_level_override": { "type": "number", "minimum": 0, "maximum": 1 }
        }
      }
    }
  }
}`
