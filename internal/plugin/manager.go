package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/bornholm/xolo/internal/core/port"
	"github.com/bornholm/xolo/pkg/pluginsdk"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
	goplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PluginEntry holds a loaded plugin's descriptor, gRPC client, and HTTP UI port.
type PluginEntry struct {
	Descriptor *proto.PluginDescriptor
	Client     proto.XoloPluginClient
	gopClient  *goplugin.Client
	HTTPPort   uint32
}

// Manager discovers and manages plugin subprocess lifecycles.
type Manager struct {
	dir         string
	configStore port.PluginConfigStore // may be nil (no UI support)
	mu          sync.RWMutex
	plugins     []*PluginEntry
}

// NewManager creates a Manager that will scan dir for plugin binaries.
// configStore is used by XoloHostService to serve GetConfig/SaveConfig to plugins.
// Pass nil if plugin UI is not needed (e.g. in tests that don't exercise UI).
func NewManager(dir string, configStore port.PluginConfigStore) *Manager {
	return &Manager{dir: dir, configStore: configStore}
}

// Start scans the plugin directory and launches each plugin subprocess.
// Missing or empty directory is not an error. Individual plugin failures
// are logged as warnings and skipped.
func (m *Manager) Start(ctx context.Context) error {
	entries, err := m.scanDir()
	if err != nil {
		// Directory missing or unreadable — treat as no plugins
		slog.WarnContext(ctx, "plugin dir unavailable, no plugins loaded",
			slog.String("dir", m.dir),
			slog.Any("error", err),
		)
		return nil
	}

	for _, path := range entries {
		entry, err := m.loadPlugin(ctx, path)
		if err != nil {
			slog.WarnContext(ctx, "failed to load plugin, skipping",
				slog.String("path", path),
				slog.Any("error", err),
			)
			continue
		}
		m.mu.Lock()
		m.plugins = append(m.plugins, entry)
		m.mu.Unlock()
		slog.InfoContext(ctx, "plugin loaded",
			slog.String("name", entry.Descriptor.Name),
			slog.String("version", entry.Descriptor.Version),
			slog.Uint64("http_ui_port", uint64(entry.HTTPPort)),
		)
	}
	return nil
}

// List returns descriptors for all successfully loaded plugins.
func (m *Manager) List() []*proto.PluginDescriptor {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*proto.PluginDescriptor, 0, len(m.plugins))
	for _, e := range m.plugins {
		out = append(out, e.Descriptor)
	}
	return out
}

// Get returns the gRPC client for a named plugin. Returns false if not found.
func (m *Manager) Get(name string) (proto.XoloPluginClient, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, e := range m.plugins {
		if e.Descriptor.Name == name {
			return e.Client, true
		}
	}
	return nil, false
}

// HTTPPort returns the HTTP UI port for a named plugin, or 0 if the plugin
// has no HTTP UI or is not loaded.
func (m *Manager) HTTPPort(pluginName string) uint32 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, e := range m.plugins {
		if e.Descriptor.Name == pluginName {
			return e.HTTPPort
		}
	}
	return 0
}

// Shutdown terminates all plugin subprocesses gracefully.
func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.plugins {
		e.gopClient.Kill()
	}
	m.plugins = nil
}

// scanDir returns paths of executable files directly in m.dir (no recursion).
func (m *Manager) scanDir() ([]string, error) {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(m.dir, e.Name())
		// Resolve symlinks and ensure they stay within the directory
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			continue
		}
		absDir, _ := filepath.Abs(m.dir)
		absResolved, _ := filepath.Abs(resolved)
		if !isWithinDir(absResolved, absDir) {
			slog.Warn("plugin symlink escapes plugin dir, skipping", slog.String("path", path))
			continue
		}
		info, err := os.Stat(resolved)
		if err != nil || info.Mode()&0111 == 0 {
			continue // not executable
		}
		paths = append(paths, resolved)
	}
	return paths, nil
}

func isWithinDir(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return len(rel) > 0 && rel[0] != '.'
}

func (m *Manager) loadPlugin(ctx context.Context, binaryPath string) (*PluginEntry, error) {
	client := goplugin.NewClient(&goplugin.ClientConfig{
		HandshakeConfig:  pluginsdk.HandshakeConfig,
		Plugins:          pluginsdk.PluginMap,
		Cmd:              exec.Command(binaryPath),
		AllowedProtocols: []goplugin.Protocol{goplugin.ProtocolGRPC},
	})

	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, err
	}

	raw, err := rpcClient.Dispense(pluginsdk.PluginName)
	if err != nil {
		client.Kill()
		return nil, err
	}

	bundle, ok := raw.(*pluginsdk.PluginClientBundle)
	if !ok {
		client.Kill()
		return nil, fmt.Errorf("unexpected plugin client type: %T", raw)
	}

	grpcClient := bundle.XoloPluginClient

	desc, err := grpcClient.Describe(ctx, &proto.DescribeRequest{})
	if err != nil {
		client.Kill()
		return nil, err
	}

	httpPort := m.initialize(ctx, grpcClient, bundle.Broker, desc.Name)

	return &PluginEntry{
		Descriptor: desc,
		Client:     grpcClient,
		gopClient:  client,
		HTTPPort:   httpPort,
	}, nil
}

// initialize calls Initialize on the plugin via the broker mechanism.
// Returns 0 if the plugin has no HTTP UI or if initialization fails.
func (m *Manager) initialize(ctx context.Context, client proto.XoloPluginClient, broker *goplugin.GRPCBroker, pluginName string) uint32 {
	if m.configStore == nil {
		return 0
	}

	brokerID := broker.NextId()
	hostSvc := NewXoloHostService(m.configStore)

	// AcceptAndServe blocks until the plugin connects, so it must run in a goroutine.
	// We call it before client.Initialize so the listener is ready when the plugin
	// calls broker.Dial inside its Initialize handler.
	go broker.AcceptAndServe(brokerID, func(opts []grpc.ServerOption) *grpc.Server {
		s := grpc.NewServer(opts...)
		proto.RegisterXoloHostServiceServer(s, hostSvc)
		return s
	})

	initCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := client.Initialize(initCtx, &proto.InitializeRequest{
		HostServiceBrokerId: brokerID,
	})
	if err != nil {
		if status.Code(err) == codes.Unimplemented {
			slog.DebugContext(ctx, "plugin does not implement Initialize, no HTTP UI",
				slog.String("plugin", pluginName),
			)
		} else {
			slog.WarnContext(ctx, "plugin Initialize failed, no HTTP UI",
				slog.String("plugin", pluginName),
				slog.Any("error", err),
			)
		}
		return 0
	}

	return resp.HttpUiPort
}
