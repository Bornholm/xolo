package setup

import (
	"context"
	"log/slog"

	"github.com/bornholm/xolo/internal/adapter/cache"
	eventsAdapter "github.com/bornholm/xolo/internal/adapter/events"
	"github.com/bornholm/xolo/internal/config"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/pkg/errors"
)

var getProviderStoreFromConfig = createFromConfigOnce(func(ctx context.Context, conf *config.Config) (port.ProviderStore, error) {
	store, err := getGormStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	emitter, err := getEventEmitterFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var providerStore port.ProviderStore = eventsAdapter.NewProviderStore(store, emitter)

	// The cache must wrap the outermost store so every mutation (from the web/API
	// handlers or hooks) flows through it and invalidates the affected entries.
	if conf.Storage.Database.Cache.Providers.Enabled {
		slog.DebugContext(ctx, "using cached provider store",
			slog.Duration("ttl", conf.Storage.Database.Cache.Providers.TTL),
			slog.Int("cache_size", conf.Storage.Database.Cache.Providers.Size))
		providerStore = cache.NewProviderStore(providerStore, conf.Storage.Database.Cache.Providers.Size, conf.Storage.Database.Cache.Providers.TTL)
	}

	return providerStore, nil
})
