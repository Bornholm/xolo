package setup

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/bornholm/xolo/internal/config"
	"github.com/bornholm/xolo/internal/core/model"
)

func migrateVirtualModelConfigs(ctx context.Context, conf *config.Config) error {
	store, err := getGormStoreFromConfig(ctx, conf)
	if err != nil {
		return err
	}

	configs, err := store.ListConfigsByPlugin(ctx, "model-auto-select")
	if err != nil {
		slog.WarnContext(ctx, "migration: could not list plugin configs", slog.Any("error", err))
		return nil // non-fatal
	}

	for _, cfg := range configs {
		if cfg.ConfigJSON == "" {
			continue
		}

		var configData struct {
			VirtualModel string `json:"virtual_model"`
		}
		if err := json.Unmarshal([]byte(cfg.ConfigJSON), &configData); err != nil {
			continue
		}

		if configData.VirtualModel == "" {
			continue
		}

		// Check if VirtualModel already exists
		existing, err := store.GetVirtualModelByName(ctx, cfg.OrgID, configData.VirtualModel)
		if err == nil && existing != nil {
			slog.DebugContext(ctx, "migration: virtual model already exists",
				slog.String("org", string(cfg.OrgID)),
				slog.String("name", configData.VirtualModel))
			continue
		}

		vm := model.NewVirtualModel(cfg.OrgID, configData.VirtualModel, "Migrated from plugin config")
		if err := store.CreateVirtualModel(ctx, vm); err != nil {
			slog.WarnContext(ctx, "migration: could not create virtual model",
				slog.String("org", string(cfg.OrgID)),
				slog.String("name", configData.VirtualModel),
				slog.Any("error", err))
			continue
		}

		slog.InfoContext(ctx, "migration: created virtual model from plugin config",
			slog.String("org", string(cfg.OrgID)),
			slog.String("name", configData.VirtualModel))
	}

	return nil
}
