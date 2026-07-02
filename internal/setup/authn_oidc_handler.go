package setup

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bornholm/xolo/internal/config"
	"github.com/bornholm/xolo/internal/http/middleware/authn/oidc"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/gitea"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"
	"github.com/markbates/goth/providers/openidConnect"
	"github.com/pkg/errors"
)

type OIDCDiscovery struct {
	Issuer                string `json:"issuer"`
	JWKSURI               string `json:"jwks_uri"`
	AuthURL               string `json:"authorization_endpoint"`
	TokenURL              string `json:"token_endpoint"`
	UserInfoEndpoint      string `json:"userinfo_endpoint"`
	IntrospectionEndpoint string `json:"introspection_endpoint"`
}

func getOIDCAuthnHandlerFromConfig(ctx context.Context, conf *config.Config) (*oidc.Handler, error) {
	sessionStore, err := getSessionStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Configure providers

	gothProviders := make([]goth.Provider, 0)
	providers := make([]oidc.Provider, 0)
	providersWithJWKS := make([]oidc.ProviderWithJWKS, 0)

	if conf.HTTP.Authn.Providers.Google.Key != "" && conf.HTTP.Authn.Providers.Google.Secret != "" {
		googleProvider := google.New(
			string(conf.HTTP.Authn.Providers.Google.Key),
			string(conf.HTTP.Authn.Providers.Google.Secret),
			fmt.Sprintf("%s/auth/oidc/providers/google/callback", conf.HTTP.BaseURL),
			conf.HTTP.Authn.Providers.Google.Scopes...,
		)

		gothProviders = append(gothProviders, googleProvider)

		providers = append(providers, oidc.Provider{
			ID:    googleProvider.Name(),
			Label: "Google",
			Icon:  "log-in",
		})

		providersWithJWKS = append(providersWithJWKS, oidc.ProviderWithJWKS{
			ID:      googleProvider.Name(),
			Label:   "Google",
			Icon:    "log-in",
			Issuer:  "https://accounts.google.com",
			JWKSURL: "https://www.googleapis.com/oauth2/v3/certs",
		})
	}

	if conf.HTTP.Authn.Providers.Github.Key != "" && conf.HTTP.Authn.Providers.Github.Secret != "" {
		githubProvider := github.New(
			string(conf.HTTP.Authn.Providers.Github.Key),
			string(conf.HTTP.Authn.Providers.Github.Secret),
			fmt.Sprintf("%s/auth/oidc/providers/github/callback", conf.HTTP.BaseURL),
			conf.HTTP.Authn.Providers.Github.Scopes...,
		)

		gothProviders = append(gothProviders, githubProvider)

		providers = append(providers, oidc.Provider{
			ID:    githubProvider.Name(),
			Label: "Github",
			Icon:  "github",
		})

		issuer := "https://github.com"
		if conf.HTTP.BaseURL != "" && conf.HTTP.BaseURL != "/" {
			issuer = conf.HTTP.BaseURL
		}
		providersWithJWKS = append(providersWithJWKS, oidc.ProviderWithJWKS{
			ID:      githubProvider.Name(),
			Label:   "Github",
			Icon:    "github",
			Issuer:  issuer,
			JWKSURL: "https://token.actions.githubusercontent.com/.well-known/jwks",
		})
	}

	if conf.HTTP.Authn.Providers.Gitea.Key != "" && conf.HTTP.Authn.Providers.Gitea.Secret != "" {
		giteaProvider := gitea.NewCustomisedURL(
			string(conf.HTTP.Authn.Providers.Gitea.Key),
			string(conf.HTTP.Authn.Providers.Gitea.Secret),
			fmt.Sprintf("%s/auth/oidc/providers/gitea/callback", conf.HTTP.BaseURL),
			string(conf.HTTP.Authn.Providers.Gitea.AuthURL),
			string(conf.HTTP.Authn.Providers.Gitea.TokenURL),
			string(conf.HTTP.Authn.Providers.Gitea.ProfileURL),
			conf.HTTP.Authn.Providers.Gitea.Scopes...,
		)

		gothProviders = append(gothProviders, giteaProvider)

		providers = append(providers, oidc.Provider{
			ID:    giteaProvider.Name(),
			Label: string(conf.HTTP.Authn.Providers.Gitea.Label),
			Icon:  "gitlab",
		})

		discoveryURL := string(conf.HTTP.Authn.Providers.Gitea.DiscoveryURL)

		discovery, err := fetchOIDCDiscovery(ctx, discoveryURL)
		if err == nil && discovery != nil && discovery.JWKSURI != "" {
			providersWithJWKS = append(providersWithJWKS, oidc.ProviderWithJWKS{
				ID:               giteaProvider.Name(),
				Label:            string(conf.HTTP.Authn.Providers.Gitea.Label),
				Icon:             "gitlab",
				Issuer:           discovery.Issuer,
				JWKSURL:          discovery.JWKSURI,
				IntrospectionURL: discovery.IntrospectionEndpoint,
				UserInfoURL:      discovery.UserInfoEndpoint,
				ClientID:         string(conf.HTTP.Authn.Providers.Gitea.Key),
				ClientSecret:     string(conf.HTTP.Authn.Providers.Gitea.Secret),
			})
		}
	}

	for _, np := range conf.HTTP.Authn.OIDCProviders {
		if np.Key == "" || np.Secret == "" {
			continue
		}

		gothProvider, provider, withJWKS, err := buildOIDCProvider(ctx, conf.HTTP.BaseURL, np)
		if err != nil {
			return nil, errors.Wrapf(err, "could not configure oidc provider %q", np.ID)
		}

		gothProviders = append(gothProviders, gothProvider)
		providers = append(providers, provider)
		if withJWKS != nil {
			providersWithJWKS = append(providersWithJWKS, *withJWKS)
		}
	}

	goth.UseProviders(gothProviders...)
	gothic.Store = sessionStore

	opts := []oidc.OptionFunc{
		oidc.WithProviders(providers...),
		oidc.WithProvidersWithJWKS(providersWithJWKS),
	}

	handler := oidc.NewHandler(
		sessionStore,
		opts...,
	)

	return handler, nil
}

