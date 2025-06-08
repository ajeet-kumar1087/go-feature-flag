package featureflag

import (
	"fmt"
	"log"
	"net/http"
)

// Config holds the configuration for the feature flag service
type Config struct {
	RedisAddr string
	Port      int
}

// New initializes and starts the feature flag service
func New(cfg Config) error {
	store := NewRedisStore(cfg.RedisAddr)
	mux := SetupRoutes(store)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Starting feature flag service on %s", addr)

	return http.ListenAndServe(addr, mux)
}
