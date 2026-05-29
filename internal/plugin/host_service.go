package plugin

import (
	"context"
	"log/slog"
	"sync"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// XoloHostService implements proto.XoloHostServiceServer.
// Config persistence uses an in-memory store keyed by (orgID, pluginName).
// This config is used by the plugin HTTP UI for display/edit; the authoritative
// per-node config lives in the pipeline graph (PluginNodeData.Config).
type XoloHostService struct {
	proto.UnimplementedXoloHostServiceServer
	providerStore     port.ProviderStore
	virtualModelStore port.VirtualModelStore
	mu                sync.RWMutex
	configs           map[string]string // key: orgID+":"+pluginName → JSON
}

// NewXoloHostService creates an XoloHostService.
func NewXoloHostService(providerStore port.ProviderStore, virtualModelStore port.VirtualModelStore) *XoloHostService {
	return &XoloHostService{
		providerStore:     providerStore,
		virtualModelStore: virtualModelStore,
		configs:           make(map[string]string),
	}
}

func (s *XoloHostService) configKey(orgID, pluginName string) string {
	return orgID + ":" + pluginName
}

// GetConfig returns the in-memory config for the plugin (defaults to "{}").
func (s *XoloHostService) GetConfig(_ context.Context, req *proto.GetConfigRequest) (*proto.GetConfigResponse, error) {
	s.mu.RLock()
	cfg, ok := s.configs[s.configKey(req.OrgId, req.PluginName)]
	s.mu.RUnlock()
	if !ok || cfg == "" {
		cfg = "{}"
	}
	return &proto.GetConfigResponse{ConfigJson: cfg}, nil
}

// SaveConfig persists the plugin config in memory so the UI can read it back.
func (s *XoloHostService) SaveConfig(_ context.Context, req *proto.SaveConfigRequest) (*proto.SaveConfigResponse, error) {
	s.mu.Lock()
	s.configs[s.configKey(req.OrgId, req.PluginName)] = req.ConfigJson
	s.mu.Unlock()
	slog.Debug("host service: config saved", slog.String("plugin", req.PluginName))
	return &proto.SaveConfigResponse{}, nil
}

// SeedConfig pre-populates the in-memory config (called before opening the plugin UI).
func (s *XoloHostService) SeedConfig(orgID, pluginName, configJSON string) {
	s.mu.Lock()
	s.configs[s.configKey(orgID, pluginName)] = configJSON
	s.mu.Unlock()
}

// ReadConfig retrieves the in-memory config (called after the plugin UI saves).
func (s *XoloHostService) ReadConfig(orgID, pluginName string) string {
	s.mu.RLock()
	cfg := s.configs[s.configKey(orgID, pluginName)]
	s.mu.RUnlock()
	if cfg == "" {
		return "{}"
	}
	return cfg
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
