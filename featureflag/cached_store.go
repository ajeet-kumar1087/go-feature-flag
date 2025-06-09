package featureflag

import "context"

type CachedStore struct {
	redis    *RedisStore
	postgres *PostgresStore
}

func NewCachedStore(redis *RedisStore, postgres *PostgresStore) *CachedStore {
	return &CachedStore{
		redis:    redis,
		postgres: postgres,
	}
}

func (cs *CachedStore) Get(ctx context.Context, key string) (*FeatureFlag, error) {
	// First, try to get the feature flag from Redis
	flag, err := cs.redis.Get(key)
	if err != nil {
		// Redis error - fallback to Postgres
		flag, err = cs.postgres.Get(ctx, key)
		if err != nil {
			return nil, err
		}
		// Cache into Redis if found
		if flag != nil {
			_ = cs.redis.Set(ctx, *flag)
		}
		return flag, nil
	}
	// If not found in Redis, try Postgres
	if flag == nil {
		flag, err = cs.postgres.Get(ctx, key)
		if err != nil {
			return nil, err
		}
		// If found in Postgres, cache it in Redis
		if flag != nil {
			err = cs.redis.Set(ctx, *flag)
			if err != nil {
				return nil, err
			}
		}
	}
	return flag, nil
}

// Persist flag into both postgres and redis
func (cs *CachedStore) Set(ctx context.Context, flag FeatureFlag) error {
	err := cs.postgres.Set(ctx, flag)
	if err != nil {
		return err
	}
	return cs.redis.Set(ctx, flag)
}

// Get all cached keys and values from Redis, falling back to Postgres if necessary.
func (cs *CachedStore) GetAll(ctx context.Context) ([]FeatureFlag, error) {
	// First, try to get all flags from Redis
	flags, err := cs.redis.GetAll()
	if err != nil {
		return nil, err
	}

	// If Redis is empty, fallback to Postgres
	if len(flags) == 0 {
		flags, err = cs.postgres.GetAll()
		if err != nil {
			return nil, err
		}
		// Cache the Postgres flags into Redis
		for _, flag := range flags {
			_ = cs.redis.Set(ctx, flag)
		}
	}

	return flags, nil
}

// Delete a feature flag from both Redis and Postgres
func (cs *CachedStore) Delete(ctx context.Context, key string) error {
	err := cs.redis.Delete(key)
	if err != nil {
		return err
	}
	return cs.postgres.Delete(ctx, key)
}
