package cache

import (
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

type Cacheable interface {
	CacheKeys() []string
}

type MultiIndexCache[V Cacheable] struct {
	cache *expirable.LRU[string, V]
	mu    sync.RWMutex
}

func NewMultiIndexCache[V Cacheable](size int, ttl time.Duration) *MultiIndexCache[V] {
	cache := expirable.NewLRU[string, V](size, nil, ttl)
	return &MultiIndexCache[V]{
		cache: cache,
	}
}

func (c *MultiIndexCache[V]) Add(item V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	keys := item.CacheKeys()
	for _, key := range keys {
		c.cache.Add(key, item)
	}
}

func (c *MultiIndexCache[V]) Get(key string) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.cache.Get(key)
}

func (c *MultiIndexCache[V]) Remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	val, ok := c.cache.Peek(key)
	if !ok {
		return
	}

	allKeys := val.CacheKeys()

	for _, k := range allKeys {
		c.cache.Remove(k)
	}
}

func (c *MultiIndexCache[V]) Len() int {
	return c.cache.Len()
}
