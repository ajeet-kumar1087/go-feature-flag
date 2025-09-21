package observability

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/core"
)

// MetricsCollector defines the interface for collecting metrics
type MetricsCollector interface {
	// Flag access metrics
	RecordFlagCheck(ctx context.Context, key string, enabled bool, duration time.Duration)
	RecordFlagGet(ctx context.Context, key string, found bool, duration time.Duration)
	RecordFlagSet(ctx context.Context, key string, success bool, duration time.Duration)
	RecordFlagDelete(ctx context.Context, key string, success bool, duration time.Duration)

	// Cache metrics
	RecordCacheHit(ctx context.Context, key string)
	RecordCacheMiss(ctx context.Context, key string)
	RecordCacheEviction(ctx context.Context, key string)

	// Storage metrics
	RecordStorageOperation(ctx context.Context, operation string, success bool, duration time.Duration)

	// Error metrics
	RecordError(ctx context.Context, operation string, errorType core.ErrorType)

	// Get metrics snapshot
	GetMetrics() core.MetricsSnapshot
}

// DefaultMetricsCollector provides a simple in-memory metrics implementation
type DefaultMetricsCollector struct {
	// Flag operation counters
	flagChecks          int64
	flagGets            int64
	flagSets            int64
	flagDeletes         int64
	flagCheckSuccesses  int64
	flagGetSuccesses    int64
	flagSetSuccesses    int64
	flagDeleteSuccesses int64

	// Cache counters
	cacheHits      int64
	cacheMisses    int64
	cacheEvictions int64

	// Storage counters
	storageOperations int64
	storageSuccesses  int64

	// Error counters
	errorsMu     sync.RWMutex
	errorsByType map[core.ErrorType]int64

	// Performance tracking
	flagCheckDurations []time.Duration
	storageDurations   []time.Duration
	durationsMu        sync.RWMutex
	maxDurationsStored int
}

// NewDefaultMetricsCollector creates a new default metrics collector
func NewDefaultMetricsCollector() *DefaultMetricsCollector {
	return &DefaultMetricsCollector{
		errorsByType:       make(map[core.ErrorType]int64),
		flagCheckDurations: make([]time.Duration, 0, 1000),
		storageDurations:   make([]time.Duration, 0, 1000),
		maxDurationsStored: 1000,
	}
}

// RecordFlagCheck records a flag check operation
func (m *DefaultMetricsCollector) RecordFlagCheck(ctx context.Context, key string, enabled bool, duration time.Duration) {
	atomic.AddInt64(&m.flagChecks, 1)
	if enabled {
		atomic.AddInt64(&m.flagCheckSuccesses, 1)
	}
	m.recordDuration(&m.flagCheckDurations, duration)
}

// RecordFlagGet records a flag get operation
func (m *DefaultMetricsCollector) RecordFlagGet(ctx context.Context, key string, found bool, duration time.Duration) {
	atomic.AddInt64(&m.flagGets, 1)
	if found {
		atomic.AddInt64(&m.flagGetSuccesses, 1)
	}
}

// RecordFlagSet records a flag set operation
func (m *DefaultMetricsCollector) RecordFlagSet(ctx context.Context, key string, success bool, duration time.Duration) {
	atomic.AddInt64(&m.flagSets, 1)
	if success {
		atomic.AddInt64(&m.flagSetSuccesses, 1)
	}
}

// RecordFlagDelete records a flag delete operation
func (m *DefaultMetricsCollector) RecordFlagDelete(ctx context.Context, key string, success bool, duration time.Duration) {
	atomic.AddInt64(&m.flagDeletes, 1)
	if success {
		atomic.AddInt64(&m.flagDeleteSuccesses, 1)
	}
}

// RecordCacheHit records a cache hit
func (m *DefaultMetricsCollector) RecordCacheHit(ctx context.Context, key string) {
	atomic.AddInt64(&m.cacheHits, 1)
}

// RecordCacheMiss records a cache miss
func (m *DefaultMetricsCollector) RecordCacheMiss(ctx context.Context, key string) {
	atomic.AddInt64(&m.cacheMisses, 1)
}

// RecordCacheEviction records a cache eviction
func (m *DefaultMetricsCollector) RecordCacheEviction(ctx context.Context, key string) {
	atomic.AddInt64(&m.cacheEvictions, 1)
}

// RecordStorageOperation records a storage operation
func (m *DefaultMetricsCollector) RecordStorageOperation(ctx context.Context, operation string, success bool, duration time.Duration) {
	atomic.AddInt64(&m.storageOperations, 1)
	if success {
		atomic.AddInt64(&m.storageSuccesses, 1)
	}
	m.recordDuration(&m.storageDurations, duration)
}

