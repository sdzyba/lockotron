package lockotron

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrNotFound = errors.New("cached value not found")
)

type fallbackFunc func(string) (interface{}, error)

type Cache struct {
	locker   *locker
	mutex    sync.RWMutex
	items    map[string]*item
	stopChan chan bool
	ticker   *time.Ticker
	config   *Config
}

func NewCache(config *Config) *Cache {
	c := &Cache{
		locker: newLocker(),
		items:  make(map[string]*item),
		config: config,
	}

	if config.CleanupInterval != NoCleaner {
		c.ticker = time.NewTicker(config.CleanupInterval)

		go func() {
			for {
				select {
				case <-c.ticker.C:
					c.DeleteExpired()
				case <-c.stopChan:
					c.ticker.Stop()

					return
				}
			}
		}()
	}

	return c
}

func (c *Cache) Close() error {
	if c.stopChan == nil || c.ticker == nil {
		return nil
	}

	close(c.stopChan)

	return nil
}

func (c *Cache) Set(key string, value interface{}) {
	c.set(key, c.config.DefaultTTL, value)
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

func (c *Cache) Fetch(key string, fallback fallbackFunc) (interface{}, error) {
	return c.fetch(key, c.config.DefaultTTL, fallback)
}

func (c *Cache) FetchEx(key string, ttl time.Duration, fallback fallbackFunc) (interface{}, error) {
	return c.fetch(key, ttl, fallback)
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

func (c *Cache) fetch(key string, ttl time.Duration, fallback fallbackFunc) (interface{}, error) {
	value, err := c.Get(key)
	if err == nil {
		return value, nil
	}

	mutex := c.locker.obtain(key)
	mutex.Lock()
	defer mutex.Unlock()
	defer c.locker.release(key)

	value, err = c.Get(key)
	if err == nil {
		return value, nil
	}

	value, err = fallback(key)
	if err != nil {
		return nil, err
	}

	return value, nil
}

func (c *Cache) set(key string, ttl time.Duration, value interface{}) {
	c.mutex.Lock()
	c.items[key] = newItem(value, ttl)
	c.mutex.Unlock()
}
