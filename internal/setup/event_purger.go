package setup

import (
	"context"
	"log/slog"

	"github.com/bornholm/xolo/internal/adapter/eventbus"
	"github.com/bornholm/xolo/internal/config"
	"github.com/pkg/errors"
)

// getEventPurgerFromConfig creates the ring-buffer purger and starts its
// background loop. Created at most once per config.
var getEventPurgerFromConfig = createFromConfigOnce(func(ctx context.Context, conf *config.Config) (*eventbus.Purger, error) {
	eventStore, err := getEventStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	settingsStore, err := getEventSettingsStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	purger := eventbus.NewPurger(eventStore, settingsStore, conf.Events.PurgeInterval, conf.Events.MaxPerOrg, conf.Events.DefaultPerOrg)

	go func() {
		purgeCtx := context.Background()
		if err := purger.Run(purgeCtx); err != nil {
			slog.ErrorContext(purgeCtx, "event purger stopped", slog.Any("error", errors.WithStack(err)))
		}
	}()

	return purger, nil
})
