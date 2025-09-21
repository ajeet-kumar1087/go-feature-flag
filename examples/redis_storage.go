//go:build examples
// +build examples

// Package main demonstrates Redis storage usage
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ajeet-kumar1087/go-feature-flag/featureflag"
)

func main() {
	fmt.Println("=== Redis Storage Example ===")

	// Configure client with Redis storage
	config := featureflag.Config{
		Storage: featureflag.StorageConfig{
			Type: "redis",
			Redis: &featureflag.RedisConfig{
				Addr:     "localhost:6379",
				Password: "", // No password for local development
				DB:       0,  // Use default DB
			},
		},
		Cache: featureflag.CacheConfig{
			Enabled: true,
			TTL:     featureflag.Duration(5 * time.Minute),
			MaxSize: 500,
		},
		Observability: featureflag.ObservabilityConfig{
			Logging: featureflag.LoggingConfig{
				Enabled: true,
				Level:   "info",
			},
		},
	}

	client, err := featureflag.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Test Redis connectivity
	if err := client.HealthCheck(ctx); err != nil {
		log.Fatalf("Redis health check failed: %v", err)
	}
	fmt.Println("✓ Redis connection healthy")

	// Create feature flags
	flags := []featureflag.FeatureFlag{
		{
			Key:         "redis-feature-1",
			Enabled:     true,
			Description: "Feature stored in Redis",
			Metadata: map[string]string{
				"storage":     "redis",
				"environment": "development",
				"team":        "backend",
			},
		},
		{
			Key:         "redis-feature-2",
			Enabled:     false,
			Description: "Another Redis feature",
			Metadata: map[string]string{
				"storage":     "redis",
				"environment": "development",
				"rollout":     "0%",
			},
		},
		{
			Key:         "redis-cache-test",
			Enabled:     true,
			Description: "Feature for cache testing",
			Metadata: map[string]string{
				"purpose": "cache-testing",
			},
		},
	}

	// Set flags
	fmt.Println("\nCreating flags in Redis:")
	for _, flag := range flags {
		if err := client.SetFlag(ctx, flag); err != nil {
			log.Printf("Failed to set flag %s: %v", flag.Key, err)
			continue
		}
		fmt.Printf("✓ Created: %s (enabled: %v)\n", flag.Key, flag.Enabled)
	}

	// Retrieve flags
	fmt.Println("\nRetrieving flags from Redis:")
	for _, flag := range flags {
		retrievedFlag, err := client.GetFlag(ctx, flag.Key)
		if err != nil {
			log.Printf("Failed to get flag %s: %v", flag.Key, err)
			continue
		}
		fmt.Printf("- %s: %v (%s)\n",
			retrievedFlag.Key, retrievedFlag.Enabled, retrievedFlag.Description)

		if len(retrievedFlag.Metadata) > 0 {
			fmt.Printf("  Metadata: %v\n", retrievedFlag.Metadata)
		}
	}

	// Demonstrate cache performance
	fmt.Println("\nCache performance test:")

	// First call - cache miss
	start := time.Now()
	client.IsEnabled(ctx, "redis-cache-test")
	firstCall := time.Since(start)

	// Second call - cache hit
	start = time.Now()
	client.IsEnabled(ctx, "redis-cache-test")
	secondCall := time.Since(start)

	fmt.Printf("First call (cache miss): %v\n", firstCall)
	fmt.Printf("Second call (cache hit): %v\n", secondCall)
	fmt.Printf("Cache speedup: %.2fx\n", float64(firstCall)/float64(secondCall))

	// Update a flag
	fmt.Println("\nUpdating flag:")
	updateFlag := flags[1] // redis-feature-2
	updateFlag.Enabled = true
	updateFlag.Description = "Updated Redis feature"
	updateFlag.Metadata["rollout"] = "50%"
	updateFlag.Metadata["updated"] = "true"

	if err := client.SetFlag(ctx, updateFlag); err != nil {
		log.Printf("Failed to update flag: %v", err)
	} else {
		fmt.Printf("✓ Updated %s to enabled: %v\n", updateFlag.Key, updateFlag.Enabled)
	}

	// Verify update
	updated, err := client.GetFlag(ctx, updateFlag.Key)
	if err != nil {
		log.Printf("Failed to verify update: %v", err)
	} else {
		fmt.Printf("Verified: %s rollout is now %s\n",
			updated.Key, updated.Metadata["rollout"])
	}

	// List all flags
	allFlags, err := client.GetAllFlags(ctx)
	if err != nil {
		log.Printf("Failed to get all flags: %v", err)
	} else {
		fmt.Printf("\nTotal flags in Redis: %d\n", len(allFlags))
	}

	// Clean up - delete test flags
	fmt.Println("\nCleaning up test flags:")
	for _, flag := range flags {
		if err := client.DeleteFlag(ctx, flag.Key); err != nil {
			log.Printf("Failed to delete flag %s: %v", flag.Key, err)
		} else {
			fmt.Printf("✓ Deleted: %s\n", flag.Key)
		}
	}

	fmt.Println("\n✅ Redis storage example completed!")
}
