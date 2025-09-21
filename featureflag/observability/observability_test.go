package observability

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestErrorTypes(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		expectedType  ErrorType
		expectedRetry bool
	}{
		{
			name:          "flag not found",
			err:           ErrFlagNotFound,
			expectedType:  ErrorTypeNotFound,
			expectedRetry: false,
		},
		{
			name:          "invalid config",
			err:           ErrInvalidConfig,
			expectedType:  ErrorTypeValidation,
			expectedRetry: false,
		},
		{
			name:          "storage failure",
			err:           ErrStorageFailure,
			expectedType:  ErrorTypeStorage,
			expectedRetry: true,
		},
		{
			name:          "connection failure",
			err:           ErrConnectionFailure,
			expectedType:  ErrorTypeConnection,
			expectedRetry: true,
		},
		{
			name:          "timeout",
			err:           ErrTimeout,
			expectedType:  ErrorTypeTimeout,
			expectedRetry: true,
		},
		{
			name:          "rate limited",
			err:           ErrRateLimited,
			expectedType:  ErrorTypeRateLimit,
			expectedRetry: true,
		},
		{
			name:          "client closed",
			err:           ErrClientClosed,
			expectedType:  ErrorTypeClient,
			expectedRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ffErr := NewError("test", "test-key", tt.err)

			if ffErr.GetType() != tt.expectedType {
				t.Errorf("expected error type %v, got %v", tt.expectedType, ffErr.GetType())
			}

			if ffErr.IsRetryable() != tt.expectedRetry {
				t.Errorf("expected retryable %v, got %v", tt.expectedRetry, ffErr.IsRetryable())
			}

			// Test error message format
			errMsg := ffErr.Error()
			if !strings.Contains(errMsg, "test operation failed for key 'test-key'") {
				t.Errorf("error message format incorrect: %s", errMsg)
			}

			// Test unwrapping
			if !errors.Is(ffErr, tt.err) {
				t.Errorf("error unwrapping failed")
			}
		})
	}
}

func TestErrorWithContext(t *testing.T) {
	context := map[string]string{
		"store_type": "redis",
		"operation":  "get",
	}

	ffErr := NewErrorWithContext("test", "test-key", ErrStorageFailure, context)

	if ffErr.GetType() != ErrorTypeStorage {
		t.Errorf("expected error type %v, got %v", ErrorTypeStorage, ffErr.GetType())
	}

	ctx := ffErr.GetContext()
	if ctx["store_type"] != "redis" {
		t.Errorf("expected context store_type=redis, got %s", ctx["store_type"])
	}

	if ctx["operation"] != "get" {
		t.Errorf("expected context operation=get, got %s", ctx["operation"])
	}
}

