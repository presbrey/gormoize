package gormoize

import (
	"sync"
	"time"

	"gorm.io/gorm"
)

// Options configures the behavior of DBCache
type Options struct {
	// CleanupInterval specifies how often to check for stale connections.
	// Only used when MaxAge > 0.
	CleanupInterval time.Duration

	// MaxAge specifies how long a connection can remain unused before being removed.
	// If 0, cleanup is disabled.
	MaxAge time.Duration
}

// DefaultOptions returns the default configuration options
func DefaultOptions() Options {
	return Options{
		CleanupInterval: 5 * time.Minute,
		MaxAge:          30 * time.Minute,
	}
}

type dbCacheEntry struct {
	db       *gorm.DB
	lastUsed time.Time
}

// ByDSN creates a new DSNCache instance with the given options.
// If opts is nil, default options are used.
func ByDSN(opts *Options) *dsnCache {
	options := DefaultOptions()
	if opts != nil {
		options = *opts
	}

	cache := &dsnCache{
		dbCache:         make(map[string]*dbCacheEntry),
		cleanupInterval: options.CleanupInterval,
		maxAge:          options.MaxAge,
	}
	if options.MaxAge > 0 {
		go cache.startCleanup()
	}
	return cache
}

// DBCache provides thread-safe caching of database connections
type dsnCache struct {
	dbCache         map[string]*dbCacheEntry
	cacheMutex      sync.RWMutex
	cleanupInterval time.Duration
	maxAge          time.Duration
}

// OpenDSN returns a gorm.Dialector for the given DSN. If a dialector for this DSN
// has already been created, the cached instance is returned. This method is
// safe for concurrent use.
func (c *dsnCache) Open(fn func(dsn string) gorm.Dialector, dsn string, opts ...gorm.Option) (*gorm.DB, error) {
	// Try to get from cache first
	c.cacheMutex.RLock()
	if entry, exists := c.dbCache[dsn]; exists {
		entry.lastUsed = time.Now() // Update last used time
		c.cacheMutex.RUnlock()
		return entry.db, nil
	}
	c.cacheMutex.RUnlock()

	// Not in cache, create new with write lock
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	// Double-check in case another goroutine created it
	if entry, exists := c.dbCache[dsn]; exists {
		entry.lastUsed = time.Now()
		return entry.db, nil
	}

	// Create new dialector and cache it
	db, err := gorm.Open(fn(dsn), opts...)
	if err != nil {
		return nil, err
	}

	c.dbCache[dsn] = &dbCacheEntry{
		db:       db,
		lastUsed: time.Now(),
	}
	return db, nil
}

// startCleanup starts the cleanup routine that removes old connections
func (c *dsnCache) startCleanup() {
	if c.maxAge <= 0 {
		return
	}
	ticker := time.NewTicker(c.cleanupInterval)
	for range ticker.C {
		c.cleanup()
	}
}

// cleanup removes connections that haven't been used for longer than maxAge
func (c *dsnCache) cleanup() {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	now := time.Now()
	for dsn, entry := range c.dbCache {
		if now.Sub(entry.lastUsed) > c.maxAge {
			sqlDB, err := entry.db.DB()
			if err == nil {
				sqlDB.Close() // Close the underlying connection
			}
			delete(c.dbCache, dsn)
		}
	}
}
