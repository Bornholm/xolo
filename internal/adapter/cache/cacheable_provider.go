package cache

import "github.com/bornholm/xolo/internal/core/model"

type CacheableProvider struct {
	model.Provider
}

// CacheKeys implements [Cacheable].
func (p *CacheableProvider) CacheKeys() []string {
	return []string{
		string(p.ID()),
	}
}

func NewCacheableProvider(p model.Provider) *CacheableProvider {
	return &CacheableProvider{p}
}

var (
	_ model.Provider = &CacheableProvider{}
	_ Cacheable      = &CacheableProvider{}
)
