//go:build integration
// +build integration

package performance

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestLoadScenarios runs comprehensive load tests against different storage backends
func TestLoadScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load tests in short mode")
	}

	t.Run("Memory Store Load Test", func(t *testing.T) {
		config := Config{
			Storage: StorageConfig{
				Type: "memory",
			},
			Cache: CacheConfig{
				Enabled: true,
				TTL:     Duration(5 * time.Minute),
				MaxSize: 10000,
			},
		}

		runLoadTest(t, config, "Memory", 50000, 100)
	})

	t.Run("Redis Store Load Test", func(t *testing.T) {
		redisURL := os.Getenv("REDIS_TEST_URL")
		if redisURL == "" {
			t.Skip("REDIS_TEST_URL not set, skipping Redis load test")
		}

		config := Config{
			Storage: StorageConfig{
				Type: "redis",
				Redis: &RedisConfig{
					Addr: "localhost:6379",
					DB:   3, // Use different DB for load tests
				},
			},
			Cache: CacheConfig{
				Enabled: true,
				TTL:     Duration(5 * time.Minute),
				MaxSize: 10000,
			},
		}

		runLoadTest(t, config, "Redis", 10000, 50)
	})

	t.Run("PostgreSQL Store Load Test", func(t *testing.T) {
		postgresURL := os.Getenv("POSTGRES_TEST_URL")
		if postgresURL == "" {
			t.Skip("POSTGRES_TEST_URL not set, skipping PostgreSQL load test")
		}

		config := Config{
			Storage: StorageConfig{
				Type: "postgres",
				Postgres: &PostgresConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "testdb",
					Username: "testuser",
					Password: "testpass",
					SSLMode:  "disable",
				},
			},
			Cache: CacheConfig{
				Enabled: true,
				TTL:     Duration(5 * time.Minute),
				MaxSize: 10000,
			},
		}

		runLoadTest(t, config, "PostgreSQL", 5000, 25)
	})
}

