package core

import (
	"context"
	"testing"
	"time"
)

// Mock implementations to verify interfaces compile correctly

type mockStore struct{}

func (m *mockStore) Get(ctx context.Context, key string) (*FeatureFlag, error) {
	return nil, nil
}

func (m *mockStore) Set(ctx context.Context, flag FeatureFlag) error {
	return nil
}

func (m *mockStore) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *mockStore) GetAll(ctx context.Context) ([]FeatureFlag, error) {
	return nil, nil
}

func (m *mockStore) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *mockStore) Close() error {
	return nil
}

type mockClient struct{}

func (m *mockClient) IsEnabled(ctx context.Context, key string) (bool, error) {
	return false, nil
}

func (m *mockClient) GetFlag(ctx context.Context, key string) (*FeatureFlag, error) {
	return nil, nil
}

func (m *mockClient) SetFlag(ctx context.Context, flag FeatureFlag) error {
	return nil
}

func (m *mockClient) DeleteFlag(ctx context.Context, key string) error {
	return nil
}

func (m *mockClient) GetAllFlags(ctx context.Context) ([]FeatureFlag, error) {
	return nil, nil
}

func (m *mockClient) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *mockClient) GetMetrics() MetricsSnapshot {
	return MetricsSnapshot{
		ErrorsByType: make(map[ErrorType]int64),
		Timestamp:    time.Now(),
	}
}

func (m *mockClient) Close() error {
	return nil
}

func TestInterfaces(t *testing.T) {
	// Test that our mock implementations satisfy the interfaces
	var _ Store = &mockStore{}
	var _ Client = &mockClient{}

	// This test passes if it compiles
	t.Log("Interfaces compile correctly")
}
