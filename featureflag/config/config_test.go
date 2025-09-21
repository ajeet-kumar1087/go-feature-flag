package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// Test default storage configuration
	if config.Storage.Type != "memory" {
		t.Errorf("Expected default storage type to be 'memory', got '%s'", config.Storage.Type)
	}

	// Test default cache configuration
	if !config.Cache.Enabled {
		t.Error("Expected cache to be enabled by default")
	}

	if time.Duration(config.Cache.TTL) != 5*time.Minute {
		t.Errorf("Expected default cache TTL to be 5 minutes, got %v", config.Cache.TTL)
	}

	if config.Cache.MaxSize != 1000 {
		t.Errorf("Expected default cache max size to be 1000, got %d", config.Cache.MaxSize)
	}

	// Test default flags
	if len(config.DefaultFlags) != 0 {
		t.Errorf("Expected no default flags, got %d", len(config.DefaultFlags))
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid default config",
			config:      DefaultConfig(),
			expectError: false,
		},
		{
			name: "invalid storage type",
			config: Config{
				Storage: StorageConfig{Type: "invalid"},
				Cache:   CacheConfig{Enabled: true, TTL: Duration(time.Minute), MaxSize: 100},
			},
			expectError: true,
			errorMsg:    "invalid storage type",
		},
		{
			name: "redis config missing when redis type",
			config: Config{
				Storage: StorageConfig{Type: "redis"},
				Cache:   CacheConfig{Enabled: true, TTL: Duration(time.Minute), MaxSize: 100},
			},
			expectError: true,
			errorMsg:    "redis configuration is required",
		},
		{
			name: "postgres config missing when postgres type",
			config: Config{
				Storage: StorageConfig{Type: "postgres"},
				Cache:   CacheConfig{Enabled: true, TTL: Duration(time.Minute), MaxSize: 100},
			},
			expectError: true,
			errorMsg:    "PostgreSQL configuration is required",
		},
		{
			name: "negative cache TTL",
			config: Config{
				Storage: StorageConfig{Type: "memory"},
				Cache:   CacheConfig{Enabled: true, TTL: Duration(-time.Minute), MaxSize: 100},
			},
			expectError: true,
			errorMsg:    "cache TTL cannot be negative",
		},
		{
			name: "negative cache max size",
			config: Config{
				Storage: StorageConfig{Type: "memory"},
				Cache:   CacheConfig{Enabled: true, TTL: Duration(time.Minute), MaxSize: -100},
			},
			expectError: true,
			errorMsg:    "cache max size cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("Expected validation error, got nil")
				} else if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no validation error, got: %v", err)
				}
			}
		})
	}
}

func TestRedisConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      RedisConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid redis config",
			config:      RedisConfig{Addr: "localhost:6379", DB: 0},
			expectError: false,
		},
		{
			name:        "empty address",
			config:      RedisConfig{Addr: "", DB: 0},
			expectError: true,
			errorMsg:    "redis address cannot be empty",
		},
		{
			name:        "invalid DB number - negative",
			config:      RedisConfig{Addr: "localhost:6379", DB: -1},
			expectError: true,
			errorMsg:    "redis DB must be between 0 and 15",
		},
		{
			name:        "invalid DB number - too high",
			config:      RedisConfig{Addr: "localhost:6379", DB: 16},
			expectError: true,
			errorMsg:    "redis DB must be between 0 and 15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("Expected validation error, got nil")
				} else if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no validation error, got: %v", err)
				}
			}
		})
	}
}

func TestPostgresConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      PostgresConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid postgres config",
			config: PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				Username: "testuser",
				SSLMode:  "disable",
			},
			expectError: false,
		},
		{
			name: "empty host",
			config: PostgresConfig{
				Host:     "",
				Port:     5432,
				Database: "testdb",
				Username: "testuser",
			},
			expectError: true,
			errorMsg:    "PostgreSQL host cannot be empty",
		},
		{
			name: "invalid port - zero",
			config: PostgresConfig{
				Host:     "localhost",
				Port:     0,
				Database: "testdb",
				Username: "testuser",
			},
			expectError: true,
			errorMsg:    "PostgreSQL port must be between 1 and 65535",
		},
		{
			name: "invalid port - too high",
			config: PostgresConfig{
				Host:     "localhost",
				Port:     65536,
				Database: "testdb",
				Username: "testuser",
			},
			expectError: true,
			errorMsg:    "PostgreSQL port must be between 1 and 65535",
		},
		{
			name: "empty database",
			config: PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "",
				Username: "testuser",
			},
			expectError: true,
			errorMsg:    "PostgreSQL database name cannot be empty",
		},
		{
			name: "empty username",
			config: PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				Username: "",
			},
			expectError: true,
			errorMsg:    "PostgreSQL username cannot be empty",
		},
		{
			name: "invalid SSL mode",
			config: PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				Username: "testuser",
				SSLMode:  "invalid",
			},
			expectError: true,
			errorMsg:    "invalid PostgreSQL SSL mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("Expected validation error, got nil")
				} else if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no validation error, got: %v", err)
				}
			}
		})
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Test JSON config file
	jsonConfig := `{
		"storage": {
			"type": "redis",
			"redis": {
				"addr": "localhost:6379",
				"db": 1
			}
		},
		"cache": {
			"enabled": false,
			"ttl": "10m",
			"max_size": 500
		},
		"default_flags": [
			{
				"key": "test-flag",
				"enabled": true,
				"description": "Test flag"
			}
		]
	}`

	jsonPath := filepath.Join(tempDir, "config.json")
	if err := os.WriteFile(jsonPath, []byte(jsonConfig), 0644); err != nil {
		t.Fatalf("Failed to write JSON config file: %v", err)
	}

	config, err := LoadConfigFromFile(jsonPath)
	if err != nil {
		t.Fatalf("Failed to load JSON config: %v", err)
	}

	// Verify loaded configuration
	if config.Storage.Type != "redis" {
		t.Errorf("Expected storage type 'redis', got '%s'", config.Storage.Type)
	}

	if config.Storage.Redis == nil {
		t.Fatal("Expected Redis config to be loaded")
	}

	if config.Storage.Redis.Addr != "localhost:6379" {
		t.Errorf("Expected Redis addr 'localhost:6379', got '%s'", config.Storage.Redis.Addr)
	}

	if config.Storage.Redis.DB != 1 {
		t.Errorf("Expected Redis DB 1, got %d", config.Storage.Redis.DB)
	}

	if config.Cache.Enabled {
		t.Error("Expected cache to be disabled")
	}

	if time.Duration(config.Cache.TTL) != 10*time.Minute {
		t.Errorf("Expected cache TTL 10m, got %v", config.Cache.TTL)
	}

	if len(config.DefaultFlags) != 1 {
		t.Errorf("Expected 1 default flag, got %d", len(config.DefaultFlags))
	}

	// Test YAML config file
	yamlConfig := `
storage:
  type: postgres
  postgres:
    host: localhost
    port: 5432
    database: testdb
    username: testuser
    password: testpass
    ssl_mode: disable
cache:
  enabled: true
  ttl: 15m
  max_size: 2000
`

	yamlPath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(yamlPath, []byte(yamlConfig), 0644); err != nil {
		t.Fatalf("Failed to write YAML config file: %v", err)
	}

	config, err = LoadConfigFromFile(yamlPath)
	if err != nil {
		t.Fatalf("Failed to load YAML config: %v", err)
	}

	// Verify loaded configuration
	if config.Storage.Type != "postgres" {
		t.Errorf("Expected storage type 'postgres', got '%s'", config.Storage.Type)
	}

	if config.Storage.Postgres == nil {
		t.Fatal("Expected Postgres config to be loaded")
	}

	if config.Storage.Postgres.Host != "localhost" {
		t.Errorf("Expected Postgres host 'localhost', got '%s'", config.Storage.Postgres.Host)
	}

	if time.Duration(config.Cache.TTL) != 15*time.Minute {
		t.Errorf("Expected cache TTL 15m, got %v", config.Cache.TTL)
	}

	// Test unsupported file format
	txtPath := filepath.Join(tempDir, "config.txt")
	if err := os.WriteFile(txtPath, []byte("invalid"), 0644); err != nil {
		t.Fatalf("Failed to write TXT config file: %v", err)
	}

	_, err = LoadConfigFromFile(txtPath)
	if err == nil {
		t.Error("Expected error for unsupported file format")
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"FEATUREFLAG_STORAGE_TYPE",
		"FEATUREFLAG_REDIS_ADDR",
		"FEATUREFLAG_REDIS_PASSWORD",
		"FEATUREFLAG_REDIS_DB",
		"FEATUREFLAG_POSTGRES_HOST",
		"FEATUREFLAG_POSTGRES_PORT",
		"FEATUREFLAG_POSTGRES_DATABASE",
		"FEATUREFLAG_POSTGRES_USERNAME",
		"FEATUREFLAG_POSTGRES_PASSWORD",
		"FEATUREFLAG_POSTGRES_SSLMODE",
		"FEATUREFLAG_CACHE_ENABLED",
		"FEATUREFLAG_CACHE_TTL",
		"FEATUREFLAG_CACHE_MAX_SIZE",
	}

	for _, envVar := range envVars {
		originalEnv[envVar] = os.Getenv(envVar)
		os.Unsetenv(envVar)
	}

	// Restore environment after test
	defer func() {
		for envVar, value := range originalEnv {
			if value != "" {
				os.Setenv(envVar, value)
			} else {
				os.Unsetenv(envVar)
			}
		}
	}()

	// Set test environment variables
	os.Setenv("FEATUREFLAG_STORAGE_TYPE", "redis")
	os.Setenv("FEATUREFLAG_REDIS_ADDR", "redis:6379")
	os.Setenv("FEATUREFLAG_REDIS_PASSWORD", "secret")
	os.Setenv("FEATUREFLAG_REDIS_DB", "2")
	os.Setenv("FEATUREFLAG_CACHE_ENABLED", "false")
	os.Setenv("FEATUREFLAG_CACHE_TTL", "30m")
	os.Setenv("FEATUREFLAG_CACHE_MAX_SIZE", "5000")

	config := LoadConfigFromEnv()

	// Verify loaded configuration
	if config.Storage.Type != "redis" {
		t.Errorf("Expected storage type 'redis', got '%s'", config.Storage.Type)
	}

	if config.Storage.Redis == nil {
		t.Fatal("Expected Redis config to be loaded")
	}

	if config.Storage.Redis.Addr != "redis:6379" {
		t.Errorf("Expected Redis addr 'redis:6379', got '%s'", config.Storage.Redis.Addr)
	}

	if config.Storage.Redis.Password != "secret" {
		t.Errorf("Expected Redis password 'secret', got '%s'", config.Storage.Redis.Password)
	}

	if config.Storage.Redis.DB != 2 {
		t.Errorf("Expected Redis DB 2, got %d", config.Storage.Redis.DB)
	}

	if config.Cache.Enabled {
		t.Error("Expected cache to be disabled")
	}

	if time.Duration(config.Cache.TTL) != 30*time.Minute {
		t.Errorf("Expected cache TTL 30m, got %v", config.Cache.TTL)
	}

	if config.Cache.MaxSize != 5000 {
		t.Errorf("Expected cache max size 5000, got %d", config.Cache.MaxSize)
	}
}

