package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/cache"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/client"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/config"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/core"
)

// TestClient_ConcurrentIsEnabled tests concurrent IsEnabled calls
func TestClient_ConcurrentIsEnabled(t *testing.T) {
	config := config.Config{
		Storage: config.StorageConfig{
			Type: "memory",
		},
		Cache: config.CacheConfig{
			Enabled: true,
			TTL:     config.Duration(5 * time.Minute),
			MaxSize: 1000,
		},
	}

	client, err := client.NewClient(config)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Pre-populate with test flags
	numFlags := 100
	for i := 0; i < numFlags; i++ {
		flag := core.FeatureFlag{
			Key:     fmt.Sprintf("flag-%d", i),
			Enabled: i%2 == 0,
		}
		if err := client.SetFlag(ctx, flag); err != nil {
			t.Fatal(err)
		}
	}

	// Test concurrent reads
	numGoroutines := runtime.NumCPU() * 4
	numOperations := 1000
	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("flag-%d", j%numFlags)
				enabled, err := client.IsEnabled(ctx, key)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
					// Verify expected result
					expected := (j%numFlags)%2 == 0
					if enabled != expected {
						t.Errorf("Goroutine %d: Expected flag %s to be %v, got %v", goroutineID, key, expected, enabled)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	totalOperations := int64(numGoroutines * numOperations)
	if successCount != totalOperations {
		t.Errorf("Expected %d successful operations, got %d (errors: %d)", totalOperations, successCount, errorCount)
	}
}

// TestClient_ConcurrentReadWrite tests concurrent reads and writes
func TestClient_ConcurrentReadWrite(t *testing.T) {
	config := config.Config{
		Storage: config.StorageConfig{
			Type: "memory",
		},
		Cache: config.CacheConfig{
			Enabled: true,
			TTL:     config.Duration(5 * time.Minute),
			MaxSize: 1000,
		},
	}

	client, err := client.NewClient(config)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Pre-populate with test flags
	numFlags := 50
	for i := 0; i < numFlags; i++ {
		flag := core.FeatureFlag{
			Key:     fmt.Sprintf("flag-%d", i),
			Enabled: false,
		}
		if err := client.SetFlag(ctx, flag); err != nil {
			t.Fatal(err)
		}
	}

	numReaders := runtime.NumCPU() * 2
	numWriters := runtime.NumCPU()
	operationsPerGoroutine := 500
	var wg sync.WaitGroup
	var readCount, writeCount, errorCount int64

	// Start readers
	wg.Add(numReaders)
	for i := 0; i < numReaders; i++ {
		go func(readerID int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("flag-%d", j%numFlags)
				_, err := client.IsEnabled(ctx, key)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&readCount, 1)
				}
			}
		}(i)
	}

	// Start writers
	wg.Add(numWriters)
	for i := 0; i < numWriters; i++ {
		go func(writerID int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("flag-%d", j%numFlags)
				flag := core.FeatureFlag{
					Key:     key,
					Enabled: j%2 == 0,
				}
				if err := client.SetFlag(ctx, flag); err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&writeCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	expectedReads := int64(numReaders * operationsPerGoroutine)
	expectedWrites := int64(numWriters * operationsPerGoroutine)

	if readCount != expectedReads {
		t.Errorf("Expected %d reads, got %d", expectedReads, readCount)
	}
	if writeCount != expectedWrites {
		t.Errorf("Expected %d writes, got %d", expectedWrites, writeCount)
	}
	if errorCount > 0 {
		t.Errorf("Unexpected errors: %d", errorCount)
	}
}

// TestMemoryStore_ConcurrentOperations tests memory store thread safety
func TestMemoryStore_ConcurrentOperations(t *testing.T) {
	store := config.NewMemoryStore()
	defer store.Close()

	ctx := context.Background()
	numGoroutines := runtime.NumCPU() * 4
	operationsPerGoroutine := 1000
	numFlags := 100

	var wg sync.WaitGroup
	var readCount, writeCount, deleteCount, errorCount int64

	// Start concurrent operations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("flag-%d", j%numFlags)

				switch j % 10 {
				case 0, 1: // 20% writes
					flag := core.FeatureFlag{
						Key:     key,
						Enabled: j%2 == 0,
					}
					if err := store.Set(ctx, flag); err != nil {
						atomic.AddInt64(&errorCount, 1)
					} else {
						atomic.AddInt64(&writeCount, 1)
					}
				case 2: // 10% deletes
					if err := store.Delete(ctx, key); err != nil {
						// Delete errors are expected when flag doesn't exist
						if !core.IsNotFoundError(err) {
							atomic.AddInt64(&errorCount, 1)
						}
					} else {
						atomic.AddInt64(&deleteCount, 1)
					}
				default: // 70% reads
					_, err := store.Get(ctx, key)
					if err != nil {
						// Read errors are expected when flag doesn't exist
						if !core.IsNotFoundError(err) {
							atomic.AddInt64(&errorCount, 1)
						}
					} else {
						atomic.AddInt64(&readCount, 1)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	if errorCount > 0 {
		t.Errorf("Unexpected errors: %d", errorCount)
	}

	t.Logf("Operations completed - Reads: %d, Writes: %d, Deletes: %d, Errors: %d",
		readCount, writeCount, deleteCount, errorCount)
}

// TestCache_ConcurrentOperations tests cache thread safety
func TestCache_ConcurrentOperations(t *testing.T) {
	cache := cache.NewCache(1000, 5*time.Minute)
	defer cache.Close()

	numGoroutines := runtime.NumCPU() * 4
	operationsPerGoroutine := 1000
	numFlags := 100

	var wg sync.WaitGroup
	var readCount, writeCount, errorCount int64

	// Start concurrent operations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("flag-%d", j%numFlags)

				if j%5 == 0 { // 20% writes
					flag := &core.FeatureFlag{
						Key:     key,
						Enabled: j%2 == 0,
					}
					cache.Set(key, flag)
					atomic.AddInt64(&writeCount, 1)
				} else { // 80% reads
					_, found := cache.Get(key)
					if found {
						atomic.AddInt64(&readCount, 1)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	t.Logf("Cache operations completed - Reads: %d, Writes: %d, Errors: %d",
		readCount, writeCount, errorCount)
}

// TestClient_ConcurrentClose tests that client can be safely closed during concurrent operations
func TestClient_ConcurrentClose(t *testing.T) {
	config := config.Config{
		Storage: config.StorageConfig{
			Type: "memory",
		},
		Cache: config.CacheConfig{
			Enabled: true,
			TTL:     config.Duration(5 * time.Minute),
			MaxSize: 1000,
		},
	}

	client, err := client.NewClient(config)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// Pre-populate with test flags
	for i := 0; i < 10; i++ {
		flag := core.FeatureFlag{
			Key:     fmt.Sprintf("flag-%d", i),
			Enabled: i%2 == 0,
		}
		client.SetFlag(ctx, flag)
	}

	var wg sync.WaitGroup
	numGoroutines := 10
	operationsPerGoroutine := 100

	// Start concurrent operations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("flag-%d", j%10)
				client.IsEnabled(ctx, key)
				// Small delay to increase chance of operations running during close
				time.Sleep(time.Microsecond)
			}
		}(i)
	}

	// Close client after a short delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		client.Close()
	}()

	wg.Wait()
	// Test passes if no panic occurs
}

// TestClient_RaceConditions tests for race conditions using race detector
func TestClient_RaceConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race condition test in short mode")
	}

	config := config.Config{
		Storage: config.StorageConfig{
			Type: "memory",
		},
		Cache: config.CacheConfig{
			Enabled: true,
			TTL:     config.Duration(1 * time.Second), // Short TTL to test expiration races
			MaxSize: 100,
		},
	}

	client, err := client.NewClient(config)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()
	numGoroutines := 20
	duration := 2 * time.Second
	var wg sync.WaitGroup

	// Start multiple goroutines doing different operations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			start := time.Now()
			operationCount := 0

			for time.Since(start) < duration {
				key := fmt.Sprintf("flag-%d", operationCount%20)

				switch operationCount % 6 {
				case 0, 1, 2: // 50% IsEnabled calls
					client.IsEnabled(ctx, key)
				case 3: // 16.7% SetFlag calls
					flag := core.FeatureFlag{
						Key:     key,
						Enabled: operationCount%2 == 0,
					}
					client.SetFlag(ctx, flag)
				case 4: // 16.7% GetFlag calls
					client.GetFlag(ctx, key)
				case 5: // 16.7% DeleteFlag calls
					client.DeleteFlag(ctx, key)
				}
				operationCount++
			}
		}(i)
	}

	wg.Wait()
	// Test passes if race detector doesn't find any races
}

