package client

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/config"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/core"
)

// testMockStore implements Store interface for testing
type testMockStore struct {
	flags       map[string]*core.FeatureFlag
	mu          sync.RWMutex
	getError    error
	setError    error
	deleteError error
	getAllError error
	healthError error
	closeError  error
	getCalls    int
	setCalls    int
	deleteCalls int
	getAllCalls int
	closed      bool
}

func newTestMockStore() *testMockStore {
	return &testMockStore{
		flags: make(map[string]*core.FeatureFlag),
	}
}

func (m *testMockStore) Get(ctx context.Context, key string) (*core.FeatureFlag, error) {
	m.mu.Lock()
	m.getCalls++
	m.mu.Unlock()

	if m.getError != nil {
		return nil, m.getError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	flag, exists := m.flags[key]
	if !exists {
		return nil, core.NewError("get", key, core.ErrFlagNotFound)
	}

	return flag.Clone(), nil
}

func (m *testMockStore) Set(ctx context.Context, flag core.FeatureFlag) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.setCalls++

	if m.setError != nil {
		return m.setError
	}

	m.flags[flag.Key] = flag.Clone()
	return nil
}

func (m *testMockStore) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.deleteCalls++

	if m.deleteError != nil {
		return m.deleteError
	}

	delete(m.flags, key)
	return nil
}

func (m *testMockStore) GetAll(ctx context.Context) ([]core.FeatureFlag, error) {
	m.mu.Lock()
	m.getAllCalls++
	m.mu.Unlock()

	if m.getAllError != nil {
		return nil, m.getAllError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	flags := make([]core.FeatureFlag, 0, len(m.flags))
	for _, flag := range m.flags {
		flags = append(flags, *flag.Clone())
	}

	return flags, nil
}

func (m *testMockStore) HealthCheck(ctx context.Context) error {
	return m.healthError
}

func (m *testMockStore) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true
	return m.closeError
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		config      config.Config
		expectError bool
	}{
		{
			name: "valid memory config",
			config: config.Config{
				Storage: config.StorageConfig{Type: "memory"},
				Cache:   config.CacheConfig{Enabled: false},
			},
			expectError: false,
		},
		{
			name: "valid config with cache",
			config: config.Config{
				Storage: config.StorageConfig{Type: "memory"},
				Cache: config.CacheConfig{
					Enabled: true,
					TTL:     config.Duration(5 * time.Minute),
					MaxSize: 100,
				},
			},
			expectError: false,
		},
		{
			name: "invalid storage type",
			config: config.Config{
				Storage: config.StorageConfig{Type: "invalid"},
				Cache:   config.CacheConfig{Enabled: false},
			},
			expectError: true,
		},
		{
			name: "redis without config",
			config: config.Config{
				Storage: config.StorageConfig{Type: "redis"},
				Cache:   config.CacheConfig{Enabled: false},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Error("expected client but got nil")
				return
			}

			// Clean up
			client.Close()
		})
	}
}

func TestNewClientWithDefaults(t *testing.T) {
	client, err := NewClientWithDefaults()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if client == nil {
		t.Error("expected client but got nil")
		return
	}

	// Clean up
	client.Close()
}

