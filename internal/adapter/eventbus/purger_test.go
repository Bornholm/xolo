package eventbus_test

import (
	"context"
	"testing"
	"time"

	"github.com/bornholm/xolo/internal/adapter/eventbus"
	xologorm "github.com/bornholm/xolo/internal/adapter/gorm"
	"github.com/bornholm/xolo/internal/core/model"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/gormlite"
	gormpkg "gorm.io/gorm"
)

func newStore(t *testing.T) *xologorm.Store {
	t.Helper()
	db, err := gormpkg.Open(gormlite.Open(":memory:"), &gormpkg.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return xologorm.NewStore(db)
}

func TestPurger_EffectiveCap(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()
	org := model.OrgID("org1")

	// global cap 1000, default 100.
	p := eventbus.NewPurger(store, store, time.Minute, 1000, 100)

	// No override → default.
	if got := p.EffectiveCap(ctx, org); got != 100 {
		t.Fatalf("expected default 100, got %d", got)
	}

	// Override within cap.
	override := 500
	if err := store.SetMaxEvents(ctx, org, &override); err != nil {
		t.Fatalf("SetMaxEvents: %v", err)
	}
	if got := p.EffectiveCap(ctx, org); got != 500 {
		t.Fatalf("expected override 500, got %d", got)
	}

	// Override above global cap → clamped.
	big := 5000
	if err := store.SetMaxEvents(ctx, org, &big); err != nil {
		t.Fatalf("SetMaxEvents: %v", err)
	}
	if got := p.EffectiveCap(ctx, org); got != 1000 {
		t.Fatalf("expected clamp to 1000, got %d", got)
	}

	// Clearing the override falls back to the default.
	if err := store.SetMaxEvents(ctx, org, nil); err != nil {
		t.Fatalf("SetMaxEvents(nil): %v", err)
	}
	if got := p.EffectiveCap(ctx, org); got != 100 {
		t.Fatalf("expected default 100 after clear, got %d", got)
	}
}
