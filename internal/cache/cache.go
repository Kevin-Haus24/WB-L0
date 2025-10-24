package cache

import (
	"encoding/json"
	"sync"
)

type Cache struct {
	mu sync.RWMutex
	m  map[string]json.RawMessage
}

func New() *Cache {
	return &Cache{m: make(map[string]json.RawMessage)}
}

func (c *Cache) Get(id string) (json.RawMessage, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.m[id]
	return val, ok
}

func (c *Cache) Set(id string, data json.RawMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[id] = data
}

func (c *Cache) LoadAll(data map[string]json.RawMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, v := range data {
		c.m[k] = v
	}
}
