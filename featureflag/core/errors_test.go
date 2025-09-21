package core

import (
	"errors"
	"testing"
)

func TestFeatureFlagError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *FeatureFlagError
		expected string
	}{
		{
			name: "error with key",
			err: &FeatureFlagError{
				Op:  "get",
				Key: "test-flag",
				Err: errors.New("not found"),
			},
			expected: "featureflag: get operation failed for key 'test-flag': not found",
		},
		{
			name: "error without key",
			err: &FeatureFlagError{
				Op:  "connect",
				Err: errors.New("connection failed"),
			},
			expected: "featureflag: connect operation failed: connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("FeatureFlagError.Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFeatureFlagError_NewMethods(t *testing.T) {
	originalErr := ErrStorageFailure
	err := NewError("test", "test-key", originalErr)

	// Test GetType
	if err.GetType() != ErrorTypeStorage {
		t.Errorf("GetType() = %v, want %v", err.GetType(), ErrorTypeStorage)
	}

	// Test IsRetryable
	if !err.IsRetryable() {
		t.Error("IsRetryable() = false, want true for storage failure")
	}

	// Test GetContext
	context := err.GetContext()
	if context == nil {
		t.Error("GetContext() returned nil")
	}
}

func TestNewErrorWithType(t *testing.T) {
	err := NewErrorWithType("test", "test-key", ErrorTypeTimeout, ErrTimeout)

	if err.GetType() != ErrorTypeTimeout {
		t.Errorf("GetType() = %v, want %v", err.GetType(), ErrorTypeTimeout)
	}

	if !err.IsRetryable() {
		t.Error("IsRetryable() = false, want true for timeout error")
	}
}

func TestNewErrorWithContext(t *testing.T) {
	context := map[string]string{
		"store_type": "redis",
		"operation":  "get",
	}

	err := NewErrorWithContext("test", "test-key", ErrStorageFailure, context)

	if err.GetType() != ErrorTypeStorage {
		t.Errorf("GetType() = %v, want %v", err.GetType(), ErrorTypeStorage)
	}

	ctx := err.GetContext()
	if ctx["store_type"] != "redis" {
		t.Errorf("GetContext()[store_type] = %v, want redis", ctx["store_type"])
	}

	if ctx["operation"] != "get" {
		t.Errorf("GetContext()[operation] = %v, want get", ctx["operation"])
	}
}

func TestFeatureFlagError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	err := &FeatureFlagError{
		Op:  "test",
		Err: originalErr,
	}

	if unwrapped := err.Unwrap(); unwrapped != originalErr {
		t.Errorf("FeatureFlagError.Unwrap() = %v, want %v", unwrapped, originalErr)
	}
}

func TestNewError(t *testing.T) {
	originalErr := errors.New("test error")
	err := NewError("get", "test-flag", originalErr)

	if err.Op != "get" {
		t.Errorf("NewError() Op = %v, want %v", err.Op, "get")
	}
	if err.Key != "test-flag" {
		t.Errorf("NewError() Key = %v, want %v", err.Key, "test-flag")
	}
	if err.Err != originalErr {
		t.Errorf("NewError() Err = %v, want %v", err.Err, originalErr)
	}
	if err.Type != ErrorTypeInternal {
		t.Errorf("NewError() Type = %v, want %v", err.Type, ErrorTypeInternal)
	}
	if err.Context == nil {
		t.Error("NewError() Context is nil")
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorType
	}{
		{"flag not found", ErrFlagNotFound, ErrorTypeNotFound},
		{"invalid config", ErrInvalidConfig, ErrorTypeValidation},
		{"storage failure", ErrStorageFailure, ErrorTypeStorage},
		{"cache failure", ErrCacheFailure, ErrorTypeCache},
		{"connection failure", ErrConnectionFailure, ErrorTypeConnection},
		{"timeout", ErrTimeout, ErrorTypeTimeout},
		{"unauthorized", ErrUnauthorized, ErrorTypeAuth},
		{"rate limited", ErrRateLimited, ErrorTypeRateLimit},
		{"client closed", ErrClientClosed, ErrorTypeClient},
		{"unknown error", errors.New("unknown"), ErrorTypeInternal},
		{"nil error", nil, ErrorTypeInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(tt.err)
			if result != tt.expected {
				t.Errorf("ClassifyError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsRetryableError_Function(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"connection failure", ErrConnectionFailure, true},
		{"timeout", ErrTimeout, true},
		{"rate limited", ErrRateLimited, true},
		{"storage failure", ErrStorageFailure, true},
		{"flag not found", ErrFlagNotFound, false},
		{"invalid config", ErrInvalidConfig, false},
		{"wrapped retryable", NewError("test", "key", ErrTimeout), true},
		{"wrapped non-retryable", NewError("test", "key", ErrFlagNotFound), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetryableError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}
