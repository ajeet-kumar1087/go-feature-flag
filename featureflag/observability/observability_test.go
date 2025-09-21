package observability

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/config"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/core"
)

func TestErrorTypes(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		expectedType  core.ErrorType
		expectedRetry bool
	}{
		{
			name:          "flag not found",
			err:           core.ErrFlagNotFound,
			expectedType:  core.ErrorTypeNotFound,
			expectedRetry: false,
		},
		{
			name:          "invalid config",
			err:           core.ErrInvalidConfig,
			expectedType:  core.ErrorTypeValidation,
			expectedRetry: false,
		},
		{
			name:          "storage failure",
			err:           core.ErrStorageFailure,
			expectedType:  core.ErrorTypeStorage,
			expectedRetry: true,
		},
		{
			name:          "connection failure",
			err:           core.ErrConnectionFailure,
			expectedType:  core.ErrorTypeConnection,
			expectedRetry: true,
		},
		{
			name:          "timeout",
			err:           core.ErrTimeout,
			expectedType:  core.ErrorTypeTimeout,
			expectedRetry: true,
		},
		{
			name:          "rate limited",
			err:           core.ErrRateLimited,
			expectedType:  core.ErrorTypeRateLimit,
			expectedRetry: true,
		},
		{
			name:          "client closed",
			err:           core.ErrClientClosed,
			expectedType:  core.ErrorTypeClient,
			expectedRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ffErr := core.NewError("test", "test-key", tt.err)

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

	ffErr := core.NewErrorWithContext("test", "test-key", core.ErrStorageFailure, context)

	if ffErr.GetType() != core.ErrorTypeStorage {
		t.Errorf("expected error type %v, got %v", core.ErrorTypeStorage, ffErr.GetType())
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
			err:      core.ErrFlagNotFound,
			expected: true,
		},
		{
			name:     "wrapped not found error",
			err:      core.NewError("test", "key", core.ErrFlagNotFound),
			expected: true,
		},
		{
			name:     "other error",
			err:      core.ErrStorageFailure,
			expected: false,
		},
		{
			name:     "wrapped other error",
			err:      core.NewError("test", "key", core.ErrStorageFailure),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := core.IsNotFoundError(tt.err)
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
			err:      core.ErrConnectionFailure,
			expected: true,
		},
		{
			name:     "wrapped retryable error",
			err:      core.NewError("test", "key", core.ErrTimeout),
			expected: true,
		},
		{
			name:     "non-retryable error",
			err:      core.ErrFlagNotFound,
			expected: false,
		},
		{
			name:     "wrapped non-retryable error",
			err:      core.NewError("test", "key", core.ErrInvalidFlag),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := core.IsRetryableError(tt.err)
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
	metrics.RecordError(ctx, "test", core.ErrorTypeStorage)

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

	if snapshot.ErrorsByType[core.ErrorTypeStorage] != 1 {
		t.Errorf("expected 1 storage error, got %d", snapshot.ErrorsByType[core.ErrorTypeStorage])
	}

	if snapshot.AvgFlagCheckDuration == 0 {
		t.Error("expected non-zero average flag check duration")
	}

	if snapshot.AvgStorageDuration == 0 {
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
	metrics.RecordError(ctx, "test", core.ErrorTypeStorage)

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
		config      config.ObservabilityConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: config.ObservabilityConfig{
				Logging: config.LoggingConfig{
					Enabled: true,
					Level:   "info",
				},
				Metrics: config.MetricsConfig{
					Enabled: true,
				},
			},
			expectError: false,
		},
		{
			name: "invalid log level",
			config: config.ObservabilityConfig{
				Logging: config.LoggingConfig{
					Enabled: true,
					Level:   "invalid",
				},
				Metrics: config.MetricsConfig{
					Enabled: true,
				},
			},
			expectError: true,
		},
		{
			name: "empty log level (should be valid)",
			config: config.ObservabilityConfig{
				Logging: config.LoggingConfig{
					Enabled: true,
					Level:   "",
				},
				Metrics: config.MetricsConfig{
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

// Note: Client integration tests are in the client package
