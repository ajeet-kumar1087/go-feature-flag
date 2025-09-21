package cache

import (
	"context"
	"sync"
	"testing"
	"time"
)

// MockStore is a simple in-memory store for testing
type MockStore struct {
	mu          sync.RWMutex
	flags       map[string]*FeatureFlag
	getCalls    int
	setCalls    int
	deleteCalls int
}

func NewMockStore() *MockStore {
	return &MockStore{
		flags: make(map[string]*FeatureFlag),
	}
}

func (m *MockStore) Get(ctx context.Context, key string) (*FeatureFlag, error) {
	m.mu.Lock()
	m.getCalls++
	flag, exists := m.flags[key]
	m.mu.Unlock()

	if !exists {
		return nil, nil
	}
	// Return a copy to avoid mutation issues
	flagCopy := *flag
	return &flagCopy, nil
}

func (m *MockStore) Set(ctx context.Context, flag FeatureFlag) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.setCalls++
	flagCopy := flag
	m.flags[flag.Key] = &flagCopy
	return nil
}

func (m *MockStore) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.deleteCalls++
	delete(m.flags, key)
	return nil
}

func (m *MockStore) GetAll(ctx context.Context) ([]FeatureFlag, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	flags := make([]FeatureFlag, 0, len(m.flags))
	for _, flag := range m.flags {
		flags = append(flags, *flag)
	}
	return flags, nil
}

func (m *MockStore) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *MockStore) Close() error {
	return nil
}

func TestCachedStore_CacheDisabled(t *testing.T) {
	mockStore := NewMockStore()
	cacheConfig := CacheConfig{
		Enabled: false,
		TTL:     Duration(5 * time.Minute),
		MaxSize: 100,
	}

	cachedStore := NewCachedStore(mockStore, cacheConfig)
	defer cachedStore.Close()

	ctx := context.Background()
	flag := FeatureFlag{
		Key:         "test-flag",
		Enabled:     true,
		Description: "Test flag",
	}

	// Set flag
	err := cachedStore.Set(ctx, flag)
	if err != nil {
		t.Fatalf("Failed to set flag: %v", err)
	}

	// Get flag - should go directly to store
	retrieved, err := cachedStore.Get(ctx, "test-flag")
	if err != nil {
		t.Fatalf("Failed to get flag: %v", err)
	}

	if retrieved == nil || retrieved.Key != "test-flag" {
		t.Error("Failed to retrieve flag from store")
	}

	// Should have called the underlying store
	if mockStore.getCalls != 1 {
		t.Errorf("Expected 1 get call to store, got %d", mockStore.getCalls)
	}
}

func TestCachedStore_CacheEnabled(t *testing.T) {
	mockStore := NewMockStore()
	cacheConfig := CacheConfig{
		Enabled: true,
		TTL:     Duration(5 * time.Minute),
		MaxSize: 100,
	}

	cachedStore := NewCachedStore(mockStore, cacheConfig)
	defer cachedStore.Close()

	ctx := context.Background()
	flag := FeatureFlag{
		Key:         "test-flag",
		Enabled:     true,
		Description: "Test flag",
	}

	// Set flag
	err := cachedStore.Set(ctx, flag)
	if err != nil {
		t.Fatalf("Failed to set flag: %v", err)
	}

	// First get - should hit cache (flag was cached during Set)
	retrieved1, err := cachedStore.Get(ctx, "test-flag")
	if err != nil {
		t.Fatalf("Failed to get flag: %v", err)
	}

	if retrieved1 == nil || retrieved1.Key != "test-flag" {
		t.Error("Failed to retrieve flag")
	}

	// Second get - should also hit cache
	retrieved2, err := cachedStore.Get(ctx, "test-flag")
	if err != nil {
		t.Fatalf("Failed to get flag from cache: %v", err)
	}

	if retrieved2 == nil || retrieved2.Key != "test-flag" {
		t.Error("Failed to retrieve flag from cache")
	}

	// Should not have called store for gets (both cache hits)
	if mockStore.getCalls != 0 {
		t.Errorf("Expected 0 get calls to store (cache hits), got %d", mockStore.getCalls)
	}

	// Test cache miss scenario
	_, err = cachedStore.Get(ctx, "non-existent-flag")
	if err != nil {
		t.Fatalf("Failed to get non-existent flag: %v", err)
	}

	// Should have called store once for the cache miss
	if mockStore.getCalls != 1 {
		t.Errorf("Expected 1 get call to store for cache miss, got %d", mockStore.getCalls)
	}
}

func TestCachedStore_CacheInvalidationOnSet(t *testing.T) {
	mockStore := NewMockStore()
	cacheConfig := CacheConfig{
		Enabled: true,
		TTL:     Duration(5 * time.Minute),
		MaxSize: 100,
	}

	cachedStore := NewCachedStore(mockStore, cacheConfig)
	defer cachedStore.Close()

	ctx := context.Background()

	// Set initial flag
	flag1 := FeatureFlag{
		Key:         "test-flag",
		Enabled:     false,
		Description: "Initial flag",
	}
	err := cachedStore.Set(ctx, flag1)
	if err != nil {
		t.Fatalf("Failed to set initial flag: %v", err)
	}

	// Get flag to verify it's cached (should hit cache since it was cached during Set)
	retrieved1, err := cachedStore.Get(ctx, "test-flag")
	if err != nil {
		t.Fatalf("Failed to get flag: %v", err)
	}
	if retrieved1.Enabled {
		t.Error("Expected initial flag to be disabled")
	}

	// Update flag
	flag2 := FeatureFlag{
		Key:         "test-flag",
		Enabled:     true,
		Description: "Updated flag",
	}
	err = cachedStore.Set(ctx, flag2)
	if err != nil {
		t.Fatalf("Failed to update flag: %v", err)
	}

	// Get flag again - should return updated value from cache
	retrieved2, err := cachedStore.Get(ctx, "test-flag")
	if err != nil {
		t.Fatalf("Failed to get updated flag: %v", err)
	}
	if !retrieved2.Enabled || retrieved2.Description != "Updated flag" {
		t.Error("Cache was not updated with new flag value")
	}

	// Should not have called store for gets (all cache hits)
	if mockStore.getCalls != 0 {
		t.Errorf("Expected 0 get calls to store (cache hits), got %d", mockStore.getCalls)
	}
}

