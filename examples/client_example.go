//go:build examples
// +build examples

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ajeet-kumar1087/go-feature-flag/featureflag"
)

func main() {
	fmt.Println("=== Feature Flag Client Example ===")

	// Example 1: Basic usage with default configuration
	fmt.Println("\n1. Basic Usage with Default Configuration")
	basicExample()

	// Example 2: Custom configuration with caching
	fmt.Println("\n2. Custom Configuration with Caching")
	customConfigExample()

	// Example 3: Default flags example
	fmt.Println("\n3. Default Flags Example")
	defaultFlagsExample()

	// Example 4: Error handling and graceful degradation
	fmt.Println("\n4. Error Handling and Graceful Degradation")
	errorHandlingExample()
}

func basicExample() {
	// Create client with default configuration (in-memory storage, caching enabled)
	client, err := featureflag.NewClientWithDefaults()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create some feature flags
	flags := []featureflag.FeatureFlag{
		{
			Key:         "new-ui",
			Enabled:     true,
			Description: "Enable new user interface",
			Metadata:    map[string]string{"team": "frontend", "version": "2.0"},
		},
		{
			Key:         "beta-features",
			Enabled:     false,
			Description: "Enable beta features for testing",
			Metadata:    map[string]string{"team": "backend", "env": "staging"},
		},
		{
			Key:         "analytics",
			Enabled:     true,
			Description: "Enable analytics tracking",
		},
	}

	// Set the flags
	for _, flag := range flags {
		if err := client.SetFlag(ctx, flag); err != nil {
			log.Printf("Failed to set flag %s: %v", flag.Key, err)
			continue
		}
		fmt.Printf("✓ Set flag: %s (enabled: %v)\n", flag.Key, flag.Enabled)
	}

	// Check if features are enabled
	fmt.Println("\nChecking feature flags:")
	for _, flag := range flags {
		enabled, err := client.IsEnabled(ctx, flag.Key)
		if err != nil {
			log.Printf("Error checking flag %s: %v", flag.Key, err)
			continue
		}
		status := "disabled"
		if enabled {
			status = "enabled"
		}
		fmt.Printf("- %s: %s\n", flag.Key, status)
	}

	// Get detailed flag information
	fmt.Println("\nDetailed flag information:")
	flag, err := client.GetFlag(ctx, "new-ui")
	if err != nil {
		log.Printf("Failed to get flag details: %v", err)
	} else {
		fmt.Printf("Flag: %s\n", flag.Key)
		fmt.Printf("  Enabled: %v\n", flag.Enabled)
		fmt.Printf("  Description: %s\n", flag.Description)
		fmt.Printf("  Created: %s\n", flag.CreatedAt.Format(time.RFC3339))
		fmt.Printf("  Updated: %s\n", flag.UpdatedAt.Format(time.RFC3339))
		if len(flag.Metadata) > 0 {
			fmt.Printf("  Metadata:\n")
			for k, v := range flag.Metadata {
				fmt.Printf("    %s: %s\n", k, v)
			}
		}
	}

	// List all flags
	allFlags, err := client.GetAllFlags(ctx)
	if err != nil {
		log.Printf("Failed to get all flags: %v", err)
	} else {
		fmt.Printf("\nTotal flags: %d\n", len(allFlags))
	}
}

func customConfigExample() {
	// Create custom configuration
	config := featureflag.Config{
		Storage: featureflag.StorageConfig{
			Type: "memory", // Use in-memory storage
		},
		Cache: featureflag.CacheConfig{
			Enabled: true,
			TTL:     featureflag.Duration(10 * time.Minute), // 10 minute TTL
			MaxSize: 500,                                    // Cache up to 500 flags
		},
	}

	client, err := featureflag.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create a flag
	flag := featureflag.FeatureFlag{
		Key:         "custom-feature",
		Enabled:     true,
		Description: "A feature with custom configuration",
	}

	if err := client.SetFlag(ctx, flag); err != nil {
		log.Printf("Failed to set flag: %v", err)
		return
	}

	fmt.Printf("✓ Created flag with custom config: %s\n", flag.Key)

	// Test multiple reads (should hit cache after first read)
	for i := 0; i < 3; i++ {
		enabled, err := client.IsEnabled(ctx, "custom-feature")
		if err != nil {
			log.Printf("Error checking flag: %v", err)
			continue
		}
		fmt.Printf("Read %d: custom-feature is %v\n", i+1, enabled)
	}
}

