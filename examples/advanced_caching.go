//go:build examples
// +build examples

// Package main demonstrates advanced caching features
package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ajeet-kumar1087/go-feature-flag/featureflag"
)

func main() {
	fmt.Println("=== Advanced Caching Example ===")

	// Configure client with aggressive caching
	config := featureflag.Config{
		Storage: featureflag.StorageConfig{
			Type: "memory", // Use memory for predictable performance
		},
		Cache: featureflag.CacheConfig{
			Enabled: true,
			TTL:     featureflag.Duration(30 * time.Second), // Short TTL for demo
			MaxSize: 100,
		},
		Observability: featureflag.ObservabilityConfig{
			Logging: featureflag.LoggingConfig{
				Enabled: true,
				Level:   "debug",
			},
			Metrics: featureflag.MetricsConfig{
				Enabled: true,
			},
		},
	}

	client, err := featureflag.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create test flags
	testFlags := []featureflag.FeatureFlag{
		{
			Key:         "cache-test-1",
			Enabled:     true,
			Description: "First cache test flag",
		},
		{
			Key:         "cache-test-2",
			Enabled:     false,
			Description: "Second cache test flag",
		},
		{
			Key:         "cache-test-3",
			Enabled:     true,
			Description: "Third cache test flag",
		},
	}

	fmt.Println("\nCreating test flags:")
	for _, flag := range testFlags {
		if err := client.SetFlag(ctx, flag); err != nil {
			log.Printf("Failed to create flag %s: %v", flag.Key, err)
		} else {
			fmt.Printf("✓ Created: %s\n", flag.Key)
		}
	}

	// Demonstrate cache miss vs cache hit performance
	fmt.Println("\n=== Cache Performance Comparison ===")

	// First call - cache miss
	fmt.Println("First call (cache miss):")
	start := time.Now()
	enabled, _ := client.IsEnabled(ctx, "cache-test-1")
	firstDuration := time.Since(start)
	fmt.Printf("- Result: %v, Duration: %v\n", enabled, firstDuration)

	// Second call - cache hit
	fmt.Println("Second call (cache hit):")
	start = time.Now()
	enabled, _ = client.IsEnabled(ctx, "cache-test-1")
	secondDuration := time.Since(start)
	fmt.Printf("- Result: %v, Duration: %v\n", enabled, secondDuration)

	if firstDuration > 0 && secondDuration > 0 {
		speedup := float64(firstDuration) / float64(secondDuration)
		fmt.Printf("- Cache speedup: %.2fx\n", speedup)
	}

	// Demonstrate concurrent access with caching
	fmt.Println("\n=== Concurrent Cache Access ===")
	concurrentRequests := 100
	var wg sync.WaitGroup
	results := make(chan time.Duration, concurrentRequests)

	start = time.Now()
	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			requestStart := time.Now()
			client.IsEnabled(ctx, "cache-test-1")
			results <- time.Since(requestStart)
		}(i)
	}

	wg.Wait()
	close(results)
	totalDuration := time.Since(start)

	// Calculate statistics
	var totalRequestTime time.Duration
	var minDuration, maxDuration time.Duration
	count := 0

	for duration := range results {
		if count == 0 {
			minDuration = duration
			maxDuration = duration
		} else {
			if duration < minDuration {
				minDuration = duration
			}
			if duration > maxDuration {
				maxDuration = duration
			}
		}
		totalRequestTime += duration
		count++
	}

	avgDuration := totalRequestTime / time.Duration(count)

	fmt.Printf("Concurrent requests: %d\n", concurrentRequests)
	fmt.Printf("Total time: %v\n", totalDuration)
	fmt.Printf("Average request time: %v\n", avgDuration)
	fmt.Printf("Min request time: %v\n", minDuration)
	fmt.Printf("Max request time: %v\n", maxDuration)
	fmt.Printf("Requests per second: %.2f\n", float64(concurrentRequests)/totalDuration.Seconds())

	// Demonstrate cache eviction with max size
	fmt.Println("\n=== Cache Eviction Test ===")
	fmt.Printf("Cache max size: %d\n", config.Cache.MaxSize)

	// Create more flags than cache can hold
	fmt.Println("Creating flags to test cache eviction:")
	for i := 0; i < 150; i++ { // More than MaxSize (100)
		flag := featureflag.FeatureFlag{
			Key:         fmt.Sprintf("eviction-test-%d", i),
			Enabled:     i%2 == 0, // Alternate enabled/disabled
			Description: fmt.Sprintf("Eviction test flag %d", i),
		}
		client.SetFlag(ctx, flag)
	}

	// Access all flags to populate cache
	fmt.Println("Accessing all flags to test cache behavior:")
	for i := 0; i < 150; i++ {
		client.IsEnabled(ctx, fmt.Sprintf("eviction-test-%d", i))
	}

	// Show metrics
	metrics := client.GetMetrics()
	fmt.Printf("\nCache Metrics:\n")
	fmt.Printf("- Total flag checks: %d\n", metrics.FlagChecks)
	fmt.Printf("- Cache hits: %d\n", metrics.CacheHits)
	fmt.Printf("- Cache misses: %d\n", metrics.CacheMisses)
	if metrics.FlagChecks > 0 {
		hitRate := float64(metrics.CacheHits) / float64(metrics.FlagChecks) * 100
		fmt.Printf("- Cache hit rate: %.2f%%\n", hitRate)
	}

	// Demonstrate TTL expiration
	fmt.Println("\n=== Cache TTL Expiration Test ===")
	fmt.Printf("Cache TTL: %v\n", time.Duration(config.Cache.TTL))

	// Access a flag to cache it
	client.IsEnabled(ctx, "cache-test-2")
	fmt.Println("Flag cached, waiting for TTL expiration...")

	// Wait for TTL to expire
	time.Sleep(time.Duration(config.Cache.TTL) + time.Second)

	// Access again - should be cache miss
	fmt.Println("Accessing flag after TTL expiration (should be cache miss):")
	start = time.Now()
	client.IsEnabled(ctx, "cache-test-2")
	postTTLDuration := time.Since(start)
	fmt.Printf("Duration: %v (cache miss expected)\n", postTTLDuration)

	// Clean up test flags
	fmt.Println("\nCleaning up test flags:")
	for _, flag := range testFlags {
		client.DeleteFlag(ctx, flag.Key)
	}
	for i := 0; i < 150; i++ {
		client.DeleteFlag(ctx, fmt.Sprintf("eviction-test-%d", i))
	}

	// Final metrics
	finalMetrics := client.GetMetrics()
	fmt.Printf("\nFinal Cache Metrics:\n")
	fmt.Printf("- Total flag checks: %d\n", finalMetrics.FlagChecks)
	fmt.Printf("- Cache hits: %d\n", finalMetrics.CacheHits)
	fmt.Printf("- Cache misses: %d\n", finalMetrics.CacheMisses)

	fmt.Println("\n✅ Advanced caching example completed!")
}