// runLoadTest executes a comprehensive load test scenario
func runLoadTest(t *testing.T, config Config, storeName string, totalOps int, numGoroutines int) {
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create %s client: %v", storeName, err)
	}
	defer client.Close()

	ctx := context.Background()

	// Pre-populate with test flags
	numFlags := 1000
	t.Logf("Pre-populating %s with %d flags...", storeName, numFlags)

	start := time.Now()
	for i := 0; i < numFlags; i++ {
		flag := FeatureFlag{
			Key:         fmt.Sprintf("load-test-flag-%d", i),
			Enabled:     i%2 == 0,
			Description: fmt.Sprintf("Load test flag %d", i),
			Metadata: map[string]string{
				"index":     fmt.Sprintf("%d", i),
				"test_type": "load",
				"store":     storeName,
			},
		}

		if err := client.SetFlag(ctx, flag); err != nil {
			t.Fatalf("Failed to pre-populate flag %d: %v", i, err)
		}
	}

	prepTime := time.Since(start)
	t.Logf("Pre-population completed in %v", prepTime)

	// Run load test
	t.Logf("Starting %s load test: %d operations across %d goroutines", storeName, totalOps, numGoroutines)

	var (
		readCount    int64
		writeCount   int64
		deleteCount  int64
		errorCount   int64
		totalLatency int64
	)

	opsPerGoroutine := totalOps / numGoroutines
	var wg sync.WaitGroup

	start = time.Now()
	wg.Add(numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer wg.Done()

			for op := 0; op < opsPerGoroutine; op++ {
				opStart := time.Now()
				flagIndex := (goroutineID*opsPerGoroutine + op) % numFlags
				flagKey := fmt.Sprintf("load-test-flag-%d", flagIndex)

				switch op % 100 {
				case 0, 1: // 2% writes
					flag := FeatureFlag{
						Key:         flagKey,
						Enabled:     op%2 == 0,
						Description: fmt.Sprintf("Updated flag %d", flagIndex),
						Metadata: map[string]string{
							"updated_by": fmt.Sprintf("goroutine-%d", goroutineID),
							"op":         fmt.Sprintf("%d", op),
						},
					}

					if err := client.SetFlag(ctx, flag); err != nil {
						atomic.AddInt64(&errorCount, 1)
					} else {
						atomic.AddInt64(&writeCount, 1)
					}

				case 2: // 1% deletes (followed by recreation)
					if err := client.DeleteFlag(ctx, flagKey); err != nil {
						atomic.AddInt64(&errorCount, 1)
					} else {
						atomic.AddInt64(&deleteCount, 1)

						// Recreate the flag immediately
						flag := FeatureFlag{
							Key:         flagKey,
							Enabled:     flagIndex%2 == 0,
							Description: fmt.Sprintf("Recreated flag %d", flagIndex),
						}
						client.SetFlag(ctx, flag)
					}

				default: // 97% reads (IsEnabled calls)
					_, err := client.IsEnabled(ctx, flagKey)
					if err != nil {
						atomic.AddInt64(&errorCount, 1)
					} else {
						atomic.AddInt64(&readCount, 1)
					}
				}

				opDuration := time.Since(opStart)
				atomic.AddInt64(&totalLatency, int64(opDuration))
			}
		}(g)
	}

	wg.Wait()
	totalDuration := time.Since(start)

	// Calculate metrics
	totalOpsCompleted := readCount + writeCount + deleteCount
	opsPerSecond := float64(totalOpsCompleted) / totalDuration.Seconds()
	avgLatency := time.Duration(totalLatency / totalOpsCompleted)

	// Report results
	t.Logf("%s Load Test Results:", storeName)
	t.Logf("  Total Duration: %v", totalDuration)
	t.Logf("  Operations Completed: %d", totalOpsCompleted)
	t.Logf("  Operations/Second: %.2f", opsPerSecond)
	t.Logf("  Average Latency: %v", avgLatency)
	t.Logf("  Reads: %d, Writes: %d, Deletes: %d", readCount, writeCount, deleteCount)
	t.Logf("  Errors: %d", errorCount)

	// Performance assertions
	if errorCount > totalOpsCompleted/100 { // Allow up to 1% errors
		t.Errorf("Too many errors: %d (%.2f%%)", errorCount, float64(errorCount)/float64(totalOpsCompleted)*100)
	}

	// Store-specific performance expectations
	switch storeName {
	case "Memory":
		if opsPerSecond < 10000 {
			t.Logf("Warning: Memory store performance below expected (%.2f ops/sec)", opsPerSecond)
		}
		if avgLatency > time.Millisecond {
			t.Logf("Warning: Memory store latency higher than expected (%v)", avgLatency)
		}
	case "Redis":
		if opsPerSecond < 1000 {
			t.Logf("Warning: Redis store performance below expected (%.2f ops/sec)", opsPerSecond)
		}
	case "PostgreSQL":
		if opsPerSecond < 500 {
			t.Logf("Warning: PostgreSQL store performance below expected (%.2f ops/sec)", opsPerSecond)
		}
	}

	// Clean up test flags
	t.Logf("Cleaning up %s test flags...", storeName)
	cleanupStart := time.Now()

	for i := 0; i < numFlags; i++ {
		flagKey := fmt.Sprintf("load-test-flag-%d", i)
		client.DeleteFlag(ctx, flagKey)
	}

	cleanupTime := time.Since(cleanupStart)
	t.Logf("Cleanup completed in %v", cleanupTime)
}

