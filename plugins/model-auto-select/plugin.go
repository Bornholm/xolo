package main

import (
	"context"
	"log/slog"

	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
)

type Plugin struct {
	proto.UnimplementedXoloPluginServer
}

func (p *Plugin) Describe(_ context.Context, _ *proto.DescribeRequest) (*proto.PluginDescriptor, error) {
	return &proto.PluginDescriptor{
		Name:         "model-auto-select",
		Version:      "0.0.1",
		Description:  "Sélectionne automatiquement le modèle LLM le plus adapté via logique floue.",
		Capabilities: []proto.PluginDescriptor_Capability{proto.PluginDescriptor_RESOLVE_MODEL},
		ConfigSchema: configSchemaJSON,
	}, nil
}

func (p *Plugin) ResolveModel(_ context.Context, in *proto.ResolveModelInput) (*proto.ResolveModelOutput, error) {
	cfg, err := parseConfig(in.GetCtx().GetConfigJson())
	if err != nil {
		slog.Warn("fuzzy-model-selector: failed to parse config", slog.Any("error", err))
		return &proto.ResolveModelOutput{ResolvedProxyName: ""}, nil
	}

	if in.RequestedModel != cfg.VirtualModel {
		return &proto.ResolveModelOutput{ResolvedProxyName: ""}, nil
	}

	signals, err := ExtractSignals(in.GetMessagesJson(), cfg.Signals)
	if err != nil {
		slog.Warn("fuzzy-model-selector: failed to extract signals", slog.Any("error", err))
		signals = make(map[string]float64)
	}

	estimatedTokens := int64(len(in.GetMessagesJson()) / 4)

	proxyName, err := Score(cfg, signals, in.GetAvailableModels(), estimatedTokens)
	if err != nil {
		slog.Warn("fuzzy-model-selector: scoring failed", slog.Any("error", err))
		return &proto.ResolveModelOutput{ResolvedProxyName: ""}, nil
	}

	slog.Debug("fuzzy-model-selector: resolved model", slog.String("proxy_name", proxyName))
	return &proto.ResolveModelOutput{ResolvedProxyName: proxyName}, nil
}