func TestIsNotFoundErrorObservability(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "direct not found error",
			err:      ErrFlagNotFound,
			expected: true,
		},
		{
			name:     "wrapped not found error",
			err:      NewError("test", "key", ErrFlagNotFound),
			expected: true,
		},
		{
			name:     "other error",
			err:      ErrStorageFailure,
			expected: false,
		},
		{
			name:     "wrapped other error",
			err:      NewError("test", "key", ErrStorageFailure),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotFoundError(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "retryable error",
			err:      ErrConnectionFailure,
			expected: true,
		},
		{
			name:     "wrapped retryable error",
			err:      NewError("test", "key", ErrTimeout),
			expected: true,
		},
		{
			name:     "non-retryable error",
			err:      ErrFlagNotFound,
			expected: false,
		},
		{
			name:     "wrapped non-retryable error",
			err:      NewError("test", "key", ErrInvalidFlag),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDefaultLogger(t *testing.T) {
	logger := NewDefaultLogger(LogLevelDebug)
	ctx := context.Background()

	// Test that logger doesn't panic
	logger.Debug(ctx, "debug message", map[string]interface{}{"key": "value"})
	logger.Info(ctx, "info message", map[string]interface{}{"key": "value"})
	logger.Warn(ctx, "warn message", map[string]interface{}{"key": "value"})
	logger.Error(ctx, "error message", map[string]interface{}{"key": "value"})

	// Test WithFields
	loggerWithFields := logger.WithFields(map[string]interface{}{"component": "test"})
	loggerWithFields.Info(ctx, "test message", map[string]interface{}{"extra": "data"})
}

func TestNoOpLogger(t *testing.T) {
	logger := NewNoOpLogger()
	ctx := context.Background()

	// Test that no-op logger doesn't panic
	logger.Debug(ctx, "debug message", map[string]interface{}{"key": "value"})
	logger.Info(ctx, "info message", map[string]interface{}{"key": "value"})
	logger.Warn(ctx, "warn message", map[string]interface{}{"key": "value"})
	logger.Error(ctx, "error message", map[string]interface{}{"key": "value"})

	// Test WithFields
	loggerWithFields := logger.WithFields(map[string]interface{}{"component": "test"})
	loggerWithFields.Info(ctx, "test message", map[string]interface{}{"extra": "data"})
}

func TestDefaultMetricsCollector(t *testing.T) {
	metrics := NewDefaultMetricsCollector()
	ctx := context.Background()

	// Record some metrics
	metrics.RecordFlagCheck(ctx, "test-flag", true, 100*time.Microsecond)
	metrics.RecordFlagGet(ctx, "test-flag", true, 200*time.Microsecond)
	metrics.RecordFlagSet(ctx, "test-flag", true, 300*time.Microsecond)
	metrics.RecordFlagDelete(ctx, "test-flag", true, 400*time.Microsecond)

	metrics.RecordCacheHit(ctx, "test-flag")
	metrics.RecordCacheMiss(ctx, "test-flag")
	metrics.RecordCacheEviction(ctx, "test-flag")

	metrics.RecordStorageOperation(ctx, "get", true, 500*time.Microsecond)
	metrics.RecordError(ctx, "test", ErrorTypeStorage)

	// Get metrics snapshot
	snapshot := metrics.GetMetrics()

	// Verify metrics
	if snapshot.FlagChecks != 1 {
		t.Errorf("expected 1 flag check, got %d", snapshot.FlagChecks)
	}

	if snapshot.FlagCheckSuccesses != 1 {
		t.Errorf("expected 1 flag check success, got %d", snapshot.FlagCheckSuccesses)
	}

	if snapshot.FlagGets != 1 {
		t.Errorf("expected 1 flag get, got %d", snapshot.FlagGets)
	}

	if snapshot.FlagSets != 1 {
		t.Errorf("expected 1 flag set, got %d", snapshot.FlagSets)
	}

	if snapshot.FlagDeletes != 1 {
		t.Errorf("expected 1 flag delete, got %d", snapshot.FlagDeletes)
	}

	if snapshot.CacheHits != 1 {
		t.Errorf("expected 1 cache hit, got %d", snapshot.CacheHits)
	}

	if snapshot.CacheMisses != 1 {
		t.Errorf("expected 1 cache miss, got %d", snapshot.CacheMisses)
	}

	if snapshot.CacheEvictions != 1 {
		t.Errorf("expected 1 cache eviction, got %d", snapshot.CacheEvictions)
	}

	if snapshot.StorageOperations != 1 {
		t.Errorf("expected 1 storage operation, got %d", snapshot.StorageOperations)
	}

	if snapshot.ErrorsByType[ErrorTypeStorage] != 1 {
		t.Errorf("expected 1 storage error, got %d", snapshot.ErrorsByType[ErrorTypeStorage])
	}

	if snapshot.AverageFlagCheckDuration == 0 {
		t.Error("expected non-zero average flag check duration")
	}

	if snapshot.AverageStorageDuration == 0 {
		t.Error("expected non-zero average storage duration")
	}
}

func TestNoOpMetricsCollector(t *testing.T) {
	metrics := NewNoOpMetricsCollector()
	ctx := context.Background()

	// Test that no-op metrics collector doesn't panic
	metrics.RecordFlagCheck(ctx, "test-flag", true, 100*time.Microsecond)
	metrics.RecordFlagGet(ctx, "test-flag", true, 200*time.Microsecond)
	metrics.RecordFlagSet(ctx, "test-flag", true, 300*time.Microsecond)
	metrics.RecordFlagDelete(ctx, "test-flag", true, 400*time.Microsecond)

	metrics.RecordCacheHit(ctx, "test-flag")
	metrics.RecordCacheMiss(ctx, "test-flag")
	metrics.RecordCacheEviction(ctx, "test-flag")

	metrics.RecordStorageOperation(ctx, "get", true, 500*time.Microsecond)
	metrics.RecordError(ctx, "test", ErrorTypeStorage)

	// Get metrics snapshot - should be empty
	snapshot := metrics.GetMetrics()

	if snapshot.FlagChecks != 0 {
		t.Errorf("expected 0 flag checks, got %d", snapshot.FlagChecks)
	}

	if len(snapshot.ErrorsByType) != 0 {
		t.Errorf("expected empty errors map, got %v", snapshot.ErrorsByType)
	}
}

func TestObservabilityConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      ObservabilityConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: ObservabilityConfig{
				Logging: LoggingConfig{
					Enabled: true,
					Level:   "info",
				},
				Metrics: MetricsConfig{
					Enabled: true,
				},
			},
			expectError: false,
		},
		{
			name: "invalid log level",
			config: ObservabilityConfig{
				Logging: LoggingConfig{
					Enabled: true,
					Level:   "invalid",
				},
				Metrics: MetricsConfig{
					Enabled: true,
				},
			},
			expectError: true,
		},
		{
			name: "empty log level (should be valid)",
			config: ObservabilityConfig{
				Logging: LoggingConfig{
					Enabled: true,
					Level:   "",
				},
				Metrics: MetricsConfig{
					Enabled: true,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no validation error, got %v", err)
			}
		})
	}
}