// RecordError records an error by type
func (m *DefaultMetricsCollector) RecordError(ctx context.Context, operation string, errorType core.ErrorType) {
	m.errorsMu.Lock()
	defer m.errorsMu.Unlock()
	m.errorsByType[errorType]++
}

// GetMetrics returns a snapshot of current metrics
func (m *DefaultMetricsCollector) GetMetrics() core.MetricsSnapshot {
	m.errorsMu.RLock()
	errorsByType := make(map[core.ErrorType]int64)
	for k, v := range m.errorsByType {
		errorsByType[k] = v
	}
	m.errorsMu.RUnlock()

	return core.MetricsSnapshot{
		FlagChecks:           atomic.LoadInt64(&m.flagChecks),
		FlagGets:             atomic.LoadInt64(&m.flagGets),
		FlagSets:             atomic.LoadInt64(&m.flagSets),
		FlagDeletes:          atomic.LoadInt64(&m.flagDeletes),
		FlagCheckSuccesses:   atomic.LoadInt64(&m.flagCheckSuccesses),
		FlagGetSuccesses:     atomic.LoadInt64(&m.flagGetSuccesses),
		FlagSetSuccesses:     atomic.LoadInt64(&m.flagSetSuccesses),
		FlagDeleteSuccesses:  atomic.LoadInt64(&m.flagDeleteSuccesses),
		CacheHits:            atomic.LoadInt64(&m.cacheHits),
		CacheMisses:          atomic.LoadInt64(&m.cacheMisses),
		CacheEvictions:       atomic.LoadInt64(&m.cacheEvictions),
		StorageOperations:    atomic.LoadInt64(&m.storageOperations),
		StorageSuccesses:     atomic.LoadInt64(&m.storageSuccesses),
		ErrorsByType:         errorsByType,
		AvgFlagCheckDuration: m.getAverageDuration(&m.flagCheckDurations),
		AvgStorageDuration:   m.getAverageDuration(&m.storageDurations),
		Timestamp:            time.Now(),
	}
}

// recordDuration records a duration, maintaining a sliding window
func (m *DefaultMetricsCollector) recordDuration(durations *[]time.Duration, duration time.Duration) {
	m.durationsMu.Lock()
	defer m.durationsMu.Unlock()

	*durations = append(*durations, duration)
	if len(*durations) > m.maxDurationsStored {
		// Remove oldest entries to maintain sliding window
		copy(*durations, (*durations)[len(*durations)-m.maxDurationsStored:])
		*durations = (*durations)[:m.maxDurationsStored]
	}
}

// getAverageDuration calculates the average duration from a slice
func (m *DefaultMetricsCollector) getAverageDuration(durations *[]time.Duration) time.Duration {
	m.durationsMu.RLock()
	defer m.durationsMu.RUnlock()

	if len(*durations) == 0 {
		return 0
	}

	var total time.Duration
	for _, d := range *durations {
		total += d
	}

	return total / time.Duration(len(*durations))
}

// NoOpMetricsCollector is a metrics collector that does nothing
type NoOpMetricsCollector struct{}

// NewNoOpMetricsCollector creates a new no-op metrics collector
func NewNoOpMetricsCollector() *NoOpMetricsCollector {
	return &NoOpMetricsCollector{}
}

// RecordFlagCheck does nothing
func (m *NoOpMetricsCollector) RecordFlagCheck(ctx context.Context, key string, enabled bool, duration time.Duration) {
}

// RecordFlagGet does nothing
func (m *NoOpMetricsCollector) RecordFlagGet(ctx context.Context, key string, found bool, duration time.Duration) {
}

// RecordFlagSet does nothing
func (m *NoOpMetricsCollector) RecordFlagSet(ctx context.Context, key string, success bool, duration time.Duration) {
}

// RecordFlagDelete does nothing
func (m *NoOpMetricsCollector) RecordFlagDelete(ctx context.Context, key string, success bool, duration time.Duration) {
}

// RecordCacheHit does nothing
func (m *NoOpMetricsCollector) RecordCacheHit(ctx context.Context, key string) {}

// RecordCacheMiss does nothing
func (m *NoOpMetricsCollector) RecordCacheMiss(ctx context.Context, key string) {}

// RecordCacheEviction does nothing
func (m *NoOpMetricsCollector) RecordCacheEviction(ctx context.Context, key string) {}

// RecordStorageOperation does nothing
func (m *NoOpMetricsCollector) RecordStorageOperation(ctx context.Context, operation string, success bool, duration time.Duration) {
}

// RecordError does nothing
func (m *NoOpMetricsCollector) RecordError(ctx context.Context, operation string, errorType core.ErrorType) {
}

// GetMetrics returns an empty metrics snapshot
func (m *NoOpMetricsCollector) GetMetrics() core.MetricsSnapshot {
	return core.MetricsSnapshot{
		ErrorsByType: make(map[core.ErrorType]int64),
		Timestamp:    time.Now(),
	}
}
