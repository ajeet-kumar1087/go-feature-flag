# Go Feature Flag Library

A high-performance, flexible feature flag library for Go applications with support for multiple storage backends, caching, and comprehensive observability.

## Features

- **Multiple Storage Backends**: In-memory, Redis, and PostgreSQL support
- **High Performance Caching**: Optional LRU cache with configurable TTL and size limits
- **Graceful Degradation**: Non-existent flags return `false` instead of errors
- **Comprehensive Observability**: Structured logging and metrics collection
- **Thread-Safe**: Concurrent access support with optimized hot paths
- **Flexible Configuration**: Environment variables, JSON/YAML files, or programmatic setup
- **Production Ready**: Extensive testing, benchmarks, and error handling

## Installation

```bash
go get github.com/ajeet-kumar1087/go-feature-flag
```

## Quick Start

### Basic Usage (In-Memory)

```go
package main

import (
    "context"
    "log"
    
    "github.com/ajeet-kumar1087/go-feature-flag/featureflag"
)

func main() {
    // Create client with default configuration (in-memory storage)
    client, err := featureflag.NewClientWithDefaults()
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()

    // Create a feature flag
    flag := featureflag.FeatureFlag{
        Key:         "new-checkout-flow",
        Enabled:     true,
        Description: "Enable the new checkout flow",
        Metadata: map[string]string{
            "team": "payments",
            "rollout": "100%",
        },
    }

    if err := client.SetFlag(ctx, flag); err != nil {
        log.Fatal(err)
    }

    // Check if feature is enabled
    if enabled, _ := client.IsEnabled(ctx, "new-checkout-flow"); enabled {
        // Use new checkout flow
        log.Println("Using new checkout flow")
    } else {
        // Use legacy checkout flow
        log.Println("Using legacy checkout flow")
    }
}
```

### Production Configuration (Redis)

```go
config := featureflag.Config{
    Storage: featureflag.StorageConfig{
        Type: "redis",
        Redis: &featureflag.RedisConfig{
            Addr:     "redis-cluster.example.com:6379",
            Password: "your-redis-password",
            DB:       0,
        },
    },
    Cache: featureflag.CacheConfig{
        Enabled: true,
        TTL:     featureflag.Duration(10 * time.Minute),
        MaxSize: 5000,
    },
    Observability: featureflag.ObservabilityConfig{
        Logging: featureflag.LoggingConfig{
            Enabled: true,
            Level:   "info",
        },
        Metrics: featureflag.MetricsConfig{
            Enabled: true,
        },
    },
}

client, err := featureflag.NewClient(config)
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

## Storage Backends

### In-Memory Storage
Perfect for development, testing, and applications that don't need persistence:

```go
config := featureflag.Config{
    Storage: featureflag.StorageConfig{
        Type: "memory",
    },
}
```

### Redis Storage
Ideal for distributed applications and high-performance scenarios:

```go
config := featureflag.Config{
    Storage: featureflag.StorageConfig{
        Type: "redis",
        Redis: &featureflag.RedisConfig{
            Addr:     "localhost:6379",
            Password: "", // Optional
            DB:       0,  // Redis database number
        },
    },
}
```

### PostgreSQL Storage
Best for applications requiring ACID compliance and complex queries:

```go
config := featureflag.Config{
    Storage: featureflag.StorageConfig{
        Type: "postgres",
        Postgres: &featureflag.PostgresConfig{
            Host:     "localhost",
            Port:     5432,
            Database: "featureflags",
            Username: "postgres",
            Password: "password",
            SSLMode:  "require", // Use "disable" for development
        },
    },
}
```

## Configuration Options

### Environment Variables

Set configuration using environment variables:

```bash
export FEATUREFLAG_STORAGE_TYPE=redis
export FEATUREFLAG_REDIS_ADDR=localhost:6379
export FEATUREFLAG_CACHE_ENABLED=true
export FEATUREFLAG_CACHE_TTL=10m
export FEATUREFLAG_LOGGING_ENABLED=true
export FEATUREFLAG_LOGGING_LEVEL=info
```

```go
config := featureflag.LoadConfigFromEnv()
client, err := featureflag.NewClient(config)
```

### Configuration Files

#### JSON Configuration

```json
{
  "storage": {
    "type": "redis",
    "redis": {
      "addr": "localhost:6379",
      "db": 0
    }
  },
  "cache": {
    "enabled": true,
    "ttl": "5m",
    "max_size": 1000
  },
  "observability": {
    "logging": {
      "enabled": true,
      "level": "info"
    },
    "metrics": {
      "enabled": true
    }
  }
}
```

#### YAML Configuration

```yaml
storage:
  type: postgres
  postgres:
    host: localhost
    port: 5432
    database: featureflags
    username: postgres
    ssl_mode: disable
