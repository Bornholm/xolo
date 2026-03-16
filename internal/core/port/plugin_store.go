package port

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
)

type PluginActivationStore interface {
	GetActivation(ctx context.Context, orgID model.OrgID, pluginName string) (*model.PluginActivation, error)
	ListActivations(ctx context.Context, orgID model.OrgID) ([]*model.PluginActivation, error)
	SaveActivation(ctx context.Context, activation *model.PluginActivation) error
	DeleteActivation(ctx context.Context, orgID model.OrgID, pluginName string) error
}

type PluginConfigStore interface {
	GetConfig(ctx context.Context, orgID model.OrgID, pluginName string, scope model.PluginConfigScope, scopeID string) (*model.PluginConfig, error)
	SaveConfig(ctx context.Context, cfg *model.PluginConfig) error
	DeleteConfig(ctx context.Context, orgID model.OrgID, pluginName string, scope model.PluginConfigScope, scopeID string) error
}
