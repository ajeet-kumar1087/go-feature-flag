package memory

import (
	"context"
	"sync"

	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/core"
)

// MemoryStore implements the Store interface using in-memory storage
// It provides thread-safe operations using sync.RWMutex
type MemoryStore struct {
	mu    sync.RWMutex
	flags map[string]*core.FeatureFlag
}

// NewMemoryStore creates a new in-memory store
func NewStore() *MemoryStore {
	return &MemoryStore{
		flags: make(map[string]*core.FeatureFlag),
	}
}

// Get retrieves a feature flag by key
func (m *MemoryStore) Get(ctx context.Context, key string) (*core.FeatureFlag, error) {
	if key == "" {
		return nil, core.NewError("get", key, core.ErrInvalidFlag)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	flag, exists := m.flags[key]
	if !exists {
		return nil, core.NewError("get", key, core.ErrFlagNotFound)
	}

	// Return a clone to prevent external modifications
	return flag.Clone(), nil
}

// Set creates or updates a feature flag
func (m *MemoryStore) Set(ctx context.Context, flag core.FeatureFlag) error {
	if err := flag.Validate(); err != nil {
		return core.NewError("set", flag.Key, err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if flag already exists to preserve CreatedAt
	if existing, exists := m.flags[flag.Key]; exists {
		flag.CreatedAt = existing.CreatedAt
	}

	// Set timestamps (will set CreatedAt only if it's zero)
	flag.SetTimestamps()

	// Store a clone to prevent external modifications
	m.flags[flag.Key] = flag.Clone()

	return nil
}

// Delete removes a feature flag
func (m *MemoryStore) Delete(ctx context.Context, key string) error {
	if key == "" {
		return core.NewError("delete", key, core.ErrInvalidFlag)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.flags[key]; !exists {
		return core.NewError("delete", key, core.ErrFlagNotFound)
	}

	delete(m.flags, key)
	return nil
}

// GetAll retrieves all feature flags
func (m *MemoryStore) GetAll(ctx context.Context) ([]core.FeatureFlag, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	flags := make([]core.FeatureFlag, 0, len(m.flags))
	for _, flag := range m.flags {
		// Add clones to prevent external modifications
		flags = append(flags, *flag.Clone())
	}

	return flags, nil
}

// HealthCheck verifies store connectivity (always healthy for memory store)
func (m *MemoryStore) HealthCheck(ctx context.Context) error {
	// Memory store is always healthy
	return nil
}

// Close cleanly shuts down the store
func (m *MemoryStore) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear the map
	m.flags = make(map[string]*core.FeatureFlag)
	return nil
}