func getRandomBytes(n int) ([]byte, error) {
	data := make([]byte, n)

	read, err := rand.Read(data)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if read != n {
		return nil, errors.Errorf("could not read %d bytes", n)
	}

	return data, nil
}

// buildOIDCProvider configures a single named OIDC provider: the goth provider
// (for interactive login), its login-button descriptor, and — when discovery
// succeeds — its JWKS/introspection/userinfo descriptor used by the oidctoken
// and oauth2token authenticators. The provider ID is reused as the goth name so
// it stays consistent across the interactive-login and API-token paths.
func buildOIDCProvider(ctx context.Context, baseURL string, np config.NamedOIDCProvider) (goth.Provider, oidc.Provider, *oidc.ProviderWithJWKS, error) {
	discoveryURL := string(np.DiscoveryURL)
	callbackURL := fmt.Sprintf("%s/auth/oidc/providers/%s/callback", baseURL, np.ID)

	gothProvider, err := openidConnect.NewNamed(
		np.ID,
		string(np.Key),
		string(np.Secret),
		callbackURL,
		discoveryURL,
		np.Scopes...,
	)
	if err != nil {
		return nil, oidc.Provider{}, nil, errors.WithStack(err)
	}

	provider := oidc.Provider{
		ID:    np.ID,
		Label: np.Label,
		Icon:  np.Icon,
	}

	var withJWKS *oidc.ProviderWithJWKS
	discovery, err := fetchOIDCDiscovery(ctx, discoveryURL)
	if err == nil && discovery != nil && discovery.JWKSURI != "" {
		withJWKS = &oidc.ProviderWithJWKS{
			ID:               np.ID,
			Label:            np.Label,
			Icon:             np.Icon,
			DiscoveryURL:     discoveryURL,
			Issuer:           discovery.Issuer,
			JWKSURL:          discovery.JWKSURI,
			IntrospectionURL: discovery.IntrospectionEndpoint,
			UserInfoURL:      discovery.UserInfoEndpoint,
			ClientID:         string(np.Key),
			ClientSecret:     string(np.Secret),
			RequiredScope:    np.RequiredScope,
			RequiredAudience: np.RequiredAudience,
		}
	}

	return gothProvider, provider, withJWKS, nil
}

func fetchOIDCDiscovery(ctx context.Context, discoveryURL string) (*OIDCDiscovery, error) {
	if discoveryURL == "" {
		return nil, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("discovery fetch failed with status %d", resp.StatusCode)
	}

	var discovery OIDCDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		return nil, errors.WithStack(err)
	}

	return &discovery, nil
}
