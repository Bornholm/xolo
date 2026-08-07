package setup

import (
	"context"

	"github.com/xolo-gateway/xolo/internal/config"
	"github.com/xolo-gateway/xolo/internal/core/port"
	"github.com/pkg/errors"
)

var getSecretStoreFromConfig = createFromConfigOnce(func(ctx context.Context, conf *config.Config) (port.SecretStore, error) {
	store, err := getGormStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return store, nil
})
