package memory

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/core"
)

func TestNewStore(t *testing.T) {
	store := NewStore()
	if store == nil {
		t.Fatal("NewStore() returned nil")
	}
	if store.flags == nil {
		t.Fatal("NewStore() did not initialize flags map")
	}
}

func TestMemoryStore_Set(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	t.Run("valid flag", func(t *testing.T) {
		flag := core.FeatureFlag{
			Key:         "test-flag",
			Enabled:     true,
			Description: "Test flag",
		}

		err := store.Set(ctx, flag)
		if err != nil {
			t.Fatalf("Set() failed: %v", err)
		}

		// Verify the flag was stored
		stored, err := store.Get(ctx, "test-flag")
		if err != nil {
			t.Fatalf("Get() failed after Set(): %v", err)
		}

		if stored.Key != flag.Key {
			t.Errorf("Expected key %s, got %s", flag.Key, stored.Key)
		}
		if stored.Enabled != flag.Enabled {
			t.Errorf("Expected enabled %v, got %v", flag.Enabled, stored.Enabled)
		}
		if stored.Description != flag.Description {
			t.Errorf("Expected description %s, got %s", flag.Description, stored.Description)
		}
	})

	t.Run("invalid flag - empty key", func(t *testing.T) {
		flag := core.FeatureFlag{
			Key:     "",
			Enabled: true,
		}

		err := store.Set(ctx, flag)
		if err == nil {
			t.Fatal("Set() should have failed with empty key")
		}
	})

	t.Run("invalid flag - invalid key characters", func(t *testing.T) {
		flag := core.FeatureFlag{
			Key:     "test@flag",
			Enabled: true,
		}

		err := store.Set(ctx, flag)
		if err == nil {
			t.Fatal("Set() should have failed with invalid key characters")
		}
	})

	t.Run("update existing flag", func(t *testing.T) {
		// First set
		flag := core.FeatureFlag{
			Key:         "update-test",
			Enabled:     false,
			Description: "Original",
		}
		err := store.Set(ctx, flag)
		if err != nil {
			t.Fatalf("First Set() failed: %v", err)
		}

		// Update
		flag.Enabled = true
		flag.Description = "Updated"
		err = store.Set(ctx, flag)
		if err != nil {
			t.Fatalf("Update Set() failed: %v", err)
		}

		// Verify update
		stored, err := store.Get(ctx, "update-test")
		if err != nil {
			t.Fatalf("Get() failed after update: %v", err)
		}

		if !stored.Enabled {
			t.Error("Flag should be enabled after update")
		}
		if stored.Description != "Updated" {
			t.Errorf("Expected description 'Updated', got %s", stored.Description)
		}
	})
}

func TestMemoryStore_Get(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	t.Run("existing flag", func(t *testing.T) {
		flag := core.FeatureFlag{
			Key:         "get-test",
			Enabled:     true,
			Description: "Get test flag",
			Metadata:    map[string]string{"env": "test"},
		}

		err := store.Set(ctx, flag)
		if err != nil {
			t.Fatalf("Set() failed: %v", err)
		}

		retrieved, err := store.Get(ctx, "get-test")
		if err != nil {
			t.Fatalf("Get() failed: %v", err)
		}

		if retrieved.Key != flag.Key {
			t.Errorf("Expected key %s, got %s", flag.Key, retrieved.Key)
		}
		if retrieved.Metadata["env"] != "test" {
			t.Errorf("Expected metadata env=test, got %s", retrieved.Metadata["env"])
		}
	})

	t.Run("non-existing flag", func(t *testing.T) {
		_, err := store.Get(ctx, "non-existing")
		if err == nil {
			t.Fatal("Get() should have failed for non-existing flag")
		}

		var ffErr *core.FeatureFlagError
		if !errors.As(err, &ffErr) {
			t.Fatal("Error should be of type core.FeatureFlagError")
		}
		if !errors.Is(ffErr.Err, core.ErrFlagNotFound) {
			t.Errorf("Expected core.ErrFlagNotFound, got %v", ffErr.Err)
		}
	})

	t.Run("empty key", func(t *testing.T) {
		_, err := store.Get(ctx, "")
		if err == nil {
			t.Fatal("Get() should have failed with empty key")
		}
	})

	t.Run("immutability - external modification", func(t *testing.T) {
		flag := core.FeatureFlag{
			Key:         "immutable-test",
			Enabled:     true,
			Description: "Original",
			Metadata:    map[string]string{"key": "value"},
		}

		err := store.Set(ctx, flag)
		if err != nil {
			t.Fatalf("Set() failed: %v", err)
		}

		retrieved, err := store.Get(ctx, "immutable-test")
		if err != nil {
			t.Fatalf("Get() failed: %v", err)
		}

		// Modify the retrieved flag
		retrieved.Description = "Modified"
		retrieved.Metadata["key"] = "modified"

		// Get again and verify original values
		retrieved2, err := store.Get(ctx, "immutable-test")
		if err != nil {
			t.Fatalf("Second Get() failed: %v", err)
		}

		if retrieved2.Description != "Original" {
			t.Error("External modification affected stored flag")
		}
		if retrieved2.Metadata["key"] != "value" {
			t.Error("External metadata modification affected stored flag")
		}
	})
}

