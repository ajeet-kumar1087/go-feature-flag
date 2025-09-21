// Package featureflag provides a backward compatibility layer for the modularized feature flag library.
// This package re-exports commonly used types and functions from the various modules.
package featureflag

import (
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/cache"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/client"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/config"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/core"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/storage/memory"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/storage/postgres"
	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/storage/redis"
)

// Re-export core types and interfaces
type (
	// Core types
	FeatureFlag = core.FeatureFlag
	Store       = core.Store

	// Configuration
	Config = config.Config

	// Client
	Client = core.Client
)

// Re-export core errors
var (
	ErrFlagNotFound   = core.ErrFlagNotFound
	ErrInvalidFlag    = core.ErrInvalidFlag
	ErrStorageFailure = core.ErrStorageFailure
	ErrInvalidConfig  = core.ErrInvalidConfig
	ErrClientClosed   = core.ErrClientClosed
)

// Storage constructors
var (
	NewMemoryStore   = memory.NewStore
	NewPostgresStore = postgres.NewStore
	NewRedisStore    = redis.NewStore
)

// Cache constructors
var (
	NewCache      = cache.NewCache
	NewCacheStore = cache.NewStore
)

// Client constructor
var (
	NewClient = client.NewClient
)

// Config constructor
var (
	DefaultConfig = config.DefaultConfig
)
