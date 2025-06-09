package featureflag

import (
	"testing"

	"github.com/redis/go-redis/v9"
)

func setupTestRedis(t *testing.T) *RedisStore {
	store := NewRedisStore("localhost:6379")
	// Clean up before each test
	store.Reset()
	return store
}

func TestRedisStore_Create(t *testing.T) {
	store := setupTestRedis(t)

	tests := []struct {
		name    string
		flag    FeatureFlag
		wantErr bool
	}{
		{
			name: "valid flag",
			flag: FeatureFlag{
				Key:     "test-feature",
				Enabled: true,
			},
			wantErr: false,
		},
		{
			name: "duplicate flag",
			flag: FeatureFlag{
				Key:     "test-feature",
				Enabled: false,
			},
			wantErr: false, // Redis Set overwrites existing keys
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.Create(tt.flag)
			if (err != nil) != tt.wantErr {
				t.Errorf("RedisStore.Create() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRedisStore_IsEnabled(t *testing.T) {
	store := setupTestRedis(t)

	// Create a test flag
	testFlag := FeatureFlag{
		Key:     "test-feature",
		Enabled: true,
	}
	store.Create(testFlag)

	tests := []struct {
		name    string
		key     string
		want    bool
		wantErr bool
	}{
		{
			name:    "existing flag",
			key:     "test-feature",
			want:    true,
			wantErr: false,
		},
		{
			name:    "non-existent flag",
			key:     "missing-feature",
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.IsEnabled(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("RedisStore.IsEnabled() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Enabled != tt.want {
				t.Errorf("RedisStore.IsEnabled() = %v, want %v", got.Enabled, tt.want)
			}
		})
	}
}

func TestRedisStore_GetAll(t *testing.T) {
	store := setupTestRedis(t)

	// Create test flags
	flags := []FeatureFlag{
		{Key: "feature1", Enabled: true},
		{Key: "feature2", Enabled: false},
	}

	for _, flag := range flags {
		store.Create(flag)
	}

	got, err := store.GetAll()
	if err != nil {
		t.Errorf("RedisStore.GetAll() error = %v", err)
		return
	}

	if len(got) != len(flags) {
		t.Errorf("RedisStore.GetAll() got %d flags, want %d", len(got), len(flags))
	}
}

func TestRedisStore_Delete(t *testing.T) {
	store := setupTestRedis(t)

	// Create a test flag
	testFlag := FeatureFlag{
		Key:     "test-feature",
		Enabled: true,
	}
	store.Create(testFlag)

	if err := store.Delete(testFlag.Key); err != nil {
		t.Errorf("RedisStore.Delete() error = %v", err)
	}

	// Verify deletion
	_, err := store.IsEnabled(testFlag.Key)
	if err != redis.Nil {
		t.Errorf("RedisStore.Delete() flag still exists")
	}
}

func TestRedisStore_Reset(t *testing.T) {
	store := setupTestRedis(t)

	// Create test flags
	flags := []FeatureFlag{
		{Key: "feature1", Enabled: true},
		{Key: "feature2", Enabled: false},
	}

	for _, flag := range flags {
		store.Create(flag)
	}

	if err := store.Reset(); err != nil {
		t.Errorf("RedisStore.Reset() error = %v", err)
	}

	// Verify all flags are deleted
	got, err := store.GetAll()
	if err != nil {
		t.Errorf("RedisStore.GetAll() error = %v", err)
	}

	if len(got) != 0 {
		t.Errorf("RedisStore.Reset() failed, got %d flags, want 0", len(got))
	}
}
