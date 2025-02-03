package gormoize

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

		assert.True(t, freshExists, "fresh item should not be cleaned up")
		assert.False(t, staleExists, "stale item should be cleaned up")
	})

	t.Run("SetMockDB sets mock database", func(t *testing.T) {
		cache := ByDSN(&Options{})
		mockDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		if err != nil {
			t.Fatalf("failed to create mock DB: %v", err)
		}

		cache.SetMockDB(mockDB)
		db := cache.Get("any")
		assert.Equal(t, mockDB, db, "mockDB was not set correctly")

		// Verify that mockDB is set correctly
		cache.cacheMutex.RLock()
		defer cache.cacheMutex.RUnlock()

		assert.Equal(t, mockDB, cache.mockDB, "mockDB was not set correctly")

		mockDB, err = cache.Open(sqlite.Open, "any")
		assert.NoError(t, err)
		assert.Equal(t, mockDB, cache.mockDB)
	})

	t.Run("Set adds entry to cache", func(t *testing.T) {
		// Create a new baseCache instance using default options
		opts := Options{
			CleanupInterval: time.Hour,
			MaxAge:          time.Hour,
		}
		cache := newBaseCache(opts)

		// Create a fake *gorm.DB instance. In a real test, you might use a mock or a real DB connection.
		fakeDB := &gorm.DB{}
		key := "test-key"

		// Call the Set method to add the fakeDB to the cache
		cache.Set(key, fakeDB)

		// Verify that the entry exists in the cache
		cache.cacheMutex.RLock()
		entry, exists := cache.dbCache[key]
		cache.cacheMutex.RUnlock()
		assert.True(t, exists, "expected key %q to be present in the cache", key)

		// Check that the stored db pointer is correct
		assert.Equal(t, fakeDB, entry.db, "expected stored db pointer to match")

		// Ensure that lastUsed is set to a recent time (within the last 2 seconds)
		assert.WithinDuration(t, time.Now(), entry.lastUsed, 2*time.Second, "expected entry.lastUsed to be recent")
	})
}
