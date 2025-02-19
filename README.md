[![Go Report Card](https://goreportcard.com/badge/github.com/presbrey/gormoize)](https://goreportcard.com/report/github.com/presbrey/gormoize)
[![codecov](https://codecov.io/gh/presbrey/gormoize/graph/badge.svg?token=DOVXA9MJAP)](https://codecov.io/gh/presbrey/gormoize)
[![Go](https://github.com/presbrey/gormoize/actions/workflows/go.yml/badge.svg)](https://github.com/presbrey/gormoize/actions/workflows/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/presbrey/gormoize.svg)](https://pkg.go.dev/github.com/presbrey/gormoize)

# gormoize

gormoize is a Go package that provides thread-safe caching for GORM database connections. It helps manage and reuse database connections efficiently while preventing connection leaks in concurrent applications.

## Features

- Thread-safe database connection caching
- Configurable connection lifetime management
- Automatic cleanup of stale connections
- Compatible with GORM v1.25+
- Zero external dependencies beyond GORM

## Usage

### Basic Usage

```go
package main

import (
    "github.com/presbrey/gormoize"
    "gorm.io/driver/sqlite"
)

func main() {
    // Create a new cache with default options
    cache := gormoize.ByDSN(nil)
    
    // Get a database connection
    db, err := cache.Open(sqlite.Open, "test.db")
    if err != nil {
        panic(err)
    }
    
    // Use the db connection with GORM as normal
    // The connection will be reused for subsequent calls with the same DSN
}
```

### Custom Configuration

```go
package main

import (
    "time"
    "github.com/presbrey/gormoize"
    "gorm.io/driver/sqlite"
)

func main() {
    // Configure custom connection management
    opts := &gormoize.Options{
        CleanupInterval: 1 * time.Minute,  // Check for stale connections every minute
        MaxAge: 10 * time.Minute,         // Remove connections unused for 10 minutes
    }
    
    cache := gormoize.ByDSN(opts)
    db, err := cache.Open(sqlite.Open, "test.db")
    if err != nil {
        panic(err)
    }
    
    // The connection will be automatically cleaned up if unused
}
```

## Thread Safety

gormoize is designed for concurrent use. All operations on the connection cache are protected by appropriate locks, ensuring that:

- Multiple goroutines can safely request connections simultaneously
- Connection cleanup doesn't interfere with active usage
- Each DSN maintains exactly one database connection

## Connection Lifecycle

- Connections are created on first request for a DSN
- Subsequent requests for the same DSN return the cached connection
- Unused connections are automatically closed and removed based on MaxAge
- If MaxAge is 0, connections remain cached indefinitely
- The cleanup routine only runs when MaxAge > 0

## Default Settings

- CleanupInterval: 5 minutes
- MaxAge: 30 minutes

These can be modified using the Options struct when creating a new cache.

## Testing with Mock Databases

gormoize provides support for mock databases in testing scenarios. You can set a mock database for your tests using the `SetMockDB` method:

```go
package main

import (
    "testing"
    "github.com/presbrey/gormoize"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

func TestMyFunction(t *testing.T) {
    // Create a new cache
    cache := gormoize.ByDSN(nil)
    
    // Create a mock database (using SQLite in-memory for this example)
    mockDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    if err != nil {
        t.Fatalf("failed to create mock DB: %v", err)
    }
    
    // Set the mock database
    cache.SetMockDB(mockDB)
    
    // Now any calls to cache.Open() will return the mock database
    // regardless of the provided DSN
    db, err := cache.Open(sqlite.Open, "any_dsn")
    if err != nil {
        t.Fatal(err)
    }
    
    // Use the mock database in your tests
    // ...
}
```

The mock database feature is particularly useful for:
- Unit testing without actual database connections
- Testing error scenarios
- Ensuring consistent test behavior
- Speeding up test execution

Note that the mock database will be used for all subsequent `Open` calls until it is cleared or the cache is stopped.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License

Copyright (c) 2025 Joe Presbrey
