package setup

import (
	"context"

	"github.com/bornholm/xolo/internal/http/middleware/authn/oidc"
	"github.com/bornholm/xolo/internal/http/middleware/authn/oidctoken"
)

func getOIDCTokenAuthnHandlerFromConfig(ctx context.Context, oidcHandler *oidc.Handler) (*oidctoken.Handler, error) {
	providers := oidcHandler.ProvidersWithJWKS()

	if len(providers) == 0 {
		return nil, nil
	}

	handler := oidctoken.NewHandler(providers)

	return handler, nil
}