func TestClientObservability(t *testing.T) {
	// Test client with observability enabled
	config := Config{
		Storage: StorageConfig{
			Type: "memory",
		},
		Cache: CacheConfig{
			Enabled: true,
			TTL:     Duration(5 * time.Minute),
			MaxSize: 100,
		},
		Observability: ObservabilityConfig{
			Logging: LoggingConfig{
				Enabled: true,
				Level:   "debug",
			},
			Metrics: MetricsConfig{
				Enabled: true,
			},
		},
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Test flag operations with observability
	flag := FeatureFlag{
		Key:         "test-flag",
		Enabled:     true,
		Description: "Test flag",
	}

	// Set flag
	err = client.SetFlag(ctx, flag)
	if err != nil {
		t.Errorf("failed to set flag: %v", err)
	}

	// Check flag
	enabled, err := client.IsEnabled(ctx, "test-flag")
	if err != nil {
		t.Errorf("failed to check flag: %v", err)
	}
	if !enabled {
		t.Error("expected flag to be enabled")
	}

	// Get flag
	retrievedFlag, err := client.GetFlag(ctx, "test-flag")
	if err != nil {
		t.Errorf("failed to get flag: %v", err)
	}
	if retrievedFlag.Key != "test-flag" {
		t.Errorf("expected flag key 'test-flag', got %s", retrievedFlag.Key)
	}

	// Test health check
	err = client.HealthCheck(ctx)
	if err != nil {
		t.Errorf("health check failed: %v", err)
	}

	// Get metrics
	metrics := client.GetMetrics()
	if metrics.FlagChecks == 0 {
		t.Error("expected non-zero flag checks in metrics")
	}

	// Delete flag
	err = client.DeleteFlag(ctx, "test-flag")
	if err != nil {
		t.Errorf("failed to delete flag: %v", err)
	}

	// Test error scenarios
	_, err = client.GetFlag(ctx, "")
	if err == nil {
		t.Error("expected error for empty flag key")
	}

	// Test graceful degradation
	enabled, err = client.IsEnabled(ctx, "non-existent-flag")
	if err != nil {
		t.Errorf("unexpected error for non-existent flag: %v", err)
	}
	if enabled {
		t.Error("expected non-existent flag to be disabled")
	}
}

func TestClientObservabilityDisabled(t *testing.T) {
	// Test client with observability disabled
	config := Config{
		Storage: StorageConfig{
			Type: "memory",
		},
		Observability: ObservabilityConfig{
			Logging: LoggingConfig{
				Enabled: false,
			},
			Metrics: MetricsConfig{
				Enabled: false,
			},
		},
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Test basic operations still work
	flag := FeatureFlag{
		Key:         "test-flag",
		Enabled:     true,
		Description: "Test flag",
	}

	err = client.SetFlag(ctx, flag)
	if err != nil {
		t.Errorf("failed to set flag: %v", err)
	}

	enabled, err := client.IsEnabled(ctx, "test-flag")
	if err != nil {
		t.Errorf("failed to check flag: %v", err)
	}
	if !enabled {
		t.Error("expected flag to be enabled")
	}

	// Metrics should be empty
	metrics := client.GetMetrics()
	if metrics.FlagChecks != 0 {
		t.Error("expected zero flag checks when metrics disabled")
	}
}
