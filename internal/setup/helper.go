package setup

import (
	"context"
	"sync"

	"github.com/bornholm/xolo/internal/config"
	"github.com/pkg/errors"
)

func createFromConfigOnce[T any](factory func(ctx context.Context, conf *config.Config) (T, error)) func(ctx context.Context, conf *config.Config) (T, error) {
	var (
		once    sync.Once
		service T
		onceErr error
	)

	return func(ctx context.Context, conf *config.Config) (T, error) {
		once.Do(func() {
			srv, err := factory(ctx, conf)
			if err != nil {
				onceErr = errors.WithStack(err)
				return
			}

			service = srv
		})
		if onceErr != nil {
			return *new(T), onceErr
		}

		return service, nil
	}
}
