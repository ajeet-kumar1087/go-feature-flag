package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/config"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/core"
)

// RedisClientInterface defines the Redis operations we need
type RedisClientInterface interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Keys(ctx context.Context, pattern string) *redis.StringSliceCmd
	Pipeline() redis.Pipeliner
	Ping(ctx context.Context) *redis.StatusCmd
	Close() error
}

// RedisStore implements the Store interface using Redis as the backend
type RedisStore struct {
	client RedisClientInterface
	prefix string
}

// NewRedisStore creates a new Redis store with the given configuration
func NewStore(config *config.RedisConfig) (*RedisStore, error) {
	if config == nil {
		return nil, core.NewError("init", "", fmt.Errorf("redis configuration cannot be nil"))
	}

	if err := config.Validate(); err != nil {
		return nil, core.NewError("init", "", err)
	}

	// Create Redis client with optimized connection pooling for high concurrency
	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,

		// Optimized connection pool settings for concurrent access
		PoolSize:        20,               // Increased pool size for higher concurrency
		MinIdleConns:    8,                // More idle connections for faster reuse
		MaxIdleConns:    15,               // Higher max idle connections
		ConnMaxIdleTime: 10 * time.Minute, // Longer idle time to reduce connection churn
		ConnMaxLifetime: 2 * time.Hour,    // Longer lifetime for better reuse

		// Optimized retry settings
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,

		// Optimized timeouts for performance
		DialTimeout:  3 * time.Second, // Faster dial timeout
		ReadTimeout:  2 * time.Second, // Faster read timeout for hot path
		WriteTimeout: 2 * time.Second, // Faster write timeout
	})

	store := &RedisStore{
		client: client,
		prefix: "featureflag:",
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := store.HealthCheck(ctx); err != nil {
		client.Close()
		return nil, core.NewError("init", "", fmt.Errorf("failed to connect to Redis: %w", err))
	}

	return store, nil
}

// Get retrieves a feature flag by key
func (r *RedisStore) Get(ctx context.Context, key string) (*core.FeatureFlag, error) {
	if key == "" {
		return nil, core.NewError("get", key, fmt.Errorf("key cannot be empty"))
	}

	redisKey := r.prefix + key
	data, err := r.client.Get(ctx, redisKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, core.NewError("get", key, core.ErrFlagNotFound)
		}
		return nil, core.NewError("get", key, fmt.Errorf("redis get failed: %w", err))
	}

	var flag core.FeatureFlag
	if err := json.Unmarshal([]byte(data), &flag); err != nil {
		return nil, core.NewError("get", key, fmt.Errorf("failed to deserialize flag: %w", err))
	}

	return &flag, nil
}

// Set creates or updates a feature flag
func (r *RedisStore) Set(ctx context.Context, flag core.FeatureFlag) error {
	if err := flag.Validate(); err != nil {
		return core.NewError("set", flag.Key, err)
	}

	// Set timestamps
	flag.SetTimestamps()

	data, err := json.Marshal(flag)
	if err != nil {
		return core.NewError("set", flag.Key, fmt.Errorf("failed to serialize flag: %w", err))
	}

	redisKey := r.prefix + flag.Key
	if err := r.client.Set(ctx, redisKey, data, 0).Err(); err != nil {
		return core.NewError("set", flag.Key, fmt.Errorf("redis set failed: %w", err))
	}

	return nil
}

// Delete removes a feature flag
func (r *RedisStore) Delete(ctx context.Context, key string) error {
	if key == "" {
		return core.NewError("delete", key, fmt.Errorf("key cannot be empty"))
	}

	redisKey := r.prefix + key
	result, err := r.client.Del(ctx, redisKey).Result()
	if err != nil {
		return core.NewError("delete", key, fmt.Errorf("redis delete failed: %w", err))
	}

	if result == 0 {
		return core.NewError("delete", key, core.ErrFlagNotFound)
	}

	return nil
}

// GetAll retrieves all feature flags
func (r *RedisStore) GetAll(ctx context.Context) ([]core.FeatureFlag, error) {
	pattern := r.prefix + "*"
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, core.NewError("getall", "", fmt.Errorf("redis keys failed: %w", err))
	}

	if len(keys) == 0 {
		return []core.FeatureFlag{}, nil
	}

	// Use pipeline for efficient batch retrieval
	pipe := r.client.Pipeline()
	if pipe == nil {
		// Fallback to individual gets if pipeline is not available (e.g., in tests)
		flags := make([]core.FeatureFlag, 0, len(keys))
		for _, key := range keys {
			data, err := r.client.Get(ctx, key).Result()
			if err != nil {
				if err == redis.Nil {
					// Skip deleted keys (race condition)
					continue
				}
				return nil, core.NewError("getall", "", fmt.Errorf("redis get failed for key %s: %w", key, err))
			}

			var flag core.FeatureFlag
			if err := json.Unmarshal([]byte(data), &flag); err != nil {
				return nil, core.NewError("getall", "", fmt.Errorf("failed to deserialize flag for key %s: %w", key, err))
			}

			flags = append(flags, flag)
		}
		return flags, nil
	}

	cmds := make([]*redis.StringCmd, len(keys))

	for i, key := range keys {
		cmds[i] = pipe.Get(ctx, key)
	}

	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, core.NewError("getall", "", fmt.Errorf("redis pipeline failed: %w", err))
	}

	flags := make([]core.FeatureFlag, 0, len(keys))
	for i, cmd := range cmds {
		data, err := cmd.Result()
		if err != nil {
			if err == redis.Nil {
				// Skip deleted keys (race condition)
				continue
			}
			return nil, core.NewError("getall", "", fmt.Errorf("redis get failed for key %s: %w", keys[i], err))
		}

		var flag core.FeatureFlag
		if err := json.Unmarshal([]byte(data), &flag); err != nil {
			return nil, core.NewError("getall", "", fmt.Errorf("failed to deserialize flag for key %s: %w", keys[i], err))
		}

		flags = append(flags, flag)
	}

	return flags, nil
}

// HealthCheck verifies store connectivity
func (r *RedisStore) HealthCheck(ctx context.Context) error {
	if err := r.client.Ping(ctx).Err(); err != nil {
		return core.NewError("healthcheck", "", fmt.Errorf("redis ping failed: %w", err))
	}
	return nil
}

// Close cleanly shuts down the store
func (r *RedisStore) Close() error {
	if err := r.client.Close(); err != nil {
		return core.NewError("close", "", fmt.Errorf("failed to close redis client: %w", err))
	}
	return nil
}
