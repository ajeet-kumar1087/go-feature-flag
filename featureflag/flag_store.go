package featureflag

import "context"

type FlagStore interface {
	// IsEnabled checks if a feature flag is enabled.
	Get(ctx context.Context, key string) (*FeatureFlag, error)

	// Enable sets the status of a feature flag.
	Set(ctx context.Context, flag FeatureFlag) error

	// GetAll returns a map of all feature flags and their enabled status.
	GetAll() ([]FeatureFlag, error)

	// Delete removes a feature flag from the store.
	Delete(ctx context.Context, flag string) error
}
