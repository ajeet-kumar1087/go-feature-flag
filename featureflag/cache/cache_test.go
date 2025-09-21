package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/core"
)

func TestCache_BasicOperations(t *testing.T) {
	cache := NewCache(10, 5*time.Minute)
	defer cache.Close()

	flag := &core.FeatureFlag{
		Key:         "test-flag",
		Enabled:     true,
		Description: "Test flag",
	}

	// Test Set and Get
	cache.Set("test-flag", flag)
	retrieved, found := cache.Get("test-flag")
	if !found {
		t.Error("Expected to find cached flag")
	}
	if retrieved.Key != flag.Key || retrieved.Enabled != flag.Enabled {
		t.Error("Retrieved flag doesn't match original")
	}

	// Test Size
	if cache.Size() != 1 {
		t.Errorf("Expected cache size 1, got %d", cache.Size())
	}

	// Test Delete
	cache.Delete("test-flag")
	_, found = cache.Get("test-flag")
	if found {
		t.Error("Expected flag to be deleted from cache")
	}

	if cache.Size() != 0 {
		t.Errorf("Expected cache size 0 after delete, got %d", cache.Size())
	}
}

func TestCache_TTLExpiration(t *testing.T) {
	cache := NewCache(10, 100*time.Millisecond)
	defer cache.Close()

	flag := &core.FeatureFlag{
		Key:     "test-flag",
		Enabled: true,
	}

	// Set flag
	cache.Set("test-flag", flag)

	// Should be available immediately
	_, found := cache.Get("test-flag")
	if !found {
		t.Error("Expected to find cached flag immediately after set")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired now
	_, found = cache.Get("test-flag")
	if found {
		t.Error("Expected flag to be expired")
	}
}

func TestCache_LRUEviction(t *testing.T) {
	cache := NewCache(3, 0) // No TTL, only LRU eviction
	defer cache.Close()

	// Fill cache to capacity
	for i := 1; i <= 3; i++ {
		flag := &core.FeatureFlag{
			Key:     fmt.Sprintf("flag-%d", i),
			Enabled: true,
		}
		cache.Set(flag.Key, flag)
	}

	// All flags should be present
	for i := 1; i <= 3; i++ {
		key := fmt.Sprintf("flag-%d", i)
		_, found := cache.Get(key)
		if !found {
			t.Errorf("Expected to find flag %s", key)
		}
	}

	// Add one more flag - should evict the least recently used (flag-1)
	flag4 := &core.FeatureFlag{
		Key:     "flag-4",
		Enabled: true,
	}
	cache.Set("flag-4", flag4)

	// flag-1 should be evicted
	_, found := cache.Get("flag-1")
	if found {
		t.Error("Expected flag-1 to be evicted")
	}

	// Other flags should still be present
	for i := 2; i <= 4; i++ {
		key := fmt.Sprintf("flag-%d", i)
		_, found := cache.Get(key)
		if !found {
			t.Errorf("Expected to find flag %s", key)
		}
	}

	if cache.Size() != 3 {
		t.Errorf("Expected cache size 3, got %d", cache.Size())
	}
}

func TestCache_UpdateExisting(t *testing.T) {
	cache := NewCache(10, 5*time.Minute)
	defer cache.Close()

	// Set initial flag
	flag1 := &core.FeatureFlag{
		Key:         "test-flag",
		Enabled:     false,
		Description: "Initial",
	}
	cache.Set("test-flag", flag1)

	// Update the flag
	flag2 := &core.FeatureFlag{
		Key:         "test-flag",
		Enabled:     true,
		Description: "Updated",
	}
	cache.Set("test-flag", flag2)

	// Should have updated value
	retrieved, found := cache.Get("test-flag")
	if !found {
		t.Error("Expected to find cached flag")
	}
	if !retrieved.Enabled || retrieved.Description != "Updated" {
		t.Error("Flag was not updated correctly")
	}

	// Size should still be 1
	if cache.Size() != 1 {
		t.Errorf("Expected cache size 1, got %d", cache.Size())
	}
}

func TestCache_Clear(t *testing.T) {
	cache := NewCache(10, 5*time.Minute)
	defer cache.Close()

	// Add multiple flags
	for i := 1; i <= 5; i++ {
		flag := &core.FeatureFlag{
			Key:     fmt.Sprintf("flag-%d", i),
			Enabled: true,
		}
		cache.Set(flag.Key, flag)
	}

	if cache.Size() != 5 {
		t.Errorf("Expected cache size 5, got %d", cache.Size())
	}

	// Clear cache
	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Expected cache size 0 after clear, got %d", cache.Size())
	}

	// All flags should be gone
	for i := 1; i <= 5; i++ {
		key := fmt.Sprintf("flag-%d", i)
		_, found := cache.Get(key)
		if found {
			t.Errorf("Expected flag %s to be cleared", key)
		}
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	cache := NewCache(1000, 5*time.Minute) // Larger cache to avoid eviction issues
	defer cache.Close()

	const numGoroutines = 10
	const numOperations = 50 // Reduced to avoid eviction

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Start multiple goroutines performing concurrent operations
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("flag-%d-%d", goroutineID, j)
				flag := &core.FeatureFlag{
					Key:     key,
					Enabled: j%2 == 0,
				}

				// Set flag
				cache.Set(key, flag)

				// Get flag immediately after setting
				retrieved, found := cache.Get(key)
				if !found {
					t.Errorf("Expected to find flag %s immediately after set", key)
					continue
				}

				if retrieved.Key != key {
					t.Errorf("Retrieved flag key mismatch: expected %s, got %s", key, retrieved.Key)
				}

				// Only delete every 20th flag to reduce contention
				if j%20 == 0 {
					cache.Delete(key)
					// Verify deletion
					_, found := cache.Get(key)
					if found {
						t.Errorf("Expected flag %s to be deleted", key)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	// Cache should still be functional
	testFlag := &core.FeatureFlag{
		Key:     "final-test",
		Enabled: true,
	}
	cache.Set("final-test", testFlag)

	retrieved, found := cache.Get("final-test")
	if !found || retrieved.Key != "final-test" {
		t.Error("Cache not functional after concurrent access")
	}
}

func TestCache_ZeroTTL(t *testing.T) {
	cache := NewCache(10, 0) // No TTL
	defer cache.Close()

	flag := &core.FeatureFlag{
		Key:     "test-flag",
		Enabled: true,
	}

	cache.Set("test-flag", flag)

	// Should be available even after some time
	time.Sleep(100 * time.Millisecond)
	_, found := cache.Get("test-flag")
	if !found {
		t.Error("Expected flag to persist with zero TTL")
	}
}

func TestCache_ZeroMaxSize(t *testing.T) {
	cache := NewCache(0, 5*time.Minute) // No size limit
	defer cache.Close()

	// Add many flags
	for i := 0; i < 1000; i++ {
		flag := &core.FeatureFlag{
			Key:     fmt.Sprintf("flag-%d", i),
			Enabled: true,
		}
		cache.Set(flag.Key, flag)
	}

	// All flags should be present
	if cache.Size() != 1000 {
		t.Errorf("Expected cache size 1000, got %d", cache.Size())
	}

	// Check a few random flags
	for i := 0; i < 1000; i += 100 {
		key := fmt.Sprintf("flag-%d", i)
		_, found := cache.Get(key)
		if !found {
			t.Errorf("Expected to find flag %s", key)
		}
	}
}

func TestCache_ExpiredItemCleanup(t *testing.T) {
	cache := NewCache(10, 50*time.Millisecond)
	defer cache.Close()

	// Add flags that will expire
	for i := 0; i < 5; i++ {
		flag := &core.FeatureFlag{
			Key:     fmt.Sprintf("flag-%d", i),
			Enabled: true,
		}
		cache.Set(flag.Key, flag)
	}

	if cache.Size() != 5 {
		t.Errorf("Expected cache size 5, got %d", cache.Size())
	}

	// Wait for expiration and cleanup
	time.Sleep(150 * time.Millisecond)

	// Size should eventually be 0 after cleanup
	// We need to wait a bit for the cleanup goroutine to run
	for i := 0; i < 10; i++ {
		if cache.Size() == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if cache.Size() != 0 {
		t.Errorf("Expected cache size 0 after cleanup, got %d", cache.Size())
	}
}
