package main

import (
	"net/http"

	"github.com/ajeet-kumar1087/go-feature-flag/featureflag"
)

func main() {
	// Initialize Redis store
	store := featureflag.NewRedisStore("localhost:6379")

	// Setup HTTP routes
	mux := featureflag.SetupRoutes(store)

	// Start the HTTP server
	http.ListenAndServe(":8080", mux)

}
