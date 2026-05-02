package services

import (
	"sync"
	"time"
)

type cacheEntry struct {
	value     []byte
	expiresAt time.Time
}

// InMemoryCache is a simple TTL-based cache safe for concurrent use.
// Implements CacheService as a fallback when Redis is unavailable.
type InMemoryCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
}

func NewInMemoryCache() *InMemoryCache {
	c := &InMemoryCache{entries: make(map[string]cacheEntry)}
	go c.evictLoop()
	return c
}

func (c *InMemoryCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[key]
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.value, true
}

func (c *InMemoryCache) Set(key string, value []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = cacheEntry{value: value, expiresAt: time.Now().Add(ttl)}
}

func (c *InMemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

// evictLoop periodically removes expired entries to prevent memory leaks.
func (c *InMemoryCache) evictLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for k, e := range c.entries {
			if now.After(e.expiresAt) {
				delete(c.entries, k)
			}
		}
		c.mu.Unlock()
	}
}