// TestConcurrencyLimits tests the system under extreme concurrency
func TestConcurrencyLimits(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency limits test in short mode")
	}

	config := Config{
		Storage: StorageConfig{
			Type: "memory",
		},
		Cache: CacheConfig{
			Enabled: true,
			TTL:     Duration(5 * time.Minute),
			MaxSize: 5000,
		},
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Pre-populate with flags
	numFlags := 100
	for i := 0; i < numFlags; i++ {
		flag := FeatureFlag{
			Key:     fmt.Sprintf("concurrency-test-flag-%d", i),
			Enabled: i%2 == 0,
		}
		client.SetFlag(ctx, flag)
	}

	// Test with very high concurrency
	maxGoroutines := runtime.NumCPU() * 50 // Much higher than typical
	operationsPerGoroutine := 1000

	t.Logf("Testing concurrency limits: %d goroutines, %d ops each", maxGoroutines, operationsPerGoroutine)

	var (
		successCount int64
		errorCount   int64
	)

	start := time.Now()
	var wg sync.WaitGroup

	wg.Add(maxGoroutines)
	for g := 0; g < maxGoroutines; g++ {
		go func(goroutineID int) {
			defer wg.Done()

			for op := 0; op < operationsPerGoroutine; op++ {
				flagIndex := op % numFlags
				flagKey := fmt.Sprintf("concurrency-test-flag-%d", flagIndex)

				_, err := client.IsEnabled(ctx, flagKey)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}(g)
	}

	wg.Wait()
	duration := time.Since(start)

	totalOps := successCount + errorCount
	opsPerSecond := float64(totalOps) / duration.Seconds()

	t.Logf("Concurrency Limits Test Results:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Total Operations: %d", totalOps)
	t.Logf("  Operations/Second: %.2f", opsPerSecond)
	t.Logf("  Success: %d, Errors: %d", successCount, errorCount)
	t.Logf("  Error Rate: %.2f%%", float64(errorCount)/float64(totalOps)*100)

	// Should handle high concurrency without excessive errors
	if errorCount > totalOps/20 { // Allow up to 5% errors under extreme load
		t.Errorf("Too many errors under high concurrency: %d (%.2f%%)", errorCount, float64(errorCount)/float64(totalOps)*100)
	}

	// Clean up
	for i := 0; i < numFlags; i++ {
		flagKey := fmt.Sprintf("concurrency-test-flag-%d", i)
		client.DeleteFlag(ctx, flagKey)
	}
}

// TestMemoryUsage tests memory usage under load
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory usage test in short mode")
	}

	// Force garbage collection before starting
	runtime.GC()
	runtime.GC()

	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	config := Config{
		Storage: StorageConfig{
			Type: "memory",
		},
		Cache: CacheConfig{
			Enabled: true,
			TTL:     Duration(10 * time.Minute),
			MaxSize: 10000,
		},
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create many flags to test memory usage
	numFlags := 10000
	t.Logf("Creating %d flags to test memory usage...", numFlags)

	for i := 0; i < numFlags; i++ {
		flag := FeatureFlag{
			Key:         fmt.Sprintf("memory-test-flag-%d", i),
			Enabled:     i%2 == 0,
			Description: fmt.Sprintf("Memory test flag %d with some description text to use more memory", i),
			Metadata: map[string]string{
				"index":       fmt.Sprintf("%d", i),
				"test_type":   "memory",
				"description": "This is additional metadata to increase memory usage per flag",
				"category":    fmt.Sprintf("category-%d", i%10),
			},
		}

		if err := client.SetFlag(ctx, flag); err != nil {
			t.Fatalf("Failed to create flag %d: %v", i, err)
		}

		// Check memory usage periodically
		if i%1000 == 999 {
			runtime.GC()
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			t.Logf("After %d flags: Alloc=%d KB, TotalAlloc=%d KB", i+1, m.Alloc/1024, m.TotalAlloc/1024)
		}
	}

	// Final memory measurement
	runtime.GC()
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	memoryUsed := m2.Alloc - m1.Alloc
	memoryPerFlag := memoryUsed / uint64(numFlags)

	t.Logf("Memory Usage Test Results:")
	t.Logf("  Initial Memory: %d KB", m1.Alloc/1024)
	t.Logf("  Final Memory: %d KB", m2.Alloc/1024)
	t.Logf("  Memory Used: %d KB", memoryUsed/1024)
	t.Logf("  Memory per Flag: %d bytes", memoryPerFlag)

	// Memory usage should be reasonable
	maxMemoryPerFlag := uint64(1024) // 1KB per flag should be more than enough
	if memoryPerFlag > maxMemoryPerFlag {
		t.Errorf("Memory usage per flag too high: %d bytes (max: %d)", memoryPerFlag, maxMemoryPerFlag)
	}

	// Test that we can still perform operations efficiently
	start := time.Now()
	for i := 0; i < 1000; i++ {
		flagKey := fmt.Sprintf("memory-test-flag-%d", i)
		client.IsEnabled(ctx, flagKey)
	}
	duration := time.Since(start)

	avgLatency := duration / 1000
	t.Logf("Performance with %d flags: avg latency %v", numFlags, avgLatency)

	if avgLatency > 10*time.Millisecond {
		t.Logf("Warning: High latency with many flags: %v", avgLatency)
	}

	// Clean up
	t.Logf("Cleaning up memory test flags...")
	for i := 0; i < numFlags; i++ {
		flagKey := fmt.Sprintf("memory-test-flag-%d", i)
		client.DeleteFlag(ctx, flagKey)
	}

	// Verify memory is released
	runtime.GC()
	runtime.GC()
	var m3 runtime.MemStats
	runtime.ReadMemStats(&m3)

	t.Logf("Memory after cleanup: %d KB", m3.Alloc/1024)
}

