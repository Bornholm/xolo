package setup

import (
	"context"

	eventsAdapter "github.com/bornholm/xolo/internal/adapter/events"
	"github.com/bornholm/xolo/internal/config"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/pkg/errors"
)

var getApplicationStoreFromConfig = createFromConfigOnce(func(ctx context.Context, conf *config.Config) (port.ApplicationStore, error) {
	store, err := getGormStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	emitter, err := getEventEmitterFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return eventsAdapter.NewApplicationStore(store, emitter), nil
})
