//go:build examples
// +build examples

// Package main demonstrates basic usage of the feature flag library
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/client"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/config"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/core"
)

func main() {
	fmt.Println("=== Basic Feature Flag Usage ===")

	// Create client with default configuration (in-memory storage, caching enabled)
	cfg := config.DefaultConfig()
	client, err := client.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create some feature flags
	flags := []core.FeatureFlag{
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

	// Example of using flags in application logic
	fmt.Println("\nApplication logic example:")
	if enabled, _ := client.IsEnabled(ctx, "new-ui"); enabled {
		fmt.Println("✓ Rendering new UI")
	} else {
		fmt.Println("- Using legacy UI")
	}

	if enabled, _ := client.IsEnabled(ctx, "beta-features"); enabled {
		fmt.Println("✓ Beta features available")
	} else {
		fmt.Println("- Beta features disabled")
	}
}
