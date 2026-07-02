package setup

import (
	"context"

	"github.com/bornholm/xolo/internal/config"
	"github.com/bornholm/xolo/internal/http/middleware/authn/oauth2token"
	"github.com/bornholm/xolo/internal/http/middleware/authn/oidc"
)

// getOAuth2TokenAuthnHandlerFromConfig builds the authenticator that validates
// incoming opaque OAuth2 access tokens (RFC 7662 introspection when available,
// otherwise the OIDC UserInfo endpoint). It returns nil (feature disabled)
// unless enabled and at least one OIDC provider can validate tokens. Providers
// are discovered from the OIDC handler so their IDs match those used for
// interactive logins, and carry their own scope/audience requirements.
func getOAuth2TokenAuthnHandlerFromConfig(_ context.Context, conf *config.Config, oidcHandler *oidc.Handler) (*oauth2token.Handler, error) {
	if !conf.HTTP.Authn.OAuth2Token.Enabled {
		return nil, nil
	}

	providers := oidcHandler.ProvidersForTokenValidation()
	if len(providers) == 0 {
		return nil, nil
	}

	opts := []oauth2token.OptionFunc{
		oauth2token.WithCacheSize(conf.HTTP.Authn.OAuth2Token.CacheSize),
	}
	if conf.HTTP.Authn.OAuth2Token.CacheTTL > 0 {
		opts = append(opts, oauth2token.WithCacheTTL(conf.HTTP.Authn.OAuth2Token.CacheTTL))
	}

	return oauth2token.NewHandler(providers, opts...), nil
}
