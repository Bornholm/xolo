package main

import (
	"context"
	"testing"

	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
)

const fullConfigJSON = `{
	"virtual_model": "auto",
	"budget_preference": 5,
	"signals": [
		{"name": "code_ratio", "type": "keyword_ratio", "patterns": ["` + "```" + `", "def "]}
	],
	"rules": "DEFINE code_ratio ( TERM high LINEAR (0.5, 1.0) ); DEFINE tag_coding ( TERM high LINEAR (0.5, 1.0) ); IF code_ratio IS high THEN tag_coding IS high;",
	"models": [{"proxy_name": "coding-model", "tags": ["coding"]}]
}`

func makeCtx(configJSON string) *proto.RequestContext {
	return &proto.RequestContext{ConfigJson: configJSON}
}

func TestResolveModel_SkipsWrongVirtualName(t *testing.T) {
	p := &Plugin{}
	in := &proto.ResolveModelInput{
		Ctx:            makeCtx(fullConfigJSON),
		RequestedModel: "gpt-4o", // ≠ "auto"
		AvailableModels: []*proto.ModelInfo{
			{ProxyName: "coding-model", TokenLimit: 128000, PromptCostPer_1KTokens: 1.0},
		},
		MessagesJson: `[{"role":"user","content":"` + "```python\\ndef foo(): pass\\n```" + `"}]`,
	}

	out, err := p.ResolveModel(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ResolvedProxyName != "" {
		t.Errorf("expected empty string (skipped), got %q", out.ResolvedProxyName)
	}
}

func TestResolveModel_SelectsBestModel(t *testing.T) {
	p := &Plugin{}
	in := &proto.ResolveModelInput{
		Ctx:            makeCtx(fullConfigJSON),
		RequestedModel: "auto",
		AvailableModels: []*proto.ModelInfo{
			{ProxyName: "coding-model", TokenLimit: 128000, PromptCostPer_1KTokens: 1.0},
		},
		MessagesJson: `[{"role":"user","content":"` + "```python\\ndef foo(): pass\\n```" + `"}]`,
	}

	out, err := p.ResolveModel(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ResolvedProxyName != "coding-model" {
		t.Errorf("expected coding-model, got %q", out.ResolvedProxyName)
	}
}

func TestResolveModel_EmptyAvailableModels(t *testing.T) {
	p := &Plugin{}
	in := &proto.ResolveModelInput{
		Ctx:             makeCtx(fullConfigJSON),
		RequestedModel:  "auto",
		AvailableModels: []*proto.ModelInfo{}, // empty
		MessagesJson:    `[{"role":"user","content":"hello"}]`,
	}

	out, err := p.ResolveModel(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ResolvedProxyName != "" {
		t.Errorf("expected empty string (no available models), got %q", out.ResolvedProxyName)
	}
}
