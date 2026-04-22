package oidctoken

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"math/big"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/bornholm/xolo/internal/http/middleware/authn"
	"github.com/pkg/errors"
)

var errInvalidToken = errors.New("invalid token")

type claims struct {
	jwt.RegisteredClaims
	Email       string `json:"email"`
	PreferredUsername string `json:"preferred_username"`
	Name        string `json:"name"`
}

type Handler struct {
	providers []Provider
}

func NewHandler(providers []Provider) *Handler {
	return &Handler{
		providers: providers,
	}
}

func (h *Handler) Authenticate(w http.ResponseWriter, r *http.Request) (*authn.User, error) {
	authorization := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authorization, "Bearer ")

	if token == "" {
		return nil, nil
	}

	for _, provider := range h.providers {
		user, err := h.validateToken(r.Context(), token, provider)
		if err == nil {
			return user, nil
		}

		if errors.Is(err, errInvalidToken) {
			continue
		}

		return nil, errors.WithStack(err)
	}

	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	return nil, authn.ErrSkipRequest
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
		Subject: cl.Subject,
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