cache:
  enabled: true
  ttl: 15m
  max_size: 2000
```

Load from file:

```go
config, err := featureflag.LoadConfigFromFile("config.yaml")
if err != nil {
    log.Fatal(err)
}

client, err := featureflag.NewClient(config)
```

## Advanced Features

### Caching

The library includes an optional LRU cache that can significantly improve performance:

```go
config := featureflag.Config{
    Cache: featureflag.CacheConfig{
        Enabled: true,
        TTL:     featureflag.Duration(10 * time.Minute), // Cache entries expire after 10 minutes
        MaxSize: 5000,                                   // Maximum 5000 entries in cache
    },
}
```

### Default Flags

Automatically load flags when the client starts:

```go
config := featureflag.Config{
    DefaultFlags: []featureflag.FeatureFlag{
        {
            Key:         "maintenance-mode",
            Enabled:     false,
            Description: "Enable maintenance mode",
        },
        {
            Key:         "new-feature-gate",
            Enabled:     true,
            Description: "Master gate for new features",
        },
    },
}
```

### Observability

Enable logging and metrics for production monitoring:

```go
config := featureflag.Config{
    Observability: featureflag.ObservabilityConfig{
        Logging: featureflag.LoggingConfig{
            Enabled: true,
            Level:   "info", // debug, info, warn, error
        },
        Metrics: featureflag.MetricsConfig{
            Enabled: true,
        },
    },
}

// Get metrics
metrics := client.GetMetrics()
fmt.Printf("Cache hit rate: %.2f%%\n", 
    float64(metrics.CacheHits)/float64(metrics.FlagChecks)*100)
