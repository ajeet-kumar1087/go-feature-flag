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

### Simple Usage (Single Import)

```go
package main

import (
    "context"
    "log"
    
    "github.com/ajeet-kumar1087/go-feature-flag"
)

func main() {
    // Create client with default configuration (in-memory storage, caching enabled)
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

    // Check if feature is enabled (primary usage pattern)
    if enabled, _ := client.IsEnabled(ctx, "new-checkout-flow"); enabled {
        // Use new checkout flow
        log.Println("✅ Using new checkout flow")
    } else {
        // Use legacy checkout flow  
        log.Println("⚪ Using legacy checkout flow")
    }
}
```

### Real-World Usage Examples

#### E-commerce Application

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    
    "github.com/ajeet-kumar1087/go-feature-flag"
)

func main() {
    // Production configuration with Redis
    config := featureflag.Config{
        Storage: featureflag.StorageConfig{
            Type: "redis",
            Redis: &featureflag.RedisConfig{
                Addr: "localhost:6379",
                DB:   0,
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

    // Set up feature flags for different features
    setupFeatureFlags(client)

    // Start HTTP server with feature-flagged endpoints
    http.HandleFunc("/checkout", checkoutHandler(client))
    http.HandleFunc("/product", productHandler(client))
    
    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func setupFeatureFlags(client featureflag.Client) {
    ctx := context.Background()
    
    flags := []featureflag.FeatureFlag{
        {
            Key:         "new-checkout-ui",
            Enabled:     true,
            Description: "Enable new checkout user interface",
            Metadata: map[string]string{
                "team":     "frontend",
                "rollout":  "100%",
                "version":  "v2.1",
            },
        },
        {
            Key:         "express-shipping",
            Enabled:     false,
            Description: "Enable express shipping option",
            Metadata: map[string]string{
                "team":     "logistics",
                "rollout":  "0%",
                "region":   "US",
            },
        },
        {
            Key:         "recommendation-engine",
            Enabled:     true,
            Description: "Enable AI-powered product recommendations",
            Metadata: map[string]string{
                "team":     "ml",
                "rollout":  "75%",
                "model":    "v3.2",
            },
        },
    }

    for _, flag := range flags {
        if err := client.SetFlag(ctx, flag); err != nil {
            log.Printf("Failed to set flag %s: %v", flag.Key, err)
        }
    }
}

func checkoutHandler(client featureflag.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        
        // Check if new checkout UI is enabled
        if enabled, _ := client.IsEnabled(ctx, "new-checkout-ui"); enabled {
            fmt.Fprintf(w, "🎨 New Checkout UI - Enhanced experience with better UX\n")
        } else {
            fmt.Fprintf(w, "📝 Legacy Checkout - Standard checkout process\n")
        }
        
        // Check if express shipping is available
        if enabled, _ := client.IsEnabled(ctx, "express-shipping"); enabled {
            fmt.Fprintf(w, "🚀 Express shipping available!\n")
        }
    }
}

func productHandler(client featureflag.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        
        fmt.Fprintf(w, "📦 Product Details\n")
        
        // Show recommendations if enabled
        if enabled, _ := client.IsEnabled(ctx, "recommendation-engine"); enabled {
            fmt.Fprintf(w, "🤖 AI Recommendations: Based on your browsing history...\n")
        }
    }
}
```

#### Microservice with Health Checks

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "net/http"
    "time"
    
    "github.com/ajeet-kumar1087/go-feature-flag"
)

type Service struct {
    client featureflag.Client
}

func main() {
    // PostgreSQL configuration for production
    config := featureflag.Config{
        Storage: featureflag.StorageConfig{
            Type: "postgres",
            Postgres: &featureflag.PostgresConfig{
                Host:     "localhost",
                Port:     5432,
                Database: "featureflags",
                Username: "postgres",
                Password: "password",
                SSLMode:  "require",
            },
        },
        Cache: featureflag.CacheConfig{
            Enabled: true,
            TTL:     featureflag.Duration(15 * time.Minute),
            MaxSize: 2000,
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
        DefaultFlags: []featureflag.FeatureFlag{
            {
                Key:         "maintenance-mode",
                Enabled:     false,
                Description: "Enable maintenance mode",
            },
            {
                Key:         "rate-limiting",
                Enabled:     true,
                Description: "Enable rate limiting",
            },
        },
    }

    client, err := featureflag.NewClient(config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    service := &Service{client: client}

    // API endpoints
    http.HandleFunc("/api/data", service.dataHandler)
    http.HandleFunc("/health", service.healthHandler)
    http.HandleFunc("/metrics", service.metricsHandler)
    
    log.Println("Microservice starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func (s *Service) dataHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // Check maintenance mode
    if enabled, _ := s.client.IsEnabled(ctx, "maintenance-mode"); enabled {
        http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
        return
    }
    
    // Check rate limiting
    if enabled, _ := s.client.IsEnabled(ctx, "rate-limiting"); enabled {
        // Implement rate limiting logic here
        log.Println("Rate limiting is active")
    }
    
    // Return data
    response := map[string]interface{}{
        "data":      "Your API data here",
        "timestamp": time.Now(),
        "version":   "v1.0",
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (s *Service) healthHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    health := map[string]interface{}{
        "status":    "healthy",
        "timestamp": time.Now(),
    }
    
    // Check feature flag service health
    if err := s.client.HealthCheck(ctx); err != nil {
        health["status"] = "unhealthy"
        health["feature_flags"] = "error: " + err.Error()
        w.WriteHeader(http.StatusServiceUnavailable)
    } else {
        health["feature_flags"] = "healthy"
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(health)
}

func (s *Service) metricsHandler(w http.ResponseWriter, r *http.Request) {
    metrics := s.client.GetMetrics()
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(metrics)
}
```

### Advanced Usage (Custom Configuration)

```go
package main

import (
    "time"
    "github.com/ajeet-kumar1087/go-feature-flag"
)

func main() {
    // Custom configuration with Redis and caching
    config := featureflag.Config{
        Storage: featureflag.StorageConfig{
            Type: "redis",
            Redis: &featureflag.RedisConfig{
                Addr: "localhost:6379",
                DB:   0,
            },
        },
        Cache: featureflag.CacheConfig{
            Enabled: true,
            TTL:     featureflag.Duration(10 * time.Minute),
            MaxSize: 1000,
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
    
    // Your application logic here...
}
```

### Modular Imports (For Advanced Users)

If you prefer explicit imports and smaller binaries:

```go
import (
    "github.com/ajeet-kumar1087/go-feature-flag/featureflag/client"
    "github.com/ajeet-kumar1087/go-feature-flag/featureflag/config"
    "github.com/ajeet-kumar1087/go-feature-flag/featureflag/core"
)

cfg := config.DefaultConfig()
client, err := client.NewClient(cfg)
```
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

## Complete Examples

### Development Setup (Memory Storage)

Perfect for local development and testing:

```go
package main

import (
    "context"
    "log"
    
    "github.com/ajeet-kumar1087/go-feature-flag"
)

func main() {
    // Quick setup with defaults
    client, err := featureflag.NewClientWithDefaults()
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()

    // Create some test flags
    testFlags := []featureflag.FeatureFlag{
        {
            Key:         "dark-mode",
            Enabled:     true,
            Description: "Enable dark mode theme",
        },
        {
            Key:         "beta-features",
            Enabled:     false,
            Description: "Enable beta features for testing",
        },
    }

    for _, flag := range testFlags {
        if err := client.SetFlag(ctx, flag); err != nil {
            log.Printf("Failed to set flag %s: %v", flag.Key, err)
        }
    }

    // Use flags in your application
    if enabled, _ := client.IsEnabled(ctx, "dark-mode"); enabled {
        log.Println("🌙 Dark mode is enabled")
    }

    if enabled, _ := client.IsEnabled(ctx, "beta-features"); enabled {
        log.Println("🧪 Beta features are enabled")
    } else {
        log.Println("🔒 Beta features are disabled")
    }
}
```

### Production Setup (Redis with Observability)

Production-ready configuration with Redis and full observability:

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/ajeet-kumar1087/go-feature-flag"
)

func main() {
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
        DefaultFlags: []featureflag.FeatureFlag{
            {
                Key:         "maintenance-mode",
                Enabled:     false,
                Description: "Global maintenance mode",
                Metadata: map[string]string{
                    "priority": "critical",
                    "team":     "platform",
                },
            },
        },
    }

    client, err := featureflag.NewClient(config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Your application logic here
    runApplication(client)
}

func runApplication(client featureflag.Client) {
    ctx := context.Background()
    
    // Simulate application runtime
    for i := 0; i < 10; i++ {
        // Check maintenance mode
        if enabled, _ := client.IsEnabled(ctx, "maintenance-mode"); enabled {
            log.Println("🚧 Application in maintenance mode")
            time.Sleep(5 * time.Second)
            continue
        }
        
        log.Printf("✅ Processing request %d", i+1)
        time.Sleep(1 * time.Second)
    }
    
    // Print metrics
    metrics := client.GetMetrics()
    log.Printf("📊 Metrics: %d flag checks, %.2f%% cache hit rate", 
        metrics.FlagChecks,
        float64(metrics.CacheHits)/float64(metrics.FlagChecks)*100)
}
```

### Enterprise Setup (PostgreSQL with Advanced Features)

Enterprise-grade setup with PostgreSQL and comprehensive configuration:

```go
package main

import (
    "context"
    "log"
    "os"
    "time"
    
    "github.com/ajeet-kumar1087/go-feature-flag"
)

func main() {
    config := featureflag.Config{
        Storage: featureflag.StorageConfig{
            Type: "postgres",
            Postgres: &featureflag.PostgresConfig{
                Host:     os.Getenv("DB_HOST"),
                Port:     5432,
                Database: os.Getenv("DB_NAME"),
                Username: os.Getenv("DB_USER"),
                Password: os.Getenv("DB_PASSWORD"),
                SSLMode:  "require",
            },
        },
        Cache: featureflag.CacheConfig{
            Enabled: true,
            TTL:     featureflag.Duration(15 * time.Minute),
            MaxSize: 10000,
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
        DefaultFlags: []featureflag.FeatureFlag{
            {
                Key:         "circuit-breaker",
                Enabled:     true,
                Description: "Enable circuit breaker pattern",
                Metadata: map[string]string{
                    "timeout":     "30s",
                    "threshold":   "5",
                    "reset_time":  "60s",
                },
            },
            {
                Key:         "feature-rollout-v2",
                Enabled:     false,
                Description: "Gradual rollout of version 2 features",
                Metadata: map[string]string{
                    "rollout_percentage": "0",
                    "target_groups":      "beta,internal",
                },
            },
        },
    }

    client, err := featureflag.NewClient(config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Demonstrate advanced usage
    demonstrateAdvancedFeatures(client)
}

func demonstrateAdvancedFeatures(client featureflag.Client) {
    ctx := context.Background()

    // 1. Feature flag with metadata usage
    flag, err := client.GetFlag(ctx, "circuit-breaker")
    if err != nil {
        log.Printf("Error getting flag: %v", err)
        return
    }

    if flag.Enabled {
        timeout := flag.Metadata["timeout"]
        threshold := flag.Metadata["threshold"]
        log.Printf("🔧 Circuit breaker enabled: timeout=%s, threshold=%s", timeout, threshold)
    }

    // 2. Gradual rollout simulation
    rolloutFlag, err := client.GetFlag(ctx, "feature-rollout-v2")
    if err == nil && rolloutFlag.Enabled {
        percentage := rolloutFlag.Metadata["rollout_percentage"]
        groups := rolloutFlag.Metadata["target_groups"]
        log.Printf("🎯 Feature rollout active: %s%% for groups: %s", percentage, groups)
    }

    // 3. Health monitoring
    if err := client.HealthCheck(ctx); err != nil {
        log.Printf("❌ Health check failed: %v", err)
    } else {
        log.Println("✅ System healthy")
    }

    // 4. Performance monitoring
    metrics := client.GetMetrics()
    log.Printf("📈 Performance: avg check duration: %v, cache hit rate: %.1f%%",
        metrics.AvgFlagCheckDuration,
        float64(metrics.CacheHits)/float64(metrics.FlagChecks)*100)
}
```

### Configuration from Environment Variables

Load configuration from environment variables for 12-factor app compliance:

```go
package main

import (
    "log"
    "os"
    
    "github.com/ajeet-kumar1087/go-feature-flag"
)

func main() {
    // Set environment variables (typically done by deployment system)
    os.Setenv("FEATUREFLAG_STORAGE_TYPE", "redis")
    os.Setenv("FEATUREFLAG_REDIS_ADDR", "localhost:6379")
    os.Setenv("FEATUREFLAG_CACHE_ENABLED", "true")
    os.Setenv("FEATUREFLAG_CACHE_TTL", "10m")
    os.Setenv("FEATUREFLAG_LOGGING_ENABLED", "true")
    os.Setenv("FEATUREFLAG_LOGGING_LEVEL", "info")

    // Load configuration from environment
    config := featureflag.LoadConfigFromEnv()
    
    client, err := featureflag.NewClient(config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    log.Println("✅ Client configured from environment variables")
}
```

### Configuration from File

Load configuration from JSON or YAML files:

```go
package main

import (
    "log"
    
    "github.com/ajeet-kumar1087/go-feature-flag"
)

func main() {
    // Load from YAML file
    config, err := featureflag.LoadConfigFromFile("config.yaml")
    if err != nil {
        log.Fatal(err)
    }

    client, err := featureflag.NewClient(config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    log.Println("✅ Client configured from YAML file")
}
```

Example `config.yaml`:

```yaml
storage:
  type: postgres
  postgres:
    host: localhost
    port: 5432
    database: featureflags
    username: postgres
    password: password
    ssl_mode: disable
cache:
  enabled: true
  ttl: 15m
  max_size: 2000
observability:
  logging:
    enabled: true
    level: info
  metrics:
    enabled: true
```

## 📚 Example Applications

The `examples/` directory contains complete, runnable examples:

### Basic Usage
```bash
cd examples/basic-usage && go run main.go
```
Demonstrates simple feature flag operations with in-memory storage.

### Production Redis Setup
```bash
# Start Redis first: docker run -d -p 6379:6379 redis:alpine
cd examples/redis-production && go run main.go
```
Shows production configuration with Redis, caching, and observability.

### Web Service Integration
```bash
cd examples/web-service && go run main.go
# Then visit: http://localhost:8080
```
Complete web service with feature-flagged endpoints and monitoring.

Each example is self-contained and demonstrates different aspects of the library.

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