func TestLoadConfig(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Create a config file
	configContent := `{
		"storage": {
			"type": "postgres",
			"postgres": {
				"host": "file-host",
				"port": 5432,
				"database": "filedb",
				"username": "fileuser"
			}
		},
		"cache": {
			"enabled": true,
			"ttl": "20m"
		}
	}`

	configPath := filepath.Join(tempDir, "config.json")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Save and clear environment
	originalHost := os.Getenv("FEATUREFLAG_POSTGRES_HOST")
	originalType := os.Getenv("FEATUREFLAG_STORAGE_TYPE")
	defer func() {
		if originalHost != "" {
			os.Setenv("FEATUREFLAG_POSTGRES_HOST", originalHost)
		} else {
			os.Unsetenv("FEATUREFLAG_POSTGRES_HOST")
		}
		if originalType != "" {
			os.Setenv("FEATUREFLAG_STORAGE_TYPE", originalType)
		} else {
			os.Unsetenv("FEATUREFLAG_STORAGE_TYPE")
		}
	}()

	// Set environment variable to override file config
	os.Setenv("FEATUREFLAG_POSTGRES_HOST", "env-host")
	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Environment variables should take precedence over file config
	if config.Storage.Type != "postgres" {
		t.Errorf("Expected storage type 'postgres', got '%s'", config.Storage.Type)
	}

	if config.Storage.Postgres.Host != "env-host" {
		t.Errorf("Expected postgres host 'env-host' from env, got '%s'", config.Storage.Postgres.Host)
	}

	if config.Storage.Postgres.Database != "filedb" {
		t.Errorf("Expected postgres database 'filedb' from file, got '%s'", config.Storage.Postgres.Database)
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())))
}
