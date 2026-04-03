package plugin

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// XoloHostService implements proto.XoloHostServiceServer.
// It gives plugin HTTP servers access to Xolo's plugin config store and model list.
type XoloHostService struct {
	proto.UnimplementedXoloHostServiceServer
	configStore        port.PluginConfigStore
	providerStore      port.ProviderStore
	virtualModelStore  port.VirtualModelStore
}

// NewXoloHostService creates an XoloHostService backed by the given stores.
func NewXoloHostService(configStore port.PluginConfigStore, providerStore port.ProviderStore, virtualModelStore port.VirtualModelStore) *XoloHostService {
	return &XoloHostService{configStore: configStore, providerStore: providerStore, virtualModelStore: virtualModelStore}
}

func (s *XoloHostService) GetConfig(ctx context.Context, req *proto.GetConfigRequest) (*proto.GetConfigResponse, error) {
	orgID := model.OrgID(req.OrgId)
	cfg, err := s.configStore.GetConfig(ctx, orgID, req.PluginName,
		model.PluginConfigScopeOrg, req.OrgId)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return &proto.GetConfigResponse{ConfigJson: "{}"}, nil
		}
		return nil, status.Errorf(codes.Internal, "get config: %v", err)
	}
	json := cfg.ConfigJSON
	if json == "" {
		json = "{}"
	}
	return &proto.GetConfigResponse{ConfigJson: json}, nil
}

func (s *XoloHostService) ListModels(ctx context.Context, req *proto.ListModelsForOrgRequest) (*proto.ListModelsForOrgResponse, error) {
	if s.providerStore == nil {
		return &proto.ListModelsForOrgResponse{}, nil
	}
	orgID := model.OrgID(req.OrgId)
	models, err := s.providerStore.ListEnabledLLMModels(ctx, orgID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list models: %v", err)
	}
	protoModels := make([]*proto.ModelInfo, 0, len(models))
	for _, m := range models {
		caps := m.Capabilities()
		protoModels = append(protoModels, &proto.ModelInfo{
			ProxyName:                  m.ProxyName(),
			RealModel:                  m.RealModel(),
			ProviderId:                 string(m.ProviderID()),
			PromptCostPer_1KTokens:     float64(m.PromptCostPer1KTokens()),
			CompletionCostPer_1KTokens: float64(m.CompletionCostPer1KTokens()),
			TokenLimit:                 m.ContextWindow(),
			IsVirtual:                  m.IsVirtual(),
			ContextLength:              m.ContextWindow(),
			SupportsVision:             caps.Vision,
			SupportsReasoning:          caps.Reasoning,
			SupportsEmbeddings:         caps.Embeddings,
			ActiveParamsBillions:       float32(m.ActiveParams()) / 1e9,
		})
	}
	// Append virtual models from the virtual model store.
	if s.virtualModelStore != nil {
		virtualModels, err := s.virtualModelStore.ListVirtualModels(ctx, orgID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "list virtual models: %v", err)
		}
		for _, vm := range virtualModels {
			protoModels = append(protoModels, &proto.ModelInfo{
				ProxyName: vm.Name(),
				IsVirtual: true,
			})
		}
	}

	return &proto.ListModelsForOrgResponse{Models: protoModels}, nil
}

func (s *XoloHostService) SaveConfig(ctx context.Context, req *proto.SaveConfigRequest) (*proto.SaveConfigResponse, error) {
	orgID := model.OrgID(req.OrgId)
	cfg := &model.PluginConfig{
		OrgID:      orgID,
		PluginName: req.PluginName,
		Scope:      model.PluginConfigScopeOrg,
		ScopeID:    req.OrgId,
		ConfigJSON: req.ConfigJson,
	}
	if err := s.configStore.SaveConfig(ctx, cfg); err != nil {
		return nil, status.Errorf(codes.Internal, "save config: %v", err)
	}
	return &proto.SaveConfigResponse{}, nil
}
