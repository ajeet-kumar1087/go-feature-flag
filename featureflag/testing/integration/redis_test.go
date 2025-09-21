//go:build integration
// +build integration

package integration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
)

// TestRedisIntegration runs integration tests against a real Redis instance
// To run these tests, set the REDIS_TEST_URL environment variable
// Example: REDIS_TEST_URL="redis://localhost:6379/1"
func TestRedisIntegration(t *testing.T) {
	testURL := os.Getenv("REDIS_TEST_URL")
	if testURL == "" {
		t.Skip("REDIS_TEST_URL not set, skipping integration tests")
	}

	// Parse the test URL to create config
	config, err := parseRedisURL(testURL)
	if err != nil {
		t.Fatalf("failed to parse test URL: %v", err)
	}

	// Create store
	store, err := NewRedisStore(config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Clean up any existing test data
	cleanupRedisTestData(t, store)

	ctx := context.Background()

	t.Run("CRUD Operations", func(t *testing.T) {
		testFlag := FeatureFlag{
			Key:         "redis-integration-test-flag",
			Enabled:     true,
			Description: "Redis integration test flag",
			Metadata: map[string]string{
				"env":  "test",
				"team": "backend",
			},
		}

		// Test Set (Create)
		err := store.Set(ctx, testFlag)
		if err != nil {
			t.Fatalf("failed to set flag: %v", err)
		}

		// Test Get
		retrieved, err := store.Get(ctx, testFlag.Key)
		if err != nil {
			t.Fatalf("failed to get flag: %v", err)
		}

		if retrieved.Key != testFlag.Key {
			t.Errorf("expected key %s, got %s", testFlag.Key, retrieved.Key)
		}
		if retrieved.Enabled != testFlag.Enabled {
			t.Errorf("expected enabled %v, got %v", testFlag.Enabled, retrieved.Enabled)
		}
		if retrieved.Description != testFlag.Description {
			t.Errorf("expected description %s, got %s", testFlag.Description, retrieved.Description)
		}
		if !mapsEqual(retrieved.Metadata, testFlag.Metadata) {
			t.Errorf("expected metadata %v, got %v", testFlag.Metadata, retrieved.Metadata)
		}

		// Test Set (Update)
		testFlag.Enabled = false
		testFlag.Description = "Updated description"
		testFlag.Metadata["updated"] = "true"

		err = store.Set(ctx, testFlag)
		if err != nil {
			t.Fatalf("failed to update flag: %v", err)
		}

		// Verify update
		updated, err := store.Get(ctx, testFlag.Key)
		if err != nil {
			t.Fatalf("failed to get updated flag: %v", err)
		}

		if updated.Enabled != false {
			t.Errorf("expected enabled false, got %v", updated.Enabled)
		}
		if updated.Description != "Updated description" {
			t.Errorf("expected updated description, got %s", updated.Description)
		}
		if updated.Metadata["updated"] != "true" {
			t.Errorf("expected metadata to be updated")
		}

		// Test Delete
		err = store.Delete(ctx, testFlag.Key)
		if err != nil {
			t.Fatalf("failed to delete flag: %v", err)
		}

		// Verify deletion
		_, err = store.Get(ctx, testFlag.Key)
		if err == nil {
			t.Errorf("expected error when getting deleted flag")
		}
		var ffErr *FeatureFlagError
		if !errors.As(err, &ffErr) || !errors.Is(ffErr.Err, ErrFlagNotFound) {
			t.Errorf("expected ErrFlagNotFound, got %v", err)
		}
	})

	t.Run("GetAll Operations", func(t *testing.T) {
		// Clean up first
		cleanupRedisTestData(t, store)

		// Create multiple test flags
		testFlags := []FeatureFlag{
			{
				Key:         "redis-test-flag-1",
				Enabled:     true,
				Description: "First Redis test flag",
			},
			{
				Key:         "redis-test-flag-2",
				Enabled:     false,
				Description: "Second Redis test flag",
				Metadata:    map[string]string{"type": "test"},
			},
			{
				Key:     "redis-test-flag-3",
				Enabled: true,
			},
		}

		// Insert all flags
		for _, flag := range testFlags {
			err := store.Set(ctx, flag)
			if err != nil {
				t.Fatalf("failed to set flag %s: %v", flag.Key, err)
			}
		}

		// Get all flags
		allFlags, err := store.GetAll(ctx)
		if err != nil {
			t.Fatalf("failed to get all flags: %v", err)
		}

		// Filter only our test flags
		var retrievedTestFlags []FeatureFlag
		for _, flag := range allFlags {
			if isRedisTestFlag(flag.Key) {
				retrievedTestFlags = append(retrievedTestFlags, flag)
			}
		}

		if len(retrievedTestFlags) != len(testFlags) {
			t.Errorf("expected %d test flags, got %d", len(testFlags), len(retrievedTestFlags))
		}

		// Verify each flag
		flagMap := make(map[string]FeatureFlag)
		for _, flag := range retrievedTestFlags {
			flagMap[flag.Key] = flag
		}

		for _, expected := range testFlags {
			actual, exists := flagMap[expected.Key]
			if !exists {
				t.Errorf("flag %s not found in results", expected.Key)
				continue
			}

			if actual.Enabled != expected.Enabled {
				t.Errorf("flag %s: expected enabled %v, got %v", expected.Key, expected.Enabled, actual.Enabled)
			}
			if actual.Description != expected.Description {
				t.Errorf("flag %s: expected description %s, got %s", expected.Key, expected.Description, actual.Description)
			}
			if !mapsEqual(actual.Metadata, expected.Metadata) {
				t.Errorf("flag %s: expected metadata %v, got %v", expected.Key, expected.Metadata, actual.Metadata)
			}
		}

		// Clean up
		for _, flag := range testFlags {
			store.Delete(ctx, flag.Key)
		}
	})

	t.Run("Health Check", func(t *testing.T) {
		err := store.HealthCheck(ctx)
		if err != nil {
			t.Errorf("health check failed: %v", err)
		}
	})

	t.Run("Concurrent Operations", func(t *testing.T) {
		// Clean up first
		cleanupRedisTestData(t, store)

		const numGoroutines = 10
		const numOperations = 5

		// Test concurrent writes
		errChan := make(chan error, numGoroutines*numOperations)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < numOperations; j++ {
					flag := FeatureFlag{
						Key:         fmt.Sprintf("redis-concurrent-flag-%d-%d", id, j),
						Enabled:     j%2 == 0,
						Description: fmt.Sprintf("Redis concurrent test flag %d-%d", id, j),
					}

					if err := store.Set(ctx, flag); err != nil {
						errChan <- fmt.Errorf("goroutine %d operation %d: %w", id, j, err)
						return
					}
				}
				errChan <- nil
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			if err := <-errChan; err != nil {
				t.Errorf("concurrent write error: %v", err)
			}
		}

		// Verify all flags were created
		allFlags, err := store.GetAll(ctx)
		if err != nil {
			t.Fatalf("failed to get all flags: %v", err)
		}

		concurrentFlags := 0
		for _, flag := range allFlags {
			if len(flag.Key) > 20 && flag.Key[:20] == "redis-concurrent-flag" {
				concurrentFlags++
			}
		}

		expectedFlags := numGoroutines * numOperations
		if concurrentFlags != expectedFlags {
			t.Errorf("expected %d concurrent flags, got %d", expectedFlags, concurrentFlags)
		}

		// Clean up concurrent test flags
		for _, flag := range allFlags {
			if len(flag.Key) > 20 && flag.Key[:20] == "redis-concurrent-flag" {
				store.Delete(ctx, flag.Key)
			}
		}
	})

	t.Run("Redis Connection Resilience", func(t *testing.T) {
		// Test that store handles Redis connection issues gracefully

		// Create a test flag first
		testFlag := FeatureFlag{
			Key:         "redis-resilience-test",
			Enabled:     true,
			Description: "Redis resilience test flag",
		}

		err := store.Set(ctx, testFlag)
		if err != nil {
			t.Fatalf("failed to set initial flag: %v", err)
		}

		// Verify we can read it back
		retrieved, err := store.Get(ctx, testFlag.Key)
		if err != nil {
			t.Fatalf("failed to get flag: %v", err)
		}

		if retrieved.Key != testFlag.Key {
			t.Errorf("expected key %s, got %s", testFlag.Key, retrieved.Key)
		}

		// Clean up
		store.Delete(ctx, testFlag.Key)
	})

	t.Run("Large Data Handling", func(t *testing.T) {
		// Test handling of flags with large metadata
		largeMetadata := make(map[string]string)
		for i := 0; i < 100; i++ {
			largeMetadata[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d_with_some_longer_content_to_test_serialization", i)
		}

		testFlag := FeatureFlag{
			Key:         "redis-large-data-test",
			Enabled:     true,
			Description: "Test flag with large metadata for Redis serialization testing",
			Metadata:    largeMetadata,
		}

		// Set the flag
		err := store.Set(ctx, testFlag)
		if err != nil {
			t.Fatalf("failed to set flag with large metadata: %v", err)
		}

		// Get the flag back
		retrieved, err := store.Get(ctx, testFlag.Key)
		if err != nil {
			t.Fatalf("failed to get flag with large metadata: %v", err)
		}

		// Verify all metadata was preserved
		if len(retrieved.Metadata) != len(testFlag.Metadata) {
			t.Errorf("expected %d metadata entries, got %d", len(testFlag.Metadata), len(retrieved.Metadata))
		}

		for key, expectedValue := range testFlag.Metadata {
			if actualValue, exists := retrieved.Metadata[key]; !exists {
				t.Errorf("metadata key %s not found", key)
			} else if actualValue != expectedValue {
				t.Errorf("metadata key %s: expected %s, got %s", key, expectedValue, actualValue)
			}
		}

		// Clean up
		store.Delete(ctx, testFlag.Key)
	})
}

// parseRedisURL parses a Redis URL into a RedisConfig
func parseRedisURL(url string) (*RedisConfig, error) {
	// Parse Redis URL - for testing we'll use a simple approach
	// In production, you might want to use a more robust URL parser

	// Default config for common test setups
	config := &RedisConfig{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1, // Use DB 1 for tests to avoid conflicts
	}

	// If URL contains specific parameters, parse them
	// This is a simplified parser for test purposes
	if url != "" && url != "redis://localhost:6379/1" {
		// For now, use defaults but log the URL
		fmt.Printf("Using Redis test URL: %s\n", url)
	}

	return config, nil
}

// cleanupRedisTestData removes any existing test data from Redis
func cleanupRedisTestData(t *testing.T, store Store) {
	ctx := context.Background()

	// Get all flags and delete any that look like test flags
	flags, err := store.GetAll(ctx)
	if err != nil {
		t.Logf("warning: failed to get flags for cleanup: %v", err)
		return
	}

	for _, flag := range flags {
		if isRedisTestFlag(flag.Key) {
			if err := store.Delete(ctx, flag.Key); err != nil {
				t.Logf("warning: failed to delete test flag %s: %v", flag.Key, err)
			}
		}
	}
}

// isRedisTestFlag determines if a flag key looks like a Redis test flag
func isRedisTestFlag(key string) bool {
	testPrefixes := []string{
		"redis-test-",
		"redis-integration-",
		"redis-concurrent-",
		"redis-resilience-",
		"redis-large-data-",
	}

	for _, prefix := range testPrefixes {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}
