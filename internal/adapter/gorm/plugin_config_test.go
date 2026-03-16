package gorm_test

import (
	"context"
	"testing"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/pkg/errors"
)

func TestPluginConfig_SaveAndGet_OrgScope(t *testing.T) {
	store := newPluginTestStore(t)
	ctx := context.Background()

	orgID := model.OrgID("org-10")
	cfg := &model.PluginConfig{
		OrgID:      orgID,
		PluginName: "time-window",
		Scope:      model.PluginConfigScopeOrg,
		ScopeID:    string(orgID),
		ConfigJSON: `{"tz":"Europe/Paris"}`,
	}

	if err := store.SaveConfig(ctx, cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	got, err := store.GetConfig(ctx, orgID, "time-window", model.PluginConfigScopeOrg, string(orgID))
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if got.ConfigJSON != `{"tz":"Europe/Paris"}` {
		t.Errorf("ConfigJSON: got %q", got.ConfigJSON)
	}
}

func TestPluginConfig_SaveAndGet_UserScope(t *testing.T) {
	store := newPluginTestStore(t)
	ctx := context.Background()

	orgID := model.OrgID("org-11")
	userID := "user-abc"
	cfg := &model.PluginConfig{
		OrgID:      orgID,
		PluginName: "time-window",
		Scope:      model.PluginConfigScopeUser,
		ScopeID:    userID,
		ConfigJSON: `{"tz":"America/New_York"}`,
	}
	if err := store.SaveConfig(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetConfig(ctx, orgID, "time-window", model.PluginConfigScopeUser, userID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ConfigJSON != `{"tz":"America/New_York"}` {
		t.Errorf("ConfigJSON: got %q", got.ConfigJSON)
	}
}

func TestPluginConfig_NotFound(t *testing.T) {
	store := newPluginTestStore(t)
	ctx := context.Background()

	_, err := store.GetConfig(ctx, model.OrgID("no-org"), "no-plugin", model.PluginConfigScopeOrg, "no-org")
	if !errors.Is(err, port.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPluginConfig_Upsert(t *testing.T) {
	store := newPluginTestStore(t)
	ctx := context.Background()

	orgID := model.OrgID("org-12")
	cfg := &model.PluginConfig{OrgID: orgID, PluginName: "p", Scope: model.PluginConfigScopeOrg, ScopeID: string(orgID), ConfigJSON: `{"v":1}`}
	if err := store.SaveConfig(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	cfg.ConfigJSON = `{"v":2}`
	if err := store.SaveConfig(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetConfig(ctx, orgID, "p", model.PluginConfigScopeOrg, string(orgID))
	if err != nil {
		t.Fatal(err)
	}
	if got.ConfigJSON != `{"v":2}` {
		t.Errorf("expected upsert: got %q", got.ConfigJSON)
	}
}

func TestPluginConfig_Delete(t *testing.T) {
	store := newPluginTestStore(t)
	ctx := context.Background()

	orgID := model.OrgID("org-13")
	cfg := &model.PluginConfig{OrgID: orgID, PluginName: "p", Scope: model.PluginConfigScopeOrg, ScopeID: string(orgID), ConfigJSON: `{}`}
	if err := store.SaveConfig(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteConfig(ctx, orgID, "p", model.PluginConfigScopeOrg, string(orgID)); err != nil {
		t.Fatal(err)
	}
	_, err := store.GetConfig(ctx, orgID, "p", model.PluginConfigScopeOrg, string(orgID))
	if !errors.Is(err, port.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}
