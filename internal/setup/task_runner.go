package setup

import (
	"context"
	"log/slog"
	"time"

	"github.com/bornholm/xolo/internal/config"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/pkg/errors"
)

var TaskRunner = NewRegistry[port.TaskRunner]()

var getTaskRunner = createFromConfigOnce(func(ctx context.Context, conf *config.Config) (port.TaskRunner, error) {
	taskRunner, err := TaskRunner.From(conf.TaskRunner.URI)
	if err != nil {
		return nil, errors.Wrapf(err, "could not retrieve task runner for uri '%s'", conf.TaskRunner.URI)
	}

	go func() {
		taskRunnerCtx := context.Background()
		backoff := time.Second
		for {
			start := time.Now()
			if err := taskRunner.Run(taskRunnerCtx); err != nil {
				slog.ErrorContext(taskRunnerCtx, "error while running task runner", slog.Any("error", errors.WithStack(err)))
			}
			time.Sleep(backoff)
			if time.Since(start) > backoff/2 {
				backoff = time.Second
			} else {
				backoff *= 2
			}
		}
	}()

	return taskRunner, nil
})
