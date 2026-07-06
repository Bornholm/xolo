package setup

import (
	"context"

	eventsAdapter "github.com/bornholm/xolo/internal/adapter/events"
	"github.com/bornholm/xolo/internal/config"
	"github.com/bornholm/xolo/internal/core/port"
)

var getVirtualModelStoreFromConfig = createFromConfigOnce(func(ctx context.Context, conf *config.Config) (port.VirtualModelStore, error) {
	store, err := getGormStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, err
	}
	emitter, err := getEventEmitterFromConfig(ctx, conf)
	if err != nil {
		return nil, err
	}
	return eventsAdapter.NewVirtualModelStore(store, emitter), nil
})