// TestCachePerformance tests cache hit rates and performance
func TestCachePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping cache performance test in short mode")
	}

	// Test with cache enabled
	configWithCache := Config{
		Storage: StorageConfig{
			Type: "memory",
		},
		Cache: CacheConfig{
			Enabled: true,
			TTL:     Duration(5 * time.Minute),
			MaxSize: 1000,
		},
	}

	clientWithCache, err := NewClient(configWithCache)
	if err != nil {
		t.Fatalf("Failed to create client with cache: %v", err)
	}
	defer clientWithCache.Close()

	// Test without cache
	configWithoutCache := Config{
		Storage: StorageConfig{
			Type: "memory",
		},
		Cache: CacheConfig{
			Enabled: false,
		},
	}

	clientWithoutCache, err := NewClient(configWithoutCache)
	if err != nil {
		t.Fatalf("Failed to create client without cache: %v", err)
	}
	defer clientWithoutCache.Close()

	ctx := context.Background()

	// Pre-populate both clients with the same flags
	numFlags := 100
	for i := 0; i < numFlags; i++ {
		flag := FeatureFlag{
			Key:     fmt.Sprintf("cache-perf-flag-%d", i),
			Enabled: i%2 == 0,
		}
		clientWithCache.SetFlag(ctx, flag)
		clientWithoutCache.SetFlag(ctx, flag)
	}

	// Warm up cache
	for i := 0; i < numFlags; i++ {
		flagKey := fmt.Sprintf("cache-perf-flag-%d", i)
		clientWithCache.IsEnabled(ctx, flagKey)
	}

	// Test performance with cache
	numOperations := 10000
	start := time.Now()
	for i := 0; i < numOperations; i++ {
		flagKey := fmt.Sprintf("cache-perf-flag-%d", i%numFlags)
		clientWithCache.IsEnabled(ctx, flagKey)
	}
	cachedDuration := time.Since(start)

	// Test performance without cache
	start = time.Now()
	for i := 0; i < numOperations; i++ {
		flagKey := fmt.Sprintf("cache-perf-flag-%d", i%numFlags)
		clientWithoutCache.IsEnabled(ctx, flagKey)
	}
	uncachedDuration := time.Since(start)

	cachedOpsPerSec := float64(numOperations) / cachedDuration.Seconds()
	uncachedOpsPerSec := float64(numOperations) / uncachedDuration.Seconds()
	speedup := cachedOpsPerSec / uncachedOpsPerSec

	t.Logf("Cache Performance Test Results:")
	t.Logf("  Cached: %v (%0.f ops/sec)", cachedDuration, cachedOpsPerSec)
	t.Logf("  Uncached: %v (%.0f ops/sec)", uncachedDuration, uncachedOpsPerSec)
	t.Logf("  Speedup: %.2fx", speedup)

	// Cache should provide significant performance improvement
	if speedup < 2.0 {
		t.Logf("Warning: Cache speedup lower than expected: %.2fx", speedup)
	}

	// Clean up
	for i := 0; i < numFlags; i++ {
		flagKey := fmt.Sprintf("cache-perf-flag-%d", i)
		clientWithCache.DeleteFlag(ctx, flagKey)
		clientWithoutCache.DeleteFlag(ctx, flagKey)
	}
}
