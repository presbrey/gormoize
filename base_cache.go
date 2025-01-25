package gormoize

import (
	"log"
	"sync"
	"time"

	"gorm.io/gorm"
)

type dbCacheEntry struct {
	db       *gorm.DB
	lastUsed time.Time
}

// baseCache provides common caching functionality with cleanup support
type baseCache struct {
	cacheMutex      sync.RWMutex
	cleanupInterval time.Duration
	maxAge          time.Duration
	dbCache         map[string]*dbCacheEntry
	stopCleanup     chan struct{}
}

// newBaseCache creates a new baseCache instance with the given options
func newBaseCache(opts Options) *baseCache {
	cache := &baseCache{
		cacheMutex:      sync.RWMutex{},
		cleanupInterval: opts.CleanupInterval,
		maxAge:          opts.MaxAge,
		dbCache:         make(map[string]*dbCacheEntry),
		stopCleanup:     make(chan struct{}),
	}
	if opts.MaxAge > 0 {
		go cache.startCleanup()
	}
	return cache
}

// lastUsed returns a map of key to last used time for all cached items
func (c *baseCache) lastUsed() map[string]time.Time {
	lastUsed := make(map[string]time.Time)
	for key, entry := range c.dbCache {
		lastUsed[key] = entry.lastUsed
	}
	return lastUsed
}

// cleanupItem removes the specified item from the cache and performs any necessary cleanup
func (c *baseCache) cleanupItem(key string) {
	if entry, exists := c.dbCache[key]; exists {
		sqlDB, err := entry.db.DB()
		if err == nil {
			sqlDB.Close()
		}
		delete(c.dbCache, key)
	}
}

// startCleanup starts the cleanup routine that removes old items
func (c *baseCache) startCleanup() {
	if c.maxAge <= 0 {
		return
	}

	// Run cleanup immediately once
	c.cleanup()

	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCleanup:
			return
		}
	}
}

// Stop stops the cleanup routine
func (c *baseCache) Stop() {
	if c.maxAge > 0 {
		close(c.stopCleanup)
	}
}

// cleanup removes items that haven't been used for longer than maxAge
func (c *baseCache) cleanup() {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	now := time.Now()
	for key, lastUsed := range c.lastUsed() {
		log.Printf("key: %s, lastUsed: %v, now: %v, maxAge: %v", key, lastUsed, now, c.maxAge)
		if now.Sub(lastUsed) > c.maxAge {
			c.cleanupItem(key)
		}
	}
}
