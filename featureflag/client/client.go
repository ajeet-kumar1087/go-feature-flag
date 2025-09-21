package client

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/cache"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/config"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/core"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/observability"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/storage/memory"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/storage/postgres"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/storage/redis"
)

// client implements the Client interface
type client struct {
	store   core.Store
	config  config.Config
	logger  observability.Logger
	metrics observability.MetricsCollector
	mu      sync.RWMutex
	closed  int32 // Use atomic for better performance in hot path
}

// NewClient creates a new feature flag client with the given configuration.
// This is the primary constructor for creating a feature flag client.
//
// The client will be configured according to the provided Config, including:
//   - Storage backend (memory, Redis, or PostgreSQL)
//   - Caching settings (TTL, max size, enabled/disabled)
//   - Observability settings (logging and metrics)
//   - Default flags to load on startup
//
// Example usage:
//
//	config := featureflag.Config{
//		Storage: featureflag.StorageConfig{
//			Type: "redis",
//			Redis: &featureflag.RedisConfig{
//				Addr: "localhost:6379",
//			},
//		},
//		Cache: featureflag.CacheConfig{
//			Enabled: true,
//			TTL:     featureflag.Duration(5 * time.Minute),
//			MaxSize: 1000,
//		},
//	}
//
//	client, err := featureflag.NewClient(config)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer client.Close()
//
// Parameters:
//   - config: Configuration for the client
//
// Returns:
//   - Client: The configured client instance
//   - error: Configuration validation errors or initialization errors
func NewClient(config config.Config) (core.Client, error) {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, core.NewError("init", "", fmt.Errorf("invalid configuration: %w", err))
	}

	// Create the appropriate store based on configuration
	store, err := createStore(config)
	if err != nil {
		return nil, core.NewError("init", "", fmt.Errorf("failed to create store: %w", err))
	}

	// Wrap with cache if enabled (will be updated with metrics later)
	if config.Cache.Enabled {
		store = cache.NewStore(store, config.Cache)
	}

	// Create logger
	var logger observability.Logger
	if config.Observability.Logging.Enabled {
		logLevel := parseLogLevel(config.Observability.Logging.Level)
		logger = observability.NewDefaultLogger(logLevel)
	} else {
		logger = observability.NewNoOpLogger()
	}

	// Create metrics collector
	var metrics observability.MetricsCollector
	if config.Observability.Metrics.Enabled {
		metrics = observability.NewDefaultMetricsCollector()
	} else {
		metrics = observability.NewNoOpMetricsCollector()
	}

	// Create client
	c := &client{
		store:   store,
		config:  config,
		logger:  logger,
		metrics: metrics,
	}

	// Set the store
	c.store = store

	// Load default flags
	if err := c.loadDefaultFlags(context.Background()); err != nil {
		// Don't fail initialization if default flags can't be loaded
		c.logger.Warn(context.Background(), "failed to load default flags", map[string]interface{}{
			"error": err.Error(),
		})
	}

	c.logger.Info(context.Background(), "feature flag client initialized", map[string]interface{}{
		"storage_type":    config.Storage.Type,
		"cache_enabled":   config.Cache.Enabled,
		"logging_enabled": config.Observability.Logging.Enabled,
		"metrics_enabled": config.Observability.Metrics.Enabled,
		"default_flags":   len(config.DefaultFlags),
	})

	return c, nil
}

// NewClientWithDefaults creates a new client with default configuration.
// This is a convenience constructor that uses sensible defaults:
//   - In-memory storage (no external dependencies)
//   - Caching enabled with 5-minute TTL and 1000 item limit
//   - Logging disabled
//   - Metrics disabled
//   - No default flags
//
// This is ideal for development, testing, or simple use cases where
// you don't need persistent storage or advanced configuration.
//
// Example usage:
//
//	client, err := featureflag.NewClientWithDefaults()
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer client.Close()
//
//	// Client is ready to use
//	enabled, _ := client.IsEnabled(ctx, "my-feature")
//
// Returns:
//   - Client: The client with default configuration
//   - error: Initialization errors (rare with default config)
func NewClientWithDefaults() (core.Client, error) {
	return NewClient(config.DefaultConfig())
}

