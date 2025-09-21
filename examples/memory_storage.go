//go:build examples
// +build examples

// Package main demonstrates in-memory storage usage
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ajeet-kumar1087/go-feature-flag/featureflag"
)

func main() {
	fmt.Println("=== In-Memory Storage Example ===")

	// Configure client with in-memory storage
	config := featureflag.Config{
		Storage: featureflag.StorageConfig{
			Type: "memory",
		},
		Cache: featureflag.CacheConfig{
			Enabled: true,
			TTL:     featureflag.Duration(10 * time.Minute),
			MaxSize: 1000,
		},
		Observability: featureflag.ObservabilityConfig{
			Logging: featureflag.LoggingConfig{
				Enabled: true,
				Level:   "info",
			},
			Metrics: featureflag.MetricsConfig{
				Enabled: true,
			},
		},
		DefaultFlags: []featureflag.FeatureFlag{
			{
				Key:         "default-feature",
				Enabled:     true,
				Description: "A default feature flag",
			},
		},
	}

	client, err := featureflag.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Verify default flag was loaded
	fmt.Println("Default flags:")
	if enabled, _ := client.IsEnabled(ctx, "default-feature"); enabled {
		fmt.Println("✓ default-feature is enabled")
	}

	// Create additional flags
	flags := []featureflag.FeatureFlag{
		{
			Key:         "memory-feature-1",
			Enabled:     true,
			Description: "First memory-stored feature",
			Metadata:    map[string]string{"storage": "memory", "priority": "high"},
		},
		{
			Key:         "memory-feature-2",
			Enabled:     false,
			Description: "Second memory-stored feature",
			Metadata:    map[string]string{"storage": "memory", "priority": "low"},
		},
	}

	// Set flags
	fmt.Println("\nCreating flags:")
	for _, flag := range flags {
		if err := client.SetFlag(ctx, flag); err != nil {
			log.Printf("Failed to set flag %s: %v", flag.Key, err)
			continue
		}
		fmt.Printf("✓ Created: %s\n", flag.Key)
	}

	// List all flags
	allFlags, err := client.GetAllFlags(ctx)
	if err != nil {
		log.Fatalf("Failed to get all flags: %v", err)
	}

	fmt.Printf("\nTotal flags in memory: %d\n", len(allFlags))
	for _, flag := range allFlags {
		fmt.Printf("- %s: %v (%s)\n", flag.Key, flag.Enabled, flag.Description)
	}

	// Demonstrate performance with multiple reads
	fmt.Println("\nPerformance test (cached reads):")
	start := time.Now()
	for i := 0; i < 1000; i++ {
		client.IsEnabled(ctx, "memory-feature-1")
	}
	duration := time.Since(start)
	fmt.Printf("1000 flag checks took: %v (avg: %v per check)\n",
		duration, duration/1000)

	// Show metrics if enabled
	if config.Observability.Metrics.Enabled {
		metrics := client.GetMetrics()
		fmt.Printf("\nMetrics:\n")
		fmt.Printf("- Flag checks: %d\n", metrics.FlagChecks)
		fmt.Printf("- Cache hits: %d\n", metrics.CacheHits)
		fmt.Printf("- Cache misses: %d\n", metrics.CacheMisses)
	}

	fmt.Println("\n✅ In-memory storage example completed!")
}
