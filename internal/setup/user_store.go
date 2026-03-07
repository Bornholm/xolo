package setup

import (
	"context"
	"log/slog"

	"github.com/bornholm/xolo/internal/adapter/cache"
	"github.com/bornholm/xolo/internal/config"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/pkg/errors"
)

var getUserStoreFromConfig = createFromConfigOnce(func(ctx context.Context, conf *config.Config) (port.UserStore, error) {
	var (
		store port.UserStore
		err   error
	)

	store, err = getGormStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if conf.Storage.Database.Cache.Users.Enabled {
		slog.DebugContext(ctx, "using cached user store", slog.Duration("ttl", conf.Storage.Database.Cache.Users.TTL), slog.Int("cache_size", conf.Storage.Database.Cache.Users.Size))
		store = cache.NewUserStore(store, conf.Storage.Database.Cache.Users.Size, conf.Storage.Database.Cache.Users.TTL)
	}

	return store, nil
})