// IsEnabled checks if a feature flag is enabled
// This is the hot path method optimized for maximum performance
func (c *client) IsEnabled(ctx context.Context, key string) (bool, error) {
	// Fast path validation - avoid allocations
	if key == "" {
		err := core.NewError("is_enabled", key, core.ErrInvalidFlag)
		c.metrics.RecordError(ctx, "is_enabled", core.ErrorTypeValidation)
		c.logger.Error(ctx, "invalid flag key", map[string]any{
			"operation": "is_enabled",
			"key":       key,
			"error":     err.Error(),
		})
		return false, err
	}

	// Optimized closed check - use atomic for maximum performance
	if atomic.LoadInt32(&c.closed) != 0 {
		err := core.NewError("is_enabled", key, core.ErrClientClosed)
		c.metrics.RecordError(ctx, "is_enabled", core.ErrorTypeClient)
		c.logger.Error(ctx, "client is closed", map[string]any{
			"operation": "is_enabled",
			"key":       key,
		})
		return false, err
	}

	// Hot path - minimize time measurement overhead for cached calls
	var start time.Time
	if c.config.Observability.Metrics.Enabled {
		start = time.Now()
	}

	flag, err := c.store.Get(ctx, key)

	var duration time.Duration
	if c.config.Observability.Metrics.Enabled {
		duration = time.Since(start)
	}

	if err != nil {
		// Graceful degradation: if flag doesn't exist or there's an error,
		// return false (disabled) instead of failing
		if core.IsNotFoundError(err) {
			if c.config.Observability.Metrics.Enabled {
				c.metrics.RecordFlagCheck(ctx, key, false, duration)
			}
			if c.config.Observability.Logging.Level == "debug" {
				c.logger.Debug(ctx, "flag not found, returning disabled", map[string]any{
					"operation": "is_enabled",
					"key":       key,
					"duration":  duration.String(),
				})
			}
			return false, nil
		}
		// For other errors, still return false but log the error
		if c.config.Observability.Metrics.Enabled {
			c.metrics.RecordError(ctx, "is_enabled", core.ClassifyError(err))
		}
		c.logger.Warn(ctx, "flag check failed, returning disabled", map[string]any{
			"operation": "is_enabled",
			"key":       key,
			"error":     err.Error(),
			"duration":  duration.String(),
		})
		return false, nil
	}

	if c.config.Observability.Metrics.Enabled {
		c.metrics.RecordFlagCheck(ctx, key, flag.Enabled, duration)
	}
	if c.config.Observability.Logging.Level == "debug" {
		c.logger.Debug(ctx, "flag check completed", map[string]any{
			"operation": "is_enabled",
			"key":       key,
			"enabled":   flag.Enabled,
			"duration":  duration.String(),
		})
	}

	return flag.Enabled, nil
}

// GetFlag retrieves a complete feature flag
func (c *client) GetFlag(ctx context.Context, key string) (*core.FeatureFlag, error) {
	start := time.Now()

	if key == "" {
		err := core.NewError("get_flag", key, core.ErrInvalidFlag)
		c.metrics.RecordError(ctx, "get_flag", core.ErrorTypeValidation)
		c.logger.Error(ctx, "invalid flag key", map[string]interface{}{
			"operation": "get_flag",
			"key":       key,
			"error":     err.Error(),
		})
		return nil, err
	}

	if atomic.LoadInt32(&c.closed) != 0 {
		err := core.NewError("get_flag", key, core.ErrClientClosed)
		c.metrics.RecordError(ctx, "get_flag", core.ErrorTypeClient)
		c.logger.Error(ctx, "client is closed", map[string]interface{}{
			"operation": "get_flag",
			"key":       key,
		})
		return nil, err
	}

	flag, err := c.store.Get(ctx, key)
	duration := time.Since(start)

	if err != nil {
		c.metrics.RecordFlagGet(ctx, key, false, duration)
		if core.IsNotFoundError(err) {
			c.logger.Debug(ctx, "flag not found", map[string]interface{}{
				"operation": "get_flag",
				"key":       key,
				"duration":  duration.String(),
			})
		} else {
			c.metrics.RecordError(ctx, "get_flag", core.ClassifyError(err))
			c.logger.Error(ctx, "flag get failed", map[string]interface{}{
				"operation": "get_flag",
				"key":       key,
				"error":     err.Error(),
				"duration":  duration.String(),
			})
		}
		return nil, err
	}

	c.metrics.RecordFlagGet(ctx, key, true, duration)
	c.logger.Debug(ctx, "flag retrieved successfully", map[string]interface{}{
		"operation": "get_flag",
		"key":       key,
		"enabled":   flag.Enabled,
		"duration":  duration.String(),
	})

	return flag, nil
}

