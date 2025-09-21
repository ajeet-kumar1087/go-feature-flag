//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestEndToEndWorkflows tests complete feature flag workflows from configuration to usage
func TestEndToEndWorkflows(t *testing.T) {
	t.Run("Complete Memory Store Workflow", func(t *testing.T) {
		// Create temporary config file
		tempDir := t.TempDir()
		configContent := `{
			"storage": {
				"type": "memory"
			},
			"cache": {
				"enabled": true,
				"ttl": "10m",
				"max_size": 1000
			},
			"default_flags": [
				{
					"key": "feature-a",
					"enabled": true,
					"description": "Feature A for testing",
					"metadata": {
						"team": "backend",
						"rollout": "100"
					}
				},
				{
					"key": "feature-b",
					"enabled": false,
					"description": "Feature B for testing"
				}
			],
			"observability": {
				"logging": {
					"enabled": true,
					"level": "info"
				},
				"metrics": {
					"enabled": true
				}
			}
		}`

		configPath := filepath.Join(tempDir, "e2e-config.json")
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		// Load configuration
		config, err := LoadConfigFromFile(configPath)
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Create client
		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close()

		ctx := context.Background()

		// Test 1: Verify default flags are loaded
		enabled, err := client.IsEnabled(ctx, "feature-a")
		if err != nil {
			t.Fatalf("Failed to check feature-a: %v", err)
		}
		if !enabled {
			t.Error("Expected feature-a to be enabled")
		}

		enabled, err = client.IsEnabled(ctx, "feature-b")
		if err != nil {
			t.Fatalf("Failed to check feature-b: %v", err)
		}
		if enabled {
			t.Error("Expected feature-b to be disabled")
		}

		// Test 2: Create new flags at runtime
		newFlag := FeatureFlag{
			Key:         "runtime-feature",
			Enabled:     true,
			Description: "Feature created at runtime",
			Metadata: map[string]string{
				"created_at": time.Now().Format(time.RFC3339),
				"test":       "e2e",
			},
		}

		err = client.SetFlag(ctx, newFlag)
		if err != nil {
			t.Fatalf("Failed to set runtime flag: %v", err)
		}

		// Test 3: Verify new flag works
		enabled, err = client.IsEnabled(ctx, "runtime-feature")
		if err != nil {
			t.Fatalf("Failed to check runtime-feature: %v", err)
		}
		if !enabled {
			t.Error("Expected runtime-feature to be enabled")
		}

		// Test 4: Update existing flag
		updatedFlag := FeatureFlag{
			Key:         "feature-b",
			Enabled:     true,
			Description: "Feature B updated at runtime",
			Metadata: map[string]string{
				"updated": "true",
			},
		}

		err = client.SetFlag(ctx, updatedFlag)
		if err != nil {
			t.Fatalf("Failed to update feature-b: %v", err)
		}

		// Verify update
		enabled, err = client.IsEnabled(ctx, "feature-b")
		if err != nil {
			t.Fatalf("Failed to check updated feature-b: %v", err)
		}
		if !enabled {
			t.Error("Expected updated feature-b to be enabled")
		}

		// Test 5: Get all flags
		allFlags, err := client.GetAllFlags(ctx)
		if err != nil {
			t.Fatalf("Failed to get all flags: %v", err)
		}

		expectedFlags := []string{"feature-a", "feature-b", "runtime-feature"}
		flagMap := make(map[string]FeatureFlag)
		for _, flag := range allFlags {
			flagMap[flag.Key] = flag
		}

		for _, expectedKey := range expectedFlags {
			if _, exists := flagMap[expectedKey]; !exists {
				t.Errorf("Expected flag %s not found in results", expectedKey)
			}
		}

		// Test 6: Delete flag
		err = client.DeleteFlag(ctx, "runtime-feature")
		if err != nil {
			t.Fatalf("Failed to delete runtime-feature: %v", err)
		}

		// Verify deletion
		enabled, err = client.IsEnabled(ctx, "runtime-feature")
		if err != nil {
			t.Fatalf("Failed to check deleted flag: %v", err)
		}
		if enabled {
			t.Error("Expected deleted flag to be disabled")
		}

		// Test 7: Performance under load
		const numOperations = 1000
		start := time.Now()

		for i := 0; i < numOperations; i++ {
			client.IsEnabled(ctx, "feature-a")
		}

		duration := time.Since(start)
		avgLatency := duration / numOperations

		t.Logf("Performance: %d operations in %v, avg latency: %v", numOperations, duration, avgLatency)

		// Should be very fast with memory store and cache
		if avgLatency > time.Millisecond {
			t.Logf("Warning: High average latency: %v", avgLatency)
		}
	})

	t.Run("Redis Store End-to-End Workflow", func(t *testing.T) {
		redisURL := os.Getenv("REDIS_TEST_URL")
		if redisURL == "" {
			t.Skip("REDIS_TEST_URL not set, skipping Redis E2E test")
		}

		config := Config{
			Storage: StorageConfig{
				Type: "redis",
				Redis: &RedisConfig{
					Addr: "localhost:6379",
					DB:   2, // Use different DB for E2E tests
				},
			},
			Cache: CacheConfig{
				Enabled: true,
				TTL:     Duration(5 * time.Minute),
				MaxSize: 500,
			},
			DefaultFlags: []FeatureFlag{
				{
					Key:         "redis-e2e-feature",
					Enabled:     true,
					Description: "Redis E2E test feature",
				},
			},
		}

		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("Failed to create Redis client: %v", err)
		}
		defer client.Close()

		ctx := context.Background()

		// Clean up any existing test data
		client.DeleteFlag(ctx, "redis-e2e-feature")
		client.DeleteFlag(ctx, "redis-e2e-runtime")

		// Test complete workflow with Redis persistence
		testFlag := FeatureFlag{
			Key:         "redis-e2e-runtime",
			Enabled:     false,
			Description: "Redis E2E runtime flag",
			Metadata: map[string]string{
				"persistence": "redis",
				"test":        "e2e",
			},
		}

		// Create flag
		err = client.SetFlag(ctx, testFlag)
		if err != nil {
			t.Fatalf("Failed to create Redis flag: %v", err)
		}

		// Verify persistence by creating a new client
		client2, err := NewClient(config)
		if err != nil {
			t.Fatalf("Failed to create second Redis client: %v", err)
		}
		defer client2.Close()

		// Flag should be persisted and available in new client
		flag, err := client2.GetFlag(ctx, "redis-e2e-runtime")
		if err != nil {
			t.Fatalf("Failed to get persisted flag: %v", err)
		}

		if flag.Key != testFlag.Key || flag.Enabled != testFlag.Enabled {
			t.Error("Flag was not properly persisted in Redis")
		}

		// Test cache behavior
		start := time.Now()
		for i := 0; i < 100; i++ {
			client.IsEnabled(ctx, "redis-e2e-runtime")
		}
		cachedDuration := time.Since(start)

		// Disable cache and test again
		config.Cache.Enabled = false
		client3, err := NewClient(config)
		if err != nil {
			t.Fatalf("Failed to create client without cache: %v", err)
		}
		defer client3.Close()

		start = time.Now()
		for i := 0; i < 100; i++ {
			client3.IsEnabled(ctx, "redis-e2e-runtime")
		}
		uncachedDuration := time.Since(start)

		t.Logf("Redis performance - Cached: %v, Uncached: %v", cachedDuration, uncachedDuration)

		// Cache should provide significant performance improvement
		if cachedDuration >= uncachedDuration {
			t.Logf("Warning: Cache did not improve performance (cached: %v, uncached: %v)", cachedDuration, uncachedDuration)
		}

		// Clean up
		client.DeleteFlag(ctx, "redis-e2e-runtime")
	})

	t.Run("PostgreSQL Store End-to-End Workflow", func(t *testing.T) {
		postgresURL := os.Getenv("POSTGRES_TEST_URL")
		if postgresURL == "" {
			t.Skip("POSTGRES_TEST_URL not set, skipping PostgreSQL E2E test")
		}

		config := Config{
			Storage: StorageConfig{
				Type: "postgres",
				Postgres: &PostgresConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "testdb",
					Username: "testuser",
					Password: "testpass",
					SSLMode:  "disable",
				},
			},
			Cache: CacheConfig{
				Enabled: true,
				TTL:     Duration(3 * time.Minute),
				MaxSize: 300,
			},
		}

		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("Failed to create PostgreSQL client: %v", err)
		}
		defer client.Close()

		ctx := context.Background()

		// Clean up any existing test data
		client.DeleteFlag(ctx, "postgres-e2e-feature")

		// Test complete workflow with PostgreSQL persistence
		testFlag := FeatureFlag{
			Key:         "postgres-e2e-feature",
			Enabled:     true,
			Description: "PostgreSQL E2E test feature",
			Metadata: map[string]string{
				"persistence": "postgres",
				"test":        "e2e",
				"timestamp":   time.Now().Format(time.RFC3339),
			},
		}

		// Create flag
		err = client.SetFlag(ctx, testFlag)
		if err != nil {
			t.Fatalf("Failed to create PostgreSQL flag: %v", err)
		}

		// Verify persistence by creating a new client
		client2, err := NewClient(config)
		if err != nil {
			t.Fatalf("Failed to create second PostgreSQL client: %v", err)
		}
		defer client2.Close()

		// Flag should be persisted and available in new client
		flag, err := client2.GetFlag(ctx, "postgres-e2e-feature")
		if err != nil {
			t.Fatalf("Failed to get persisted flag: %v", err)
		}

		if flag.Key != testFlag.Key || flag.Enabled != testFlag.Enabled {
			t.Error("Flag was not properly persisted in PostgreSQL")
		}

		if flag.Metadata["persistence"] != "postgres" {
			t.Error("Flag metadata was not properly persisted")
		}

		// Test transaction-like behavior with multiple operations
		flags := []FeatureFlag{
			{Key: "postgres-batch-1", Enabled: true, Description: "Batch test 1"},
			{Key: "postgres-batch-2", Enabled: false, Description: "Batch test 2"},
			{Key: "postgres-batch-3", Enabled: true, Description: "Batch test 3"},
		}

		// Create multiple flags
		for _, f := range flags {
			err = client.SetFlag(ctx, f)
			if err != nil {
				t.Fatalf("Failed to create batch flag %s: %v", f.Key, err)
			}
		}

		// Verify all flags exist
		allFlags, err := client.GetAllFlags(ctx)
		if err != nil {
			t.Fatalf("Failed to get all flags: %v", err)
		}

		batchFlagCount := 0
		for _, flag := range allFlags {
			if len(flag.Key) >= 14 && flag.Key[:14] == "postgres-batch" {
				batchFlagCount++
			}
		}

		if batchFlagCount != 3 {
			t.Errorf("Expected 3 batch flags, found %d", batchFlagCount)
		}

		// Clean up batch flags
		for _, f := range flags {
			client.DeleteFlag(ctx, f.Key)
		}

		// Clean up main test flag
		client.DeleteFlag(ctx, "postgres-e2e-feature")
	})

	t.Run("Multi-Client Concurrent Workflow", func(t *testing.T) {
		// Test multiple clients working with the same storage concurrently
		config := Config{
			Storage: StorageConfig{
				Type: "memory",
			},
			Cache: CacheConfig{
				Enabled: true,
				TTL:     Duration(2 * time.Minute),
				MaxSize: 200,
			},
		}

		const numClients = 5
		const numOperations = 100

		clients := make([]Client, numClients)
		for i := 0; i < numClients; i++ {
			client, err := NewClient(config)
			if err != nil {
				t.Fatalf("Failed to create client %d: %v", i, err)
			}
			clients[i] = client
			defer client.Close()
		}

		ctx := context.Background()
		var wg sync.WaitGroup
		errorChan := make(chan error, numClients*numOperations)

		// Each client performs operations concurrently
		wg.Add(numClients)
		for clientID := 0; clientID < numClients; clientID++ {
			go func(id int, client Client) {
				defer wg.Done()

				for op := 0; op < numOperations; op++ {
					flagKey := fmt.Sprintf("multi-client-flag-%d-%d", id, op)

					// Create flag
					flag := FeatureFlag{
						Key:         flagKey,
						Enabled:     op%2 == 0,
						Description: fmt.Sprintf("Flag from client %d operation %d", id, op),
						Metadata: map[string]string{
							"client": fmt.Sprintf("%d", id),
							"op":     fmt.Sprintf("%d", op),
						},
					}

					if err := client.SetFlag(ctx, flag); err != nil {
						errorChan <- fmt.Errorf("client %d op %d set: %w", id, op, err)
						continue
					}

					// Check flag
					enabled, err := client.IsEnabled(ctx, flagKey)
					if err != nil {
						errorChan <- fmt.Errorf("client %d op %d check: %w", id, op, err)
						continue
					}

					if enabled != flag.Enabled {
						errorChan <- fmt.Errorf("client %d op %d: expected %v, got %v", id, op, flag.Enabled, enabled)
						continue
					}

					// Occasionally delete flags
					if op%10 == 9 {
						if err := client.DeleteFlag(ctx, flagKey); err != nil {
							errorChan <- fmt.Errorf("client %d op %d delete: %w", id, op, err)
						}
					}
				}
			}(clientID, clients[clientID])
		}

		wg.Wait()
		close(errorChan)

		// Check for errors
		errorCount := 0
		for err := range errorChan {
			t.Errorf("Concurrent operation error: %v", err)
			errorCount++
		}

		if errorCount > 0 {
			t.Errorf("Total errors in concurrent workflow: %d", errorCount)
		}

		// Verify final state
		allFlags, err := clients[0].GetAllFlags(ctx)
		if err != nil {
			t.Fatalf("Failed to get final flags: %v", err)
		}

		multiClientFlags := 0
		for _, flag := range allFlags {
			if len(flag.Key) >= 18 && flag.Key[:18] == "multi-client-flag-" {
				multiClientFlags++
			}
		}

		t.Logf("Final state: %d multi-client flags remaining", multiClientFlags)
	})

	t.Run("Cache Invalidation Workflow", func(t *testing.T) {
		config := Config{
			Storage: StorageConfig{
				Type: "memory",
			},
			Cache: CacheConfig{
				Enabled: true,
				TTL:     Duration(1 * time.Minute),
				MaxSize: 100,
			},
		}

		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close()

		ctx := context.Background()

		// Create initial flag
		flag := FeatureFlag{
			Key:         "cache-invalidation-test",
			Enabled:     false,
			Description: "Cache invalidation test flag",
		}

		err = client.SetFlag(ctx, flag)
		if err != nil {
			t.Fatalf("Failed to set initial flag: %v", err)
		}

		// Check flag (should cache it)
		enabled, err := client.IsEnabled(ctx, "cache-invalidation-test")
		if err != nil {
			t.Fatalf("Failed to check initial flag: %v", err)
		}
		if enabled {
			t.Error("Expected initial flag to be disabled")
		}

		// Update flag
		flag.Enabled = true
		err = client.SetFlag(ctx, flag)
		if err != nil {
			t.Fatalf("Failed to update flag: %v", err)
		}

		// Check flag immediately (cache should be invalidated)
		enabled, err = client.IsEnabled(ctx, "cache-invalidation-test")
		if err != nil {
			t.Fatalf("Failed to check updated flag: %v", err)
		}
		if !enabled {
			t.Error("Expected updated flag to be enabled (cache should be invalidated)")
		}

		// Delete flag
		err = client.DeleteFlag(ctx, "cache-invalidation-test")
		if err != nil {
			t.Fatalf("Failed to delete flag: %v", err)
		}

		// Check flag (should return false for deleted flag)
		enabled, err = client.IsEnabled(ctx, "cache-invalidation-test")
		if err != nil {
			t.Fatalf("Failed to check deleted flag: %v", err)
		}
		if enabled {
			t.Error("Expected deleted flag to be disabled")
		}
	})
}
