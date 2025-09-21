package performance

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// BenchmarkClient_IsEnabled_MemoryStore benchmarks the hot path with memory store
func BenchmarkClient_IsEnabled_MemoryStore(b *testing.B) {
	config := Config{
		Storage: StorageConfig{
			Type: "memory",
		},
		Cache: CacheConfig{
			Enabled: false, // Test without cache first
		},
		Observability: ObservabilityConfig{
			Logging: LoggingConfig{
				Enabled: false, // Disable logging for pure performance test
			},
			Metrics: MetricsConfig{
				Enabled: false, // Disable metrics for pure performance test
			},
		},
	}

	client, err := NewClient(config)
	if err != nil {
		b.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Pre-populate with test flags
	for i := 0; i < 100; i++ {
		flag := FeatureFlag{
			Key:     fmt.Sprintf("flag-%d", i),
			Enabled: i%2 == 0,
		}
		client.SetFlag(ctx, flag)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("flag-%d", i%100)
			client.IsEnabled(ctx, key)
			i++
		}
	})
}

// BenchmarkClient_IsEnabled_WithCache benchmarks the hot path with cache enabled
func BenchmarkClient_IsEnabled_WithCache(b *testing.B) {
	config := Config{
		Storage: StorageConfig{
			Type: "memory",
		},
		Cache: CacheConfig{
			Enabled: true,
			TTL:     Duration(5 * time.Minute),
			MaxSize: 1000,
		},
		Observability: ObservabilityConfig{
			Logging: LoggingConfig{
				Enabled: false,
			},
			Metrics: MetricsConfig{
				Enabled: false,
			},
		},
	}

	client, err := NewClient(config)
	if err != nil {
		b.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Pre-populate with test flags
	for i := 0; i < 100; i++ {
		flag := FeatureFlag{
			Key:     fmt.Sprintf("flag-%d", i),
			Enabled: i%2 == 0,
		}
		client.SetFlag(ctx, flag)
	}

	// Warm up cache
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("flag-%d", i)
		client.IsEnabled(ctx, key)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("flag-%d", i%100)
			client.IsEnabled(ctx, key)
			i++
		}
	})
}

// BenchmarkClient_IsEnabled_WithMetrics benchmarks with metrics enabled
func BenchmarkClient_IsEnabled_WithMetrics(b *testing.B) {
	config := Config{
		Storage: StorageConfig{
			Type: "memory",
		},
		Cache: CacheConfig{
			Enabled: true,
			TTL:     Duration(5 * time.Minute),
			MaxSize: 1000,
		},
		Observability: ObservabilityConfig{
			Logging: LoggingConfig{
				Enabled: false,
			},
			Metrics: MetricsConfig{
				Enabled: true,
			},
		},
	}

	client, err := NewClient(config)
	if err != nil {
		b.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Pre-populate with test flags
	for i := 0; i < 100; i++ {
		flag := FeatureFlag{
			Key:     fmt.Sprintf("flag-%d", i),
			Enabled: i%2 == 0,
		}
		client.SetFlag(ctx, flag)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("flag-%d", i%100)
			client.IsEnabled(ctx, key)
			i++
		}
	})
}

// BenchmarkMemoryStore_ConcurrentAccess benchmarks memory store under concurrent load
func BenchmarkMemoryStore_ConcurrentAccess(b *testing.B) {
	store := NewMemoryStore()
	defer store.Close()

	ctx := context.Background()

	// Pre-populate store
	for i := 0; i < 1000; i++ {
		flag := FeatureFlag{
			Key:     fmt.Sprintf("flag-%d", i),
			Enabled: i%2 == 0,
		}
		store.Set(ctx, flag)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("flag-%d", i%1000)
			store.Get(ctx, key)
			i++
		}
	})
}

// BenchmarkCache_ConcurrentReadWrite benchmarks cache under mixed read/write load
func BenchmarkCache_ConcurrentReadWrite(b *testing.B) {
	cache := NewCache(1000, 5*time.Minute)
	defer cache.Close()

	// Pre-populate cache
	for i := 0; i < 100; i++ {
		flag := &FeatureFlag{
			Key:     fmt.Sprintf("flag-%d", i),
			Enabled: i%2 == 0,
		}
		cache.Set(flag.Key, flag)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("flag-%d", i%100)
			if i%10 == 0 {
				// 10% writes
				flag := &FeatureFlag{
					Key:     key,
					Enabled: i%2 == 0,
				}
				cache.Set(key, flag)
			} else {
				// 90% reads
				cache.Get(key)
			}
			i++
		}
	})
}

// BenchmarkClient_MixedOperations benchmarks mixed client operations
func BenchmarkClient_MixedOperations(b *testing.B) {
	config := Config{
		Storage: StorageConfig{
			Type: "memory",
		},
		Cache: CacheConfig{
			Enabled: true,
			TTL:     Duration(5 * time.Minute),
			MaxSize: 1000,
		},
		Observability: ObservabilityConfig{
			Logging: LoggingConfig{
				Enabled: false,
			},
			Metrics: MetricsConfig{
				Enabled: true,
			},
		},
	}

	client, err := NewClient(config)
	if err != nil {
		b.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Pre-populate with test flags
	for i := 0; i < 100; i++ {
		flag := FeatureFlag{
			Key:     fmt.Sprintf("flag-%d", i),
			Enabled: i%2 == 0,
		}
		client.SetFlag(ctx, flag)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("flag-%d", i%100)

			switch i % 100 {
			case 0, 1: // 2% writes
				flag := FeatureFlag{
					Key:     key,
					Enabled: i%2 == 0,
				}
				client.SetFlag(ctx, flag)
			case 2: // 1% deletes
				client.DeleteFlag(ctx, key)
			case 3: // 1% GetFlag calls
				client.GetFlag(ctx, key)
			default: // 96% IsEnabled calls (hot path)
				client.IsEnabled(ctx, key)
			}
			i++
		}
	})
}

// BenchmarkClient_HighConcurrency tests performance under very high concurrency
func BenchmarkClient_HighConcurrency(b *testing.B) {
	config := Config{
		Storage: StorageConfig{
			Type: "memory",
		},
		Cache: CacheConfig{
			Enabled: true,
			TTL:     Duration(5 * time.Minute),
			MaxSize: 10000,
		},
		Observability: ObservabilityConfig{
			Logging: LoggingConfig{
				Enabled: false,
			},
			Metrics: MetricsConfig{
				Enabled: false, // Disable for maximum performance
			},
		},
	}

	client, err := NewClient(config)
	if err != nil {
		b.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Pre-populate with many flags
	for i := 0; i < 1000; i++ {
		flag := FeatureFlag{
			Key:     fmt.Sprintf("flag-%d", i),
			Enabled: i%2 == 0,
		}
		client.SetFlag(ctx, flag)
	}

	// Set high parallelism
	b.SetParallelism(100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("flag-%d", i%1000)
			client.IsEnabled(ctx, key)
			i++
		}
	})
}
