package loki

import (
	"time"
)

type cleaner struct {
	interval time.Duration
	stopChan chan bool
	cache    *Cache
	ticker   *time.Ticker
}

func newCleaner(cache *Cache, interval time.Duration) *cleaner {
	return &cleaner{
		stopChan: make(chan bool),
		cache:    cache,
		interval: interval,
	}
}

func (c *cleaner) Start() {
	c.ticker = time.NewTicker(c.interval)

	for {
		select {
		case <-c.ticker.C:
			c.cache.DeleteExpired()
		case <-c.stopChan:
			c.ticker.Stop()

			return
		}
	}
}

func (c *cleaner) Stop() {
	close(c.stopChan)
}
