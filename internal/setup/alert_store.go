package setup

import (
	"context"

	"github.com/bornholm/xolo/internal/config"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/pkg/errors"
)

var getAlertStoreFromConfig = createFromConfigOnce(func(ctx context.Context, conf *config.Config) (port.AlertStore, error) {
	store, err := getGormStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return store, nil
})

var getAlertIncidentStoreFromConfig = createFromConfigOnce(func(ctx context.Context, conf *config.Config) (port.AlertIncidentStore, error) {
	store, err := getGormStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return store, nil
})
