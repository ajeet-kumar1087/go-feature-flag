package cache

import (
	"container/list"
	"sync"
	"time"

	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/core"
)

// cacheItem represents a cached feature flag with expiration
type cacheItem struct {
	flag      *core.FeatureFlag
	expiresAt time.Time
	element   *list.Element // for LRU tracking
}

// EvictionCallback is called when an item is evicted from the cache
type EvictionCallback func(key string, flag *core.FeatureFlag)

// Cache is an in-memory cache with TTL and LRU eviction
type Cache struct {
	mu               sync.RWMutex
	items            map[string]*cacheItem
	lruList          *list.List
	maxSize          int
	ttl              time.Duration
	stopCh           chan struct{}
	stopOnce         sync.Once
	evictionCallback EvictionCallback
}

// NewCache creates a new cache with the specified configuration
func NewCache(maxSize int, ttl time.Duration) *Cache {
	cache := &Cache{
		items:   make(map[string]*cacheItem),
		lruList: list.New(),
		maxSize: maxSize,
		ttl:     ttl,
		stopCh:  make(chan struct{}),
	}

	// Start cleanup goroutine if TTL is enabled
	if ttl > 0 {
		go cache.cleanupExpired()
	}

	return cache
}

// NewCacheWithEvictionCallback creates a new cache with eviction callback
func NewCacheWithEvictionCallback(maxSize int, ttl time.Duration, callback EvictionCallback) *Cache {
	cache := &Cache{
		items:            make(map[string]*cacheItem),
		lruList:          list.New(),
		maxSize:          maxSize,
		ttl:              ttl,
		stopCh:           make(chan struct{}),
		evictionCallback: callback,
	}

	// Start cleanup goroutine if TTL is enabled
	if ttl > 0 {
		go cache.cleanupExpired()
	}

	return cache
}

// Get retrieves a feature flag from the cache
// Optimized for read-heavy workloads with minimal lock contention
func (c *Cache) Get(key string) (*core.FeatureFlag, bool) {
	// First, try a read lock to check if item exists and is not expired
	c.mu.RLock()
	item, exists := c.items[key]
	if !exists {
		c.mu.RUnlock()
		return nil, false
	}

	// Check if item has expired (read-only check)
	expired := c.ttl > 0 && time.Now().After(item.expiresAt)
	if expired {
		c.mu.RUnlock()
		// Need write lock to remove expired item
		c.mu.Lock()
		// Double-check after acquiring write lock (item might have been updated)
		if item, exists := c.items[key]; exists && c.ttl > 0 && time.Now().After(item.expiresAt) {
			c.removeItem(key, item)
		}
		c.mu.Unlock()
		return nil, false
	}

	// Item exists and is not expired, get the flag value
	flag := item.flag
	c.mu.RUnlock()

	// Update LRU position with write lock (only if necessary)
	// This is a separate operation to minimize write lock time
	c.mu.Lock()
	// Double-check item still exists (might have been evicted)
	if currentItem, exists := c.items[key]; exists && currentItem == item {
		c.lruList.MoveToFront(item.element)
	}
	c.mu.Unlock()

	return flag, true
}

// Set stores a feature flag in the cache
func (c *Cache) Set(key string, flag *core.FeatureFlag) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if item already exists
	if existingItem, exists := c.items[key]; exists {
		// Update existing item
		existingItem.flag = flag
		if c.ttl > 0 {
			existingItem.expiresAt = time.Now().Add(c.ttl)
		}
		c.lruList.MoveToFront(existingItem.element)
		return
	}

	// Create new item
	expiresAt := time.Time{}
	if c.ttl > 0 {
		expiresAt = time.Now().Add(c.ttl)
	}

	element := c.lruList.PushFront(key)
	item := &cacheItem{
		flag:      flag,
		expiresAt: expiresAt,
		element:   element,
	}

	c.items[key] = item

	// Evict oldest items if cache is full
	c.evictIfNeeded()
}

// Delete removes a feature flag from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, exists := c.items[key]; exists {
		c.removeItem(key, item)
	}
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*cacheItem)
	c.lruList.Init()
}

// Size returns the current number of items in the cache
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Close stops the cleanup goroutine
func (c *Cache) Close() {
	c.stopOnce.Do(func() {
		close(c.stopCh)
	})
}

// removeItem removes an item from both the map and LRU list
// Must be called with lock held
func (c *Cache) removeItem(key string, item *cacheItem) {
	delete(c.items, key)
	c.lruList.Remove(item.element)

	// Call eviction callback if set
	if c.evictionCallback != nil {
		c.evictionCallback(key, item.flag)
	}
}

// evictIfNeeded removes the least recently used items if cache is at capacity
// Must be called with lock held
func (c *Cache) evictIfNeeded() {
	for c.maxSize > 0 && len(c.items) > c.maxSize {
		// Remove least recently used item (back of list)
		oldest := c.lruList.Back()
		if oldest != nil {
			key := oldest.Value.(string)
			if item, exists := c.items[key]; exists {
				c.removeItem(key, item)
			}
		}
	}
}

// cleanupExpired periodically removes expired items
func (c *Cache) cleanupExpired() {
	ticker := time.NewTicker(c.ttl / 2) // Clean up twice per TTL period
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.removeExpiredItems()
		case <-c.stopCh:
			return
		}
	}
}

// removeExpiredItems removes all expired items from the cache
func (c *Cache) removeExpiredItems() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ttl <= 0 {
		return
	}

	now := time.Now()
	var keysToRemove []string

	for key, item := range c.items {
		if now.After(item.expiresAt) {
			keysToRemove = append(keysToRemove, key)
		}
	}

	for _, key := range keysToRemove {
		if item, exists := c.items[key]; exists {
			c.removeItem(key, item)
		}
	}
}
