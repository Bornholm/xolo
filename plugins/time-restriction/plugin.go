package main

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/bornholm/xolo/pkg/pluginsdk"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
)

// Plugin implémente proto.XoloPluginServer.
type Plugin struct {
	proto.UnimplementedXoloPluginServer

	hostMu     sync.Mutex
	hostClient pluginsdk.HostClient
}

// SetHostClient implémente pluginsdk.HostClientSetter : le runtime l'appelle une
// fois la connexion au XoloHostService établie (dans Initialize).
func (p *Plugin) SetHostClient(c pluginsdk.HostClient) {
	p.hostMu.Lock()
	defer p.hostMu.Unlock()
	p.hostClient = c
}

func (p *Plugin) getHostClient() pluginsdk.HostClient {
	p.hostMu.Lock()
	defer p.hostMu.Unlock()
	return p.hostClient
}

// emitBlocked émet, de façon non bloquante, un événement lorsqu'une requête est
// bloquée hors des plages horaires autorisées.
func (p *Plugin) emitBlocked(reqCtx *proto.RequestContext, reason string) {
	hc := p.getHostClient()
	if hc == nil {
		return
	}
	evt := pluginsdk.Event{
		PluginName: "time-restriction",
		Type:       "request.blocked",
		Severity:   "warning",
		Message:    "Requête bloquée : " + reason,
		Attributes: map[string]string{"reason": reason},
	}
	if reqCtx != nil {
		evt.OrgID = reqCtx.GetOrgId()
		evt.UserID = reqCtx.GetUserId()
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := hc.EmitEvent(ctx, evt); err != nil {
			slog.Warn("time-restriction: could not emit event", slog.Any("error", err))
		}
	}()
}

func (p *Plugin) Describe(_ context.Context, _ *proto.DescribeRequest) (*proto.PluginDescriptor, error) {
	return &proto.PluginDescriptor{
		Name:            "time-restriction",
		Version:         "0.0.1",
		Description:     "Restreint l'accès au proxy LLM selon des plages horaires hebdomadaires configurables par organisation.",
		Capabilities:    []proto.PluginDescriptor_Capability{proto.PluginDescriptor_PRE_REQUEST},
		ConfigSchema:    configSchemaJSON,
		DefaultRequired: true,
		InputPorts: []*proto.PortDescriptor{
			{Name: "request", PortType: "request", Required: true},
		},
		OutputPorts: []*proto.PortDescriptor{
			{Name: "request", PortType: "request", Required: true},
		},
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
		p.emitBlocked(in.GetCtx(), "hors des plages horaires autorisées")
		return &proto.PreRequestOutput{
			Allowed:         false,
			RejectionReason: "Accès refusé : hors des plages horaires autorisées.",
		}, nil
	}
	return &proto.PreRequestOutput{Allowed: true}, nil
}
