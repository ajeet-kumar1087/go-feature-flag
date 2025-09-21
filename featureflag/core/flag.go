package core

import (
	"errors"
	"strings"
	"time"
)

// FeatureFlag represents a feature flag with metadata and validation.
// This is the core data structure for feature flags in the system.
//
// Key naming conventions:
//   - Use lowercase letters, numbers, hyphens, and underscores only
//   - Use descriptive names like "new-checkout-flow" or "beta_features"
//   - Avoid spaces and special characters
//
// Example usage:
//
//	flag := featureflag.FeatureFlag{
//		Key:         "new-ui-design",
//		Enabled:     true,
//		Description: "Enable the new UI design for all users",
//		Metadata: map[string]string{
//			"team":        "frontend",
//			"environment": "production",
//			"rollout":     "100%",
//		},
//	}
//
// Fields:
//   - Key: Unique identifier for the flag (required, validated)
//   - Enabled: Whether the flag is currently enabled
//   - Description: Human-readable description of the flag's purpose
//   - CreatedAt: Timestamp when the flag was first created (auto-managed)
//   - UpdatedAt: Timestamp when the flag was last modified (auto-managed)
//   - Metadata: Additional key-value pairs for custom data
type FeatureFlag struct {
	// Key is the unique identifier for this feature flag.
	// Must contain only alphanumeric characters, hyphens, and underscores.
	// Cannot be empty or contain only whitespace.
	Key string `json:"key" yaml:"key"`

	// Enabled indicates whether this feature flag is currently active.
	// When true, the feature should be enabled for users.
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Description provides a human-readable explanation of what this flag controls.
	// Optional but recommended for maintainability.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// CreatedAt is the timestamp when this flag was first created.
	// Automatically set by SetTimestamps() method.
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`

	// UpdatedAt is the timestamp when this flag was last modified.
	// Automatically updated by SetTimestamps() method on each change.
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`

	// Metadata contains additional key-value pairs for custom information.
	// Common uses include team ownership, rollout percentages, environments, etc.
	// Optional and can be nil.
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// Validate checks if the FeatureFlag has valid data according to the system rules.
// This method is called automatically when setting flags through the client.
//
// Validation rules:
//   - Key cannot be empty or contain only whitespace
//   - Key can only contain alphanumeric characters, hyphens, and underscores
//   - All other fields are optional and have no specific validation
//
// Returns:
//   - error: Description of the validation failure, or nil if valid
//
// Example usage:
//
//	flag := featureflag.FeatureFlag{Key: "my-feature", Enabled: true}
//	if err := flag.Validate(); err != nil {
//		log.Printf("Invalid flag: %v", err)
//	}
func (f *FeatureFlag) Validate() error {
	if strings.TrimSpace(f.Key) == "" {
		return errors.New("feature flag key cannot be empty")
	}

	// Key should only contain alphanumeric characters, hyphens, and underscores
	for _, char := range f.Key {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_') {
			return errors.New("feature flag key can only contain alphanumeric characters, hyphens, and underscores")
		}
	}

	return nil
}

// SetTimestamps updates the CreatedAt and UpdatedAt fields with current time.
// This method is called automatically by the client when storing flags.
//
// Behavior:
//   - If CreatedAt is zero (never set), it will be set to current time
//   - UpdatedAt is always set to current time
//   - Uses time.Now() for timestamp generation
//
// This method is typically not called directly by application code,
// as the client handles timestamp management automatically.
func (f *FeatureFlag) SetTimestamps() {
	now := time.Now()
	if f.CreatedAt.IsZero() {
		f.CreatedAt = now
	}
	f.UpdatedAt = now
}

// Clone creates a deep copy of the FeatureFlag.
// This is useful when you need to modify a flag without affecting the original.
// All fields including the Metadata map are copied to new memory locations.
//
// Returns:
//   - *FeatureFlag: A new FeatureFlag instance with identical data
//
// Example usage:
//
//	original := &featureflag.FeatureFlag{
//		Key: "test-flag",
//		Enabled: true,
//		Metadata: map[string]string{"env": "prod"},
//	}
//
//	copy := original.Clone()
//	copy.Enabled = false // Doesn't affect original
//	copy.Metadata["env"] = "dev" // Doesn't affect original
func (f *FeatureFlag) Clone() *FeatureFlag {
	clone := &FeatureFlag{
		Key:         f.Key,
		Enabled:     f.Enabled,
		Description: f.Description,
		CreatedAt:   f.CreatedAt,
		UpdatedAt:   f.UpdatedAt,
	}

	if f.Metadata != nil {
		clone.Metadata = make(map[string]string)
		for k, v := range f.Metadata {
			clone.Metadata[k] = v
		}
	}

	return clone
}
