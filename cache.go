package imp

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrNotFound = errors.New("cached value not found")
)

type Cache struct {
	mutex      sync.RWMutex
	items      map[string]*item
	defaultTTL time.Duration
}

func NewCache(config *Config) *Cache {
	c := &Cache{
		items:      make(map[string]*item),
		defaultTTL: config.DefaultTTL,
	}

	if config.CleanupInterval != NoCleaner {
		cleaner := newCleaner(c, config.CleanupInterval)
		go cleaner.Start()
	}

	return c
}

func (c *Cache) Set(key string, value interface{}) {
	c.set(key, c.defaultTTL, value)
}

func (c *Cache) SetEx(key string, ttl time.Duration, value interface{}) {
	c.set(key, ttl, value)
}

func (c *Cache) Get(key string) (interface{}, error) {
	c.mutex.RLock()
	item, ok := c.items[key]
	c.mutex.RUnlock()
	if ok {
		return item.value, nil
	}

	return nil, ErrNotFound
}

func (c *Cache) Delete(key string) {
	c.mutex.Lock()
	delete(c.items, key)
	c.mutex.Unlock()
}

func (c *Cache) Fetch(key string, fallbackFunc func(string) (interface{}, error)) (interface{}, error) {
	return c.fetch(key, c.defaultTTL, fallbackFunc)
}

func (c *Cache) FetchEx(key string, ttl time.Duration, fallbackFunc func(string) (interface{}, error)) (interface{}, error) {
	return c.fetch(key, ttl, fallbackFunc)
}

func (c *Cache) DeleteAll() {
	c.mutex.Lock()
	c.items = make(map[string]*item)
	c.mutex.Unlock()
}

func (c *Cache) DeleteExpired() {
	now := time.Now().UnixNano()

	c.mutex.Lock()
	for key, item := range c.items {
		if item.isExpirable() && now > item.ttl {
			delete(c.items, key)
		}
	}
	c.mutex.Unlock()
}

func (c *Cache) fetch(key string, ttl time.Duration, fallbackFunc func(string) (interface{}, error)) (interface{}, error) {
	c.mutex.Lock()
	item, ok := c.items[key]
	if ok {
		c.mutex.Unlock()

		return item.value, nil
	}

	value, err := fallbackFunc(key)
	if err != nil {
		c.mutex.Unlock()

		return nil, err
	}

	c.items[key] = newItem(value, ttl)
	c.mutex.Unlock()

	return value, nil
}

func (c *Cache) set(key string, ttl time.Duration, value interface{}) {
	c.mutex.Lock()
	c.items[key] = newItem(value, ttl)
	c.mutex.Unlock()
}
