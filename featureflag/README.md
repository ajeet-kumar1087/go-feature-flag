# Go Feature Flag Library - Architecture & Developer Guide

This directory contains the modular implementation of the Go Feature Flag library, designed for high performance, scalability, and maintainability.

## 🏗️ Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        Application Layer                         │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │   Web Service   │  │   CLI Tool      │  │  Background Job │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
└─────────────────────────────┬───────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                     Feature Flag Client                         │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                    Client Interface                         │ │
│  │  • IsEnabled()  • GetFlag()  • SetFlag()  • DeleteFlag()   │ │
│  │  • GetAllFlags()  • HealthCheck()  • GetMetrics()          │ │
│  └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────┬───────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                      Caching Layer (Optional)                   │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                    LRU Cache                                │ │
│  │  • TTL-based expiration  • Size-based eviction             │ │
│  │  • Thread-safe operations  • Cache hit/miss metrics        │ │
│  └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────┬───────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                       Storage Layer                             │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │  Memory Store   │  │   Redis Store   │  │ PostgreSQL Store│ │
│  │                 │  │                 │  │                 │ │
│  │ • Development   │  │ • Production    │  │ • Enterprise    │ │
│  │ • Testing       │  │ • Distributed   │  │ • ACID          │ │
│  │ • Single node   │  │ • High perf     │  │ • Complex query │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
└─────────────────────────────┬───────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                    Observability Layer                          │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │  Logging                    │  Metrics Collection           │ │
│  │  • Structured logging       │  • Operation counters         │ │
│  │  • Configurable levels      │  • Performance metrics        │ │
│  │  • Context propagation      │  • Cache statistics           │ │
│  │  • Error tracking           │  • Error classification       │ │
│  └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## 🔧 Component Architecture

### Core Components

#### 1. Client Layer (`client/`)
The main entry point for all feature flag operations.

```go
// High-level client interface
type Client interface {
    IsEnabled(ctx context.Context, key string) (bool, error)    // Hot path - optimized
    GetFlag(ctx context.Context, key string) (*FeatureFlag, error)
    SetFlag(ctx context.Context, flag FeatureFlag) error
    DeleteFlag(ctx context.Context, key string) error
    GetAllFlags(ctx context.Context) ([]FeatureFlag, error)
    HealthCheck(ctx context.Context) error
    GetMetrics() MetricsSnapshot
    Close() error
}
```

**Key Features:**
- **Hot Path Optimization**: `IsEnabled()` is optimized for maximum throughput
- **Graceful Degradation**: Returns `false` for non-existent flags instead of errors
- **Context Support**: All operations respect context cancellation and timeouts
- **Thread Safety**: Concurrent access with minimal locking overhead

#### 2. Storage Layer (`storage/`)
Pluggable storage backends for different deployment scenarios.

```go
// Storage interface implemented by all backends
type Store interface {
    Get(ctx context.Context, key string) (*FeatureFlag, error)
    Set(ctx context.Context, flag FeatureFlag) error
    Delete(ctx context.Context, key string) error
    GetAll(ctx context.Context) ([]FeatureFlag, error)
    HealthCheck(ctx context.Context) error
    Close() error
}
```

**Available Implementations:**

| Backend | Use Case | Pros | Cons |
|---------|----------|------|------|
| **Memory** | Development, Testing | Fast, No dependencies | Not persistent, Single node |
| **Redis** | Production, Distributed | High performance, Persistence | External dependency |
| **PostgreSQL** | Enterprise | ACID, Complex queries | Heavier, More complex |

#### 3. Caching Layer (`cache/`)
Optional high-performance caching with LRU eviction and TTL expiration.

```go
// Cache configuration
type CacheConfig struct {
    Enabled bool          // Enable/disable caching
    TTL     Duration      // Time-to-live for cache entries
    MaxSize int           // Maximum number of cached items
}
```

