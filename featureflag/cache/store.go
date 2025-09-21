package cache

import (
	"context"
	"time"

	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/config"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/core"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/observability"
)

// CachedStore wraps any Store implementation with an in-memory cache
type CachedStore struct {
	store   core.Store
	cache   *Cache
	metrics observability.MetricsCollector
}

// NewCachedStore creates a new cached store wrapper
func NewStore(store core.Store, cacheConfig config.CacheConfig) *CachedStore {
	var cache *Cache
	if cacheConfig.Enabled {
		cache = NewCache(cacheConfig.MaxSize, time.Duration(cacheConfig.TTL))
	}

	return &CachedStore{
		store:   store,
		cache:   cache,
		metrics: observability.NewNoOpMetricsCollector(), // Default to no-op, will be set by client
	}
}

// NewCachedStoreWithMetrics creates a new cached store wrapper with metrics
func NewStoreWithMetrics(store core.Store, cacheConfig config.CacheConfig, metrics observability.MetricsCollector) *CachedStore {
	cs := &CachedStore{
		store:   store,
		metrics: metrics,
	}

	if cacheConfig.Enabled {
		// Create cache with eviction callback for metrics
		evictionCallback := func(key string, flag *core.FeatureFlag) {
			metrics.RecordCacheEviction(context.Background(), key)
		}
		cs.cache = NewCacheWithEvictionCallback(cacheConfig.MaxSize, time.Duration(cacheConfig.TTL), evictionCallback)
	}

	return cs
}

// Get retrieves a feature flag, checking cache first
func (cs *CachedStore) Get(ctx context.Context, key string) (*core.FeatureFlag, error) {
	// Check cache first if enabled
	if cs.cache != nil {
		if flag, found := cs.cache.Get(key); found {
			cs.metrics.RecordCacheHit(ctx, key)
			return flag, nil
		}
		cs.metrics.RecordCacheMiss(ctx, key)
	}

	// Cache miss or cache disabled - get from underlying store
	start := time.Now()
	flag, err := cs.store.Get(ctx, key)
	duration := time.Since(start)

	cs.metrics.RecordStorageOperation(ctx, "get", err == nil, duration)

	if err != nil {
		return nil, err
	}

	// Cache the result if cache is enabled and flag was found
	if cs.cache != nil && flag != nil {
		cs.cache.Set(key, flag)
	}

	return flag, nil
}

// Set creates or updates a feature flag in both cache and store
func (cs *CachedStore) Set(ctx context.Context, flag core.FeatureFlag) error {
	// Update the underlying store first
	start := time.Now()
	err := cs.store.Set(ctx, flag)
	duration := time.Since(start)

	cs.metrics.RecordStorageOperation(ctx, "set", err == nil, duration)

	if err != nil {
		return err
	}

	// Update cache if enabled
	if cs.cache != nil {
		cs.cache.Set(flag.Key, &flag)
	}

	return nil
}

// Delete removes a feature flag from both cache and store
func (cs *CachedStore) Delete(ctx context.Context, key string) error {
	// Delete from underlying store first
	start := time.Now()
	err := cs.store.Delete(ctx, key)
	duration := time.Since(start)

	cs.metrics.RecordStorageOperation(ctx, "delete", err == nil, duration)

	if err != nil {
		return err
	}

	// Remove from cache if enabled
	if cs.cache != nil {
		cs.cache.Delete(key)
	}

	return nil
}

// GetAll retrieves all feature flags from the underlying store
// Note: This bypasses cache for consistency and performance reasons
func (cs *CachedStore) GetAll(ctx context.Context) ([]core.FeatureFlag, error) {
	start := time.Now()
	flags, err := cs.store.GetAll(ctx)
	duration := time.Since(start)

	cs.metrics.RecordStorageOperation(ctx, "get_all", err == nil, duration)

	if err != nil {
		return nil, err
	}

	// Optionally warm the cache with all flags
	if cs.cache != nil {
		for _, flag := range flags {
			cs.cache.Set(flag.Key, &flag)
		}
	}

	return flags, nil
}

// HealthCheck verifies connectivity of the underlying store
func (cs *CachedStore) HealthCheck(ctx context.Context) error {
	return cs.store.HealthCheck(ctx)
}

// Close cleanly shuts down both cache and store
func (cs *CachedStore) Close() error {
	if cs.cache != nil {
		cs.cache.Close()
	}
	return cs.store.Close()
}
