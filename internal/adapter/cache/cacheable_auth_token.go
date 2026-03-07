package cache

import "github.com/bornholm/xolo/internal/core/model"

type CacheableAuthToken struct {
	model.AuthToken
}

// CacheKeys implements [Cacheable].
func (t *CacheableAuthToken) CacheKeys() []string {
	return []string{
		t.Value(),
		string(t.ID()),
	}
}

func NewCacheableAuthToken(authToken model.AuthToken) *CacheableAuthToken {
	return &CacheableAuthToken{authToken}
}

var (
	_ model.AuthToken = &CacheableAuthToken{}
	_ Cacheable       = &CacheableAuthToken{}
)