func TestMemoryStore_Delete(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	t.Run("existing flag", func(t *testing.T) {
		flag := core.FeatureFlag{
			Key:     "delete-test",
			Enabled: true,
		}

		err := store.Set(ctx, flag)
		if err != nil {
			t.Fatalf("Set() failed: %v", err)
		}

		err = store.Delete(ctx, "delete-test")
		if err != nil {
			t.Fatalf("Delete() failed: %v", err)
		}

		// Verify flag is deleted
		_, err = store.Get(ctx, "delete-test")
		if err == nil {
			t.Fatal("Get() should have failed after Delete()")
		}
	})

	t.Run("non-existing flag", func(t *testing.T) {
		err := store.Delete(ctx, "non-existing")
		if err == nil {
			t.Fatal("Delete() should have failed for non-existing flag")
		}

		var ffErr *core.FeatureFlagError
		if !errors.As(err, &ffErr) {
			t.Fatal("Error should be of type core.FeatureFlagError")
		}
		if !errors.Is(ffErr.Err, core.ErrFlagNotFound) {
			t.Errorf("Expected core.ErrFlagNotFound, got %v", ffErr.Err)
		}
	})

	t.Run("empty key", func(t *testing.T) {
		err := store.Delete(ctx, "")
		if err == nil {
			t.Fatal("Delete() should have failed with empty key")
		}
	})
}

func TestMemoryStore_GetAll(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	t.Run("empty store", func(t *testing.T) {
		flags, err := store.GetAll(ctx)
		if err != nil {
			t.Fatalf("GetAll() failed: %v", err)
		}
		if len(flags) != 0 {
			t.Errorf("Expected 0 flags, got %d", len(flags))
		}
	})

	t.Run("multiple flags", func(t *testing.T) {
		flags := []core.FeatureFlag{
			{Key: "flag1", Enabled: true, Description: "First flag"},
			{Key: "flag2", Enabled: false, Description: "Second flag"},
			{Key: "flag3", Enabled: true, Description: "Third flag"},
		}

		for _, flag := range flags {
			err := store.Set(ctx, flag)
			if err != nil {
				t.Fatalf("Set() failed for %s: %v", flag.Key, err)
			}
		}

		retrieved, err := store.GetAll(ctx)
		if err != nil {
			t.Fatalf("GetAll() failed: %v", err)
		}

		if len(retrieved) != len(flags) {
			t.Errorf("Expected %d flags, got %d", len(flags), len(retrieved))
		}

		// Create a map for easier verification
		retrievedMap := make(map[string]core.FeatureFlag)
		for _, flag := range retrieved {
			retrievedMap[flag.Key] = flag
		}

		for _, original := range flags {
			retrieved, exists := retrievedMap[original.Key]
			if !exists {
				t.Errorf("Flag %s not found in GetAll() result", original.Key)
				continue
			}
			if retrieved.Enabled != original.Enabled {
				t.Errorf("Flag %s: expected enabled %v, got %v", original.Key, original.Enabled, retrieved.Enabled)
			}
		}
	})

	t.Run("immutability", func(t *testing.T) {
		// Use a fresh store for this test to avoid interference
		freshStore := NewStore()

		flag := core.FeatureFlag{
			Key:         "immutable-getall",
			Enabled:     true,
			Description: "Original",
			Metadata:    map[string]string{"key": "value"},
		}

		err := freshStore.Set(ctx, flag)
		if err != nil {
			t.Fatalf("Set() failed: %v", err)
		}

		flags, err := freshStore.GetAll(ctx)
		if err != nil {
			t.Fatalf("GetAll() failed: %v", err)
		}

		// Find and modify our specific flag
		var targetFlag *core.FeatureFlag
		for i := range flags {
			if flags[i].Key == "immutable-getall" {
				targetFlag = &flags[i]
				break
			}
		}

		if targetFlag == nil {
			t.Fatal("Target flag not found in GetAll() result")
		}

		// Modify the returned flag
		targetFlag.Description = "Modified"
		if targetFlag.Metadata != nil {
			targetFlag.Metadata["key"] = "modified"
		}

		// Get again and verify original values
		flags2, err := freshStore.GetAll(ctx)
		if err != nil {
			t.Fatalf("Second GetAll() failed: %v", err)
		}

		// Find our flag again
		var targetFlag2 *core.FeatureFlag
		for i := range flags2 {
			if flags2[i].Key == "immutable-getall" {
				targetFlag2 = &flags2[i]
				break
			}
		}

		if targetFlag2 == nil {
			t.Fatal("Target flag not found in second GetAll() result")
		}

		if targetFlag2.Description != "Original" {
			t.Error("External modification affected stored flag")
		}
		if targetFlag2.Metadata != nil && targetFlag2.Metadata["key"] != "value" {
			t.Error("External metadata modification affected stored flag")
		}
	})
}

func TestMemoryStore_HealthCheck(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	err := store.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("HealthCheck() failed: %v", err)
	}
}

