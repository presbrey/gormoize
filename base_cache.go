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
	cacheMutex sync.RWMutex
	dbCache    map[string]*dbCacheEntry

	cleanupInterval time.Duration
	maxAge          time.Duration
	mockDB          *gorm.DB
	stopCleanup     chan struct{}
}

// newBaseCache creates a new baseCache instance with the given options
func newBaseCache(opts Options) *baseCache {
	cache := &baseCache{
		cacheMutex: sync.RWMutex{},
		dbCache:    make(map[string]*dbCacheEntry),

		cleanupInterval: opts.CleanupInterval,
		maxAge:          opts.MaxAge,
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
	entry, exists := c.dbCache[key]
	if !exists {
		return
	}

	// remove the specified item from the cache
	delete(c.dbCache, key)

	// close the database connection
	sqlDB, err := entry.db.DB()
	if err == nil {
		sqlDB.Close()
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

// mockDB returns the mock DB used for testing
func (c *baseCache) SetMockDB(db *gorm.DB) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	c.mockDB = db
}

// Stop stops the cleanup routine and closes all database connections
func (c *baseCache) Stop() {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	// Clean up all database connections
	for key := range c.dbCache {
		c.cleanupItem(key)
	}

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