// TestCache_ConcurrentTTLExpiration tests concurrent access during TTL expiration
func TestCache_ConcurrentTTLExpiration(t *testing.T) {
	cache := cache.NewCache(100, 50*time.Millisecond) // Very short TTL
	defer cache.Close()

	numGoroutines := 10
	var wg sync.WaitGroup

	// Pre-populate cache
	for i := 0; i < 20; i++ {
		flag := &core.FeatureFlag{
			Key:     fmt.Sprintf("flag-%d", i),
			Enabled: i%2 == 0,
		}
		cache.Set(flag.Key, flag)
	}

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				key := fmt.Sprintf("flag-%d", j%20)

				if j%10 == 0 {
					// Refresh some items
					flag := &core.FeatureFlag{
						Key:     key,
						Enabled: j%2 == 0,
					}
					cache.Set(key, flag)
				} else {
					// Try to get items (some may be expired)
					cache.Get(key)
				}

				// Small delay to allow TTL expiration
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	// Test passes if no race conditions or panics occur
}

// TestClient_StressTest performs a comprehensive stress test
func TestClient_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	config := config.Config{
		Storage: config.StorageConfig{
			Type: "memory",
		},
		Cache: config.CacheConfig{
			Enabled: true,
			TTL:     config.Duration(2 * time.Second),
			MaxSize: 500,
		},
	}

	client, err := client.NewClient(config)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()
	numGoroutines := runtime.NumCPU() * 8
	testDuration := 5 * time.Second
	var wg sync.WaitGroup
	var totalOperations int64

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			start := time.Now()
			operations := 0

			for time.Since(start) < testDuration {
				key := fmt.Sprintf("flag-%d", operations%200)

				switch operations % 20 {
				case 0, 1: // 10% writes
					flag := core.FeatureFlag{
						Key:     key,
						Enabled: operations%2 == 0,
					}
					client.SetFlag(ctx, flag)
				case 2: // 5% deletes
					client.DeleteFlag(ctx, key)
				case 3: // 5% GetFlag calls
					client.GetFlag(ctx, key)
				case 4: // 5% GetAllFlags calls
					client.GetAllFlags(ctx)
				default: // 75% IsEnabled calls (hot path)
					client.IsEnabled(ctx, key)
				}
				operations++
			}

			atomic.AddInt64(&totalOperations, int64(operations))
		}(i)
	}

	wg.Wait()

	opsPerSecond := float64(totalOperations) / testDuration.Seconds()
	t.Logf("Stress test completed: %d total operations, %.0f ops/sec", totalOperations, opsPerSecond)

	// Verify we achieved reasonable throughput
	if opsPerSecond < 10000 {
		t.Logf("Warning: Low throughput detected: %.0f ops/sec", opsPerSecond)
	}
}