**Performance Benefits:**
- **Sub-microsecond latency** for cached flags
- **Reduced storage load** by caching frequently accessed flags
- **Configurable TTL** to balance freshness vs performance
- **LRU eviction** to manage memory usage

#### 4. Configuration System (`config/`)
Flexible configuration loading from multiple sources.

```go
// Configuration precedence (highest to lowest):
// 1. Environment variables
// 2. Configuration files (JSON/YAML)
// 3. Default values

config := featureflag.LoadConfig("config.yaml")  // Load with precedence
config := featureflag.LoadConfigFromFile("config.yaml")  // File only
config := featureflag.LoadConfigFromEnv()  // Environment only
config := featureflag.DefaultConfig()  // Defaults only
```

#### 5. Observability System (`observability/`)
Comprehensive monitoring and debugging capabilities.

```go
// Observability features
type ObservabilityConfig struct {
    Logging LoggingConfig  // Structured logging with levels
    Metrics MetricsConfig  // Performance and operation metrics
}
```

**Logging Features:**
- **Structured logging** with contextual information
- **Configurable levels**: debug, info, warn, error
- **Operation tracking** with duration and success/failure
- **Error classification** for better debugging

**Metrics Features:**
- **Operation counters**: flag checks, gets, sets, deletes
- **Performance metrics**: average durations, cache hit rates
- **Error tracking**: categorized by error type
- **Cache statistics**: hits, misses, evictions

## 📁 Directory Structure

```
featureflag/
├── client/                 # Client implementation
│   ├── client.go          # Main client logic
│   └── client_test.go     # Client tests
├── config/                 # Configuration system
│   ├── config.go          # Config types and loading
│   └── config_test.go     # Config tests
├── core/                   # Core types and interfaces
│   ├── interfaces.go      # Client and Store interfaces
│   ├── flag.go           # FeatureFlag type and validation
│   ├── errors.go         # Error types and handling
│   └── *_test.go         # Core tests
├── storage/               # Storage implementations
│   ├── memory/           # In-memory storage
│   ├── redis/            # Redis storage
│   └── postgres/         # PostgreSQL storage
├── cache/                 # Caching layer
│   ├── cache.go          # LRU cache implementation
│   ├── store.go          # Cached store wrapper
│   └── *_test.go         # Cache tests
├── observability/         # Logging and metrics
│   ├── logger.go         # Logging implementation
│   ├── metrics.go        # Metrics collection
│   └── *_test.go         # Observability tests
└── testing/              # Testing utilities
    ├── performance/      # Performance benchmarks
    └── integration/      # Integration tests
```

## 🚀 Performance Characteristics

### Hot Path Optimization

The `IsEnabled()` method is optimized for maximum performance:

```go
// Optimized hot path with minimal allocations
func (c *client) IsEnabled(ctx context.Context, key string) (bool, error) {
    // 1. Fast validation (no allocations)
    if key == "" { return false, ErrInvalidFlag }
    
    // 2. Atomic closed check (lock-free)
    if atomic.LoadInt32(&c.closed) != 0 { return false, ErrClientClosed }
    
    // 3. Cache lookup (sub-microsecond if cached)
    flag, err := c.store.Get(ctx, key)
    
    // 4. Graceful degradation (return false for missing flags)
    if IsNotFoundError(err) { return false, nil }
    
    return flag.Enabled, nil
}
```

### Performance Benchmarks

Typical performance characteristics:

| Operation | Without Cache | With Cache | Notes |
|-----------|---------------|------------|-------|
| `IsEnabled()` | 10-100μs | <1μs | Depends on storage backend |
| `GetFlag()` | 10-100μs | <1μs | Includes metadata retrieval |
| `SetFlag()` | 100μs-1ms | N/A | Write operations bypass cache |
| Cache Hit Rate | N/A | 95%+ | For typical workloads |

### Concurrency Model

```go
// Thread-safe design principles:
// 1. Immutable data structures where possible
// 2. Atomic operations for hot paths
// 3. Read-write locks for infrequent operations
// 4. Lock-free cache implementation
// 5. Context-based cancellation
```

