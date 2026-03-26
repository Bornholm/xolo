package plugin_test

import (
	"context"
	"testing"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/bornholm/xolo/internal/plugin"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
)

// stubConfigStore is an in-memory PluginConfigStore for tests.
type stubConfigStore struct {
	data map[string]*model.PluginConfig
}

func newStubConfigStore() *stubConfigStore {
	return &stubConfigStore{data: make(map[string]*model.PluginConfig)}
}

func (s *stubConfigStore) key(orgID model.OrgID, pluginName, scope, scopeID string) string {
	return string(orgID) + "|" + pluginName + "|" + scope + "|" + scopeID
}

func (s *stubConfigStore) GetConfig(_ context.Context, orgID model.OrgID, pluginName string, scope model.PluginConfigScope, scopeID string) (*model.PluginConfig, error) {
	k := s.key(orgID, pluginName, string(scope), scopeID)
	cfg, ok := s.data[k]
	if !ok {
		return nil, port.ErrNotFound
	}
	return cfg, nil
}

func (s *stubConfigStore) SaveConfig(_ context.Context, cfg *model.PluginConfig) error {
	k := s.key(cfg.OrgID, cfg.PluginName, string(cfg.Scope), cfg.ScopeID)
	s.data[k] = cfg
	return nil
}

func (s *stubConfigStore) DeleteConfig(_ context.Context, orgID model.OrgID, pluginName string, scope model.PluginConfigScope, scopeID string) error {
	k := s.key(orgID, pluginName, string(scope), scopeID)
	delete(s.data, k)
	return nil
}

func (s *stubConfigStore) ListConfigsByPlugin(_ context.Context, pluginName string) ([]model.PluginConfig, error) {
	var configs []model.PluginConfig
	for _, cfg := range s.data {
		if cfg.PluginName == pluginName {
			configs = append(configs, *cfg)
		}
	}
	return configs, nil
}

func TestGetConfig_NotFound_ReturnsEmpty(t *testing.T) {
	svc := plugin.NewXoloHostService(newStubConfigStore())
	resp, err := svc.GetConfig(context.Background(), &proto.GetConfigRequest{
		OrgId: "org-1", PluginName: "my-plugin",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ConfigJson != "{}" {
		t.Errorf("expected '{}', got %q", resp.ConfigJson)
	}
}

func TestSaveAndGetConfig_RoundTrip(t *testing.T) {
	store := newStubConfigStore()
	svc := plugin.NewXoloHostService(store)

	_, err := svc.SaveConfig(context.Background(), &proto.SaveConfigRequest{
		OrgId:      "org-1",
		PluginName: "my-plugin",
		ConfigJson: `{"key":"value"}`,
	})
	if err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	resp, err := svc.GetConfig(context.Background(), &proto.GetConfigRequest{
		OrgId: "org-1", PluginName: "my-plugin",
	})
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if resp.ConfigJson != `{"key":"value"}` {
		t.Errorf("expected '{\"key\":\"value\"}', got %q", resp.ConfigJson)
	}
}
