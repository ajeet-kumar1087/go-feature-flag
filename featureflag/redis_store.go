package featureflag

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisStore(addr string) *RedisStore {
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	return &RedisStore{
		client: rdb,
		ctx:    context.Background(),
	}
}

// IsEnabled checks if a feature flag is enabled in Redis.
func (rdb *RedisStore) IsEnabled(feature string) (*FeatureFlag, error) {
	val, err := rdb.client.Get(rdb.ctx, feature).Result()
	if err != nil {
		return nil, err
	}
	var flag FeatureFlag
	if err := json.Unmarshal([]byte(val), &flag); err != nil {
		return nil, err
	}
	return &flag, nil
}

// Create a new feature flag in Redis with a default value.
func (rdb *RedisStore) Create(flag FeatureFlag) error {
	data, err := json.Marshal(flag)
	if err != nil {
		return err
	}
	rdb.client.Set(rdb.ctx, flag.Key, data, 0)
	return nil
}

// Enable sets the status of a feature flag in Redis.
func (rdb *RedisStore) Enable(flag FeatureFlag) error {
	data, err := json.Marshal(flag)
	if err != nil {
		return err
	}
	return rdb.client.Set(rdb.ctx, flag.Key, data, 0).Err()
}

// GetAll retrieves all feature flags and their enabled status from Redis.
func (rdb *RedisStore) GetAll() ([]FeatureFlag, error) {
	keys, err := rdb.client.Keys(rdb.ctx, "*").Result()
	if err != nil {
		return nil, err
	}
	var result []FeatureFlag
	for _, key := range keys {
		val, err := rdb.client.Get(rdb.ctx, key).Result()
		if err != nil {
			continue
		}
		var flag FeatureFlag
		if err := json.Unmarshal([]byte(val), &flag); err == nil {
			result = append(result, flag)
		}
	}
	return result, nil
}

// Delete removes a feature flag from Redis.
func (rdb *RedisStore) Delete(feature string) error {
	return rdb.client.Del(rdb.ctx, feature).Err()
}

func (rdb *RedisStore) Reset() error {
	// Resetting all feature flags by deleting them from Redis.
	keys, _ := rdb.client.Keys(rdb.ctx, "*").Result()
	for _, k := range keys {
		rdb.client.Del(rdb.ctx, k)
	}
	return nil
}
