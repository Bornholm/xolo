package main

import (
	"context"
	"encoding/json"
	"math"

	fuzzy "github.com/bornholm/go-fuzzy"
	"github.com/bornholm/go-fuzzy/dsl"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
)

const PluginName = "fuzzy-evaluator"
const PluginVersion = "0.1.0"

type Plugin struct {
	proto.UnimplementedXoloPluginServer
}

func (p *Plugin) Describe(_ context.Context, _ *proto.DescribeRequest) (*proto.PluginDescriptor, error) {
	return &proto.PluginDescriptor{
		Name:    PluginName,
		Version: PluginVersion,
		Description: "Moteur d'inférence floue générique. Lit des variables numériques depuis les ports d'entrée " +
			"et produit un ou plusieurs résultats numériques sur les ports de sortie.",
		Capabilities: []proto.PluginDescriptor_Capability{proto.PluginDescriptor_PRE_REQUEST},
		// Default ports shown before configuration.
		InputPorts: []*proto.PortDescriptor{
			{Name: "complexity", PortType: "number"},
			{Name: "budget_pressure", PortType: "number"},
			{Name: "energy_cost", PortType: "number"},
		},
		OutputPorts: []*proto.PortDescriptor{
			{Name: "power_level", PortType: "number"},
		},
	}, nil
}

func (p *Plugin) PreRequest(_ context.Context, in *proto.PreRequestInput) (*proto.PreRequestOutput, error) {
	cfg := parseConfig(in.GetCtx().GetConfigJson())

	var inputs map[string]interface{}
	if in.InputsJson != "" && in.InputsJson != "{}" {
		json.Unmarshal([]byte(in.InputsJson), &inputs) //nolint:errcheck
	}
	if inputs == nil {
		inputs = make(map[string]interface{})
	}

	outputValues, err := runFuzzyInference(cfg, inputs)
	if err != nil {
		// Fallback: emit 0.5 for every declared output.
		outputValues = make(map[string]float64)
		for _, out := range cfg.Outputs {
			outputValues[out.Name] = 0.5
		}
	}

	out := make(map[string]interface{}, len(outputValues))
	for k, v := range outputValues {
		out[k] = v
	}
	b, _ := json.Marshal(out)

	return &proto.PreRequestOutput{
		Allowed:     true,
		OutputsJson: string(b),
	}, nil
}

// ─── Config ───────────────────────────────────────────────────────────────────

// PortDef names a single input or output port (always type "number" for fuzzy).
type PortDef struct {
	Name string `json:"name"`
}

type Config struct {
	RulesDSL string    `json:"rules_dsl"`
	Inputs   []PortDef `json:"inputs"`
	Outputs  []PortDef `json:"outputs"`
}

func defaultConfig() Config {
	return Config{
		RulesDSL: DefaultRulesDSL,
		Inputs: []PortDef{
			{Name: "complexity"},
			{Name: "budget_pressure"},
			{Name: "energy_cost"},
		},
		Outputs: []PortDef{
			{Name: "power_level"},
		},
	}
}

func parseConfig(raw string) Config {
	cfg := defaultConfig()
	if raw != "" && raw != "{}" {
		json.Unmarshal([]byte(raw), &cfg) //nolint:errcheck
		if cfg.RulesDSL == "" {
			cfg.RulesDSL = DefaultRulesDSL
		}
		// Backward-compat: old configs had "output_name" string.
		var legacy struct {
			OutputName string `json:"output_name"`
		}
		if json.Unmarshal([]byte(raw), &legacy) == nil && legacy.OutputName != "" && len(cfg.Outputs) == 0 {
			cfg.Outputs = []PortDef{{Name: legacy.OutputName}}
		}
	}
	return cfg
}

// ─── Fuzzy inference ─────────────────────────────────────────────────────────

func runFuzzyInference(cfg Config, inputs map[string]interface{}) (map[string]float64, error) {
	result, err := dsl.ParseRulesAndVariables(cfg.RulesDSL)
	if err != nil {
		return nil, err
	}

	engine := fuzzy.NewEngine(fuzzy.Centroid(200))
	engine.Variables(result.Variables...)
	engine.Rules(result.Rules...)

	values := fuzzy.Values{}
	for _, inp := range cfg.Inputs {
		if v, ok := inputs[inp.Name]; ok {
			values[inp.Name] = toFloat64(v)
		}
	}

	results, err := engine.Infer(values)
	if err != nil {
		return nil, err
	}

	outputs := make(map[string]float64, len(cfg.Outputs))
	for _, out := range cfg.Outputs {
		val, err := engine.Defuzzify(out.Name, results)
		if err == nil {
			outputs[out.Name] = math.Max(0, math.Min(1, val))
		}
	}
	return outputs, nil
}

func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	}
	return 0
}