```

## API Reference

### Client Interface

```go
type Client interface {
    // Check if a feature flag is enabled (primary method)
    IsEnabled(ctx context.Context, key string) (bool, error)
    
    // Get complete flag information
    GetFlag(ctx context.Context, key string) (*FeatureFlag, error)
    
    // Create or update a flag
    SetFlag(ctx context.Context, flag FeatureFlag) error
    
    // Delete a flag
    DeleteFlag(ctx context.Context, key string) error
    
    // Get all flags
    GetAllFlags(ctx context.Context) ([]FeatureFlag, error)
    
    // Health check
    HealthCheck(ctx context.Context) error
    
    // Get metrics (if enabled)
    GetMetrics() MetricsSnapshot
    
    // Clean shutdown
    Close() error
}
```

### FeatureFlag Structure

```go
type FeatureFlag struct {
    Key         string            `json:"key"`
    Enabled     bool              `json:"enabled"`
    Description string            `json:"description,omitempty"`
    CreatedAt   time.Time         `json:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at"`
    Metadata    map[string]string `json:"metadata,omitempty"`
}
```

## Examples

See the `examples/` directory for comprehensive examples:

- [`basic_usage.go`](examples/basic_usage.go) - Simple feature flag usage
- [`memory_storage.go`](examples/memory_storage.go) - In-memory storage example
- [`redis_storage.go`](examples/redis_storage.go) - Redis storage example
- [`postgres_example.go`](examples/postgres_example.go) - PostgreSQL storage example
- [`advanced_caching.go`](examples/advanced_caching.go) - Caching performance examples
- [`advanced_configuration.go`](examples/advanced_configuration.go) - Configuration examples

## Performance

The library is optimized for high-performance scenarios:

- **Hot Path Optimization**: `IsEnabled()` is optimized for maximum throughput
- **Concurrent Access**: Thread-safe with minimal locking overhead
- **Caching**: Optional LRU cache with configurable TTL and size limits
- **Graceful Degradation**: Non-existent flags return `false` without errors

Typical performance (with caching enabled):
- 1M+ flag checks per second on modern hardware
- Sub-microsecond latency for cached flags
- Minimal memory allocation in hot paths

## Requirements

- **Go**: 1.19 or higher
- **Redis**: 6.0 or higher (if using Redis storage)
- **PostgreSQL**: 12 or higher (if using PostgreSQL storage)

## Database Setup

### PostgreSQL

Run the schema creation script:

```bash
psql -d your_database -f db/postgres_schema.sql
```

Or create the table manually:

```sql
CREATE TABLE IF NOT EXISTS feature_flags (
    key VARCHAR(255) PRIMARY KEY,
    enabled BOOLEAN NOT NULL DEFAULT false,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB
);

CREATE INDEX IF NOT EXISTS idx_feature_flags_enabled ON feature_flags(enabled);
CREATE INDEX IF NOT EXISTS idx_feature_flags_updated_at ON feature_flags(updated_at);
```

## Testing

Run the test suite:

```bash
# Unit tests
go test ./featureflag

# Integration tests (requires Redis and PostgreSQL)
go test ./featureflag -tags=integration

# Benchmarks
go test ./featureflag -bench=.

# Coverage
go test ./featureflag -cover
```

## Migration Guide

### From HTTP Service Approach

If you're migrating from an HTTP-based feature flag service:

1. **Replace HTTP calls with direct client calls**:
   ```go
   // Old HTTP approach
   resp, err := http.Get("http://feature-service/flags/my-feature")
   
   // New client approach
   enabled, err := client.IsEnabled(ctx, "my-feature")
   ```

2. **Update configuration**:
   - Replace HTTP endpoints with storage configuration
   - Configure caching for better performance than HTTP calls
   - Enable observability for monitoring

3. **Handle errors differently**:
   - HTTP errors become storage/validation errors
   - Network timeouts become context cancellation
   - 404 responses become graceful `false` returns

4. **Performance improvements**:
   - Eliminate network latency
   - Add caching for frequently accessed flags
   - Reduce serialization overhead

## Best Practices

### Flag Naming
- Use descriptive, lowercase names with hyphens: `new-checkout-flow`
- Include context when helpful: `mobile-app-dark-mode`
- Avoid abbreviations: `authentication-enabled` not `auth-on`

### Error Handling
```go
// IsEnabled provides graceful degradation
enabled, err := client.IsEnabled(ctx, "my-feature")
if err != nil {
    // Log error but continue with default behavior
    log.Printf("Flag check failed: %v", err)
    enabled = false // Safe default
}

if enabled {
    // New feature code
} else {
    // Legacy/default code
}
```

### Production Configuration
- Use persistent storage (Redis or PostgreSQL)
- Enable caching with appropriate TTL
- Enable observability (logging and metrics)
- Set up health checks
- Use environment variables for sensitive configuration

### Performance Optimization
- Use `IsEnabled()` for hot paths (optimized for speed)
- Use `GetFlag()` only when you need metadata
- Configure appropriate cache settings for your workload
- Monitor cache hit rates and adjust TTL accordingly

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests for your changes
4. Ensure all tests pass (`go test ./...`)
5. Run linting (`golangci-lint run`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.