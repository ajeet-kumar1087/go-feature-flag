package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/core"
	"gopkg.in/yaml.v3"
)

// Config holds all configuration options for the feature flag library.
// This is the main configuration structure used to customize client behavior.
//
// Configuration can be loaded from multiple sources:
//   - Programmatically by creating a Config struct
//   - From JSON or YAML files using LoadConfigFromFile
//   - From environment variables using LoadConfigFromEnv
//   - Combined approach using LoadConfig with precedence rules
//
// Example programmatic configuration:
//
//	config := featureflag.Config{
//		Storage: featureflag.StorageConfig{
//			Type: "redis",
//			Redis: &featureflag.RedisConfig{
//				Addr: "localhost:6379",
//			},
//		},
//		Cache: featureflag.CacheConfig{
//			Enabled: true,
//			TTL:     featureflag.Duration(10 * time.Minute),
//		},
//	}
//
// All fields have sensible defaults available through DefaultConfig().
type Config struct {
	// Storage configuration specifies which backend to use for persistence.
	// Supported types: "memory", "redis", "postgres"
	Storage StorageConfig `json:"storage" yaml:"storage"`

	// Cache configuration controls the optional caching layer.
	// Caching can significantly improve performance for frequently accessed flags.
	Cache CacheConfig `json:"cache" yaml:"cache"`

	// Observability configuration enables logging and metrics collection.
	// Useful for monitoring and debugging in production environments.
	Observability ObservabilityConfig `json:"observability" yaml:"observability"`

	// DefaultFlags are automatically loaded when the client starts.
	// Useful for ensuring critical flags exist with known default values.
	// These flags are only created if they don't already exist in storage.
	DefaultFlags []core.FeatureFlag `json:"default_flags,omitempty" yaml:"default_flags,omitempty"`
}

// StorageConfig defines storage backend configuration
type StorageConfig struct {
	Type     string          `json:"type" yaml:"type"` // "redis", "postgres", "memory"
	Redis    *RedisConfig    `json:"redis,omitempty" yaml:"redis,omitempty"`
	Postgres *PostgresConfig `json:"postgres,omitempty" yaml:"postgres,omitempty"`
}

// CacheConfig defines caching configuration
type CacheConfig struct {
	Enabled bool     `json:"enabled" yaml:"enabled"`
	TTL     Duration `json:"ttl" yaml:"ttl"`
	MaxSize int      `json:"max_size" yaml:"max_size"`
}

// Duration is a wrapper around time.Duration that supports JSON/YAML marshaling
type Duration time.Duration

// MarshalJSON implements json.Marshaler
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON implements json.Unmarshaler
func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	duration, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(duration)
	return nil
}

// MarshalYAML implements yaml.Marshaler
func (d Duration) MarshalYAML() (interface{}, error) {
	return time.Duration(d).String(), nil
}

// UnmarshalYAML implements yaml.Unmarshaler
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	duration, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(duration)
	return nil
}

// RedisConfig defines Redis-specific configuration
type RedisConfig struct {
	Addr     string `json:"addr" yaml:"addr"`
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
	DB       int    `json:"db" yaml:"db"`
}

// PostgresConfig defines PostgreSQL-specific configuration
type PostgresConfig struct {
	Host     string `json:"host" yaml:"host"`
	Port     int    `json:"port" yaml:"port"`
	Database string `json:"database" yaml:"database"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
	SSLMode  string `json:"ssl_mode" yaml:"ssl_mode"`
}

// ObservabilityConfig defines logging and metrics configuration
type ObservabilityConfig struct {
	// Logging configuration
	Logging LoggingConfig `json:"logging" yaml:"logging"`

	// Metrics configuration
	Metrics MetricsConfig `json:"metrics" yaml:"metrics"`
}

// LoggingConfig defines logging configuration
type LoggingConfig struct {
	Enabled bool   `json:"enabled" yaml:"enabled"`
	Level   string `json:"level" yaml:"level"` // "debug", "info", "warn", "error"
}

// MetricsConfig defines metrics collection configuration
type MetricsConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`
}

