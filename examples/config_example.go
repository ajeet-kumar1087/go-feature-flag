//go:build examples
// +build examples

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ajeet-kumar1087/go-feature-flag/featureflag"
)

func main() {
	fmt.Println("=== Feature Flag Configuration System Demo ===\n")

	// Example 1: Default configuration
	fmt.Println("1. Default Configuration:")
	defaultConfig := featureflag.DefaultConfig()
	fmt.Printf("   Storage Type: %s\n", defaultConfig.Storage.Type)
	fmt.Printf("   Cache Enabled: %t\n", defaultConfig.Cache.Enabled)
	fmt.Printf("   Cache TTL: %v\n", time.Duration(defaultConfig.Cache.TTL))
	fmt.Printf("   Cache Max Size: %d\n", defaultConfig.Cache.MaxSize)
	fmt.Println()

	// Example 2: Programmatic configuration
	fmt.Println("2. Programmatic Configuration:")
	programmaticConfig := featureflag.Config{
		Storage: featureflag.StorageConfig{
			Type: "redis",
			Redis: &featureflag.RedisConfig{
				Addr: "localhost:6379",
				DB:   0,
			},
		},
		Cache: featureflag.CacheConfig{
			Enabled: true,
			TTL:     featureflag.Duration(10 * time.Minute),
			MaxSize: 2000,
		},
		DefaultFlags: []featureflag.FeatureFlag{
			{
				Key:         "example-feature",
				Enabled:     true,
				Description: "Example feature flag",
			},
		},
	}

	if err := programmaticConfig.Validate(); err != nil {
		log.Printf("   Configuration validation failed: %v\n", err)
	} else {
		fmt.Printf("   ✓ Configuration is valid\n")
		fmt.Printf("   Storage Type: %s\n", programmaticConfig.Storage.Type)
		fmt.Printf("   Redis Address: %s\n", programmaticConfig.Storage.Redis.Addr)
		fmt.Printf("   Default Flags: %d\n", len(programmaticConfig.DefaultFlags))
	}
	fmt.Println()

	// Example 3: Environment variable configuration
	fmt.Println("3. Environment Variable Configuration:")

	// Set some environment variables for demonstration
	os.Setenv("FEATUREFLAG_STORAGE_TYPE", "postgres")
	os.Setenv("FEATUREFLAG_POSTGRES_HOST", "localhost")
	os.Setenv("FEATUREFLAG_POSTGRES_PORT", "5432")
	os.Setenv("FEATUREFLAG_POSTGRES_DATABASE", "featureflags")
	os.Setenv("FEATUREFLAG_POSTGRES_USERNAME", "postgres")
	os.Setenv("FEATUREFLAG_POSTGRES_SSLMODE", "disable")
	os.Setenv("FEATUREFLAG_CACHE_ENABLED", "true")
	os.Setenv("FEATUREFLAG_CACHE_TTL", "15m")
	os.Setenv("FEATUREFLAG_CACHE_MAX_SIZE", "5000")

	envConfig := featureflag.LoadConfigFromEnv()
	if err := envConfig.Validate(); err != nil {
		log.Printf("   Environment configuration validation failed: %v\n", err)
	} else {
		fmt.Printf("   ✓ Environment configuration is valid\n")
		fmt.Printf("   Storage Type: %s\n", envConfig.Storage.Type)
		if envConfig.Storage.Postgres != nil {
			fmt.Printf("   Postgres Host: %s\n", envConfig.Storage.Postgres.Host)
			fmt.Printf("   Postgres Database: %s\n", envConfig.Storage.Postgres.Database)
		}
		fmt.Printf("   Cache TTL: %v\n", time.Duration(envConfig.Cache.TTL))
	}
	fmt.Println()

	// Example 4: JSON configuration file
	fmt.Println("4. JSON Configuration File:")

	// Create a temporary JSON config file
	jsonConfig := `{
		"storage": {
			"type": "redis",
			"redis": {
				"addr": "redis:6379",
				"password": "secret",
				"db": 1
			}
		},
		"cache": {
			"enabled": false,
			"ttl": "30m",
			"max_size": 10000
		},
		"default_flags": [
			{
				"key": "json-feature",
				"enabled": true,
				"description": "Feature loaded from JSON"
			},
			{
				"key": "another-feature",
				"enabled": false,
				"description": "Another feature from JSON"
			}
		]
	}`

	// Write to temporary file
	tmpFile, err := os.CreateTemp("", "config*.json")
	if err != nil {
		log.Printf("   Failed to create temp file: %v\n", err)
	} else {
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.WriteString(jsonConfig); err != nil {
			log.Printf("   Failed to write config: %v\n", err)
		} else {
			tmpFile.Close()

			fileConfig, err := featureflag.LoadConfigFromFile(tmpFile.Name())
			if err != nil {
				log.Printf("   Failed to load JSON config: %v\n", err)
			} else {
				fmt.Printf("   ✓ JSON configuration loaded successfully\n")
				fmt.Printf("   Storage Type: %s\n", fileConfig.Storage.Type)
				if fileConfig.Storage.Redis != nil {
					fmt.Printf("   Redis Address: %s\n", fileConfig.Storage.Redis.Addr)
					fmt.Printf("   Redis DB: %d\n", fileConfig.Storage.Redis.DB)
				}
				fmt.Printf("   Cache Enabled: %t\n", fileConfig.Cache.Enabled)
				fmt.Printf("   Default Flags: %d\n", len(fileConfig.DefaultFlags))
			}
		}
	}
	fmt.Println()

	// Example 5: YAML configuration file
	fmt.Println("5. YAML Configuration File:")

	yamlConfig := `
storage:
  type: postgres
  postgres:
    host: db.example.com
    port: 5432
    database: production_flags
    username: flaguser
    password: flagpass
    ssl_mode: require
cache:
  enabled: true
  ttl: 1h
  max_size: 50000
default_flags:
  - key: yaml-feature
    enabled: true
    description: Feature loaded from YAML
  - key: production-feature
    enabled: false
    description: Production feature from YAML
`

	// Write to temporary YAML file
	tmpYamlFile, err := os.CreateTemp("", "config*.yaml")
	if err != nil {
		log.Printf("   Failed to create temp YAML file: %v\n", err)
	} else {
		defer os.Remove(tmpYamlFile.Name())

		if _, err := tmpYamlFile.WriteString(yamlConfig); err != nil {
			log.Printf("   Failed to write YAML config: %v\n", err)
		} else {
			tmpYamlFile.Close()

			yamlFileConfig, err := featureflag.LoadConfigFromFile(tmpYamlFile.Name())
			if err != nil {
				log.Printf("   Failed to load YAML config: %v\n", err)
			} else {
				fmt.Printf("   ✓ YAML configuration loaded successfully\n")
				fmt.Printf("   Storage Type: %s\n", yamlFileConfig.Storage.Type)
				if yamlFileConfig.Storage.Postgres != nil {
					fmt.Printf("   Postgres Host: %s\n", yamlFileConfig.Storage.Postgres.Host)
					fmt.Printf("   Postgres SSL Mode: %s\n", yamlFileConfig.Storage.Postgres.SSLMode)
				}
				fmt.Printf("   Cache TTL: %v\n", time.Duration(yamlFileConfig.Cache.TTL))
				fmt.Printf("   Default Flags: %d\n", len(yamlFileConfig.DefaultFlags))
			}
		}
	}
	fmt.Println()

	// Example 6: Configuration precedence (env vars override file)
	fmt.Println("6. Configuration Precedence (Environment overrides File):")

	// Use the YAML file but override with environment variables
	if tmpYamlFile != nil {
		// Set environment variable to override storage type
		os.Setenv("FEATUREFLAG_STORAGE_TYPE", "memory")
		os.Setenv("FEATUREFLAG_CACHE_TTL", "2h")

		finalConfig, err := featureflag.LoadConfig(tmpYamlFile.Name())
		if err != nil {
			log.Printf("   Failed to load final config: %v\n", err)
		} else {
			fmt.Printf("   ✓ Configuration loaded with precedence\n")
			fmt.Printf("   Storage Type: %s (overridden by env var)\n", finalConfig.Storage.Type)
			fmt.Printf("   Cache TTL: %v (overridden by env var)\n", time.Duration(finalConfig.Cache.TTL))
			fmt.Printf("   Default Flags: %d (from file)\n", len(finalConfig.DefaultFlags))
		}
	}

	// Clean up environment variables
	envVarsToClean := []string{
		"FEATUREFLAG_STORAGE_TYPE",
		"FEATUREFLAG_POSTGRES_HOST",
		"FEATUREFLAG_POSTGRES_PORT",
		"FEATUREFLAG_POSTGRES_DATABASE",
		"FEATUREFLAG_POSTGRES_USERNAME",
		"FEATUREFLAG_POSTGRES_SSLMODE",
		"FEATUREFLAG_CACHE_ENABLED",
		"FEATUREFLAG_CACHE_TTL",
		"FEATUREFLAG_CACHE_MAX_SIZE",
	}

	for _, envVar := range envVarsToClean {
		os.Unsetenv(envVar)
	}

	fmt.Println("\n=== Configuration System Demo Complete ===")
}
