//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestConfigurationIntegration tests configuration loading from files and environment variables
func TestConfigurationIntegration(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	t.Run("JSON Configuration File Integration", func(t *testing.T) {
		// Create a comprehensive JSON config file
		jsonConfig := `{
			"storage": {
				"type": "memory"
			},
			"cache": {
				"enabled": true,
				"ttl": "10m",
				"max_size": 500
			},
			"default_flags": [
				{
					"key": "json-config-test-flag",
					"enabled": true,
					"description": "Test flag from JSON config",
					"metadata": {
						"source": "json",
						"env": "test"
					}
				},
				{
					"key": "json-config-disabled-flag",
					"enabled": false,
					"description": "Disabled test flag from JSON config"
				}
			],
			"observability": {
				"logging": {
					"enabled": true,
					"level": "info"
				},
				"metrics": {
					"enabled": false
				}
			}
		}`

		jsonPath := filepath.Join(tempDir, "test-config.json")
		if err := os.WriteFile(jsonPath, []byte(jsonConfig), 0644); err != nil {
			t.Fatalf("Failed to write JSON config file: %v", err)
		}

		// Load configuration and create client
		config, err := LoadConfigFromFile(jsonPath)
		if err != nil {
			t.Fatalf("Failed to load JSON config: %v", err)
		}

		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("Failed to create client with JSON config: %v", err)
		}
		defer client.Close()

		ctx := context.Background()

		// Verify default flags were loaded
		flag1, err := client.GetFlag(ctx, "json-config-test-flag")
		if err != nil {
			t.Fatalf("Failed to get default flag: %v", err)
		}

		if !flag1.Enabled {
			t.Error("Expected json-config-test-flag to be enabled")
		}
		if flag1.Description != "Test flag from JSON config" {
			t.Errorf("Expected description 'Test flag from JSON config', got '%s'", flag1.Description)
		}
		if flag1.Metadata["source"] != "json" {
			t.Errorf("Expected metadata source 'json', got '%s'", flag1.Metadata["source"])
		}

		flag2, err := client.GetFlag(ctx, "json-config-disabled-flag")
		if err != nil {
			t.Fatalf("Failed to get second default flag: %v", err)
		}

		if flag2.Enabled {
			t.Error("Expected json-config-disabled-flag to be disabled")
		}

		// Test that configuration settings are applied
		enabled, err := client.IsEnabled(ctx, "json-config-test-flag")
		if err != nil {
			t.Fatalf("Failed to check flag: %v", err)
		}
		if !enabled {
			t.Error("Expected flag to be enabled")
		}
	})

	t.Run("YAML Configuration File Integration", func(t *testing.T) {
		// Create a comprehensive YAML config file
		yamlConfig := `
storage:
  type: memory
cache:
  enabled: true
  ttl: 15m
  max_size: 1000
default_flags:
  - key: yaml-config-test-flag
    enabled: true
    description: Test flag from YAML config
    metadata:
      source: yaml
      env: test
      team: backend
  - key: yaml-config-feature-flag
    enabled: false
    description: Feature flag from YAML config
    metadata:
      feature: new-ui
observability:
  logging:
    enabled: false
  metrics:
    enabled: true
`

		yamlPath := filepath.Join(tempDir, "test-config.yaml")
		if err := os.WriteFile(yamlPath, []byte(yamlConfig), 0644); err != nil {
			t.Fatalf("Failed to write YAML config file: %v", err)
		}

		// Load configuration and create client
		config, err := LoadConfigFromFile(yamlPath)
		if err != nil {
			t.Fatalf("Failed to load YAML config: %v", err)
		}

		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("Failed to create client with YAML config: %v", err)
		}
		defer client.Close()

		ctx := context.Background()

		// Verify default flags were loaded
		flag1, err := client.GetFlag(ctx, "yaml-config-test-flag")
		if err != nil {
			t.Fatalf("Failed to get default flag: %v", err)
		}

		if !flag1.Enabled {
			t.Error("Expected yaml-config-test-flag to be enabled")
		}
		if flag1.Metadata["source"] != "yaml" {
			t.Errorf("Expected metadata source 'yaml', got '%s'", flag1.Metadata["source"])
		}
		if flag1.Metadata["team"] != "backend" {
			t.Errorf("Expected metadata team 'backend', got '%s'", flag1.Metadata["team"])
		}

		flag2, err := client.GetFlag(ctx, "yaml-config-feature-flag")
		if err != nil {
			t.Fatalf("Failed to get second default flag: %v", err)
		}

		if flag2.Enabled {
			t.Error("Expected yaml-config-feature-flag to be disabled")
		}
		if flag2.Metadata["feature"] != "new-ui" {
			t.Errorf("Expected metadata feature 'new-ui', got '%s'", flag2.Metadata["feature"])
		}

		// Verify cache configuration was applied
		if time.Duration(config.Cache.TTL) != 15*time.Minute {
			t.Errorf("Expected cache TTL 15m, got %v", config.Cache.TTL)
		}
		if config.Cache.MaxSize != 1000 {
			t.Errorf("Expected cache max size 1000, got %d", config.Cache.MaxSize)
		}
	})

	t.Run("Environment Variable Override Integration", func(t *testing.T) {
		// Create a base config file
		baseConfig := `{
			"storage": {
				"type": "memory"
			},
			"cache": {
				"enabled": true,
				"ttl": "5m",
				"max_size": 100
			},
			"default_flags": [
				{
					"key": "env-override-test-flag",
					"enabled": false,
					"description": "Test flag for env override"
				}
			]
		}`

		configPath := filepath.Join(tempDir, "base-config.json")
		if err := os.WriteFile(configPath, []byte(baseConfig), 0644); err != nil {
			t.Fatalf("Failed to write base config file: %v", err)
		}

		// Save original environment
		originalEnv := make(map[string]string)
		envVars := []string{
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

		// Set environment variables to override config file
		os.Setenv("FEATUREFLAG_CACHE_ENABLED", "false")
		os.Setenv("FEATUREFLAG_CACHE_TTL", "30m")
		os.Setenv("FEATUREFLAG_CACHE_MAX_SIZE", "2000")

		// Load configuration with environment override
		config, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("Failed to load config with env override: %v", err)
		}

		// Verify environment variables took precedence
		if config.Cache.Enabled {
			t.Error("Expected cache to be disabled from environment variable")
		}
		if time.Duration(config.Cache.TTL) != 30*time.Minute {
			t.Errorf("Expected cache TTL 30m from env, got %v", config.Cache.TTL)
		}
		if config.Cache.MaxSize != 2000 {
			t.Errorf("Expected cache max size 2000 from env, got %d", config.Cache.MaxSize)
		}

		// Create client and verify it works
		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("Failed to create client with env override config: %v", err)
		}
		defer client.Close()

		ctx := context.Background()

		// Verify default flag was still loaded from file
		flag, err := client.GetFlag(ctx, "env-override-test-flag")
		if err != nil {
			t.Fatalf("Failed to get default flag: %v", err)
		}

		if flag.Enabled {
			t.Error("Expected env-override-test-flag to be disabled")
		}
	})

	t.Run("Redis Configuration Integration", func(t *testing.T) {
		// Skip if Redis is not available
		redisURL := os.Getenv("REDIS_TEST_URL")
		if redisURL == "" {
			t.Skip("REDIS_TEST_URL not set, skipping Redis config integration test")
		}

		// Create Redis config file
		redisConfig := `{
			"storage": {
				"type": "redis",
				"redis": {
					"addr": "localhost:6379",
					"db": 1
				}
			},
			"cache": {
				"enabled": true,
				"ttl": "10m",
				"max_size": 500
			},
			"default_flags": [
				{
					"key": "redis-config-test-flag",
					"enabled": true,
					"description": "Test flag for Redis config integration"
				}
			]
		}`

		redisConfigPath := filepath.Join(tempDir, "redis-config.json")
		if err := os.WriteFile(redisConfigPath, []byte(redisConfig), 0644); err != nil {
			t.Fatalf("Failed to write Redis config file: %v", err)
		}

		// Load configuration and create client
		config, err := LoadConfigFromFile(redisConfigPath)
		if err != nil {
			t.Fatalf("Failed to load Redis config: %v", err)
		}

		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("Failed to create client with Redis config: %v", err)
		}
		defer client.Close()

		ctx := context.Background()

		// Verify default flag was loaded into Redis
		flag, err := client.GetFlag(ctx, "redis-config-test-flag")
		if err != nil {
			t.Fatalf("Failed to get default flag from Redis: %v", err)
		}

		if !flag.Enabled {
			t.Error("Expected redis-config-test-flag to be enabled")
		}

		// Test that we can perform operations
		testFlag := FeatureFlag{
			Key:         "redis-config-runtime-flag",
			Enabled:     true,
			Description: "Runtime flag for Redis config test",
		}

		err = client.SetFlag(ctx, testFlag)
		if err != nil {
			t.Fatalf("Failed to set runtime flag: %v", err)
		}

		enabled, err := client.IsEnabled(ctx, "redis-config-runtime-flag")
		if err != nil {
			t.Fatalf("Failed to check runtime flag: %v", err)
		}
		if !enabled {
			t.Error("Expected runtime flag to be enabled")
		}

		// Clean up
		client.DeleteFlag(ctx, "redis-config-test-flag")
		client.DeleteFlag(ctx, "redis-config-runtime-flag")
	})

	t.Run("PostgreSQL Configuration Integration", func(t *testing.T) {
		// Skip if PostgreSQL is not available
		postgresURL := os.Getenv("POSTGRES_TEST_URL")
		if postgresURL == "" {
			t.Skip("POSTGRES_TEST_URL not set, skipping PostgreSQL config integration test")
		}

		// Create PostgreSQL config file
		postgresConfig := `{
			"storage": {
				"type": "postgres",
				"postgres": {
					"host": "localhost",
					"port": 5432,
					"database": "testdb",
					"username": "testuser",
					"password": "testpass",
					"ssl_mode": "disable"
				}
			},
			"cache": {
				"enabled": true,
				"ttl": "5m",
				"max_size": 200
			},
			"default_flags": [
				{
					"key": "postgres-config-test-flag",
					"enabled": false,
					"description": "Test flag for PostgreSQL config integration",
					"metadata": {
						"db": "postgres",
						"test": "integration"
					}
				}
			]
		}`

		postgresConfigPath := filepath.Join(tempDir, "postgres-config.json")
		if err := os.WriteFile(postgresConfigPath, []byte(postgresConfig), 0644); err != nil {
			t.Fatalf("Failed to write PostgreSQL config file: %v", err)
		}

		// Load configuration and create client
		config, err := LoadConfigFromFile(postgresConfigPath)
		if err != nil {
			t.Fatalf("Failed to load PostgreSQL config: %v", err)
		}

		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("Failed to create client with PostgreSQL config: %v", err)
		}
		defer client.Close()

		ctx := context.Background()

		// Verify default flag was loaded into PostgreSQL
		flag, err := client.GetFlag(ctx, "postgres-config-test-flag")
		if err != nil {
			t.Fatalf("Failed to get default flag from PostgreSQL: %v", err)
		}

		if flag.Enabled {
			t.Error("Expected postgres-config-test-flag to be disabled")
		}
		if flag.Metadata["db"] != "postgres" {
			t.Errorf("Expected metadata db 'postgres', got '%s'", flag.Metadata["db"])
		}

		// Test that we can perform operations
		testFlag := FeatureFlag{
			Key:         "postgres-config-runtime-flag",
			Enabled:     true,
			Description: "Runtime flag for PostgreSQL config test",
			Metadata: map[string]string{
				"created_by": "integration_test",
			},
		}

		err = client.SetFlag(ctx, testFlag)
		if err != nil {
			t.Fatalf("Failed to set runtime flag: %v", err)
		}

		enabled, err := client.IsEnabled(ctx, "postgres-config-runtime-flag")
		if err != nil {
			t.Fatalf("Failed to check runtime flag: %v", err)
		}
		if !enabled {
			t.Error("Expected runtime flag to be enabled")
		}

		// Verify metadata was preserved
		retrievedFlag, err := client.GetFlag(ctx, "postgres-config-runtime-flag")
		if err != nil {
			t.Fatalf("Failed to get runtime flag: %v", err)
		}
		if retrievedFlag.Metadata["created_by"] != "integration_test" {
			t.Errorf("Expected metadata created_by 'integration_test', got '%s'", retrievedFlag.Metadata["created_by"])
		}

		// Clean up
		client.DeleteFlag(ctx, "postgres-config-test-flag")
		client.DeleteFlag(ctx, "postgres-config-runtime-flag")
	})

	t.Run("Configuration Validation Integration", func(t *testing.T) {
		// Test that invalid configurations are properly rejected
		invalidConfigs := []struct {
			name   string
			config string
		}{
			{
				name: "invalid storage type",
				config: `{
					"storage": {
						"type": "invalid"
					}
				}`,
			},
			{
				name: "missing Redis config",
				config: `{
					"storage": {
						"type": "redis"
					}
				}`,
			},
			{
				name: "invalid cache TTL",
				config: `{
					"storage": {
						"type": "memory"
					},
					"cache": {
						"enabled": true,
						"ttl": "invalid"
					}
				}`,
			},
		}

		for _, tc := range invalidConfigs {
			t.Run(tc.name, func(t *testing.T) {
				configPath := filepath.Join(tempDir, fmt.Sprintf("invalid-%s.json", tc.name))
				if err := os.WriteFile(configPath, []byte(tc.config), 0644); err != nil {
					t.Fatalf("Failed to write invalid config file: %v", err)
				}

				config, err := LoadConfigFromFile(configPath)
				if err != nil {
					// Some validation happens during loading
					return
				}

				// Validation should fail when creating client
				_, err = NewClient(config)
				if err == nil {
					t.Errorf("Expected error when creating client with invalid config, but got none")
				}
			})
		}
	})
}
