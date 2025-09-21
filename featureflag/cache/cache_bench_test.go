package cache

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func BenchmarkCache_Get(b *testing.B) {
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
			cache.Get(key)
			i++
		}
	})
}

func BenchmarkCache_Set(b *testing.B) {
	cache := NewCache(1000, 5*time.Minute)
	defer cache.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			flag := &FeatureFlag{
				Key:     fmt.Sprintf("flag-%d", i),
				Enabled: i%2 == 0,
			}
			cache.Set(flag.Key, flag)
			i++
		}
	})
}

func BenchmarkCachedStore_Get_CacheHit(b *testing.B) {
	mockStore := NewMockStore()
	cacheConfig := CacheConfig{
		Enabled: true,
		TTL:     Duration(5 * time.Minute),
		MaxSize: 1000,
	}

	cachedStore := NewCachedStore(mockStore, cacheConfig)
	defer cachedStore.Close()

	ctx := context.Background()

	// Pre-populate cache
	for i := 0; i < 100; i++ {
		flag := FeatureFlag{
			Key:     fmt.Sprintf("flag-%d", i),
			Enabled: i%2 == 0,
		}
		cachedStore.Set(ctx, flag)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("flag-%d", i%100)
			cachedStore.Get(ctx, key)
			i++
		}
	})
}

func BenchmarkCachedStore_Get_CacheMiss(b *testing.B) {
	mockStore := NewMockStore()
	cacheConfig := CacheConfig{
		Enabled: true,
		TTL:     Duration(5 * time.Minute),
		MaxSize: 1000,
	}

	cachedStore := NewCachedStore(mockStore, cacheConfig)
	defer cachedStore.Close()

	ctx := context.Background()

	// Pre-populate underlying store (not cache)
	for i := 0; i < 100; i++ {
		flag := FeatureFlag{
			Key:     fmt.Sprintf("flag-%d", i),
			Enabled: i%2 == 0,
		}
		mockStore.Set(ctx, flag)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("flag-%d", i%100)
			cachedStore.Get(ctx, key)
			i++
		}
	})
}

func BenchmarkCachedStore_Set(b *testing.B) {
	mockStore := NewMockStore()
	cacheConfig := CacheConfig{
		Enabled: true,
		TTL:     Duration(5 * time.Minute),
		MaxSize: 1000,
	}

	cachedStore := NewCachedStore(mockStore, cacheConfig)
	defer cachedStore.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			flag := FeatureFlag{
				Key:     fmt.Sprintf("flag-%d", i),
				Enabled: i%2 == 0,
			}
			cachedStore.Set(ctx, flag)
			i++
		}
	})
}
