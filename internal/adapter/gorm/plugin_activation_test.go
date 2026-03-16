package gorm_test

import (
	"context"
	"testing"
	"time"

	gorma "github.com/bornholm/xolo/internal/adapter/gorm"
	"github.com/bornholm/xolo/internal/core/model"
	gormlite "github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"

	_ "github.com/ncruces/go-sqlite3/embed"
)

func newPluginTestStore(t *testing.T) *gorma.Store {
	t.Helper()
	db, err := gorm.Open(gormlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	store := gorma.NewStore(db)
	// Trigger AutoMigrate by making a call; fail fast if migration fails.
	ctx := context.Background()
	if _, err := store.ListActivations(ctx, model.OrgID("init")); err != nil {
		t.Fatalf("AutoMigrate via ListActivations: %v", err)
	}
	return store
}

func TestPluginActivation_SaveAndGet(t *testing.T) {
	store := newPluginTestStore(t)
	ctx := context.Background()

	act := &model.PluginActivation{
		OrgID:      model.OrgID("org-1"),
		PluginName: "time-window",
		Enabled:    true,
		Required:   false,
		Order:      10,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := store.SaveActivation(ctx, act); err != nil {
		t.Fatalf("SaveActivation: %v", err)
	}

	got, err := store.GetActivation(ctx, model.OrgID("org-1"), "time-window")
	if err != nil {
		t.Fatalf("GetActivation: %v", err)
	}
	if got.PluginName != "time-window" {
		t.Errorf("PluginName: got %q, want %q", got.PluginName, "time-window")
	}
	if !got.Enabled {
		t.Error("Enabled: expected true")
	}
}

func TestPluginActivation_ListSortedByOrder(t *testing.T) {
	store := newPluginTestStore(t)
	ctx := context.Background()

	orgID := model.OrgID("org-2")
	for _, a := range []*model.PluginActivation{
		{OrgID: orgID, PluginName: "c", Enabled: true, Order: 30},
		{OrgID: orgID, PluginName: "a", Enabled: true, Order: 10},
		{OrgID: orgID, PluginName: "b", Enabled: true, Order: 20},
	} {
		if err := store.SaveActivation(ctx, a); err != nil {
			t.Fatal(err)
		}
	}

	list, err := store.ListActivations(ctx, orgID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 activations, got %d", len(list))
	}
	if list[0].PluginName != "a" || list[1].PluginName != "b" || list[2].PluginName != "c" {
		t.Errorf("unexpected order: %v", []string{list[0].PluginName, list[1].PluginName, list[2].PluginName})
	}
}

func TestPluginActivation_DeleteActivation(t *testing.T) {
	store := newPluginTestStore(t)
	ctx := context.Background()

	orgID := model.OrgID("org-3")
	act := &model.PluginActivation{OrgID: orgID, PluginName: "myplugin", Enabled: true}
	if err := store.SaveActivation(ctx, act); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteActivation(ctx, orgID, "myplugin"); err != nil {
		t.Fatal(err)
	}
	list, err := store.ListActivations(ctx, orgID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list after delete, got %d", len(list))
	}
}

func TestPluginActivation_UpsertPreservesOrder(t *testing.T) {
	store := newPluginTestStore(t)
	ctx := context.Background()

	orgID := model.OrgID("org-4")
	act := &model.PluginActivation{OrgID: orgID, PluginName: "plug", Enabled: true, Order: 5}
	if err := store.SaveActivation(ctx, act); err != nil {
		t.Fatal(err)
	}
	// Update with new order
	act.Order = 99
	act.Enabled = false
	if err := store.SaveActivation(ctx, act); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetActivation(ctx, orgID, "plug")
	if err != nil {
		t.Fatal(err)
	}
	if got.Order != 99 {
		t.Errorf("Order: got %d, want 99", got.Order)
	}
	if got.Enabled {
		t.Error("Enabled: expected false after upsert")
	}
}
