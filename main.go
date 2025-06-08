package main

import (
	"log"

	"github.com/ajeet-kumar1087/go-feature-flag/featureflag"
)

func main() {
	cfg := featureflag.Config{
		RedisAddr: "localhost:6379",
		Port:      8080,
	}

	if err := featureflag.New(cfg); err != nil {
		log.Fatalf("Failed to start feature flag service: %v", err)
	}
}
