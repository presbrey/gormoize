package gormoize

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBaseCache(t *testing.T) {
	t.Run("cleanup removes old items", func(t *testing.T) {
		// Create base cache with short intervals for testing
		opts := Options{
			CleanupInterval: 100 * time.Millisecond,
			MaxAge:          2000 * time.Millisecond,
		}

		cache := newBaseCache(opts)
		defer cache.Stop() // Ensure cleanup goroutine is stopped

		// Add test items
		freshDB, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		staleDB, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})

		cache.cacheMutex.Lock()
		cache.dbCache["fresh"] = &dbCacheEntry{
			db:       freshDB,
			lastUsed: time.Now(),
		}
		cache.dbCache["stale"] = &dbCacheEntry{
			db:       staleDB,
			lastUsed: time.Now().Add(-10 * time.Second),
		}
		cache.cacheMutex.Unlock()

		// No need to wait as long since cleanup runs immediately now
		time.Sleep(500 * time.Millisecond) // Small wait to ensure cleanup completes

		cache.cacheMutex.RLock()
		_, freshExists := cache.dbCache["fresh"]
		_, staleExists := cache.dbCache["stale"]
		cache.cacheMutex.RUnlock()

		if !freshExists {
			t.Error("fresh item should not be cleaned up")
		}
		if staleExists {
			t.Error("stale item should be cleaned up")
		}
	})

}