// SetFlag creates or updates a feature flag
func (c *client) SetFlag(ctx context.Context, flag core.FeatureFlag) error {
	start := time.Now()

	if err := flag.Validate(); err != nil {
		validationErr := core.NewError("set_flag", flag.Key, err)
		c.metrics.RecordError(ctx, "set_flag", core.ErrorTypeValidation)
		c.logger.Error(ctx, "invalid flag", map[string]interface{}{
			"operation": "set_flag",
			"key":       flag.Key,
			"error":     validationErr.Error(),
		})
		return validationErr
	}

	if atomic.LoadInt32(&c.closed) != 0 {
		err := core.NewError("set_flag", flag.Key, core.ErrClientClosed)
		c.metrics.RecordError(ctx, "set_flag", core.ErrorTypeClient)
		c.logger.Error(ctx, "client is closed", map[string]interface{}{
			"operation": "set_flag",
			"key":       flag.Key,
		})
		return err
	}

	// Set timestamps
	flag.SetTimestamps()

	err := c.store.Set(ctx, flag)
	duration := time.Since(start)

	if err != nil {
		c.metrics.RecordFlagSet(ctx, flag.Key, false, duration)
		c.metrics.RecordError(ctx, "set_flag", core.ClassifyError(err))
		c.logger.Error(ctx, "flag set failed", map[string]interface{}{
			"operation": "set_flag",
			"key":       flag.Key,
			"enabled":   flag.Enabled,
			"error":     err.Error(),
			"duration":  duration.String(),
		})
		return err
	}

	c.metrics.RecordFlagSet(ctx, flag.Key, true, duration)
	c.logger.Info(ctx, "flag set successfully", map[string]interface{}{
		"operation": "set_flag",
		"key":       flag.Key,
		"enabled":   flag.Enabled,
		"duration":  duration.String(),
	})

	return nil
}

// DeleteFlag removes a feature flag
func (c *client) DeleteFlag(ctx context.Context, key string) error {
	start := time.Now()

	if key == "" {
		err := core.NewError("delete_flag", key, core.ErrInvalidFlag)
		c.metrics.RecordError(ctx, "delete_flag", core.ErrorTypeValidation)
		c.logger.Error(ctx, "invalid flag key", map[string]interface{}{
			"operation": "delete_flag",
			"key":       key,
			"error":     err.Error(),
		})
		return err
	}

	if atomic.LoadInt32(&c.closed) != 0 {
		err := core.NewError("delete_flag", key, core.ErrClientClosed)
		c.metrics.RecordError(ctx, "delete_flag", core.ErrorTypeClient)
		c.logger.Error(ctx, "client is closed", map[string]interface{}{
			"operation": "delete_flag",
			"key":       key,
		})
		return err
	}

	err := c.store.Delete(ctx, key)
	duration := time.Since(start)

	if err != nil {
		c.metrics.RecordFlagDelete(ctx, key, false, duration)
		c.metrics.RecordError(ctx, "delete_flag", core.ClassifyError(err))
		c.logger.Error(ctx, "flag delete failed", map[string]interface{}{
			"operation": "delete_flag",
			"key":       key,
			"error":     err.Error(),
			"duration":  duration.String(),
		})
		return err
	}

	c.metrics.RecordFlagDelete(ctx, key, true, duration)
	c.logger.Info(ctx, "flag deleted successfully", map[string]interface{}{
		"operation": "delete_flag",
		"key":       key,
		"duration":  duration.String(),
	})

	return nil
}

