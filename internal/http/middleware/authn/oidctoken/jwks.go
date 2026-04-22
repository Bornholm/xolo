package oidctoken

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/pkg/errors"
)

const jwksCacheDuration = 1 * time.Hour

type cachedJWKS struct {
	JWKS      JWKS
	ExpiresAt time.Time
}

type jwksCache struct {
	mu    sync.RWMutex
	items map[string]cachedJWKS
}

var cache jwksCache

func init() {
	cache.items = map[string]cachedJWKS{}
}

func (c *jwksCache) Get(jwksURL string) (JWKS, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[jwksURL]
	if !ok || time.Now().After(item.ExpiresAt) {
		return JWKS{}, false
	}

	return item.JWKS, true
}

func (c *jwksCache) Set(jwksURL string, jwks JWKS) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[jwksURL] = cachedJWKS{
		JWKS:      jwks,
		ExpiresAt: time.Now().Add(jwksCacheDuration),
	}
}

func (c *jwksCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = map[string]cachedJWKS{}
}

func fetchJWKS(ctx context.Context, jwksURL string) (JWKS, error) {
	if cached, ok := cache.Get(jwksURL); ok {
		return cached, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return JWKS{}, errors.WithStack(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return JWKS{}, errors.WithStack(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return JWKS{}, errors.Errorf("jwks fetch failed with status %d", resp.StatusCode)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		slog.ErrorContext(ctx, "could not decode JWKS", slog.Any("error", errors.WithStack(err)))
		return JWKS{}, errors.WithStack(err)
	}

	cache.Set(jwksURL, jwks)

	return jwks, nil
}