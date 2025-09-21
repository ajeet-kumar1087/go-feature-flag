//go:build examples
// +build examples

// Package main demonstrates PostgreSQL storage usage
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ajeet-kumar1087/go-feature-flag/featureflag"
)

func main() {
	fmt.Println("=== PostgreSQL Storage Example ===")

	// Configure client with PostgreSQL storage
	config := featureflag.Config{
		Storage: featureflag.StorageConfig{
			Type: "postgres",
			Postgres: &featureflag.PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "featureflags",
				Username: "postgres",
				Password: "password",
				SSLMode:  "disable", // Use "require" for production
			},
		},
		Cache: featureflag.CacheConfig{
			Enabled: true,
			TTL:     featureflag.Duration(15 * time.Minute),
			MaxSize: 2000,
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

	// Test PostgreSQL connectivity
	if err := client.HealthCheck(ctx); err != nil {
		log.Fatalf("PostgreSQL health check failed: %v", err)
	}
	fmt.Println("✓ PostgreSQL connection healthy")

	// Create some example feature flags
	flags := []featureflag.FeatureFlag{
		{
			Key:         "postgres-ui-feature",
			Enabled:     true,
			Description: "UI feature stored in PostgreSQL",
			Metadata: map[string]string{
				"team":        "frontend",
				"environment": "production",
				"rollout":     "100%",
				"storage":     "postgres",
			},
		},
		{
			Key:         "postgres-beta-features",
			Enabled:     false,
			Description: "Beta features in PostgreSQL",
			Metadata: map[string]string{
				"team":        "product",
				"environment": "staging",
				"rollout":     "0%",
				"storage":     "postgres",
			},
		},
		{
			Key:         "postgres-analytics",
			Enabled:     true,
			Description: "Analytics feature with PostgreSQL persistence",
			Metadata: map[string]string{
				"team":        "data",
				"environment": "production",
				"critical":    "false",
				"storage":     "postgres",
			},
		},
	}

	// Set feature flags
	fmt.Println("\nCreating feature flags in PostgreSQL:")
	for _, flag := range flags {
		if err := client.SetFlag(ctx, flag); err != nil {
			log.Printf("Failed to set flag %s: %v", flag.Key, err)
			continue
		}
		fmt.Printf("✓ Created flag: %s (enabled: %v)\n", flag.Key, flag.Enabled)
	}

	// Retrieve and display all flags
	fmt.Println("\nRetrieving all flags from PostgreSQL:")
	allFlags, err := client.GetAllFlags(ctx)
	if err != nil {
		log.Fatalf("Failed to get all flags: %v", err)
	}

	for _, flag := range allFlags {
		fmt.Printf("- %s: %v", flag.Key, flag.Enabled)
		if flag.Description != "" {
			fmt.Printf(" (%s)", flag.Description)
		}
		fmt.Println()

		if len(flag.Metadata) > 0 {
			fmt.Printf("  Metadata: %v\n", flag.Metadata)
		}
		fmt.Printf("  Created: %s, Updated: %s\n",
			flag.CreatedAt.Format(time.RFC3339),
			flag.UpdatedAt.Format(time.RFC3339))
	}

	// Demonstrate individual flag retrieval
	fmt.Println("\nRetrieving specific flags:")
	for _, key := range []string{"postgres-ui-feature", "postgres-beta-features", "nonexistent-flag"} {
		flag, err := client.GetFlag(ctx, key)
		if err != nil {
			fmt.Printf("- %s: Error - %v\n", key, err)
		} else {
			fmt.Printf("- %s: %v\n", key, flag.Enabled)
		}
	}

	// Update a flag
	fmt.Println("\nUpdating postgres-beta-features flag:")
	betaFlag, err := client.GetFlag(ctx, "postgres-beta-features")
	if err != nil {
		log.Printf("Failed to get postgres-beta-features flag: %v", err)
	} else {
		betaFlag.Enabled = true
		betaFlag.Description = "Beta features now enabled for limited rollout"
		betaFlag.Metadata["rollout"] = "25%"
		betaFlag.Metadata["updated_by"] = "admin"

		if err := client.SetFlag(ctx, *betaFlag); err != nil {
			log.Printf("Failed to update postgres-beta-features flag: %v", err)
		} else {
			fmt.Println("✓ Updated postgres-beta-features flag")
		}
	}

	// Verify the update
	updatedFlag, err := client.GetFlag(ctx, "postgres-beta-features")
	if err != nil {
		log.Printf("Failed to get updated flag: %v", err)
	} else {
		fmt.Printf("- postgres-beta-features: %v (%s)\n", updatedFlag.Enabled, updatedFlag.Description)
		fmt.Printf("  Rollout: %s, Updated by: %s\n",
			updatedFlag.Metadata["rollout"],
			updatedFlag.Metadata["updated_by"])
	}

	// Demonstrate persistence across connections
	fmt.Println("\nDemonstrating PostgreSQL persistence:")
	fmt.Println("Flags will persist even after client restart...")

	// Demonstrate flag deletion
	fmt.Println("\nCleaning up test flags:")
	for _, flag := range flags {
		if err := client.DeleteFlag(ctx, flag.Key); err != nil {
			log.Printf("Failed to delete flag %s: %v", flag.Key, err)
		} else {
			fmt.Printf("✓ Deleted: %s\n", flag.Key)
		}
	}

	// Final count
	finalFlags, err := client.GetAllFlags(ctx)
	if err != nil {
		log.Printf("Failed to get final flag count: %v", err)
	} else {
		fmt.Printf("\nFinal flag count: %d\n", len(finalFlags))
	}

	fmt.Println("\n✅ PostgreSQL storage example completed!")
}
