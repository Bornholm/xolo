package main

import (
	"context"
	"log/slog"
	"time"

	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
)

// Plugin implémente proto.XoloPluginServer.
type Plugin struct {
	proto.UnimplementedXoloPluginServer
}

func (p *Plugin) Describe(_ context.Context, _ *proto.DescribeRequest) (*proto.PluginDescriptor, error) {
	return &proto.PluginDescriptor{
		Name:            "time-restriction",
		Version:         "0.0.1",
		Description:     "Restreint l'accès au proxy LLM selon des plages horaires hebdomadaires configurables par organisation.",
		Capabilities:    []proto.PluginDescriptor_Capability{proto.PluginDescriptor_PRE_REQUEST},
		ConfigSchema:    configSchemaJSON,
		DefaultRequired: true,
	}, nil
}

func (p *Plugin) PreRequest(_ context.Context, in *proto.PreRequestInput) (*proto.PreRequestOutput, error) {
	cfg, err := parseConfig(in.GetCtx().GetConfigJson())
	if err != nil {
		slog.Warn("time-restriction: failed to parse config, denying request", slog.Any("error", err))
		return &proto.PreRequestOutput{
			Allowed:         false,
			RejectionReason: "Accès refusé : hors des plages horaires autorisées.",
		}, nil
	}
	allowed, err := isAllowed(time.Now(), cfg)
	if err != nil {
		slog.Warn("time-restriction: configuration error, denying request", slog.Any("error", err))
		return &proto.PreRequestOutput{
			Allowed:         false,
			RejectionReason: "Accès refusé : erreur de configuration du plugin.",
		}, nil
	}
	if !allowed {
		slog.Debug("time-restriction: request denied (outside allowed time slots)")
		return &proto.PreRequestOutput{
			Allowed:         false,
			RejectionReason: "Accès refusé : hors des plages horaires autorisées.",
		}, nil
	}
	return &proto.PreRequestOutput{Allowed: true}, nil
}
