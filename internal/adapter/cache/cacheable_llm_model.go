package cache

import "github.com/bornholm/xolo/internal/core/model"

type CacheableLLMModel struct {
	model.LLMModel
}

// CacheKeys implements [Cacheable].
//
// A model is indexed both by its ID and by its (org, proxy name) pair, the two
// ways the hot proxy path looks it up. When the model is later removed from the
// cache, [MultiIndexCache.Remove] peeks the stored value and drops every key it
// exposes — so an invalidation by ID also clears a stale proxy-name entry.
func (m *CacheableLLMModel) CacheKeys() []string {
	return []string{
		string(m.ID()),
		getLLMModelProxyNameCacheKey(m.OrgID(), m.ProxyName()),
	}
}

func NewCacheableLLMModel(m model.LLMModel) *CacheableLLMModel {
	return &CacheableLLMModel{m}
}

func getLLMModelProxyNameCacheKey(orgID model.OrgID, proxyName string) string {
	return getCompositeCacheKey(string(orgID), proxyName)
}

var (
	_ model.LLMModel = &CacheableLLMModel{}
	_ Cacheable      = &CacheableLLMModel{}
)
