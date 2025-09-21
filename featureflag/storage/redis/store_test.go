package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/config"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/core"
	"github.com/redis/go-redis/v9"
)

// mockRedisClient implements a simple mock for Redis client
type mockRedisClient struct {
	data    map[string]string
	mu      sync.RWMutex
	closed  bool
	pingErr error
	getErr  error
	setErr  error
	delErr  error
	keysErr error
}

func newMockRedisClient() *mockRedisClient {
	return &mockRedisClient{
		data: make(map[string]string),
	}
}

func (m *mockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx, "get", key)
	if m.getErr != nil {
		cmd.SetErr(m.getErr)
		return cmd
	}

	m.mu.RLock()
	value, exists := m.data[key]
	m.mu.RUnlock()

	if !exists {
		cmd.SetErr(redis.Nil)
		return cmd
	}

	cmd.SetVal(value)
	return cmd
}

func (m *mockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx, "set", key, value)
	if m.setErr != nil {
		cmd.SetErr(m.setErr)
		return cmd
	}

	// Handle different value types
	var strValue string
	switch v := value.(type) {
	case string:
		strValue = v
	case []byte:
		strValue = string(v)
	default:
		strValue = fmt.Sprintf("%v", v)
	}

	m.mu.Lock()
	m.data[key] = strValue
	m.mu.Unlock()

	cmd.SetVal("OK")
	return cmd
}

func (m *mockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx, "del")
	if m.delErr != nil {
		cmd.SetErr(m.delErr)
		return cmd
	}

	m.mu.Lock()
	deleted := int64(0)
	for _, key := range keys {
		if _, exists := m.data[key]; exists {
			delete(m.data, key)
			deleted++
		}
	}
	m.mu.Unlock()

	cmd.SetVal(deleted)
	return cmd
}

func (m *mockRedisClient) Keys(ctx context.Context, pattern string) *redis.StringSliceCmd {
	cmd := redis.NewStringSliceCmd(ctx, "keys", pattern)
	if m.keysErr != nil {
		cmd.SetErr(m.keysErr)
		return cmd
	}

	m.mu.RLock()
	var keys []string
	for key := range m.data {
		// Simple pattern matching for "prefix*"
		if pattern == "featureflag:*" && len(key) >= 12 && key[:12] == "featureflag:" {
			keys = append(keys, key)
		}
	}
	m.mu.RUnlock()

	cmd.SetVal(keys)
	return cmd
}

func (m *mockRedisClient) Pipeline() redis.Pipeliner {
	// For testing, we'll return nil and handle this in the test
	// This is a limitation of our mock, but acceptable for unit tests
	return nil
}

func (m *mockRedisClient) Ping(ctx context.Context) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx, "ping")
	if m.pingErr != nil {
		cmd.SetErr(m.pingErr)
		return cmd
	}
	cmd.SetVal("PONG")
	return cmd
}

func (m *mockRedisClient) Close() error {
	m.closed = true
	return nil
}

// createMockRedisStore creates a RedisStore with a mock client for testing
func createMockRedisStore() *RedisStore {
	return &RedisStore{
		client: newMockRedisClient(),
		prefix: "featureflag:",
	}
}

func TestNewRedisStore(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.RedisConfig
		expectError bool
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
		},
		{
			name: "invalid config - empty addr",
			config: &config.RedisConfig{
				Addr: "",
				DB:   0,
			},
			expectError: true,
		},
		{
			name: "invalid config - invalid DB",
			config: &config.RedisConfig{
				Addr: "localhost:6379",
				DB:   16,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewStore(tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				if store != nil {
					t.Error("expected nil store but got non-nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if store == nil {
					t.Error("expected non-nil store but got nil")
				}
			}
		})
	}
}

func TestRedisStore_Get(t *testing.T) {
	store := createMockRedisStore()
	mockClient := store.client.(*mockRedisClient)
	ctx := context.Background()

	// Test data
	testFlag := core.FeatureFlag{
		Key:         "test-flag",
		Enabled:     true,
		Description: "Test flag",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	flagData, _ := json.Marshal(testFlag)
	mockClient.data["featureflag:test-flag"] = string(flagData)

	tests := []struct {
		name        string
		key         string
		setupMock   func()
		expectError bool
		expectFlag  *core.FeatureFlag
	}{
		{
			name:        "successful get",
			key:         "test-flag",
			setupMock:   func() {},
			expectError: false,
			expectFlag:  &testFlag,
		},
		{
			name:        "empty key",
			key:         "",
			setupMock:   func() {},
			expectError: true,
			expectFlag:  nil,
		},
		{
			name: "flag not found",
			key:  "nonexistent",
			setupMock: func() {
				mockClient.getErr = redis.Nil
			},
			expectError: true,
			expectFlag:  nil,
		},
		{
			name: "redis error",
			key:  "test-flag",
			setupMock: func() {
				mockClient.getErr = errors.New("redis connection error")
			},
			expectError: true,
			expectFlag:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock state
			mockClient.getErr = nil
			tt.setupMock()

			flag, err := store.Get(ctx, tt.key)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if flag == nil {
					t.Error("expected flag but got nil")
				} else if flag.Key != tt.expectFlag.Key {
					t.Errorf("expected flag key %s, got %s", tt.expectFlag.Key, flag.Key)
				}
			}
		})
	}
}

