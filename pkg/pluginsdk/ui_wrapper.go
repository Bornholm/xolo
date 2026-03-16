package pluginsdk

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/hashicorp/go-plugin"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
)

// uiWrapper wraps an XoloPluginServer and handles Initialize by:
// 1. Connecting to XoloHostService via the broker
// 2. Starting the plugin's HTTP server
// 3. Returning the HTTP port
type uiWrapper struct {
	proto.XoloPluginServer
	pluginName string
	uiHandler  http.Handler
	broker     *plugin.GRPCBroker // set by GRPCServer via setBroker before any gRPC call
}

func newUIWrapper(impl proto.XoloPluginServer, pluginName string, uiHandler http.Handler) *uiWrapper {
	return &uiWrapper{
		XoloPluginServer: impl,
		pluginName:       pluginName,
		uiHandler:        uiHandler,
	}
}

// setBroker is called by XoloPluginGRPC.GRPCServer before any gRPC connections.
func (w *uiWrapper) setBroker(broker *plugin.GRPCBroker) {
	w.broker = broker
}

// Initialize connects to XoloHostService and starts the HTTP server.
func (w *uiWrapper) Initialize(_ context.Context, req *proto.InitializeRequest) (*proto.InitializeResponse, error) {
	if w.broker == nil {
		return nil, fmt.Errorf("broker not set: GRPCServer must be called before Initialize")
	}

	conn, err := w.broker.Dial(req.HostServiceBrokerId)
	if err != nil {
		return nil, fmt.Errorf("dial XoloHostService: %w", err)
	}

	hostClient := newGRPCHostClient(conn)

	// Inject HostClient and plugin name into every request context.
	wrapped := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		ctx := contextWithHostClient(r.Context(), hostClient)
		ctx = contextWithPluginName(ctx, w.pluginName)
		w.uiHandler.ServeHTTP(rw, r.WithContext(ctx))
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	srv := &http.Server{Handler: wrapped}
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			_ = err
		}
	}()

	port := uint32(ln.Addr().(*net.TCPAddr).Port)
	return &proto.InitializeResponse{HttpUiPort: port}, nil
}
