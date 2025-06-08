package featureflag

import (
	"net/http"
)

func SetupRoutes(store Store) *http.ServeMux {
	mux := http.NewServeMux()

	// Feature flag endpoints
	mux.HandleFunc("/api/v1/flags/get", GetFlagHandler(store))
	mux.HandleFunc("/api/v1/flags/create", AddFlagHandler(store))

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok"}`))
	})

	return mux
}
