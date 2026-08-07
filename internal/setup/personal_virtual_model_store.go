package setup

import (
	"context"

	"github.com/xolo-gateway/xolo/internal/config"
	"github.com/xolo-gateway/xolo/internal/core/port"
)

var getPersonalVirtualModelStoreFromConfig = createFromConfigOnce(func(ctx context.Context, conf *config.Config) (port.PersonalVirtualModelStore, error) {
	store, err := getGormStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, err
	}
	return store, nil
})
