package core

import (
	"errors"
	"fmt"
)

// Common errors
var (
	ErrFlagNotFound      = errors.New("feature flag not found")
	ErrInvalidConfig     = errors.New("invalid configuration")
	ErrStorageFailure    = errors.New("storage operation failed")
	ErrCacheFailure      = errors.New("cache operation failed")
	ErrInvalidFlag       = errors.New("invalid feature flag")
	ErrConnectionFailure = errors.New("connection failure")
	ErrTimeout           = errors.New("operation timeout")
	ErrUnauthorized      = errors.New("unauthorized access")
	ErrRateLimited       = errors.New("rate limited")
	ErrClientClosed      = errors.New("client is closed")
	ErrInvalidOperation  = errors.New("invalid operation")
)

// ErrorType represents different categories of errors
type ErrorType string

const (
	ErrorTypeNotFound   ErrorType = "not_found"
	ErrorTypeValidation ErrorType = "validation"
	ErrorTypeStorage    ErrorType = "storage"
	ErrorTypeCache      ErrorType = "cache"
	ErrorTypeConnection ErrorType = "connection"
	ErrorTypeTimeout    ErrorType = "timeout"
	ErrorTypeAuth       ErrorType = "authorization"
	ErrorTypeRateLimit  ErrorType = "rate_limit"
	ErrorTypeClient     ErrorType = "client"
	ErrorTypeInternal   ErrorType = "internal"
)

// FeatureFlagError provides detailed error information
type FeatureFlagError struct {
	Op        string            // Operation that failed
	Key       string            // Flag key (if applicable)
	Type      ErrorType         // Error category
	Err       error             // Underlying error
	Retryable bool              // Whether the operation can be retried
	Context   map[string]string // Additional context information
}

func (e *FeatureFlagError) Error() string {
	if e.Key != "" {
		return fmt.Sprintf("featureflag: %s operation failed for key '%s': %v", e.Op, e.Key, e.Err)
	}
	return fmt.Sprintf("featureflag: %s operation failed: %v", e.Op, e.Err)
}

func (e *FeatureFlagError) Unwrap() error {
	return e.Err
}

// IsRetryable returns whether the error indicates a retryable operation
func (e *FeatureFlagError) IsRetryable() bool {
	return e.Retryable
}

// GetType returns the error type
func (e *FeatureFlagError) GetType() ErrorType {
	return e.Type
}

// GetContext returns additional context information
func (e *FeatureFlagError) GetContext() map[string]string {
	if e.Context == nil {
		return make(map[string]string)
	}
	return e.Context
}

// NewError creates a new FeatureFlagError
func NewError(op, key string, err error) *FeatureFlagError {
	return &FeatureFlagError{
		Op:        op,
		Key:       key,
		Type:      ClassifyError(err),
		Err:       err,
		Retryable: isRetryableError(err),
		Context:   make(map[string]string),
	}
}

// NewErrorWithType creates a new FeatureFlagError with specific type
func NewErrorWithType(op, key string, errType ErrorType, err error) *FeatureFlagError {
	return &FeatureFlagError{
		Op:        op,
		Key:       key,
		Type:      errType,
		Err:       err,
		Retryable: isRetryableError(err),
		Context:   make(map[string]string),
	}
}

// NewErrorWithContext creates a new FeatureFlagError with additional context
func NewErrorWithContext(op, key string, err error, context map[string]string) *FeatureFlagError {
	return &FeatureFlagError{
		Op:        op,
		Key:       key,
		Type:      ClassifyError(err),
		Err:       err,
		Retryable: isRetryableError(err),
		Context:   context,
	}
}

// classifyError determines the error type based on the underlying error
func ClassifyError(err error) ErrorType {
	if err == nil {
		return ErrorTypeInternal
	}

	switch err {
	case ErrFlagNotFound:
		return ErrorTypeNotFound
	case ErrInvalidFlag, ErrInvalidConfig:
		return ErrorTypeValidation
	case ErrStorageFailure:
		return ErrorTypeStorage
	case ErrCacheFailure:
		return ErrorTypeCache
	case ErrConnectionFailure:
		return ErrorTypeConnection
	case ErrTimeout:
		return ErrorTypeTimeout
	case ErrUnauthorized:
		return ErrorTypeAuth
	case ErrRateLimited:
		return ErrorTypeRateLimit
	case ErrClientClosed, ErrInvalidOperation:
		return ErrorTypeClient
	default:
		return ErrorTypeInternal
	}
}

// isRetryableError determines if an error indicates a retryable operation
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	switch err {
	case ErrConnectionFailure, ErrTimeout, ErrRateLimited, ErrStorageFailure:
		return true
	default:
		return false
	}
}

// IsNotFoundError checks if an error is a "not found" error
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's our custom error type
	if ffErr, ok := err.(*FeatureFlagError); ok {
		return ffErr.Type == ErrorTypeNotFound || ffErr.Err == ErrFlagNotFound
	}

	// Check if it's the direct error
	return err == ErrFlagNotFound
}

// IsRetryableError checks if an error indicates a retryable operation
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's our custom error type
	if ffErr, ok := err.(*FeatureFlagError); ok {
		return ffErr.IsRetryable()
	}

	return isRetryableError(err)
}
