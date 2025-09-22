package main

import (
	"context"
	"log"
	"time"

	featureflag "github.com/ajeet-kumar1087/go-feature-flag"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/config"
)

func main() {
	// Production configuration with Redis
	config := featureflag.Config{
		Storage: featureflag.StorageConfig{
			Type: "redis",
			Redis: &featureflag.RedisConfig{
				Addr: "localhost:6379",
				DB:   0,
			},
		},
		Cache: featureflag.CacheConfig{
			Enabled: true,
			TTL:     featureflag.Duration(10 * time.Minute),
			MaxSize: 5000,
		},
		Observability: featureflag.ObservabilityConfig{
			Logging: config.LoggingConfig{
				Enabled: true,
				Level:   "info",
			},
			Metrics: featureflag.MetricsConfig{
				Enabled: true,
			},
		},
		DefaultFlags: []featureflag.FeatureFlag{
			{
				Key:         "maintenance-mode",
				Enabled:     false,
				Description: "Global maintenance mode",
				Metadata: map[string]string{
					"priority": "critical",
					"team":     "platform",
				},
			},
		},
	}

	client, err := featureflag.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Simulate production workload
	log.Println("🚀 Starting production simulation...")

	for i := 0; i < 100; i++ {
		// Check maintenance mode
		if enabled, _ := client.IsEnabled(ctx, "maintenance-mode"); enabled {
			log.Println("🚧 Application in maintenance mode")
			break
		}

		// Simulate feature flag checks
		checkFeatureFlags(client, ctx)

		time.Sleep(100 * time.Millisecond)
	}

	// Print final metrics
	metrics := client.GetMetrics()
	log.Printf("📊 Final Metrics:")
	log.Printf("   Flag checks: %d", metrics.FlagChecks)
	log.Printf("   Cache hits: %d", metrics.CacheHits)
	log.Printf("   Cache misses: %d", metrics.CacheMisses)
	if metrics.FlagChecks > 0 {
		hitRate := float64(metrics.CacheHits) / float64(metrics.FlagChecks) * 100
		log.Printf("   Cache hit rate: %.2f%%", hitRate)
	}
}

func checkFeatureFlags(client featureflag.Client, ctx context.Context) {
	flags := []string{
		"new-ui",
		"beta-features",
		"premium-tier",
		"analytics-tracking",
	}

	for _, flagKey := range flags {
		if enabled, _ := client.IsEnabled(ctx, flagKey); enabled {
			log.Printf("✅ %s is enabled", flagKey)
		}
	}
}
