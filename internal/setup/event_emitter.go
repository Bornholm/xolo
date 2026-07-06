package setup

import (
	"context"
	"log/slog"

	"github.com/bornholm/xolo/internal/adapter/eventbus"
	"github.com/bornholm/xolo/internal/config"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/pkg/errors"
)

// getEventEmitterFromConfig creates the async event emitter and starts its
// background worker (following the task-runner goroutine pattern). It is created
// at most once per config.
var getEventEmitterFromConfig = createFromConfigOnce(func(ctx context.Context, conf *config.Config) (port.EventEmitter, error) {
	eventStore, err := getEventStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	emitter := eventbus.NewAsyncEmitter(eventStore, conf.Events.EmitBufferSize)

	go func() {
		emitterCtx := context.Background()
		if err := emitter.Run(emitterCtx); err != nil {
			slog.ErrorContext(emitterCtx, "event emitter stopped", slog.Any("error", errors.WithStack(err)))
		}
	}()

	return emitter, nil
})
