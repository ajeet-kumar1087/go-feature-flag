//go:build examples
// +build examples

// Package main provides a comprehensive demonstration of the feature flag library
// Run this example to see all features in action
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ajeet-kumar1087/go-feature-flag/featureflag"
)

func main() {
	fmt.Println("🚀 Go Feature Flag Library - Comprehensive Example")
	fmt.Println("==================================================")

	// Example 1: Basic usage with default configuration
	fmt.Println("\n📝 Example 1: Basic Usage")
	basicUsageExample()

	// Example 2: Custom configuration
	fmt.Println("\n⚙️  Example 2: Custom Configuration")
	customConfigurationExample()

	// Example 3: Advanced features
	fmt.Println("\n🔧 Example 3: Advanced Features")
	advancedFeaturesExample()

	// Example 4: Error handling and edge cases
	fmt.Println("\n🛡️  Example 4: Error Handling")
	errorHandlingExample()

	// Example 5: Performance demonstration
	fmt.Println("\n⚡ Example 5: Performance")
	performanceExample()

	fmt.Println("\n✅ All examples completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("- Check out individual example files in examples/ directory")
	fmt.Println("- Read the README.md for detailed documentation")
	fmt.Println("- See docs/migration-guide.md if migrating from HTTP service")
}

func basicUsageExample() {
	// Create client with defaults (in-memory storage, caching enabled)
	client, err := featureflag.NewClientWithDefaults()
	if err != nil {
		log.Printf("Failed to create client: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	// Create a simple feature flag
	flag := featureflag.FeatureFlag{
		Key:         "welcome-message",
		Enabled:     true,
		Description: "Show welcome message to users",
		Metadata: map[string]string{
			"team":    "frontend",
			"version": "1.0",
		},
	}

	// Set the flag
	if err := client.SetFlag(ctx, flag); err != nil {
		log.Printf("Failed to set flag: %v", err)
		return
	}
	fmt.Printf("✓ Created flag: %s\n", flag.Key)

	// Check if the feature is enabled
	enabled, err := client.IsEnabled(ctx, "welcome-message")
	if err != nil {
		log.Printf("Error checking flag: %v", err)
		return
	}

	if enabled {
		fmt.Println("👋 Welcome! This message is controlled by a feature flag.")
	} else {
		fmt.Println("No welcome message (feature disabled)")
	}

	// Get detailed flag information
	retrievedFlag, err := client.GetFlag(ctx, "welcome-message")
	if err != nil {
		log.Printf("Failed to get flag details: %v", err)
		return
	}

	fmt.Printf("Flag details:\n")
	fmt.Printf("  Key: %s\n", retrievedFlag.Key)
	fmt.Printf("  Enabled: %v\n", retrievedFlag.Enabled)
	fmt.Printf("  Description: %s\n", retrievedFlag.Description)
	fmt.Printf("  Team: %s\n", retrievedFlag.Metadata["team"])
}

func customConfigurationExample() {
	// Create custom configuration
	config := featureflag.Config{
		Storage: featureflag.StorageConfig{
			Type: "memory", // Use in-memory for this example
		},
		Cache: featureflag.CacheConfig{
			Enabled: true,
			TTL:     featureflag.Duration(30 * time.Second),
			MaxSize: 100,
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
				Description: "A default feature loaded at startup",
			},
		},
	}

	client, err := featureflag.NewClient(config)
	if err != nil {
		log.Printf("Failed to create client: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	// Check that default flag was loaded
	if enabled, _ := client.IsEnabled(ctx, "default-feature"); enabled {
		fmt.Println("✓ Default feature flag is active")
	}

	// Create additional flags
	flags := []featureflag.FeatureFlag{
		{
			Key:         "dark-mode",
			Enabled:     false,
			Description: "Enable dark mode theme",
			Metadata:    map[string]string{"ui": "theme"},
		},
		{
			Key:         "beta-features",
			Enabled:     true,
			Description: "Enable beta features",
			Metadata:    map[string]string{"rollout": "25%"},
		},
	}

	for _, flag := range flags {
		if err := client.SetFlag(ctx, flag); err != nil {
			log.Printf("Failed to set flag %s: %v", flag.Key, err)
		} else {
			fmt.Printf("✓ Created: %s (enabled: %v)\n", flag.Key, flag.Enabled)
		}
	}

	// Show metrics
	metrics := client.GetMetrics()
	fmt.Printf("Metrics: %d flag checks, %d cache hits\n",
		metrics.FlagChecks, metrics.CacheHits)
}

func advancedFeaturesExample() {
	client, err := featureflag.NewClientWithDefaults()
	if err != nil {
		log.Printf("Failed to create client: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	// Create flags with rich metadata
	advancedFlag := featureflag.FeatureFlag{
		Key:         "advanced-analytics",
		Enabled:     true,
		Description: "Advanced analytics with A/B testing",
		Metadata: map[string]string{
			"experiment_id":   "exp_001",
			"variant":         "treatment",
			"rollout_percent": "50",
			"owner":           "data-team",
			"jira_ticket":     "FEAT-123",
			"environment":     "production",
		},
	}

	if err := client.SetFlag(ctx, advancedFlag); err != nil {
		log.Printf("Failed to set advanced flag: %v", err)
		return
	}

	// Demonstrate flag usage in application logic
	if enabled, _ := client.IsEnabled(ctx, "advanced-analytics"); enabled {
		fmt.Println("📊 Advanced analytics enabled")

		// Get flag details for experiment configuration
		flag, err := client.GetFlag(ctx, "advanced-analytics")
		if err == nil {
			fmt.Printf("   Experiment ID: %s\n", flag.Metadata["experiment_id"])
			fmt.Printf("   Variant: %s\n", flag.Metadata["variant"])
			fmt.Printf("   Rollout: %s%%\n", flag.Metadata["rollout_percent"])
		}
	}

	// Demonstrate flag updates
	advancedFlag.Metadata["rollout_percent"] = "75"
	advancedFlag.Metadata["updated_at"] = time.Now().Format(time.RFC3339)

	if err := client.SetFlag(ctx, advancedFlag); err != nil {
		log.Printf("Failed to update flag: %v", err)
	} else {
		fmt.Println("✓ Updated rollout percentage to 75%")
	}

	// List all flags
	allFlags, err := client.GetAllFlags(ctx)
	if err != nil {
		log.Printf("Failed to get all flags: %v", err)
	} else {
		fmt.Printf("Total flags in system: %d\n", len(allFlags))
	}
}

func errorHandlingExample() {
	client, err := featureflag.NewClientWithDefaults()
	if err != nil {
		log.Printf("Failed to create client: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	// Demonstrate graceful degradation for non-existent flags
	fmt.Println("Checking non-existent flag (graceful degradation):")
	enabled, err := client.IsEnabled(ctx, "non-existent-flag")
	if err != nil {
		fmt.Printf("❌ Unexpected error: %v\n", err)
	} else {
		fmt.Printf("✓ Non-existent flag returns: %v (safe default)\n", enabled)
	}

	// Demonstrate validation errors
	fmt.Println("\nTesting validation errors:")
	invalidFlag := featureflag.FeatureFlag{
		Key:     "", // Invalid empty key
		Enabled: true,
	}

	err = client.SetFlag(ctx, invalidFlag)
	if err != nil {
		fmt.Printf("✓ Validation caught invalid flag: %v\n", err)
	}

	// Demonstrate context cancellation
	fmt.Println("\nTesting context cancellation:")
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = client.IsEnabled(cancelCtx, "test-flag")
	if err != nil {
		fmt.Printf("✓ Context cancellation handled: %v\n", err)
	}

	// Demonstrate operations after close
	fmt.Println("\nTesting operations after client close:")
	testClient, _ := featureflag.NewClientWithDefaults()
	testClient.Close()

	_, err = testClient.IsEnabled(ctx, "test")
	if err != nil {
		fmt.Printf("✓ Post-close operation error: %v\n", err)
	}
}

func performanceExample() {
	config := featureflag.Config{
		Storage: featureflag.StorageConfig{
			Type: "memory",
		},
		Cache: featureflag.CacheConfig{
			Enabled: true,
			TTL:     featureflag.Duration(1 * time.Minute),
			MaxSize: 1000,
		},
		Observability: featureflag.ObservabilityConfig{
			Metrics: featureflag.MetricsConfig{
				Enabled: true,
			},
		},
	}

	client, err := featureflag.NewClient(config)
	if err != nil {
		log.Printf("Failed to create client: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	// Create a test flag
	testFlag := featureflag.FeatureFlag{
		Key:     "performance-test",
		Enabled: true,
	}
	client.SetFlag(ctx, testFlag)

	// Measure performance
	iterations := 10000

	fmt.Printf("Running %d flag checks...\n", iterations)
	start := time.Now()

	for i := 0; i < iterations; i++ {
		client.IsEnabled(ctx, "performance-test")
	}

	duration := time.Since(start)

	fmt.Printf("Performance results:\n")
	fmt.Printf("  Total time: %v\n", duration)
	fmt.Printf("  Average per check: %v\n", duration/time.Duration(iterations))
	fmt.Printf("  Checks per second: %.0f\n", float64(iterations)/duration.Seconds())

	// Show cache performance
	metrics := client.GetMetrics()
	if metrics.FlagChecks > 0 {
		hitRate := float64(metrics.CacheHits) / float64(metrics.FlagChecks) * 100
		fmt.Printf("  Cache hit rate: %.2f%%\n", hitRate)
	}
}
