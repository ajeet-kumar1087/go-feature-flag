package main

import (
	"context"
	"log"

	featureflag "github.com/ajeet-kumar1087/go-feature-flag"
)

func main() {
	// Create client with default configuration (in-memory storage, caching enabled)
	client, err := featureflag.NewClientWithDefaults()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create a feature flag
	flag := featureflag.FeatureFlag{
		Key:         "new-checkout-flow",
		Enabled:     true,
		Description: "Enable the new checkout flow",
		Metadata: map[string]string{
			"team":    "payments",
			"rollout": "100%",
		},
	}

	if err := client.SetFlag(ctx, flag); err != nil {
		log.Fatal(err)
	}

	// Check if feature is enabled (primary usage pattern)
	if enabled, _ := client.IsEnabled(ctx, "new-checkout-flow"); enabled {
		// Use new checkout flow
		log.Println("✅ Using new checkout flow")
	} else {
		// Use legacy checkout flow
		log.Println("⚪ Using legacy checkout flow")
	}

	// Demonstrate graceful degradation
	if enabled, _ := client.IsEnabled(ctx, "non-existent-flag"); enabled {
		log.Println("This won't print")
	} else {
		log.Println("🔒 Non-existent flag returns false (graceful degradation)")
	}

	// Get complete flag information
	retrievedFlag, err := client.GetFlag(ctx, "new-checkout-flow")
	if err != nil {
		log.Printf("Error getting flag: %v", err)
	} else {
		log.Printf("📋 Flag details: %+v", retrievedFlag)
	}

	// Health check
	if err := client.HealthCheck(ctx); err != nil {
		log.Printf("❌ Health check failed: %v", err)
	} else {
		log.Println("✅ System healthy")
	}
}
