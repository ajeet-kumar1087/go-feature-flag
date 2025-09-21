// Package featureflag provides a comprehensive feature flag library for Go applications.
// This package offers a simple, single-import interface to all feature flag functionality
// including multiple storage backends, caching, observability, and flexible configuration.
//
// Basic usage:
//
//	import "github.com/ajeet-kumar1087/go-feature-flag"
//
//	// Create a client with default settings (memory storage, caching enabled)
//	client, err := featureflag.NewClient(featureflag.DefaultConfig())
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer client.Close()
//
//	// Create and set a feature flag
//	flag := featureflag.FeatureFlag{
//		Key:         "new-feature",
//		Enabled:     true,
//		Description: "Enable new feature",
//	}
//	client.SetFlag(ctx, flag)
//
//	// Check if feature is enabled
//	if enabled, _ := client.IsEnabled(ctx, "new-feature"); enabled {
//		// Use new feature
//	}
//
// Advanced usage with custom configuration:
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
//			TTL:     featureflag.Duration(10 * time.Minute),
//			MaxSize: 1000,
//		},
//	}
//	client, err := featureflag.NewClient(config)
package featureflag

import (
	"time"

	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/cache"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/client"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/config"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/core"
	memorystorage "github.com/ajeet-kumar1087/go-feature-flag/featureflag/storage/memory"
	postgresstorage "github.com/ajeet-kumar1087/go-feature-flag/featureflag/storage/postgres"
	redisstorage "github.com/ajeet-kumar1087/go-feature-flag/featureflag/storage/redis"
)

// Core types - re-exported for convenience
type (
	// FeatureFlag represents a feature flag with metadata and validation
	FeatureFlag = core.FeatureFlag

	// Client is the main interface for feature flag operations
	Client = core.Client

	// Store defines the interface for feature flag storage backends
	Store = core.Store

	// MetricsSnapshot provides a point-in-time view of metrics
	MetricsSnapshot = core.MetricsSnapshot
)

// Configuration types
type (
	// Config holds all configuration options for the feature flag library
	Config = config.Config

	// StorageConfig defines storage backend configuration
	StorageConfig = config.StorageConfig

	// CacheConfig defines caching configuration
	CacheConfig = config.CacheConfig

	// RedisConfig defines Redis-specific configuration
	RedisConfig = config.RedisConfig

	// PostgresConfig defines PostgreSQL-specific configuration
	PostgresConfig = config.PostgresConfig

	// ObservabilityConfig defines logging and metrics configuration
	ObservabilityConfig = config.ObservabilityConfig

	// Duration is a wrapper around time.Duration that supports JSON/YAML marshaling
	Duration = config.Duration
)

// Error types and common errors
type (
	// ErrorType represents different categories of errors
	ErrorType = core.ErrorType

	// FeatureFlagError provides detailed error information
	FeatureFlagError = core.FeatureFlagError
)

// Common error variables
var (
	// ErrFlagNotFound indicates a feature flag was not found
	ErrFlagNotFound = core.ErrFlagNotFound

	// ErrInvalidFlag indicates an invalid feature flag
	ErrInvalidFlag = core.ErrInvalidFlag

	// ErrStorageFailure indicates a storage operation failed
	ErrStorageFailure = core.ErrStorageFailure

	// ErrInvalidConfig indicates invalid configuration
	ErrInvalidConfig = core.ErrInvalidConfig

	// ErrClientClosed indicates the client is closed
	ErrClientClosed = core.ErrClientClosed

	// ErrConnectionFailure indicates a connection failure
	ErrConnectionFailure = core.ErrConnectionFailure

	// ErrTimeout indicates an operation timeout
	ErrTimeout = core.ErrTimeout
)

// Error type constants
const (
	ErrorTypeNotFound   = core.ErrorTypeNotFound
	ErrorTypeValidation = core.ErrorTypeValidation
	ErrorTypeStorage    = core.ErrorTypeStorage
	ErrorTypeCache      = core.ErrorTypeCache
	ErrorTypeConnection = core.ErrorTypeConnection
	ErrorTypeTimeout    = core.ErrorTypeTimeout
	ErrorTypeAuth       = core.ErrorTypeAuth
	ErrorTypeRateLimit  = core.ErrorTypeRateLimit
	ErrorTypeClient     = core.ErrorTypeClient
	ErrorTypeInternal   = core.ErrorTypeInternal
)

// NewClient creates a new feature flag client with the given configuration.
// This is the main entry point for using the feature flag library.
func NewClient(cfg Config) (Client, error) {
	return client.NewClient(cfg)
}

// NewClientWithDefaults creates a new feature flag client with default configuration.
// Uses in-memory storage with caching enabled - suitable for development and testing.
func NewClientWithDefaults() (Client, error) {
	return client.NewClient(config.DefaultConfig())
}

// DefaultConfig returns a configuration with sensible defaults.
// Uses in-memory storage, caching enabled, and observability disabled.
func DefaultConfig() Config {
	return config.DefaultConfig()
}

// Storage constructor functions

// NewMemoryStore creates a new memory-based feature flag store.
// Suitable for development, testing, and single-instance deployments.
func NewMemoryStore() Store {
	return memorystorage.NewStore()
}

// NewRedisStore creates a new Redis-based feature flag store.
// Suitable for production deployments requiring persistence and scalability.
func NewRedisStore(cfg *RedisConfig) (Store, error) {
	return redisstorage.NewStore(cfg)
}

// NewPostgresStore creates a new PostgreSQL-based feature flag store.
// Suitable for production deployments requiring ACID compliance and complex queries.
func NewPostgresStore(cfg *PostgresConfig) (Store, error) {
	return postgresstorage.NewStore(cfg)
}

// Cache constructor functions

// NewCache creates a new cache instance with the specified configuration.
// Used internally by the cached store wrapper.
func NewCache(maxSize int, ttl time.Duration) *cache.Cache {
	return cache.NewCache(maxSize, ttl)
}

// NewCachedStore creates a cached store wrapper around any store implementation.
// Improves performance by caching frequently accessed feature flags.
func NewCachedStore(store Store, cacheConfig CacheConfig) Store {
	return cache.NewStore(store, cacheConfig)
}

// Utility functions

// IsNotFoundError checks if an error is a "not found" error.
func IsNotFoundError(err error) bool {
	return core.IsNotFoundError(err)
}

// IsRetryableError checks if an error indicates a retryable operation.
func IsRetryableError(err error) bool {
	return core.IsRetryableError(err)
}

// ClassifyError determines the error type based on the underlying error.
func ClassifyError(err error) ErrorType {
	return core.ClassifyError(err)
}