func TestMemoryStore_Close(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Add some flags
	flag := core.FeatureFlag{Key: "test", Enabled: true}
	err := store.Set(ctx, flag)
	if err != nil {
		t.Fatalf("Set() failed: %v", err)
	}

	err = store.Close()
	if err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	// Verify flags are cleared
	flags, err := store.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll() failed after Close(): %v", err)
	}
	if len(flags) != 0 {
		t.Errorf("Expected 0 flags after Close(), got %d", len(flags))
	}
}

// TestMemoryStore_ConcurrentAccess tests thread safety
func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	const numGoroutines = 100
	const numOperations = 100

	var wg sync.WaitGroup

	// Test concurrent writes
	t.Run("concurrent writes", func(t *testing.T) {
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()

				for j := 0; j < numOperations; j++ {
					flag := core.FeatureFlag{
						Key:         fmt.Sprintf("flag-%d-%d", id, j),
						Enabled:     j%2 == 0,
						Description: fmt.Sprintf("Flag %d-%d", id, j),
					}

					err := store.Set(ctx, flag)
					if err != nil {
						t.Errorf("Concurrent Set() failed: %v", err)
					}
				}
			}(i)
		}

		wg.Wait()

		// Verify all flags were stored
		flags, err := store.GetAll(ctx)
		if err != nil {
			t.Fatalf("GetAll() failed after concurrent writes: %v", err)
		}

		expectedCount := numGoroutines * numOperations
		if len(flags) != expectedCount {
			t.Errorf("Expected %d flags after concurrent writes, got %d", expectedCount, len(flags))
		}
	})

	// Test concurrent reads
	t.Run("concurrent reads", func(t *testing.T) {
		// First, add a flag to read
		testFlag := core.FeatureFlag{
			Key:     "concurrent-read-test",
			Enabled: true,
		}
		err := store.Set(ctx, testFlag)
		if err != nil {
			t.Fatalf("Set() failed: %v", err)
		}

		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()

				for j := 0; j < numOperations; j++ {
					_, err := store.Get(ctx, "concurrent-read-test")
					if err != nil {
						t.Errorf("Concurrent Get() failed: %v", err)
					}
				}
			}()
		}

		wg.Wait()
	})

	// Test mixed concurrent operations
	t.Run("mixed concurrent operations", func(t *testing.T) {
		wg.Add(numGoroutines * 3) // readers, writers, deleters

		// Concurrent readers
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					store.GetAll(ctx)
				}
			}(i)
		}

		// Concurrent writers
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					flag := core.FeatureFlag{
						Key:     fmt.Sprintf("mixed-%d-%d", id, j),
						Enabled: true,
					}
					store.Set(ctx, flag)
				}
			}(i)
		}

		// Concurrent deleters (some will fail, that's expected)
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					store.Delete(ctx, fmt.Sprintf("mixed-%d-%d", id, j))
				}
			}(i)
		}

		wg.Wait()
	})
}

func TestMemoryStore_TimestampHandling(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	t.Run("timestamps are set automatically", func(t *testing.T) {
		flag := core.FeatureFlag{
			Key:     "timestamp-test",
			Enabled: true,
		}

		before := time.Now()
		err := store.Set(ctx, flag)
		if err != nil {
			t.Fatalf("Set() failed: %v", err)
		}
		after := time.Now()

		stored, err := store.Get(ctx, "timestamp-test")
		if err != nil {
			t.Fatalf("Get() failed: %v", err)
		}

		if stored.CreatedAt.IsZero() {
			t.Error("CreatedAt should be set")
		}
		if stored.UpdatedAt.IsZero() {
			t.Error("UpdatedAt should be set")
		}
		if stored.CreatedAt.Before(before) || stored.CreatedAt.After(after) {
			t.Error("CreatedAt should be within expected time range")
		}
		if stored.UpdatedAt.Before(before) || stored.UpdatedAt.After(after) {
			t.Error("UpdatedAt should be within expected time range")
		}
	})

	t.Run("update preserves CreatedAt", func(t *testing.T) {
		flag := core.FeatureFlag{
			Key:     "update-timestamp-test",
			Enabled: false,
		}

		err := store.Set(ctx, flag)
		if err != nil {
			t.Fatalf("First Set() failed: %v", err)
		}

		first, err := store.Get(ctx, "update-timestamp-test")
		if err != nil {
			t.Fatalf("First Get() failed: %v", err)
		}

		// Wait a bit to ensure different timestamps
		time.Sleep(10 * time.Millisecond)

		// Update the flag
		flag.Enabled = true
		err = store.Set(ctx, flag)
		if err != nil {
			t.Fatalf("Update Set() failed: %v", err)
		}

		updated, err := store.Get(ctx, "update-timestamp-test")
		if err != nil {
			t.Fatalf("Updated Get() failed: %v", err)
		}

		if !updated.CreatedAt.Equal(first.CreatedAt) {
			t.Error("CreatedAt should be preserved on update")
		}
		if !updated.UpdatedAt.After(first.UpdatedAt) {
			t.Error("UpdatedAt should be newer on update")
		}
	})
}
