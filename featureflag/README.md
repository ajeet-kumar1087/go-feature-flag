# Feature Flag Library - Code Organization

This directory contains the core feature flag library implementation, organized using a logical file naming convention for better maintainability and readability.

## File Organization

### 📋 Core Files
- `client.go` - Main client interface and implementation
- `client_test.go` - Client tests
- `interfaces.go` - Core interfaces (Store, Logger, MetricsCollector)
- `interfaces_test.go` - Interface tests
- `flag.go` - FeatureFlag type and validation
- `flag_test.go` - Flag tests
- `errors.go` - Error types and handling
- `errors_test.go` - Error tests

### ⚙️ Configuration
- `config.go` - Configuration types and loading
- `config_test.go` - Configuration tests

### 💾 Storage Implementations
- `store_memory.go` - In-memory storage implementation
- `store_memory_test.go` - Memory store tests
- `store_redis.go` - Redis storage implementation
- `store_redis_test.go` - Redis store tests
- `store_postgres.go` - PostgreSQL storage implementation
- `store_postgres_test.go` - PostgreSQL store tests

### 🚀 Caching Layer
- `cache.go` - LRU cache implementation
- `cache_test.go` - Cache tests
- `cache_bench_test.go` - Cache benchmarks
- `cache_store.go` - Cached store wrapper
- `cache_store_test.go` - Cached store tests

### 📊 Observability
- `observability_logger.go` - Logging implementation
- `observability_metrics.go` - Metrics collection
- `observability_test.go` - Observability tests

### 🌐 HTTP Interface
- `http_handler.go` - HTTP handlers for REST API
- `http_routes.go` - HTTP route definitions

### 🧪 Testing & Performance
- `test_concurrency.go` - Concurrency tests
- `test_performance.go` - Performance benchmarks
- `test_load.go` - Load testing
- `test_integration_*.go` - Integration tests for various components

## Benefits of This Organization

### 🎯 **Improved Readability**
- Files are grouped by functionality using clear prefixes
- Easy to find related code (e.g., all storage implementations start with `store_`)
- Consistent naming convention across the codebase

### 🔧 **Better Maintainability**
- Related functionality is clearly grouped
- Easy to add new storage backends (follow `store_*.go` pattern)
- Test files are clearly associated with their implementation files

### 📦 **Single Package Design**
- All files remain in the same Go package for simplicity
- No complex import dependencies between subpackages
- Maintains backward compatibility with existing imports

### 🚀 **Developer Experience**
- IDE file explorers show files in logical groups
- Easy to navigate between related files
- Clear separation of concerns

## Adding New Components

When adding new functionality, follow these naming conventions:

- **Storage backends**: `store_<backend>.go` and `store_<backend>_test.go`
- **Cache implementations**: `cache_<type>.go` and `cache_<type>_test.go`
- **Observability features**: `observability_<feature>.go`
- **HTTP endpoints**: `http_<feature>.go`
- **Tests**: `test_<type>.go` for specialized test files

This organization makes the codebase more maintainable while preserving the simplicity of a single Go package.