// GetAllFlags retrieves all feature flags
func (c *client) GetAllFlags(ctx context.Context) ([]core.FeatureFlag, error) {
	start := time.Now()

	if atomic.LoadInt32(&c.closed) != 0 {
		err := core.NewError("get_all_flags", "", core.ErrClientClosed)
		c.metrics.RecordError(ctx, "get_all_flags", core.ErrorTypeClient)
		c.logger.Error(ctx, "client is closed", map[string]interface{}{
			"operation": "get_all_flags",
		})
		return nil, err
	}

	flags, err := c.store.GetAll(ctx)
	duration := time.Since(start)

	if err != nil {
		c.metrics.RecordError(ctx, "get_all_flags", core.ClassifyError(err))
		c.logger.Error(ctx, "get all flags failed", map[string]interface{}{
			"operation": "get_all_flags",
			"error":     err.Error(),
			"duration":  duration.String(),
		})
		return nil, err
	}

	c.logger.Debug(ctx, "retrieved all flags successfully", map[string]interface{}{
		"operation": "get_all_flags",
		"count":     len(flags),
		"duration":  duration.String(),
	})

	return flags, nil
}

// HealthCheck verifies the health of the client and its dependencies
func (c *client) HealthCheck(ctx context.Context) error {
	start := time.Now()

	if atomic.LoadInt32(&c.closed) != 0 {
		err := core.NewError("health_check", "", core.ErrClientClosed)
		c.logger.Error(ctx, "health check failed - client is closed", map[string]interface{}{
			"operation": "health_check",
		})
		return err
	}

	// Check store health
	if err := c.store.HealthCheck(ctx); err != nil {
		duration := time.Since(start)
		c.metrics.RecordError(ctx, "health_check", core.ClassifyError(err))
		c.logger.Error(ctx, "health check failed - store unhealthy", map[string]interface{}{
			"operation": "health_check",
			"error":     err.Error(),
			"duration":  duration.String(),
		})
		return core.NewError("health_check", "", fmt.Errorf("store health check failed: %w", err))
	}

	duration := time.Since(start)
	c.logger.Debug(ctx, "health check passed", map[string]interface{}{
		"operation": "health_check",
		"duration":  duration.String(),
	})

	return nil
}

// GetMetrics returns current metrics snapshot
func (c *client) GetMetrics() core.MetricsSnapshot {
	return c.metrics.GetMetrics()
}

// Close cleanly shuts down the client
func (c *client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return nil // Already closed
	}

	c.logger.Info(context.Background(), "shutting down feature flag client", map[string]interface{}{
		"operation": "close",
	})

	// closed flag is already set by CompareAndSwapInt32 above

	if c.store != nil {
		if err := c.store.Close(); err != nil {
			c.logger.Error(context.Background(), "failed to close store", map[string]interface{}{
				"operation": "close",
				"error":     err.Error(),
			})
			return err
		}
	}

	c.logger.Info(context.Background(), "feature flag client shut down successfully", map[string]interface{}{
		"operation": "close",
	})

	return nil
}

// loadDefaultFlags loads the default flags specified in configuration
func (c *client) loadDefaultFlags(ctx context.Context) error {
	if len(c.config.DefaultFlags) == 0 {
		return nil
	}

	for _, flag := range c.config.DefaultFlags {
		// Check if flag already exists
		existing, err := c.store.Get(ctx, flag.Key)
		if err != nil && !core.IsNotFoundError(err) {
			// If there's an error other than "not found", continue with next flag
			continue
		}

		// Only set the flag if it doesn't exist (don't override existing flags)
		if existing == nil {
			flag.SetTimestamps()
			if err := c.store.Set(ctx, flag); err != nil {
				// Continue with other flags if one fails
				continue
			}
		}
	}

	return nil
}

// createStore creates the appropriate store based on configuration
func createStore(config config.Config) (core.Store, error) {
	switch config.Storage.Type {
	case "memory":
		return memory.NewStore(), nil

	case "redis":
		if config.Storage.Redis == nil {
			return nil, fmt.Errorf("redis configuration is required for redis storage")
		}
		return redis.NewStore(config.Storage.Redis)

	case "postgres":
		if config.Storage.Postgres == nil {
			return nil, fmt.Errorf("postgres configuration is required for postgres storage")
		}
		return postgres.NewStore(config.Storage.Postgres)

	default:
		return nil, fmt.Errorf("unsupported storage type: %s", config.Storage.Type)
	}
}

// parseLogLevel converts a string log level to LogLevel
func parseLogLevel(level string) observability.LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return observability.LogLevelDebug
	case "info":
		return observability.LogLevelInfo
	case "warn":
		return observability.LogLevelWarn
	case "error":
		return observability.LogLevelError
	default:
		return observability.LogLevelInfo
	}
}
