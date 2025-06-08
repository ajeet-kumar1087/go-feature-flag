package featureflag

import (
	"net/http"
)

func SetupRoutes(store Store) *http.ServeMux {
	mux := http.NewServeMux()

	// Feature flag endpoints
	mux.HandleFunc("/flags/get", GetFlagHandler(store))
	mux.HandleFunc("/flags/create", AddFlagHandler(store))
	mux.HandleFunc("/flags/enable", EnableFlagHandler(store))
	mux.HandleFunc("/flags/all", GetAllFlagsHandler(store))
	mux.HandleFunc("/flags/delete", DeleteFlagHandler(store))
	mux.HandleFunc("/flags/reset", ResetFlagsHandler(store))

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok"}`))
	})

	return mux
}