## 🔌 Extensibility Points

### Adding New Storage Backends

1. Implement the `Store` interface:

```go
type MyCustomStore struct {
    // Your storage implementation
}

func (s *MyCustomStore) Get(ctx context.Context, key string) (*FeatureFlag, error) {
    // Implementation
}

// ... implement other Store methods
```

2. Add configuration support:

```go
type MyCustomConfig struct {
    Endpoint string `json:"endpoint"`
    APIKey   string `json:"api_key"`
}

// Add to StorageConfig
type StorageConfig struct {
    Type     string           `json:"type"`
    MyCustom *MyCustomConfig  `json:"my_custom,omitempty"`
    // ... other configs
}
```

3. Register in client factory:

```go
func createStore(config Config) (Store, error) {
    switch config.Storage.Type {
    case "my_custom":
        return NewMyCustomStore(config.Storage.MyCustom)
    // ... other cases
    }
}
```

### Custom Observability

Implement custom loggers and metrics collectors:

```go
// Custom logger
type MyLogger struct{}

func (l *MyLogger) Info(ctx context.Context, msg string, fields map[string]any) {
    // Your logging implementation
}

// Custom metrics collector
type MyMetrics struct{}

func (m *MyMetrics) RecordFlagCheck(ctx context.Context, key string, enabled bool, duration time.Duration) {
    // Your metrics implementation
}
```

## 🧪 Testing Strategy

### Unit Tests
- **Component isolation**: Each component tested independently
- **Mock interfaces**: Use mocks for external dependencies
- **Edge cases**: Comprehensive error condition testing
- **Performance**: Benchmark critical paths

### Integration Tests
- **Storage backends**: Test against real Redis/PostgreSQL instances
- **End-to-end**: Full client lifecycle testing
- **Concurrency**: Multi-threaded access patterns
- **Failure scenarios**: Network failures, timeouts, etc.

### Performance Tests
- **Load testing**: High-throughput scenarios
- **Memory profiling**: Allocation and GC pressure
- **Latency testing**: P99 response times
- **Cache effectiveness**: Hit rate optimization

## 🔧 Development Guidelines

### Code Organization Principles

1. **Single Responsibility**: Each package has a clear, focused purpose
2. **Interface Segregation**: Small, focused interfaces
3. **Dependency Inversion**: Depend on abstractions, not concretions
4. **Open/Closed**: Open for extension, closed for modification

### Performance Guidelines

1. **Hot Path Optimization**: Minimize allocations in `IsEnabled()`
2. **Context Propagation**: Always respect context cancellation
3. **Graceful Degradation**: Prefer availability over consistency
4. **Cache-Friendly**: Design for effective caching

### Error Handling

```go
// Error classification for better handling
type ErrorType int

const (
    ErrorTypeNotFound ErrorType = iota
    ErrorTypeValidation
    ErrorTypeStorage
    ErrorTypeCache
    ErrorTypeConnection
    ErrorTypeTimeout
    // ... more types
)

// Rich error information
type FeatureFlagError struct {
    Type      ErrorType
    Operation string
    Key       string
    Cause     error
}
```

## 🚀 Contributing

### Adding New Features

1. **Design**: Start with interface design and documentation
2. **Implementation**: Follow existing patterns and conventions
3. **Testing**: Comprehensive unit and integration tests
4. **Documentation**: Update architecture docs and examples
5. **Performance**: Benchmark critical paths

### Code Review Checklist

- [ ] Interface design follows existing patterns
- [ ] Error handling is comprehensive and classified
- [ ] Performance impact is measured and acceptable
- [ ] Tests cover happy path and error conditions
- [ ] Documentation is updated
- [ ] Backward compatibility is maintained

This architecture provides a solid foundation for building scalable, maintainable feature flag systems while maintaining high performance and developer productivity.