// DefaultConfig returns a configuration with sensible defaults.
// This configuration is suitable for development and testing environments.
//
// Default settings:
//   - Storage: In-memory (no external dependencies)
//   - Cache: Enabled with 5-minute TTL and 1000 item limit
//   - Logging: Disabled
//   - Metrics: Disabled
//   - Default flags: None
//
// For production use, consider customizing the configuration with:
//   - Persistent storage (Redis or PostgreSQL)
//   - Appropriate cache settings for your workload
//   - Enabled observability features
//
// Returns:
//   - Config: Configuration with default values
func DefaultConfig() Config {
	return Config{
		Storage: StorageConfig{
			Type: "memory",
		},
		Cache: CacheConfig{
			Enabled: true,
			TTL:     Duration(5 * time.Minute),
			MaxSize: 1000,
		},
		Observability: ObservabilityConfig{
			Logging: LoggingConfig{
				Enabled: false,
				Level:   "info",
			},
			Metrics: MetricsConfig{
				Enabled: false,
			},
		},
		DefaultFlags: []core.FeatureFlag{},
	}
}

// LoadConfig loads configuration from multiple sources with precedence:
// 1. Environment variables (highest precedence)
// 2. Configuration file
// 3. Default values (lowest precedence)
func LoadConfig(configPath string) (Config, error) {
	// Start with default configuration
	config := DefaultConfig()

	// Load from file if provided
	if configPath != "" {
		fileConfig, err := LoadConfigFromFile(configPath)
		if err != nil {
			return config, fmt.Errorf("failed to load config from file: %w", err)
		}
		config = mergeConfigs(config, fileConfig)
	}

	// Override with environment variables
	envConfig := LoadConfigFromEnv()
	config = mergeConfigs(config, envConfig)

	// Validate the final configuration
	if err := config.Validate(); err != nil {
		return config, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// LoadConfigFromFile loads configuration from a JSON or YAML file
func LoadConfigFromFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config

	// Determine file type by extension
	if strings.HasSuffix(strings.ToLower(path), ".json") {
		err = json.Unmarshal(data, &config)
	} else if strings.HasSuffix(strings.ToLower(path), ".yaml") || strings.HasSuffix(strings.ToLower(path), ".yml") {
		err = yaml.Unmarshal(data, &config)
	} else {
		return Config{}, fmt.Errorf("unsupported config file format, use .json, .yaml, or .yml")
	}

	if err != nil {
		return Config{}, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// LoadConfigFromEnv loads configuration from environment variables
func LoadConfigFromEnv() Config {
	config := Config{}

	// Storage configuration
	if storageType := os.Getenv("FEATUREFLAG_STORAGE_TYPE"); storageType != "" {
		config.Storage.Type = storageType
	}

	// Redis configuration
	if redisAddr := os.Getenv("FEATUREFLAG_REDIS_ADDR"); redisAddr != "" {
		if config.Storage.Redis == nil {
			config.Storage.Redis = &RedisConfig{}
		}
		config.Storage.Redis.Addr = redisAddr
	}

	if redisPassword := os.Getenv("FEATUREFLAG_REDIS_PASSWORD"); redisPassword != "" {
		if config.Storage.Redis == nil {
			config.Storage.Redis = &RedisConfig{}
		}
		config.Storage.Redis.Password = redisPassword
	}

	if redisDB := os.Getenv("FEATUREFLAG_REDIS_DB"); redisDB != "" {
		if db, err := strconv.Atoi(redisDB); err == nil {
			if config.Storage.Redis == nil {
				config.Storage.Redis = &RedisConfig{}
			}
			config.Storage.Redis.DB = db
		}
	}

	// PostgreSQL configuration
	if pgHost := os.Getenv("FEATUREFLAG_POSTGRES_HOST"); pgHost != "" {
		if config.Storage.Postgres == nil {
			config.Storage.Postgres = &PostgresConfig{}
		}
		config.Storage.Postgres.Host = pgHost
	}

	if pgPort := os.Getenv("FEATUREFLAG_POSTGRES_PORT"); pgPort != "" {
		if port, err := strconv.Atoi(pgPort); err == nil {
			if config.Storage.Postgres == nil {
				config.Storage.Postgres = &PostgresConfig{}
			}
			config.Storage.Postgres.Port = port
		}
	}

	if pgDatabase := os.Getenv("FEATUREFLAG_POSTGRES_DATABASE"); pgDatabase != "" {
		if config.Storage.Postgres == nil {
			config.Storage.Postgres = &PostgresConfig{}
		}
		config.Storage.Postgres.Database = pgDatabase
	}

	if pgUsername := os.Getenv("FEATUREFLAG_POSTGRES_USERNAME"); pgUsername != "" {
		if config.Storage.Postgres == nil {
			config.Storage.Postgres = &PostgresConfig{}
		}
		config.Storage.Postgres.Username = pgUsername
	}

	if pgPassword := os.Getenv("FEATUREFLAG_POSTGRES_PASSWORD"); pgPassword != "" {
		if config.Storage.Postgres == nil {
			config.Storage.Postgres = &PostgresConfig{}
		}
		config.Storage.Postgres.Password = pgPassword
	}

	if pgSSLMode := os.Getenv("FEATUREFLAG_POSTGRES_SSLMODE"); pgSSLMode != "" {
		if config.Storage.Postgres == nil {
			config.Storage.Postgres = &PostgresConfig{}
		}
		config.Storage.Postgres.SSLMode = pgSSLMode
	}

	// Cache configuration
	if cacheEnabled := os.Getenv("FEATUREFLAG_CACHE_ENABLED"); cacheEnabled != "" {
		if enabled, err := strconv.ParseBool(cacheEnabled); err == nil {
			config.Cache.Enabled = enabled
		}
	}

	if cacheTTL := os.Getenv("FEATUREFLAG_CACHE_TTL"); cacheTTL != "" {
		if ttl, err := time.ParseDuration(cacheTTL); err == nil {
			config.Cache.TTL = Duration(ttl)
		}
	}

	if cacheMaxSize := os.Getenv("FEATUREFLAG_CACHE_MAX_SIZE"); cacheMaxSize != "" {
		if maxSize, err := strconv.Atoi(cacheMaxSize); err == nil {
			config.Cache.MaxSize = maxSize
		}
	}

	// Observability configuration
	if loggingEnabled := os.Getenv("FEATUREFLAG_LOGGING_ENABLED"); loggingEnabled != "" {
		if enabled, err := strconv.ParseBool(loggingEnabled); err == nil {
			config.Observability.Logging.Enabled = enabled
		}
	}

	if loggingLevel := os.Getenv("FEATUREFLAG_LOGGING_LEVEL"); loggingLevel != "" {
		config.Observability.Logging.Level = loggingLevel
	}

	if metricsEnabled := os.Getenv("FEATUREFLAG_METRICS_ENABLED"); metricsEnabled != "" {
		if enabled, err := strconv.ParseBool(metricsEnabled); err == nil {
			config.Observability.Metrics.Enabled = enabled
		}
	}

	return config
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate storage type
	validStorageTypes := map[string]bool{
		"memory":   true,
		"redis":    true,
		"postgres": true,
	}

	if !validStorageTypes[c.Storage.Type] {
		return fmt.Errorf("invalid storage type '%s', must be one of: memory, redis, postgres", c.Storage.Type)
	}

	// Validate Redis configuration if Redis is selected
	if c.Storage.Type == "redis" {
		if c.Storage.Redis == nil {
			return fmt.Errorf("redis configuration is required when storage type is 'redis'")
		}
		if err := c.Storage.Redis.Validate(); err != nil {
			return fmt.Errorf("invalid Redis configuration: %w", err)
		}
	}

	// Validate PostgreSQL configuration if PostgreSQL is selected
	if c.Storage.Type == "postgres" {
		if c.Storage.Postgres == nil {
			return fmt.Errorf("PostgreSQL configuration is required when storage type is 'postgres'")
		}
		if err := c.Storage.Postgres.Validate(); err != nil {
			return fmt.Errorf("invalid PostgreSQL configuration: %w", err)
		}
	}

	// Validate cache configuration
	if err := c.Cache.Validate(); err != nil {
		return fmt.Errorf("invalid cache configuration: %w", err)
	}

	// Validate observability configuration
	if err := c.Observability.Validate(); err != nil {
		return fmt.Errorf("invalid observability configuration: %w", err)
	}

	// Validate default flags
	for i, flag := range c.DefaultFlags {
		if err := flag.Validate(); err != nil {
			return fmt.Errorf("invalid default flag at index %d: %w", i, err)
		}
	}

	return nil
}

// Validate checks if the Redis configuration is valid
func (r *RedisConfig) Validate() error {
	if r.Addr == "" {
		return fmt.Errorf("redis address cannot be empty")
	}

	if r.DB < 0 || r.DB > 15 {
		return fmt.Errorf("redis DB must be between 0 and 15")
	}

	return nil
}

// Validate checks if the PostgreSQL configuration is valid
func (p *PostgresConfig) Validate() error {
	if p.Host == "" {
		return fmt.Errorf("PostgreSQL host cannot be empty")
	}

	if p.Port <= 0 || p.Port > 65535 {
		return fmt.Errorf("PostgreSQL port must be between 1 and 65535")
	}

	if p.Database == "" {
		return fmt.Errorf("PostgreSQL database name cannot be empty")
	}

	if p.Username == "" {
		return fmt.Errorf("PostgreSQL username cannot be empty")
	}

	validSSLModes := map[string]bool{
		"disable":     true,
		"allow":       true,
		"prefer":      true,
		"require":     true,
		"verify-ca":   true,
		"verify-full": true,
	}

	if p.SSLMode != "" && !validSSLModes[p.SSLMode] {
		return fmt.Errorf("invalid PostgreSQL SSL mode '%s'", p.SSLMode)
	}

	return nil
}

// Validate checks if the cache configuration is valid
func (c *CacheConfig) Validate() error {
	if time.Duration(c.TTL) < 0 {
		return fmt.Errorf("cache TTL cannot be negative")
	}

	if c.MaxSize < 0 {
		return fmt.Errorf("cache max size cannot be negative")
	}

	return nil
}

// Validate checks if the observability configuration is valid
func (o *ObservabilityConfig) Validate() error {
	if err := o.Logging.Validate(); err != nil {
		return fmt.Errorf("invalid logging configuration: %w", err)
	}

	if err := o.Metrics.Validate(); err != nil {
		return fmt.Errorf("invalid metrics configuration: %w", err)
	}

	return nil
}

// Validate checks if the logging configuration is valid
func (l *LoggingConfig) Validate() error {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if l.Level != "" && !validLevels[strings.ToLower(l.Level)] {
		return fmt.Errorf("invalid logging level '%s', must be one of: debug, info, warn, error", l.Level)
	}

	return nil
}

// Validate checks if the metrics configuration is valid
func (m *MetricsConfig) Validate() error {
	// No specific validation needed for metrics config currently
	return nil
}

// mergeConfigs merges two configurations, with the second taking precedence
func mergeConfigs(base, override Config) Config {
	result := base

	// Merge storage configuration
	if override.Storage.Type != "" {
		result.Storage.Type = override.Storage.Type
	}

	if override.Storage.Redis != nil {
		if result.Storage.Redis == nil {
			result.Storage.Redis = &RedisConfig{}
		}
		if override.Storage.Redis.Addr != "" {
			result.Storage.Redis.Addr = override.Storage.Redis.Addr
		}
		if override.Storage.Redis.Password != "" {
			result.Storage.Redis.Password = override.Storage.Redis.Password
		}
		if override.Storage.Redis.DB != 0 {
			result.Storage.Redis.DB = override.Storage.Redis.DB
		}
	}

	if override.Storage.Postgres != nil {
		if result.Storage.Postgres == nil {
			result.Storage.Postgres = &PostgresConfig{}
		}
		if override.Storage.Postgres.Host != "" {
			result.Storage.Postgres.Host = override.Storage.Postgres.Host
		}
		if override.Storage.Postgres.Port != 0 {
			result.Storage.Postgres.Port = override.Storage.Postgres.Port
		}
		if override.Storage.Postgres.Database != "" {
			result.Storage.Postgres.Database = override.Storage.Postgres.Database
		}
		if override.Storage.Postgres.Username != "" {
			result.Storage.Postgres.Username = override.Storage.Postgres.Username
		}
		if override.Storage.Postgres.Password != "" {
			result.Storage.Postgres.Password = override.Storage.Postgres.Password
		}
		if override.Storage.Postgres.SSLMode != "" {
			result.Storage.Postgres.SSLMode = override.Storage.Postgres.SSLMode
		}
	}

	// Merge cache configuration
	if override.Cache.Enabled != base.Cache.Enabled {
		result.Cache.Enabled = override.Cache.Enabled
	}
	if time.Duration(override.Cache.TTL) != 0 {
		result.Cache.TTL = override.Cache.TTL
	}
	if override.Cache.MaxSize != 0 {
		result.Cache.MaxSize = override.Cache.MaxSize
	}

	// Merge observability configuration
	if override.Observability.Logging.Enabled != base.Observability.Logging.Enabled {
		result.Observability.Logging.Enabled = override.Observability.Logging.Enabled
	}
	if override.Observability.Logging.Level != "" {
		result.Observability.Logging.Level = override.Observability.Logging.Level
	}
	if override.Observability.Metrics.Enabled != base.Observability.Metrics.Enabled {
		result.Observability.Metrics.Enabled = override.Observability.Metrics.Enabled
	}

	// Merge default flags
	if len(override.DefaultFlags) > 0 {
		result.DefaultFlags = override.DefaultFlags
	}

	return result
}
