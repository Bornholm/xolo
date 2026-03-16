package pluginsdk

import (
	"context"
	"net/http"

	"github.com/hashicorp/go-plugin"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
)

// noopInitWrapper wraps any XoloPluginServer and adds a no-op Initialize
// that returns http_ui_port: 0. This ensures plugins using Serve() never
// return UNIMPLEMENTED for Initialize.
type noopInitWrapper struct {
	proto.XoloPluginServer
}

func (w *noopInitWrapper) Initialize(_ context.Context, _ *proto.InitializeRequest) (*proto.InitializeResponse, error) {
	return &proto.InitializeResponse{HttpUiPort: 0}, nil
}

// WrapWithNoopInit wraps impl with a no-op Initialize. Exported for testing.
func WrapWithNoopInit(impl proto.XoloPluginServer) proto.XoloPluginServer {
	return &noopInitWrapper{impl}
}

// Serve starts the plugin gRPC server. Call this from your plugin binary's main().
// The impl is automatically wrapped to handle Initialize with a no-op (returns port 0).
func Serve(impl proto.XoloPluginServer) {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			PluginName: &XoloPluginGRPC{Impl: WrapWithNoopInit(impl)},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}

// ServeWithUI starts the plugin gRPC server with an embedded HTTP UI.
// pluginName must match the name returned by Describe().
// uiHandler serves the plugin's configuration UI.
func ServeWithUI(impl proto.XoloPluginServer, pluginName string, uiHandler http.Handler) {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			PluginName: &XoloPluginGRPC{Impl: newUIWrapper(impl, pluginName, uiHandler)},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
