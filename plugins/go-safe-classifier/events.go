package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bornholm/xolo/pkg/pluginsdk"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
)

// emitSafetyDetectedEvent envoie un événement "safety.detected" non bloquant
// au host Xolo. L'appel gRPC est fait dans une goroutine avec un timeout
// court pour ne pas ralentir le pipeline.
func (p *Plugin) emitSafetyDetectedEvent(in *proto.PreRequestInput, label string, score float32, cfg Config) {
	hc := p.getHostClient()
	if hc == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		evt := pluginsdk.Event{
			PluginName: "go-safe-classifier",
			OrgID:      in.GetCtx().GetOrgId(),
			UserID:     in.GetCtx().GetUserId(),
			Type:       "safety.detected",
			Severity:   "warning",
			Message:    fmt.Sprintf("Requête bloquée : classifiée %q (score=%.2f)", label, score),
			Attributes: map[string]string{
				"label":     label,
				"score":     fmt.Sprintf("%.4f", score),
				"threshold": fmt.Sprintf("%.4f", cfg.Threshold),
				"language":  cfg.Language,
				"model":     cfg.Model,
			},
		}
		if err := hc.EmitEvent(ctx, evt); err != nil {
			slog.Warn("go-safe-classifier: could not emit event", slog.Any("error", err))
		}
	}()
}