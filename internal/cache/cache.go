package cache

import (
	"encoding/json"
	"sync"
)

// Cache хранит JSON по ключу order_uid
type Cache struct {
	mu sync.RWMutex               // защищает map от одновременного доступа нескольких горутин
	m  map[string]json.RawMessage // данные кэша
}

// New создаёт пустой кэш
func New() *Cache {
	return &Cache{m: make(map[string]json.RawMessage)} // инициализация map
}

// Get вытаскивает JSON по ключу
func (c *Cache) Get(id string) (json.RawMessage, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.m[id]
	return val, ok // возвращаем JSON
}

// Set кладёт JSON в кэш
func (c *Cache) Set(id string, data json.RawMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[id] = data
}

// LoadAll массово грузит данные
func (c *Cache) LoadAll(data map[string]json.RawMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, v := range data {
		c.m[k] = v
	}
}
