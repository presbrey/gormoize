package gormoize

import (
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

// DBCache provides thread-safe caching of database connections
type dsnCache struct {
	*baseCache
}

// ByDSN creates a new DSNCache instance with the given options.
// If opts is nil, default options are used.
func ByDSN(opts *Options) *dsnCache {
	options := DefaultOptions()
	if opts != nil {
		options = *opts
	}

	cache := &dsnCache{}
	cache.baseCache = newBaseCache(options)
	return cache
}

// Get returns a cached gorm.DB instance for the given DSN if it exists.
// If no instance exists for the DSN, returns nil.
// This method is safe for concurrent use.
func (c *dsnCache) Get(dsn string) *gorm.DB {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	if entry, exists := c.dbCache[dsn]; exists {
		entry.lastUsed = time.Now()
		return entry.db
	}
	return nil
}

// Open returns a gorm.Dialector for the given DSN. If a dialector for this DSN
// has already been created, the cached instance is returned. This method is
// safe for concurrent use.
func (c *dsnCache) Open(fn func(dsn string) gorm.Dialector, dsn string, opts ...gorm.Option) (*gorm.DB, error) {
	// Try to get from cache first
	if db := c.Get(dsn); db != nil {
		return db, nil
	}

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