func TestCachedStore_CacheInvalidationOnDelete(t *testing.T) {
	mockStore := NewMockStore()
	cacheConfig := CacheConfig{
		Enabled: true,
		TTL:     Duration(5 * time.Minute),
		MaxSize: 100,
	}

	cachedStore := NewCachedStore(mockStore, cacheConfig)
	defer cachedStore.Close()

	ctx := context.Background()
	flag := FeatureFlag{
		Key:         "test-flag",
		Enabled:     true,
		Description: "Test flag",
	}

	// Set and cache flag
	err := cachedStore.Set(ctx, flag)
	if err != nil {
		t.Fatalf("Failed to set flag: %v", err)
	}

	// Get flag to ensure it's cached (should hit cache since it was cached during Set)
	_, err = cachedStore.Get(ctx, "test-flag")
	if err != nil {
		t.Fatalf("Failed to get flag: %v", err)
	}

	// Delete flag
	err = cachedStore.Delete(ctx, "test-flag")
	if err != nil {
		t.Fatalf("Failed to delete flag: %v", err)
	}

	// Get flag again - should return nil (not found) and hit store
	retrieved, err := cachedStore.Get(ctx, "test-flag")
	if err != nil {
		t.Fatalf("Failed to get deleted flag: %v", err)
	}
	if retrieved != nil {
		t.Error("Expected deleted flag to return nil")
	}

	// Should have called store once for get after delete (cache miss)
	if mockStore.getCalls != 1 {
		t.Errorf("Expected 1 get call to store after delete, got %d", mockStore.getCalls)
	}
}

func TestCachedStore_GetAll(t *testing.T) {
	mockStore := NewMockStore()
	cacheConfig := CacheConfig{
		Enabled: true,
		TTL:     Duration(5 * time.Minute),
		MaxSize: 100,
	}

	cachedStore := NewCachedStore(mockStore, cacheConfig)
	defer cachedStore.Close()

	ctx := context.Background()

	// Set multiple flags
	flags := []FeatureFlag{
		{Key: "flag-1", Enabled: true, Description: "Flag 1"},
		{Key: "flag-2", Enabled: false, Description: "Flag 2"},
		{Key: "flag-3", Enabled: true, Description: "Flag 3"},
	}

	for _, flag := range flags {
		err := cachedStore.Set(ctx, flag)
		if err != nil {
			t.Fatalf("Failed to set flag %s: %v", flag.Key, err)
		}
	}

	// Get all flags
	allFlags, err := cachedStore.GetAll(ctx)
	if err != nil {
		t.Fatalf("Failed to get all flags: %v", err)
	}

	if len(allFlags) != 3 {
		t.Errorf("Expected 3 flags, got %d", len(allFlags))
	}

	// Verify flags are now cached by getting them individually
	for _, flag := range flags {
		retrieved, err := cachedStore.Get(ctx, flag.Key)
		if err != nil {
			t.Fatalf("Failed to get cached flag %s: %v", flag.Key, err)
		}
		if retrieved == nil || retrieved.Key != flag.Key {
			t.Errorf("Flag %s not properly cached", flag.Key)
		}
	}

	// Should not have called store for individual gets (cache hits)
	if mockStore.getCalls != 0 {
		t.Errorf("Expected 0 individual get calls to store after GetAll, got %d", mockStore.getCalls)
	}
}

func TestCachedStore_HealthCheck(t *testing.T) {
	mockStore := NewMockStore()
	cacheConfig := CacheConfig{
		Enabled: true,
		TTL:     Duration(5 * time.Minute),
		MaxSize: 100,
	}

	cachedStore := NewCachedStore(mockStore, cacheConfig)
	defer cachedStore.Close()

	ctx := context.Background()
	err := cachedStore.HealthCheck(ctx)
	if err != nil {
		t.Errorf("HealthCheck failed: %v", err)
	}
}

func TestCachedStore_TTLExpiration(t *testing.T) {
	mockStore := NewMockStore()
	cacheConfig := CacheConfig{
		Enabled: true,
		TTL:     Duration(100 * time.Millisecond),
		MaxSize: 100,
	}

	cachedStore := NewCachedStore(mockStore, cacheConfig)
	defer cachedStore.Close()

	ctx := context.Background()
	flag := FeatureFlag{
		Key:         "test-flag",
		Enabled:     true,
		Description: "Test flag",
	}

	// Set flag
	err := cachedStore.Set(ctx, flag)
	if err != nil {
		t.Fatalf("Failed to set flag: %v", err)
	}

	// Get flag immediately - should hit cache
	_, err = cachedStore.Get(ctx, "test-flag")
	if err != nil {
		t.Fatalf("Failed to get flag: %v", err)
	}

	// Wait for cache expiration
	time.Sleep(150 * time.Millisecond)

	// Get flag again - should hit store due to expiration
	_, err = cachedStore.Get(ctx, "test-flag")
	if err != nil {
		t.Fatalf("Failed to get expired flag: %v", err)
	}

	// Should have called store twice (once after expiration)
	if mockStore.getCalls != 1 {
		t.Errorf("Expected 1 get call to store after expiration, got %d", mockStore.getCalls)
	}
}
