package core

import (
	"context"
	"time"
)

// Client is the main interface for feature flag operations.
// It provides methods to check, create, update, and delete feature flags,
// as well as health checking and metrics collection.
//
// Example usage:
//
//	client, err := featureflag.NewClientWithDefaults()
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer client.Close()
//
//	// Check if a feature is enabled
//	enabled, err := client.IsEnabled(ctx, "new-feature")
//	if err != nil {
//		log.Printf("Error checking flag: %v", err)
//	}
//
//	if enabled {
//		// Use new feature
//	}
type Client interface {
	// IsEnabled checks if a feature flag is enabled.
	// This is the primary method for feature flag evaluation in application code.
	// It returns false for non-existent flags (graceful degradation) and only
	// returns an error for validation issues or when the client is closed.
	//
	// Parameters:
	//   - ctx: Context for the operation
	//   - key: The feature flag key to check
	//
	// Returns:
	//   - bool: true if the flag exists and is enabled, false otherwise
	//   - error: validation errors or client state errors
	IsEnabled(ctx context.Context, key string) (bool, error)

	// GetFlag retrieves a complete feature flag with all its metadata.
	// Unlike IsEnabled, this method returns an error if the flag doesn't exist.
	//
	// Parameters:
	//   - ctx: Context for the operation
	//   - key: The feature flag key to retrieve
	//
	// Returns:
	//   - *FeatureFlag: The complete flag data including metadata and timestamps
	//   - error: ErrFlagNotFound if the flag doesn't exist, or other errors
	GetFlag(ctx context.Context, key string) (*FeatureFlag, error)

	// SetFlag creates or updates a feature flag.
	// If the flag already exists, it will be updated with the new values.
	// Timestamps are automatically managed.
	//
	// Parameters:
	//   - ctx: Context for the operation
	//   - flag: The feature flag to create or update
	//
	// Returns:
	//   - error: validation errors or storage errors
	SetFlag(ctx context.Context, flag FeatureFlag) error

	// DeleteFlag removes a feature flag permanently.
	// This operation cannot be undone. Returns an error if the flag doesn't exist.
	//
	// Parameters:
	//   - ctx: Context for the operation
	//   - key: The feature flag key to delete
	//
	// Returns:
	//   - error: ErrFlagNotFound if the flag doesn't exist, or other errors
	DeleteFlag(ctx context.Context, key string) error

	// GetAllFlags retrieves all feature flags from storage.
	// This method can be expensive for large numbers of flags and should be
	// used sparingly in production environments.
	//
	// Parameters:
	//   - ctx: Context for the operation
	//
	// Returns:
	//   - []FeatureFlag: All flags in the system
	//   - error: storage errors
	GetAllFlags(ctx context.Context) ([]FeatureFlag, error)

	// HealthCheck verifies the health of the client and its dependencies.
	// This includes checking storage connectivity and cache status.
	// Useful for application health endpoints and monitoring.
	//
	// Parameters:
	//   - ctx: Context for the operation
	//
	// Returns:
	//   - error: nil if healthy, error describing the issue if unhealthy
	HealthCheck(ctx context.Context) error

	// GetMetrics returns current metrics snapshot if metrics are enabled.
	// Returns zero values if metrics collection is disabled.
	// Useful for monitoring cache performance and operation statistics.
	//
	// Returns:
	//   - MetricsSnapshot: Current metrics data
	GetMetrics() MetricsSnapshot

	// Close cleanly shuts down the client and releases all resources.
	// After calling Close, all other methods will return ErrClientClosed.
	// It's safe to call Close multiple times.
	//
	// Returns:
	//   - error: errors encountered during shutdown
	Close() error
}

// Store defines the interface for feature flag storage backends.
// Implementations include in-memory, Redis, and PostgreSQL stores.
// This interface is primarily used internally by the Client implementation.
//
// Storage implementations should be thread-safe and handle concurrent access.
// All methods should respect context cancellation and timeouts.
type Store interface {
	// Get retrieves a feature flag by key from the storage backend.
	//
	// Parameters:
	//   - ctx: Context for the operation
	//   - key: The feature flag key to retrieve
	//
	// Returns:
	//   - *FeatureFlag: The flag data if found
	//   - error: ErrFlagNotFound if not found, or storage errors
	Get(ctx context.Context, key string) (*FeatureFlag, error)

	// Set creates or updates a feature flag in the storage backend.
	// The implementation should handle both creation and updates transparently.
	//
	// Parameters:
	//   - ctx: Context for the operation
	//   - flag: The feature flag to store
	//
	// Returns:
	//   - error: storage errors or validation errors
	Set(ctx context.Context, flag FeatureFlag) error

	// Delete removes a feature flag from the storage backend.
	//
	// Parameters:
	//   - ctx: Context for the operation
	//   - key: The feature flag key to delete
	//
	// Returns:
	//   - error: ErrFlagNotFound if not found, or storage errors
	Delete(ctx context.Context, key string) error

	// GetAll retrieves all feature flags from the storage backend.
	// Implementations should be efficient for large datasets.
	//
	// Parameters:
	//   - ctx: Context for the operation
	//
	// Returns:
	//   - []FeatureFlag: All flags in storage
	//   - error: storage errors
	GetAll(ctx context.Context) ([]FeatureFlag, error)

	// HealthCheck verifies connectivity and health of the storage backend.
	// Should perform a lightweight operation to verify the store is accessible.
	//
	// Parameters:
	//   - ctx: Context for the operation
	//
	// Returns:
	//   - error: nil if healthy, error describing the issue if unhealthy
	HealthCheck(ctx context.Context) error

	// Close cleanly shuts down the storage backend and releases resources.
	// Should be safe to call multiple times.
	//
	// Returns:
	//   - error: errors encountered during shutdown
	Close() error
}

// MetricsSnapshot provides a point-in-time view of metrics
type MetricsSnapshot struct {
	// Flag operation counts
	FlagChecks  int64 `json:"flag_checks"`
	FlagGets    int64 `json:"flag_gets"`
	FlagSets    int64 `json:"flag_sets"`
	FlagDeletes int64 `json:"flag_deletes"`

	// Flag operation success rates
	FlagCheckSuccesses  int64 `json:"flag_check_successes"`
	FlagGetSuccesses    int64 `json:"flag_get_successes"`
	FlagSetSuccesses    int64 `json:"flag_set_successes"`
	FlagDeleteSuccesses int64 `json:"flag_delete_successes"`

	// Cache metrics
	CacheHits      int64 `json:"cache_hits"`
	CacheMisses    int64 `json:"cache_misses"`
	CacheEvictions int64 `json:"cache_evictions"`

	// Storage operation metrics
	StorageOperations int64 `json:"storage_operations"`
	StorageSuccesses  int64 `json:"storage_successes"`

	// Error counts by type
	ErrorsByType map[ErrorType]int64 `json:"errors_by_type"`

	// Performance metrics
	AvgFlagCheckDuration time.Duration `json:"avg_flag_check_duration"`
	AvgStorageDuration   time.Duration `json:"avg_storage_duration"`

	// Timestamp when snapshot was taken
	Timestamp time.Time `json:"timestamp"`
}
