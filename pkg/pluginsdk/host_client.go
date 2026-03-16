package pluginsdk

import (
	"context"
	"fmt"

	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
	"google.golang.org/grpc"
)

// HostClient is the interface plugins use to read and write their configuration
// via Xolo's XoloHostService gRPC.
type HostClient interface {
	GetConfig(ctx context.Context, orgID, pluginName string) (string, error)
	SaveConfig(ctx context.Context, orgID, pluginName, configJSON string) error
}

// grpcHostClient implements HostClient over gRPC.
type grpcHostClient struct {
	client proto.XoloHostServiceClient
}

func newGRPCHostClient(conn *grpc.ClientConn) *grpcHostClient {
	return &grpcHostClient{client: proto.NewXoloHostServiceClient(conn)}
}

func (c *grpcHostClient) GetConfig(ctx context.Context, orgID, pluginName string) (string, error) {
	resp, err := c.client.GetConfig(ctx, &proto.GetConfigRequest{
		OrgId:      orgID,
		PluginName: pluginName,
	})
	if err != nil {
		return "", fmt.Errorf("GetConfig gRPC: %w", err)
	}
	if resp.ConfigJson == "" {
		return "{}", nil
	}
	return resp.ConfigJson, nil
}

func (c *grpcHostClient) SaveConfig(ctx context.Context, orgID, pluginName, configJSON string) error {
	_, err := c.client.SaveConfig(ctx, &proto.SaveConfigRequest{
		OrgId:      orgID,
		PluginName: pluginName,
		ConfigJson: configJSON,
	})
	if err != nil {
		return fmt.Errorf("SaveConfig gRPC: %w", err)
	}
	return nil
}
