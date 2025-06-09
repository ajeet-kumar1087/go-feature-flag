package featureflag

type FeatureFlag struct {
	Key         string `json:"key"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description,omitempty"`
}