func defaultFlagsExample() {
	// Configuration with default flags
	config := featureflag.Config{
		Storage: featureflag.StorageConfig{
			Type: "memory",
		},
		Cache: featureflag.CacheConfig{
			Enabled: false, // Disable cache for this example
		},
		DefaultFlags: []featureflag.FeatureFlag{
			{
				Key:         "maintenance-mode",
				Enabled:     false,
				Description: "Enable maintenance mode",
				Metadata:    map[string]string{"priority": "high"},
			},
			{
				Key:         "debug-logging",
				Enabled:     true,
				Description: "Enable debug logging",
				Metadata:    map[string]string{"level": "debug"},
			},
			{
				Key:         "feature-x",
				Enabled:     false,
				Description: "New experimental feature X",
				Metadata:    map[string]string{"experimental": "true"},
			},
		},
	}

	client, err := featureflag.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	fmt.Println("Default flags loaded automatically:")

	// Check all default flags
	for _, defaultFlag := range config.DefaultFlags {
		enabled, err := client.IsEnabled(ctx, defaultFlag.Key)
		if err != nil {
			log.Printf("Error checking default flag %s: %v", defaultFlag.Key, err)
			continue
		}
		status := "disabled"
		if enabled {
			status = "enabled"
		}
		fmt.Printf("- %s: %s\n", defaultFlag.Key, status)
	}

	// Show that we can still modify flags
	fmt.Println("\nModifying a default flag:")
	updatedFlag := featureflag.FeatureFlag{
		Key:         "maintenance-mode",
		Enabled:     true, // Enable maintenance mode
		Description: "Enable maintenance mode (updated)",
		Metadata:    map[string]string{"priority": "critical", "updated": "true"},
	}

	if err := client.SetFlag(ctx, updatedFlag); err != nil {
		log.Printf("Failed to update flag: %v", err)
	} else {
		enabled, _ := client.IsEnabled(ctx, "maintenance-mode")
		fmt.Printf("✓ Updated maintenance-mode: %v\n", enabled)
	}
}

func errorHandlingExample() {
	client, err := featureflag.NewClientWithDefaults()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Test graceful degradation - checking non-existent flag
	fmt.Println("Checking non-existent flag (should return false, no error):")
	enabled, err := client.IsEnabled(ctx, "non-existent-flag")
	if err != nil {
		fmt.Printf("Error (unexpected): %v\n", err)
	} else {
		fmt.Printf("✓ Non-existent flag returns: %v (graceful degradation)\n", enabled)
	}

	// Test getting non-existent flag (should return error)
	fmt.Println("\nGetting non-existent flag (should return error):")
	flag, err := client.GetFlag(ctx, "non-existent-flag")
	if err != nil {
		fmt.Printf("✓ Expected error: %v\n", err)
	} else {
		fmt.Printf("Unexpected success: %+v\n", flag)
	}

	// Test invalid flag operations
	fmt.Println("\nTesting invalid operations:")

	// Empty key
	enabled, err = client.IsEnabled(ctx, "")
	if err != nil {
		fmt.Printf("✓ Empty key error: %v\n", err)
	}

	// Invalid flag
	invalidFlag := featureflag.FeatureFlag{
		Key:     "", // Invalid empty key
		Enabled: true,
	}
	err = client.SetFlag(ctx, invalidFlag)
	if err != nil {
		fmt.Printf("✓ Invalid flag error: %v\n", err)
	}

	// Test operations after close
	fmt.Println("\nTesting operations after client close:")
	client.Close()

	enabled, err = client.IsEnabled(ctx, "test")
	if err != nil {
		fmt.Printf("✓ Operation after close error: %v\n", err)
	}
}
