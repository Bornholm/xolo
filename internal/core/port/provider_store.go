package port

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
)

type ProviderStore interface {
	// Providers
	CreateProvider(ctx context.Context, p model.Provider) error
	GetProviderByID(ctx context.Context, id model.ProviderID) (model.Provider, error)
	ListProviders(ctx context.Context, orgID model.OrgID) ([]model.Provider, error)
	SaveProvider(ctx context.Context, p model.Provider) error
	DeleteProvider(ctx context.Context, id model.ProviderID) error

	// LLM Models
	CreateLLMModel(ctx context.Context, m model.LLMModel) error
	GetLLMModelByID(ctx context.Context, id model.LLMModelID) (model.LLMModel, error)
	GetLLMModelByProxyName(ctx context.Context, orgID model.OrgID, proxyName string) (model.LLMModel, error)
	ListLLMModels(ctx context.Context, orgID model.OrgID) ([]model.LLMModel, error)
	ListEnabledLLMModels(ctx context.Context, orgID model.OrgID) ([]model.LLMModel, error)
	SaveLLMModel(ctx context.Context, m model.LLMModel) error
	DeleteLLMModel(ctx context.Context, id model.LLMModelID) error
}
