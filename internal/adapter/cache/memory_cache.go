package cache

import (
	"sync"

	"github.com/example/wb-order-service/internal/domain"
)

type MemoryOrderCache struct {
	mu    sync.RWMutex
	store map[string]domain.Order
}

func NewMemoryOrderCache() *MemoryOrderCache {
	return &MemoryOrderCache{store: make(map[string]domain.Order)}
}

func (c *MemoryOrderCache) Get(id string) (domain.Order, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	o, ok := c.store[id]
	return o, ok
}

func (c *MemoryOrderCache) Set(id string, o domain.Order) {
	c.mu.Lock()
	c.store[id] = o
	c.mu.Unlock()
}
