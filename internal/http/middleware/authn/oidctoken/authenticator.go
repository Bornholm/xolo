package oidctoken

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"log/slog"
	"math/big"
	"net/http"
	"strings"

	"github.com/bornholm/xolo/internal/http/middleware/authn"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
)

var errInvalidToken = errors.New("invalid token")

type claims struct {
	jwt.RegisteredClaims
	Email             string `json:"email"`
	PreferredUsername string `json:"preferred_username"`
	Name              string `json:"name"`
}

type Options struct {
	CookieNames []string
}

type OptionFunc func(*Options)

func WithCookieNames(names ...string) OptionFunc {
	return func(o *Options) {
		o.CookieNames = names
	}
}

type Handler struct {
	providers []Provider
	options   Options
}

func NewHandler(providers []Provider, funcs ...OptionFunc) *Handler {
	opts := Options{
		CookieNames: []string{"oauth_id_token"},
	}
	for _, f := range funcs {
		f(&opts)
	}

	return &Handler{
		providers: providers,
		options:   opts,
	}
}

func (h *Handler) Authenticate(w http.ResponseWriter, r *http.Request) (*authn.User, error) {
	tokens := extractTokens(r, h.options.CookieNames)
	slog.Debug("oidctoken.Authenticate: tokens extracted", "count", len(tokens))

	for _, token := range tokens {
		slog.Debug("oidctoken.Authenticate: trying token", "prefix", token[:50])
		for _, provider := range h.providers {
			slog.Debug("oidctoken.Authenticate: trying provider", "provider", provider.ID)
			user, err := h.validateToken(r.Context(), token, provider)
			if err == nil && user != nil {
				slog.Debug("oidctoken.Authenticate: success", "user", user.Subject)
				return user, nil
			}

			if errors.Is(err, errInvalidToken) {
				slog.Debug("oidctoken.Authenticate: invalid token", "error", err)
				continue
			}

			slog.Debug("oidctoken.Authenticate: error", "error", err)
			return nil, errors.WithStack(err)
		}
	}

	return nil, nil
}

func extractTokens(r *http.Request, cookieNames []string) []string {
	var tokens []string

	authHeader := r.Header.Get("Authorization")
	if token := strings.TrimPrefix(authHeader, "Bearer "); token != "" {
		tokens = append(tokens, token)
	}

	for _, name := range cookieNames {
		if cookie, err := r.Cookie(name); err == nil && cookie.Value != "" {
			tokens = append(tokens, cookie.Value)
		}
	}

	return tokens
}

func (h *Handler) validateToken(ctx context.Context, rawToken string, provider Provider) (*authn.User, error) {
	if provider.JWKSURL == "" {
		return nil, errInvalidToken
	}

	jwks, err := fetchJWKS(ctx, provider.JWKSURL)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	token, err := jwt.ParseWithClaims(rawToken, &claims{}, func(t *jwt.Token) (interface{}, error) {
		kid, ok := t.Header["kid"].(string)
		if !ok {
			return nil, errors.New("token missing kid")
		}

		for _, key := range jwks.Keys {
			if key.Kid != kid {
				continue
			}

			if key.Kty != "RSA" {
				return nil, errors.Errorf("unsupported key type: %s", key.Kty)
			}

			return parseRSAPublicKey(key)
		}

		return nil, errors.New("key not found")
	}, jwt.WithIssuer(provider.Issuer))

	if err != nil {
		return nil, errInvalidToken
	}

	cl, ok := token.Claims.(*claims)
	if !ok {
		return nil, errInvalidToken
	}

	if cl.Subject == "" {
		return nil, errInvalidToken
	}

	user := &authn.User{
		Email:    cl.Email,
		Provider: provider.ID,
		Subject:  cl.Subject,
	}

	if cl.PreferredUsername != "" {
		user.DisplayName = cl.PreferredUsername
	} else if cl.Name != "" {
		user.DisplayName = cl.Name
	}

	return user, nil
}

func parseRSAPublicKey(key JWK) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := int(new(big.Int).SetBytes(eBytes).Int64())

	return &rsa.PublicKey{N: n, E: e}, nil
}

var _ authn.Authenticator = &Handler{}