func TestRedisStore_Set(t *testing.T) {
	store := createMockRedisStore()
	mockClient := store.client.(*mockRedisClient)
	ctx := context.Background()

	tests := []struct {
		name        string
		flag        core.FeatureFlag
		setupMock   func()
		expectError bool
	}{
		{
			name: "successful set",
			flag: core.FeatureFlag{
				Key:         "test-flag",
				Enabled:     true,
				Description: "Test flag",
			},
			setupMock:   func() {},
			expectError: false,
		},
		{
			name: "invalid flag - empty key",
			flag: core.FeatureFlag{
				Key:     "",
				Enabled: true,
			},
			setupMock:   func() {},
			expectError: true,
		},
		{
			name: "redis error",
			flag: core.FeatureFlag{
				Key:         "test-flag",
				Enabled:     true,
				Description: "Test flag",
			},
			setupMock: func() {
				mockClient.setErr = errors.New("redis connection error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock state
			mockClient.setErr = nil
			tt.setupMock()

			err := store.Set(ctx, tt.flag)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				// Verify flag was stored
				if tt.flag.Key != "" {
					redisKey := "featureflag:" + tt.flag.Key
					if _, exists := mockClient.data[redisKey]; !exists {
						t.Error("flag was not stored in mock Redis")
					}
				}
			}
		})
	}
}

