package pluginsdk

import "context"

type contextKey int

const (
	hostClientKey contextKey = iota
	pluginNameKey
)

// HostClientFromContext retrieves the HostClient injected by ServeWithUI middleware.
// Returns nil if not present (indicates a bug in ServeWithUI wiring).
func HostClientFromContext(ctx context.Context) HostClient {
	v, _ := ctx.Value(hostClientKey).(HostClient)
	return v
}

// PluginNameFromContext retrieves the plugin name injected by ServeWithUI middleware.
func PluginNameFromContext(ctx context.Context) string {
	v, _ := ctx.Value(pluginNameKey).(string)
	return v
}

func contextWithHostClient(ctx context.Context, client HostClient) context.Context {
	return context.WithValue(ctx, hostClientKey, client)
}

func contextWithPluginName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, pluginNameKey, name)
}
