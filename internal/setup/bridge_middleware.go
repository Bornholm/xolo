package setup

import (
	"context"
	"net/http"

	"github.com/bornholm/xolo/internal/config"
	"github.com/bornholm/xolo/internal/http/middleware/bridge"
	"github.com/pkg/errors"
)

func getBridgeMiddlewareFromConfig(ctx context.Context, conf *config.Config) (func(http.Handler) http.Handler, error) {
	userStore, err := getUserStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	bridgeMiddleware := bridge.Middleware(userStore, conf.HTTP.Authn.ActiveByDefault, conf.HTTP.Authn.DefaultAdmins...)

	return bridgeMiddleware, nil
}
