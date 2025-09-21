//go:build examples
// +build examples

// Package main demonstrates advanced configuration scenarios
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ajeet-kumar1087/go-feature-flag/featureflag"
)

func main() {
	fmt.Println("=== Advanced Configuration Example ===")

	// Example 1: Production-like configuration
	fmt.Println("\n1. Production Configuration:")
	productionConfig := featureflag.Config{
		Storage: featureflag.StorageConfig{
			Type: "postgres",
			Postgres: &featureflag.PostgresConfig{
				Host:     "prod-db.example.com",
				Port:     5432,
				Database: "feature_flags",
				Username: "ff_service",
				Password: "secure_password",
				SSLMode:  "require",
			},
		},
		Cache: featureflag.CacheConfig{
			Enabled: true,
			TTL:     featureflag.Duration(1 * time.Hour), // Long TTL for production
			MaxSize: 10000,                               // Large cache for production
		},
		Observability: featureflag.ObservabilityConfig{
			Logging: featureflag.LoggingConfig{
				Enabled: true,
				Level:   "warn", // Only warnings and errors in production
			},
			Metrics: featureflag.MetricsConfig{
				Enabled: true, // Always enable metrics in production
			},
		},
		DefaultFlags: []featureflag.FeatureFlag{
			{
				Key:         "maintenance-mode",
				Enabled:     false,
				Description: "Enable maintenance mode",
				Metadata:    map[string]string{"critical": "true"},
			},
			{
				Key:         "feature-rollout-gate",
				Enabled:     true,
				Description: "Master gate for new feature rollouts",
				Metadata:    map[string]string{"type": "gate"},
			},
		},
	}

	fmt.Printf("Storage: %s (SSL: %s)\n", productionConfig.Storage.Type, productionConfig.Storage.Postgres.SSLMode)
	fmt.Printf("Cache: TTL=%v, MaxSize=%d\n", time.Duration(productionConfig.Cache.TTL), productionConfig.Cache.MaxSize)
	fmt.Printf("Logging: Level=%s\n", productionConfig.Observability.Logging.Level)
	fmt.Printf("Default flags: %d\n", len(productionConfig.DefaultFlags))

	// Example 2: Development configuration
	fmt.Println("\n2. Development Configuration:")
	devConfig := featureflag.Config{
		Storage: featureflag.StorageConfig{
			Type: "memory", // Fast, no external dependencies
		},
		Cache: featureflag.CacheConfig{
			Enabled: false, // Disable cache for immediate feedback
		},
		Observability: featureflag.ObservabilityConfig{
			Logging: featureflag.LoggingConfig{
				Enabled: true,
				Level:   "debug", // Verbose logging for development
			},
			Metrics: featureflag.MetricsConfig{
				Enabled: false, // Metrics not needed in dev
			},
		},
		DefaultFlags: []featureflag.FeatureFlag{
			{
				Key:         "dev-debug-mode",
				Enabled:     true,
				Description: "Enable debug features for development",
			},
			{
				Key:         "experimental-features",
				Enabled:     true,
				Description: "Enable all experimental features in dev",
			},
		},
	}

	fmt.Printf("Storage: %s\n", devConfig.Storage.Type)
	fmt.Printf("Cache: Enabled=%v\n", devConfig.Cache.Enabled)
	fmt.Printf("Logging: Level=%s\n", devConfig.Observability.Logging.Level)
	fmt.Printf("Default flags: %d\n", len(devConfig.DefaultFlags))

	// Example 3: Environment-based configuration
	fmt.Println("\n3. Environment-Based Configuration:")

	// Set environment variables
	os.Setenv("FEATUREFLAG_STORAGE_TYPE", "redis")
	os.Setenv("FEATUREFLAG_REDIS_ADDR", "redis-cluster.example.com:6379")
	os.Setenv("FEATUREFLAG_REDIS_PASSWORD", "redis_secret")
	os.Setenv("FEATUREFLAG_CACHE_ENABLED", "true")
	os.Setenv("FEATUREFLAG_CACHE_TTL", "30m")
	os.Setenv("FEATUREFLAG_LOGGING_ENABLED", "true")
	os.Setenv("FEATUREFLAG_LOGGING_LEVEL", "info")
	os.Setenv("FEATUREFLAG_METRICS_ENABLED", "true")

	envConfig := featureflag.LoadConfigFromEnv()
	fmt.Printf("Storage: %s\n", envConfig.Storage.Type)
	if envConfig.Storage.Redis != nil {
		fmt.Printf("Redis: %s\n", envConfig.Storage.Redis.Addr)
	}
	fmt.Printf("Cache: TTL=%v\n", time.Duration(envConfig.Cache.TTL))

	// Example 4: Configuration validation
	fmt.Println("\n4. Configuration Validation:")

	// Valid configuration
	validConfig := featureflag.DefaultConfig()
	if err := validConfig.Validate(); err != nil {
		fmt.Printf("❌ Default config validation failed: %v\n", err)
	} else {
		fmt.Printf("✓ Default config is valid\n")
	}

	// Invalid configuration examples
	invalidConfigs := []struct {
		name   string
		config featureflag.Config
	}{
		{
			name: "Invalid storage type",
			config: featureflag.Config{
				Storage: featureflag.StorageConfig{
					Type: "invalid-storage",
				},
			},
		},
		{
			name: "Redis config missing",
			config: featureflag.Config{
				Storage: featureflag.StorageConfig{
					Type: "redis",
					// Redis config is nil
				},
			},
		},
		{
			name: "Invalid cache TTL",
			config: featureflag.Config{
				Storage: featureflag.StorageConfig{
					Type: "memory",
				},
				Cache: featureflag.CacheConfig{
					TTL: featureflag.Duration(-1 * time.Hour), // Negative TTL
				},
			},
		},
	}

	for _, test := range invalidConfigs {
		if err := test.config.Validate(); err != nil {
			fmt.Printf("✓ %s: %v\n", test.name, err)
		} else {
			fmt.Printf("❌ %s: Expected validation error but got none\n", test.name)
		}
	}

	// Example 5: Configuration with working client
	fmt.Println("\n5. Working Configuration Example:")

	workingConfig := featureflag.Config{
		Storage: featureflag.StorageConfig{
			Type: "memory",
		},
		Cache: featureflag.CacheConfig{
			Enabled: true,
			TTL:     featureflag.Duration(5 * time.Minute),
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
				Key:         "config-example-flag",
				Enabled:     true,
				Description: "Example flag from configuration",
				Metadata: map[string]string{
					"source": "configuration",
					"type":   "example",
				},
			},
		},
	}

	client, err := featureflag.NewClient(workingConfig)
	if err != nil {
		log.Printf("Failed to create client: %v", err)
	} else {
		defer client.Close()

		ctx := context.Background()

		// Test the default flag
		if enabled, _ := client.IsEnabled(ctx, "config-example-flag"); enabled {
			fmt.Println("✓ Default flag from configuration is working")
		}

		// Create additional flag
		testFlag := featureflag.FeatureFlag{
			Key:         "runtime-flag",
			Enabled:     false,
			Description: "Flag created at runtime",
		}

		if err := client.SetFlag(ctx, testFlag); err != nil {
			log.Printf("Failed to set runtime flag: %v", err)
		} else {
			fmt.Println("✓ Runtime flag created successfully")
		}

		// Show all flags
		allFlags, err := client.GetAllFlags(ctx)
		if err != nil {
			log.Printf("Failed to get all flags: %v", err)
		} else {
			fmt.Printf("Total flags: %d\n", len(allFlags))
			for _, flag := range allFlags {
				fmt.Printf("- %s: %v\n", flag.Key, flag.Enabled)
			}
		}
	}

	// Clean up environment variables
	envVars := []string{
		"FEATUREFLAG_STORAGE_TYPE",
		"FEATUREFLAG_REDIS_ADDR",
		"FEATUREFLAG_REDIS_PASSWORD",
		"FEATUREFLAG_CACHE_ENABLED",
		"FEATUREFLAG_CACHE_TTL",
		"FEATUREFLAG_LOGGING_ENABLED",
		"FEATUREFLAG_LOGGING_LEVEL",
		"FEATUREFLAG_METRICS_ENABLED",
	}

	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}

	fmt.Println("\n✅ Advanced configuration example completed!")
}
