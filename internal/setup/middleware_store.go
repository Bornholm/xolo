package setup

import (
	"context"

	"github.com/bornholm/xolo/internal/config"
	"github.com/bornholm/xolo/internal/core/port"
)

var getMiddlewareStoreFromConfig = createFromConfigOnce(func(ctx context.Context, conf *config.Config) (port.MiddlewareStore, error) {
	store, err := getGormStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, err
	}
	return store, nil
})
