package eventbus

import (
	"context"
	"log/slog"
	"time"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/pkg/errors"
)

// Purger enforces the per-org ring-buffer retention by periodically evicting
// non-pinned events beyond each org's effective cap. The effective cap is the
// per-org override (or the default) clamped to the global hard cap.
type Purger struct {
	eventStore    port.EventStore
	settingsStore port.EventSettingsStore
	interval      time.Duration
	maxPerOrg     int
	defaultPerOrg int
}

func NewPurger(eventStore port.EventStore, settingsStore port.EventSettingsStore, interval time.Duration, maxPerOrg, defaultPerOrg int) *Purger {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	return &Purger{
		eventStore:    eventStore,
		settingsStore: settingsStore,
		interval:      interval,
		maxPerOrg:     maxPerOrg,
		defaultPerOrg: defaultPerOrg,
	}
}

// Run purges overflow on a ticker until ctx is cancelled.
func (p *Purger) Run(ctx context.Context) error {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			p.purgeAll(ctx)
		}
	}
}

// EffectiveCap resolves the retention cap for an org: its override or the
// default, clamped to the global maximum.
func (p *Purger) EffectiveCap(ctx context.Context, orgID model.OrgID) int {
	keepN := p.defaultPerOrg
	if override, err := p.settingsStore.GetMaxEvents(ctx, orgID); err == nil && override != nil {
		keepN = *override
	}
	if keepN <= 0 {
		keepN = p.defaultPerOrg
	}
	if p.maxPerOrg > 0 && keepN > p.maxPerOrg {
		keepN = p.maxPerOrg
	}
	return keepN
}

func (p *Purger) purgeAll(ctx context.Context) {
	orgIDs, err := p.eventStore.ListEventOrgIDs(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "event purger: could not list org ids", slog.Any("error", errors.WithStack(err)))
		return
	}
	for _, orgID := range orgIDs {
		keepN := p.EffectiveCap(ctx, orgID)
		deleted, err := p.eventStore.EvictOverflow(ctx, orgID, keepN)
		if err != nil {
			slog.ErrorContext(ctx, "event purger: eviction failed",
				slog.String("orgID", string(orgID)),
				slog.Any("error", errors.WithStack(err)))
			continue
		}
		if deleted > 0 {
			slog.DebugContext(ctx, "event purger: evicted overflow events",
				slog.String("orgID", string(orgID)),
				slog.Int64("deleted", deleted),
				slog.Int("keepN", keepN))
		}
	}
}
