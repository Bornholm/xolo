package setup

import (
	"context"

	"github.com/bornholm/xolo/internal/config"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/pkg/errors"
)

var getApplicationStoreFromConfig = createFromConfigOnce(func(ctx context.Context, conf *config.Config) (port.ApplicationStore, error) {
	store, err := getGormStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return store, nil
})
