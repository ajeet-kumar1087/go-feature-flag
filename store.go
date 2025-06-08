package featureflag

type Store interface {
	// IsEnabled checks if a feature flag is enabled.
	IsEnabled(key string) (*FeatureFlag, error)

	// Enable sets the status of a feature flag.
	Enable(flag FeatureFlag) error

	// Create initializes a new feature flag with a default value.
	Create(flag FeatureFlag) error

	// GetAll returns a map of all feature flags and their enabled status.
	GetAll() ([]FeatureFlag, error)

	// Delete removes a feature flag from the store.
	Delete(flag string) error

	// Reset resets all feature flags to their default state.
	Reset() error
}

type FeatureFlag struct {
	Key         string `json:"key"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description,omitempty"`
}
