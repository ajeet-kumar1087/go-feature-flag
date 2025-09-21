package core

import (
	"testing"
	"time"
)

func TestFeatureFlag_Validate(t *testing.T) {
	tests := []struct {
		name    string
		flag    FeatureFlag
		wantErr bool
	}{
		{
			name: "valid flag",
			flag: FeatureFlag{
				Key:     "test-flag",
				Enabled: true,
			},
			wantErr: false,
		},
		{
			name: "empty key",
			flag: FeatureFlag{
				Key:     "",
				Enabled: true,
			},
			wantErr: true,
		},
		{
			name: "whitespace key",
			flag: FeatureFlag{
				Key:     "   ",
				Enabled: true,
			},
			wantErr: true,
		},
		{
			name: "invalid characters in key",
			flag: FeatureFlag{
				Key:     "test@flag",
				Enabled: true,
			},
			wantErr: true,
		},
		{
			name: "valid key with underscores and hyphens",
			flag: FeatureFlag{
				Key:     "test_flag-123",
				Enabled: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.flag.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("FeatureFlag.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFeatureFlag_SetTimestamps(t *testing.T) {
	flag := &FeatureFlag{
		Key:     "test-flag",
		Enabled: true,
	}

	// First call should set both CreatedAt and UpdatedAt
	flag.SetTimestamps()
	if flag.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if flag.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}

	createdAt := flag.CreatedAt
	time.Sleep(time.Millisecond) // Ensure time difference

	// Second call should only update UpdatedAt
	flag.SetTimestamps()
	if !flag.CreatedAt.Equal(createdAt) {
		t.Error("CreatedAt should not change on subsequent calls")
	}
	if flag.UpdatedAt.Equal(createdAt) {
		t.Error("UpdatedAt should be updated on subsequent calls")
	}
}

func TestFeatureFlag_Clone(t *testing.T) {
	original := &FeatureFlag{
		Key:         "test-flag",
		Enabled:     true,
		Description: "Test flag",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata: map[string]string{
			"env":     "test",
			"version": "1.0",
		},
	}

	clone := original.Clone()

	// Verify all fields are copied
	if clone.Key != original.Key {
		t.Error("Key not cloned correctly")
	}
	if clone.Enabled != original.Enabled {
		t.Error("Enabled not cloned correctly")
	}
	if clone.Description != original.Description {
		t.Error("Description not cloned correctly")
	}
	if !clone.CreatedAt.Equal(original.CreatedAt) {
		t.Error("CreatedAt not cloned correctly")
	}
	if !clone.UpdatedAt.Equal(original.UpdatedAt) {
		t.Error("UpdatedAt not cloned correctly")
	}

	// Verify metadata is deep copied
	if len(clone.Metadata) != len(original.Metadata) {
		t.Error("Metadata not cloned correctly")
	}
	for k, v := range original.Metadata {
		if clone.Metadata[k] != v {
			t.Error("Metadata values not cloned correctly")
		}
	}

	// Verify it's a deep copy by modifying clone
	clone.Metadata["new"] = "value"
	if _, exists := original.Metadata["new"]; exists {
		t.Error("Metadata should be deep copied, not shared")
	}
}
