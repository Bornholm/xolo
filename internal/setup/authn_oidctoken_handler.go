package setup

import (
	"context"

	"github.com/bornholm/xolo/internal/config"
	"github.com/bornholm/xolo/internal/http/middleware/authn/oidc"
	"github.com/bornholm/xolo/internal/http/middleware/authn/oidctoken"
)

func getOIDCTokenAuthnHandlerFromConfig(ctx context.Context, conf *config.Config, oidcHandler *oidc.Handler) (*oidctoken.Handler, error) {
	providers := oidcHandler.ProvidersWithJWKS()

	if len(providers) == 0 {
		return nil, nil
	}

	opts := []oidctoken.OptionFunc{}
	if len(conf.HTTP.Authn.CookiesToCheck) > 0 {
		opts = append(opts, oidctoken.WithCookieNames(conf.HTTP.Authn.CookiesToCheck...))
	}
	if conf.HTTP.Authn.OIDCTokenIgnoreExpiry {
		opts = append(opts, oidctoken.WithIgnoreTokenExpiry())
	}

	handler := oidctoken.NewHandler(providers, opts...)

	return handler, nil
}