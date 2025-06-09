package featureflag

import (
	"context"
	"encoding/json"
	"errors"

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
func (rdb *RedisStore) Get(key string) (*FeatureFlag, error) {
	val, err := rdb.client.Get(rdb.ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil // Key doesn't exist
		}
		return nil, err // other types of error
	}
	var flag FeatureFlag
	error := json.Unmarshal([]byte(val), &flag)
	if error != nil {
		return nil, err
	}
	return &flag, nil
}

// Create a new feature flag in Redis with a default value.
func (rdb *RedisStore) Set(ctx context.Context, flag FeatureFlag) error {
	data, err := json.Marshal(flag)
	if err != nil {
		return err
	}
	rdb.client.Set(rdb.ctx, flag.Key, data, 0)
	return nil
}

// // Enable sets the status of a feature flag in Redis.
// func (rdb *RedisStore) Enable(flag FeatureFlag) error {
// 	data, err := json.Marshal(flag)
// 	if err != nil {
// 		return err
// 	}
// 	return rdb.client.Set(rdb.ctx, flag.Key, data, 0).Err()
// }

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
func (rdb *RedisStore) Delete(key string) error {
	return rdb.client.Del(rdb.ctx, key).Err()
}
