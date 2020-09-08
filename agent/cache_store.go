package agent

import (
	"net/http"
	"sync"
)

type CacheStore interface {
	Get(*http.Request) *Cache
	Put(*http.Request, *Cache)
	Clear()
}

type cacheStore struct {
	mu    sync.RWMutex
	table map[string]*Cache
}

func NewCacheStore() CacheStore {
	return &cacheStore{
		mu:    sync.RWMutex{},
		table: make(map[string]*Cache),
	}
}

func (c *cacheStore) Get(r *http.Request) *Cache {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c, ok := c.table[r.URL.String()]; ok && c != nil {
		return c
	}

	return nil
}

func (c *cacheStore) Put(r *http.Request, cache *Cache) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.table[r.URL.String()] = cache
}

func (c *cacheStore) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.table = make(map[string]*Cache)
}