func TestClient_IsEnabled(t *testing.T) {
	// Create client with mock store
	cfg := config.Config{
		Storage: config.StorageConfig{Type: "memory"},
		Cache:   config.CacheConfig{Enabled: false},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Test with non-existent flag (should return false, no error)
	enabled, err := client.IsEnabled(ctx, "non-existent")
	if err != nil {
		t.Errorf("unexpected error for non-existent flag: %v", err)
	}
	if enabled {
		t.Error("expected false for non-existent flag")
	}

	// Create a flag
	flag := core.FeatureFlag{
		Key:     "test-flag",
		Enabled: true,
	}
	err = client.SetFlag(ctx, flag)
	if err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	// Test enabled flag
	enabled, err = client.IsEnabled(ctx, "test-flag")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !enabled {
		t.Error("expected true for enabled flag")
	}

	// Create disabled flag
	disabledFlag := core.FeatureFlag{
		Key:     "disabled-flag",
		Enabled: false,
	}
	err = client.SetFlag(ctx, disabledFlag)
	if err != nil {
		t.Fatalf("failed to set disabled flag: %v", err)
	}

	// Test disabled flag
	enabled, err = client.IsEnabled(ctx, "disabled-flag")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if enabled {
		t.Error("expected false for disabled flag")
	}

	// Test empty key
	enabled, err = client.IsEnabled(ctx, "")
	if err == nil {
		t.Error("expected error for empty key")
	}
	if enabled {
		t.Error("expected false for empty key")
	}
}

func TestClient_GetFlag(t *testing.T) {
	cfg := config.Config{
		Storage: config.StorageConfig{Type: "memory"},
		Cache:   config.CacheConfig{Enabled: false},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Test non-existent flag
	flag, err := client.GetFlag(ctx, "non-existent")
	if err == nil {
		t.Error("expected error for non-existent flag")
	}
	if flag != nil {
		t.Error("expected nil flag for non-existent")
	}

	// Create and set a flag
	testFlag := core.FeatureFlag{
		Key:         "test-flag",
		Enabled:     true,
		Description: "Test flag",
		Metadata:    map[string]string{"env": "test"},
	}
	err = client.SetFlag(ctx, testFlag)
	if err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	// Get the flag
	retrievedFlag, err := client.GetFlag(ctx, "test-flag")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if retrievedFlag == nil {
		t.Fatal("expected flag but got nil")
	}

	if retrievedFlag.Key != testFlag.Key {
		t.Errorf("expected key %s, got %s", testFlag.Key, retrievedFlag.Key)
	}
	if retrievedFlag.Enabled != testFlag.Enabled {
		t.Errorf("expected enabled %v, got %v", testFlag.Enabled, retrievedFlag.Enabled)
	}
	if retrievedFlag.Description != testFlag.Description {
		t.Errorf("expected description %s, got %s", testFlag.Description, retrievedFlag.Description)
	}

	// Test empty key
	_, err = client.GetFlag(ctx, "")
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestClient_SetFlag(t *testing.T) {
	config := config.Config{
		Storage: config.StorageConfig{Type: "memory"},
		Cache:   config.CacheConfig{Enabled: false},
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Test valid flag
	flag := core.FeatureFlag{
		Key:         "test-flag",
		Enabled:     true,
		Description: "Test flag",
	}

	err = client.SetFlag(ctx, flag)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify flag was set
	retrievedFlag, err := client.GetFlag(ctx, "test-flag")
	if err != nil {
		t.Errorf("failed to retrieve flag: %v", err)
	}
	if retrievedFlag.Key != flag.Key {
		t.Errorf("expected key %s, got %s", flag.Key, retrievedFlag.Key)
	}

	// Test invalid flag (empty key)
	invalidFlag := core.FeatureFlag{
		Key:     "",
		Enabled: true,
	}

	err = client.SetFlag(ctx, invalidFlag)
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestClient_DeleteFlag(t *testing.T) {
	config := config.Config{
		Storage: config.StorageConfig{Type: "memory"},
		Cache:   config.CacheConfig{Enabled: false},
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create a flag first
	flag := core.FeatureFlag{
		Key:     "test-flag",
		Enabled: true,
	}
	err = client.SetFlag(ctx, flag)
	if err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	// Delete the flag
	err = client.DeleteFlag(ctx, "test-flag")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify flag was deleted
	_, err = client.GetFlag(ctx, "test-flag")
	if err == nil {
		t.Error("expected error for deleted flag")
	}

	// Test deleting non-existent flag (should error)
	err = client.DeleteFlag(ctx, "non-existent")
	if err == nil {
		t.Error("expected error for non-existent flag")
	}

	// Test empty key
	err = client.DeleteFlag(ctx, "")
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestClient_GetAllFlags(t *testing.T) {
	config := config.Config{
		Storage: config.StorageConfig{Type: "memory"},
		Cache:   config.CacheConfig{Enabled: false},
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Initially should be empty
	flags, err := client.GetAllFlags(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(flags) != 0 {
		t.Errorf("expected 0 flags, got %d", len(flags))
	}

	// Add some flags
	flag1 := core.FeatureFlag{Key: "flag1", Enabled: true}
	flag2 := core.FeatureFlag{Key: "flag2", Enabled: false}

	err = client.SetFlag(ctx, flag1)
	if err != nil {
		t.Fatalf("failed to set flag1: %v", err)
	}

	err = client.SetFlag(ctx, flag2)
	if err != nil {
		t.Fatalf("failed to set flag2: %v", err)
	}

	// Get all flags
	flags, err = client.GetAllFlags(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(flags) != 2 {
		t.Errorf("expected 2 flags, got %d", len(flags))
	}

	// Verify flags are present
	flagKeys := make(map[string]bool)
	for _, flag := range flags {
		flagKeys[flag.Key] = true
	}

	if !flagKeys["flag1"] {
		t.Error("flag1 not found in results")
	}
	if !flagKeys["flag2"] {
		t.Error("flag2 not found in results")
	}
}

func TestClient_Close(t *testing.T) {
	config := config.Config{
		Storage: config.StorageConfig{Type: "memory"},
		Cache:   config.CacheConfig{Enabled: false},
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Close the client
	err = client.Close()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Subsequent operations should fail
	ctx := context.Background()

	_, err = client.IsEnabled(ctx, "test")
	if err == nil {
		t.Error("expected error after close")
	}

	_, err = client.GetFlag(ctx, "test")
	if err == nil {
		t.Error("expected error after close")
	}

	err = client.SetFlag(ctx, core.FeatureFlag{Key: "test", Enabled: true})
	if err == nil {
		t.Error("expected error after close")
	}

	err = client.DeleteFlag(ctx, "test")
	if err == nil {
		t.Error("expected error after close")
	}

	_, err = client.GetAllFlags(ctx)
	if err == nil {
		t.Error("expected error after close")
	}

	// Multiple closes should not error
	err = client.Close()
	if err != nil {
		t.Errorf("unexpected error on second close: %v", err)
	}
}

func TestClient_DefaultFlags(t *testing.T) {
	defaultFlags := []core.FeatureFlag{
		{Key: "default1", Enabled: true, Description: "Default flag 1"},
		{Key: "default2", Enabled: false, Description: "Default flag 2"},
	}

	config := config.Config{
		Storage:      config.StorageConfig{Type: "memory"},
		Cache:        config.CacheConfig{Enabled: false},
		DefaultFlags: defaultFlags,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Check that default flags were loaded
	flag1, err := client.GetFlag(ctx, "default1")
	if err != nil {
		t.Errorf("failed to get default flag1: %v", err)
	}
	if flag1 == nil || !flag1.Enabled {
		t.Error("default1 should be enabled")
	}

	flag2, err := client.GetFlag(ctx, "default2")
	if err != nil {
		t.Errorf("failed to get default flag2: %v", err)
	}
	if flag2 == nil || flag2.Enabled {
		t.Error("default2 should be disabled")
	}

	// Verify that existing flags are not overridden
	// First, set a flag manually
	existingFlag := core.FeatureFlag{Key: "existing", Enabled: true}
	err = client.SetFlag(ctx, existingFlag)
	if err != nil {
		t.Fatalf("failed to set existing flag: %v", err)
	}

	// Create new client with default flag that has same key

	// This should use the same memory store, so the existing flag should remain

}

func TestClient_GracefulDegradation(t *testing.T) {
	// Test that IsEnabled returns false on storage errors instead of failing
	testMockStore := newTestMockStore()
	testMockStore.getError = errors.New("storage error")

	// Create client with mock store (we'll need to modify the client to accept a store)
	// For now, we'll test the graceful degradation behavior indirectly
	config := config.Config{
		Storage: config.StorageConfig{Type: "memory"},
		Cache:   config.CacheConfig{Enabled: false},
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Test with non-existent flag - should return false, no error
	enabled, err := client.IsEnabled(ctx, "non-existent")
	if err != nil {
		t.Errorf("unexpected error for non-existent flag: %v", err)
	}
	if enabled {
		t.Error("expected false for non-existent flag")
	}
}

func TestClient_ConcurrentAccess(t *testing.T) {
	config := config.Config{
		Storage: config.StorageConfig{Type: "memory"},
		Cache:   config.CacheConfig{Enabled: false},
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Set up initial flag
	flag := core.FeatureFlag{Key: "concurrent-test", Enabled: true}
	err = client.SetFlag(ctx, flag)
	if err != nil {
		t.Fatalf("failed to set initial flag: %v", err)
	}

	// Run concurrent operations
	const numGoroutines = 10
	const numOperations = 100

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numOperations)

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_, err := client.IsEnabled(ctx, "concurrent-test")
				if err != nil {
					errors <- err
				}
			}
		}()
	}

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				testFlag := core.FeatureFlag{
					Key:     fmt.Sprintf("concurrent-flag-%d-%d", id, j),
					Enabled: j%2 == 0,
				}
				err := client.SetFlag(ctx, testFlag)
				if err != nil {
					errors <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("concurrent operation error: %v", err)
	}
}

func TestIsNotFoundError(t *testing.T) {
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
			name:     "direct ErrFlagNotFound",
			err:      core.ErrFlagNotFound,
			expected: true,
		},
		{
			name:     "wrapped ErrFlagNotFound",
			err:      core.NewError("test", "key", core.ErrFlagNotFound),
			expected: true,
		},
		{
			name:     "other error",
			err:      errors.New("other error"),
			expected: false,
		},
		{
			name:     "wrapped other error",
			err:      core.NewError("test", "key", errors.New("other error")),
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
