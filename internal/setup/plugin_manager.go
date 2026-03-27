package setup

import (
	"context"
	"sync"

	"github.com/bornholm/xolo/internal/config"
	"github.com/bornholm/xolo/internal/plugin"
)

var (
	pluginManagerOnce sync.Once
	pluginManagerVal  *plugin.Manager
	pluginManagerErr  error
)

func getPluginManagerFromConfig(ctx context.Context, conf *config.Config) (*plugin.Manager, error) {
	pluginManagerOnce.Do(func() {
		configStore, err := getPluginConfigStoreFromConfig(ctx, conf)
		if err != nil {
			pluginManagerErr = err
			return
		}
		providerStore, err := getProviderStoreFromConfig(ctx, conf)
		if err != nil {
			pluginManagerErr = err
			return
		}
		virtualModelStore, err := getVirtualModelStoreFromConfig(ctx, conf)
		if err != nil {
			pluginManagerErr = err
			return
		}
		mgr := plugin.NewManager(conf.Plugins.Dir, configStore, providerStore, virtualModelStore)
		if err := mgr.Start(ctx); err != nil {
			pluginManagerErr = err
			return
		}
		pluginManagerVal = mgr
	})
	return pluginManagerVal, pluginManagerErr
}
