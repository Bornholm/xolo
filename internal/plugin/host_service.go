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
// It gives plugin HTTP servers access to Xolo's plugin config store.
type XoloHostService struct {
	proto.UnimplementedXoloHostServiceServer
	configStore port.PluginConfigStore
}

// NewXoloHostService creates an XoloHostService backed by the given config store.
func NewXoloHostService(configStore port.PluginConfigStore) *XoloHostService {
	return &XoloHostService{configStore: configStore}
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
