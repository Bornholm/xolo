package setup

import (
	"context"

	eventsAdapter "github.com/bornholm/xolo/internal/adapter/events"
	"github.com/bornholm/xolo/internal/config"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/pkg/errors"
)

var getInviteStoreFromConfig = createFromConfigOnce(func(ctx context.Context, conf *config.Config) (port.InviteStore, error) {
	store, err := getGormStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	emitter, err := getEventEmitterFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return eventsAdapter.NewInviteStore(store, emitter), nil
})
