package gormoize_test

import (
	"os"
	"testing"

	"github.com/presbrey/gormoize"
	"gorm.io/driver/sqlite"
)

func TestOpen(t *testing.T) {
	// Clean up test databases after tests
	defer func() {
		os.Remove("test1.db")
		os.Remove("test2.db")
	}()

	cache := gormoize.ByDSN(nil)

	tests := []struct {
		name string
		dsn  string
	}{
		{
			name: "same DSN returns same database connection",
			dsn:  "test1.db",
		},
		{
			name: "different DSN returns different database connection",
			dsn:  "test2.db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First call
			db1, err := cache.Open(sqlite.Open, tt.dsn)
			if err != nil {
				t.Errorf("failed to open database: %v", err)
			}
			if db1 == nil {
				t.Error("expected non-nil database connection")
			}

			// Test database connectivity
			err = db1.Raw("SELECT 1").Error
			if err != nil {
				t.Errorf("database connection not working: %v", err)
			}

			// Second call with same DSN
			db2, err := cache.Open(sqlite.Open, tt.dsn)
			if err != nil {
				t.Errorf("failed to open database: %v", err)
			}

			// Verify we got the same connection back
			if db1 != db2 {
				t.Error("expected same database connection for same DSN")
			}
		})
	}
}

func TestGet(t *testing.T) {
	// Clean up test database after tests
	defer os.Remove("test_get.db")

	cache := gormoize.ByDSN(nil)
	dsn := "test_get.db"

	// Test getting non-existent connection
	t.Run("get non-existent connection returns nil", func(t *testing.T) {
		db := cache.Get(dsn)
		if db != nil {
			t.Error("expected nil for non-existent connection")
		}
	})

	// Create a connection first using Open
	db1, err := cache.Open(sqlite.Open, dsn)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	t.Run("get existing connection returns same instance", func(t *testing.T) {
		db2 := cache.Get(dsn)
		if db2 == nil {
			t.Error("expected non-nil database connection")
		}
		if db1 != db2 {
			t.Error("expected same database connection for same DSN")
		}

		// Test database connectivity
		err := db2.Raw("SELECT 1").Error
		if err != nil {
			t.Errorf("database connection not working: %v", err)
		}
	})

	t.Run("multiple gets return same instance", func(t *testing.T) {
		db2 := cache.Get(dsn)
		db3 := cache.Get(dsn)
		if db2 != db3 {
			t.Error("expected same database connection for multiple gets")
		}
	})
}