func TestRedisStore_Delete(t *testing.T) {
	store := createMockRedisStore()
	mockClient := store.client.(*mockRedisClient)
	ctx := context.Background()

	// Setup test data
	mockClient.data["featureflag:test-flag"] = `{"key":"test-flag","enabled":true}`

	tests := []struct {
		name        string
		key         string
		setupMock   func()
		expectError bool
	}{
		{
			name:        "successful delete",
			key:         "test-flag",
			setupMock:   func() {},
			expectError: false,
		},
		{
			name:        "empty key",
			key:         "",
			setupMock:   func() {},
			expectError: true,
		},
		{
			name:        "flag not found",
			key:         "nonexistent",
			setupMock:   func() {},
			expectError: true,
		},
		{
			name: "redis error",
			key:  "test-flag",
			setupMock: func() {
				mockClient.delErr = errors.New("redis connection error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock state
			mockClient.delErr = nil
			tt.setupMock()

			err := store.Delete(ctx, tt.key)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRedisStore_GetAll(t *testing.T) {
	store := createMockRedisStore()
	mockClient := store.client.(*mockRedisClient)
	ctx := context.Background()

	// Setup test data
	flag1 := core.FeatureFlag{Key: "flag1", Enabled: true}
	flag2 := core.FeatureFlag{Key: "flag2", Enabled: false}

	flag1Data, _ := json.Marshal(flag1)
	flag2Data, _ := json.Marshal(flag2)

	mockClient.data["featureflag:flag1"] = string(flag1Data)
	mockClient.data["featureflag:flag2"] = string(flag2Data)

	tests := []struct {
		name        string
		setupMock   func()
		expectError bool
		expectCount int
	}{
		{
			name:        "successful get all",
			setupMock:   func() {},
			expectError: false,
			expectCount: 2,
		},
		{
			name: "keys error",
			setupMock: func() {
				mockClient.keysErr = errors.New("redis connection error")
			},
			expectError: true,
			expectCount: 0,
		},
		{
			name: "no flags",
			setupMock: func() {
				mockClient.data = make(map[string]string)
			},
			expectError: false,
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock state
			mockClient.keysErr = nil
			tt.setupMock()

			flags, err := store.GetAll(ctx)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(flags) != tt.expectCount {
					t.Errorf("expected %d flags, got %d", tt.expectCount, len(flags))
				}
			}
		})
	}
}

func TestRedisStore_HealthCheck(t *testing.T) {
	store := createMockRedisStore()
	mockClient := store.client.(*mockRedisClient)
	ctx := context.Background()

	tests := []struct {
		name        string
		setupMock   func()
		expectError bool
	}{
		{
			name:        "successful health check",
			setupMock:   func() {},
			expectError: false,
		},
		{
			name: "ping error",
			setupMock: func() {
				mockClient.pingErr = errors.New("redis connection error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock state
			mockClient.pingErr = nil
			tt.setupMock()

			err := store.HealthCheck(ctx)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRedisStore_Close(t *testing.T) {
	store := createMockRedisStore()
	mockClient := store.client.(*mockRedisClient)

	err := store.Close()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !mockClient.closed {
		t.Error("expected client to be closed")
	}
}

// Test concurrent access to Redis store
func TestRedisStore_ConcurrentAccess(t *testing.T) {
	store := createMockRedisStore()
	ctx := context.Background()

	// Test concurrent writes
	t.Run("concurrent writes", func(t *testing.T) {
		const numGoroutines = 10
		const numOperations = 100

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					flag := core.FeatureFlag{
						Key:         fmt.Sprintf("concurrent-flag-%d-%d", id, j),
						Enabled:     j%2 == 0,
						Description: fmt.Sprintf("Concurrent test flag %d-%d", id, j),
					}
					err := store.Set(ctx, flag)
					if err != nil {
						t.Errorf("failed to set flag: %v", err)
					}
				}
			}(i)
		}

		wg.Wait()
	})

	// Test concurrent reads
	t.Run("concurrent reads", func(t *testing.T) {
		// First, set up some test data
		testFlag := core.FeatureFlag{
			Key:         "read-test-flag",
			Enabled:     true,
			Description: "Flag for concurrent read testing",
		}
		err := store.Set(ctx, testFlag)
		if err != nil {
			t.Fatalf("failed to set test flag: %v", err)
		}

		const numGoroutines = 10
		const numReads = 100

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < numReads; j++ {
					flag, err := store.Get(ctx, testFlag.Key)
					if err != nil {
						t.Errorf("failed to get flag: %v", err)
						return
					}
					if flag.Key != testFlag.Key {
						t.Errorf("unexpected flag key: got %s, want %s", flag.Key, testFlag.Key)
					}
				}
			}()
		}

		wg.Wait()
	})
}

// Test Redis store with invalid JSON data
func TestRedisStore_InvalidJSONHandling(t *testing.T) {
	store := createMockRedisStore()
	mockClient := store.client.(*mockRedisClient)
	ctx := context.Background()

	// Set invalid JSON data directly in mock
	mockClient.data["featureflag:invalid-json"] = "invalid json data"

	_, err := store.Get(ctx, "invalid-json")
	if err == nil {
		t.Error("expected error when getting flag with invalid JSON")
	}

	// Verify error contains deserialization information
	if !strings.Contains(err.Error(), "deserialize") {
		t.Errorf("expected deserialization error, got: %v", err)
	}
}

// Test Redis store with metadata
func TestRedisStore_MetadataHandling(t *testing.T) {
	store := createMockRedisStore()
	ctx := context.Background()

	testFlag := core.FeatureFlag{
		Key:         "metadata-test",
		Enabled:     true,
		Description: "Flag with metadata",
		Metadata: map[string]string{
			"environment": "test",
			"version":     "1.0.0",
			"owner":       "team-alpha",
		},
	}

	// Test Set with metadata
	err := store.Set(ctx, testFlag)
	if err != nil {
		t.Fatalf("failed to set flag with metadata: %v", err)
	}

	// Test Get with metadata
	retrievedFlag, err := store.Get(ctx, testFlag.Key)
	if err != nil {
		t.Fatalf("failed to get flag with metadata: %v", err)
	}

	// Verify metadata is preserved
	if len(retrievedFlag.Metadata) != len(testFlag.Metadata) {
		t.Errorf("metadata length mismatch: got %d, want %d", len(retrievedFlag.Metadata), len(testFlag.Metadata))
	}

	for key, expectedValue := range testFlag.Metadata {
		if actualValue, exists := retrievedFlag.Metadata[key]; !exists {
			t.Errorf("metadata key %s not found", key)
		} else if actualValue != expectedValue {
			t.Errorf("metadata value mismatch for key %s: got %s, want %s", key, actualValue, expectedValue)
		}
	}
}

// Integration test helper - only runs if Redis is available
func TestRedisStore_Integration(t *testing.T) {
	// Skip integration test in unit test mode
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := &config.RedisConfig{
		Addr: "localhost:6379",
		DB:   1, // Use DB 1 for testing
	}

	store, err := NewStore(config)
	if err != nil {
		t.Skipf("Redis not available for integration test: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Test basic operations
	testFlag := core.FeatureFlag{
		Key:         "integration-test",
		Enabled:     true,
		Description: "Integration test flag",
	}

	// Test Set
	err = store.Set(ctx, testFlag)
	if err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	// Test Get
	retrievedFlag, err := store.Get(ctx, testFlag.Key)
	if err != nil {
		t.Fatalf("failed to get flag: %v", err)
	}

	if retrievedFlag.Key != testFlag.Key || retrievedFlag.Enabled != testFlag.Enabled {
		t.Errorf("retrieved flag doesn't match: expected %+v, got %+v", testFlag, retrievedFlag)
	}

	// Test GetAll
	flags, err := store.GetAll(ctx)
	if err != nil {
		t.Fatalf("failed to get all flags: %v", err)
	}

	found := false
	for _, flag := range flags {
		if flag.Key == testFlag.Key {
			found = true
			break
		}
	}
	if !found {
		t.Error("test flag not found in GetAll results")
	}

	// Test Delete
	err = store.Delete(ctx, testFlag.Key)
	if err != nil {
		t.Fatalf("failed to delete flag: %v", err)
	}

	// Verify deletion
	_, err = store.Get(ctx, testFlag.Key)
	if err == nil {
		t.Error("expected error when getting deleted flag")
	}
}
