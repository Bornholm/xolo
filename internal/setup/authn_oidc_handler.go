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
	Issuer   string `json:"issuer"`
	JWKSURI  string `json:"jwks_uri"`
	AuthURL  string `json:"authorization_endpoint"`
	TokenURL string `json:"token_endpoint"`
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
			Icon:  "fa-google",
		})

		providersWithJWKS = append(providersWithJWKS, oidc.ProviderWithJWKS{
			ID:      googleProvider.Name(),
			Label:   "Google",
			Icon:    "fa-google",
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
			Icon:  "fa-github",
		})

		issuer := "https://github.com"
		if conf.HTTP.BaseURL != "" && conf.HTTP.BaseURL != "/" {
			issuer = conf.HTTP.BaseURL
		}
		providersWithJWKS = append(providersWithJWKS, oidc.ProviderWithJWKS{
			ID:      githubProvider.Name(),
			Label:   "Github",
			Icon:    "fa-github",
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
			Icon:  "fa-git-alt",
		})

		discoveryURL := string(conf.HTTP.Authn.Providers.Gitea.DiscoveryURL)

		jwksURL, issuer, err := fetchOIDCDiscovery(ctx, discoveryURL)
		if err == nil && jwksURL != "" {
			providersWithJWKS = append(providersWithJWKS, oidc.ProviderWithJWKS{
				ID:      giteaProvider.Name(),
				Label:   string(conf.HTTP.Authn.Providers.Gitea.Label),
				Icon:    "fa-git-alt",
				Issuer:  issuer,
				JWKSURL: jwksURL,
			})
		}
	}

	if conf.HTTP.Authn.Providers.OIDC.Key != "" && conf.HTTP.Authn.Providers.OIDC.Secret != "" {
		discoveryURL := string(conf.HTTP.Authn.Providers.OIDC.DiscoveryURL)
		oidcProvider, err := openidConnect.New(
			string(conf.HTTP.Authn.Providers.OIDC.Key),
			string(conf.HTTP.Authn.Providers.OIDC.Secret),
			fmt.Sprintf("%s/auth/oidc/providers/openid-connect/callback", conf.HTTP.BaseURL),
			discoveryURL,
			conf.HTTP.Authn.Providers.OIDC.Scopes...,
		)
		if err != nil {
			return nil, errors.Wrap(err, "could not configure oidc provider")
		}

		gothProviders = append(gothProviders, oidcProvider)

		providers = append(providers, oidc.Provider{
			ID:    oidcProvider.Name(),
			Label: string(conf.HTTP.Authn.Providers.OIDC.Label),
			Icon:  string(conf.HTTP.Authn.Providers.OIDC.Icon),
		})

		jwksURL, issuer, err := fetchOIDCDiscovery(ctx, discoveryURL)
		if err == nil && jwksURL != "" {
			providersWithJWKS = append(providersWithJWKS, oidc.ProviderWithJWKS{
				ID:           oidcProvider.Name(),
				Label:        string(conf.HTTP.Authn.Providers.OIDC.Label),
				Icon:         string(conf.HTTP.Authn.Providers.OIDC.Icon),
				DiscoveryURL: discoveryURL,
				Issuer:       issuer,
				JWKSURL:      jwksURL,
			})
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

func fetchOIDCDiscovery(ctx context.Context, discoveryURL string) (jwksURL, issuer string, err error) {
	if discoveryURL == "" {
		return "", "", nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return "", "", errors.WithStack(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", errors.WithStack(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", errors.Errorf("discovery fetch failed with status %d", resp.StatusCode)
	}

	var discovery OIDCDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		return "", "", errors.WithStack(err)
	}

	return discovery.JWKSURI, discovery.Issuer, nil
}
