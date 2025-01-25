package gormoize_test

import (
	"os"
	"testing"

	"github.com/presbrey/gormoize"
	"gorm.io/driver/sqlite"
)

func TestOpenDSN(t *testing.T) {
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
