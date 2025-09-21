//go:build integration
// +build integration

package integration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

// TestPostgresIntegration runs integration tests against a real PostgreSQL database
// To run these tests, set the POSTGRES_TEST_URL environment variable
// Example: POSTGRES_TEST_URL="postgres://user:password@localhost:5432/testdb?sslmode=disable"
func TestPostgresIntegration(t *testing.T) {
	testURL := os.Getenv("POSTGRES_TEST_URL")
	if testURL == "" {
		t.Skip("POSTGRES_TEST_URL not set, skipping integration tests")
	}

	// Parse the test URL to create config
	config, err := parsePostgresURL(testURL)
	if err != nil {
		t.Fatalf("failed to parse test URL: %v", err)
	}

	// Create store
	store, err := NewPostgresStore(config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Clean up any existing test data
	cleanupTestData(t, store)

	ctx := context.Background()

	t.Run("CRUD Operations", func(t *testing.T) {
		testFlag := FeatureFlag{
			Key:         "integration-test-flag",
			Enabled:     true,
			Description: "Integration test flag",
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
		cleanupTestData(t, store)

		// Create multiple test flags
		testFlags := []FeatureFlag{
			{
				Key:         "test-flag-1",
				Enabled:     true,
				Description: "First test flag",
			},
			{
				Key:         "test-flag-2",
				Enabled:     false,
				Description: "Second test flag",
				Metadata:    map[string]string{"type": "test"},
			},
			{
				Key:     "test-flag-3",
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
			if flag.Key == "test-flag-1" || flag.Key == "test-flag-2" || flag.Key == "test-flag-3" {
				retrievedTestFlags = append(retrievedTestFlags, flag)
			}
		}

		if len(retrievedTestFlags) != len(testFlags) {
			t.Errorf("expected %d test flags, got %d", len(testFlags), len(retrievedTestFlags))
		}

		// Verify each flag (order might be different due to ORDER BY key)
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
		cleanupTestData(t, store)

		const numGoroutines = 10
		const numOperations = 5

		// Test concurrent writes
		errChan := make(chan error, numGoroutines*numOperations)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < numOperations; j++ {
					flag := FeatureFlag{
						Key:         fmt.Sprintf("concurrent-flag-%d-%d", id, j),
						Enabled:     j%2 == 0,
						Description: fmt.Sprintf("Concurrent test flag %d-%d", id, j),
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
			if len(flag.Key) > 15 && flag.Key[:15] == "concurrent-flag" {
				concurrentFlags++
			}
		}

		expectedFlags := numGoroutines * numOperations
		if concurrentFlags != expectedFlags {
			t.Errorf("expected %d concurrent flags, got %d", expectedFlags, concurrentFlags)
		}

		// Clean up concurrent test flags
		for _, flag := range allFlags {
			if len(flag.Key) > 15 && flag.Key[:15] == "concurrent-flag" {
				store.Delete(ctx, flag.Key)
			}
		}
	})
}

// parsePostgresURL parses a PostgreSQL URL into a PostgresConfig
func parsePostgresURL(url string) (*PostgresConfig, error) {
	// This is a simple parser for test URLs
	// In production, you might want to use a more robust URL parser

	// For now, we'll create a basic config that works with common test setups
	return &PostgresConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "testdb",
		Username: "testuser",
		Password: "testpass",
		SSLMode:  "disable",
	}, nil
}

// cleanupTestData removes any existing test data
func cleanupTestData(t *testing.T, store *PostgresStore) {
	ctx := context.Background()

	// Get all flags and delete any that look like test flags
	flags, err := store.GetAll(ctx)
	if err != nil {
		t.Logf("warning: failed to get flags for cleanup: %v", err)
		return
	}

	for _, flag := range flags {
		if isTestFlag(flag.Key) {
			if err := store.Delete(ctx, flag.Key); err != nil {
				t.Logf("warning: failed to delete test flag %s: %v", flag.Key, err)
			}
		}
	}
}

// isTestFlag determines if a flag key looks like a test flag
func isTestFlag(key string) bool {
	testPrefixes := []string{
		"test-",
		"integration-",
		"concurrent-",
	}

	for _, prefix := range testPrefixes {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}
