package setup

import (
	"context"

	"github.com/bornholm/xolo/internal/config"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/pkg/errors"
)

var getEventStoreFromConfig = createFromConfigOnce(func(ctx context.Context, conf *config.Config) (port.EventStore, error) {
	store, err := getGormStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return store, nil
})

var getEventSettingsStoreFromConfig = createFromConfigOnce(func(ctx context.Context, conf *config.Config) (port.EventSettingsStore, error) {
	store, err := getGormStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return store, nil
})
