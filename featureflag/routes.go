package featureflag

import (
	"net/http"
)

func SetupRoutes(store FlagStore) *http.ServeMux {
	mux := http.NewServeMux()

	// Feature flag endpoints
	mux.HandleFunc("/flags/", GetFlagHandler(store))
	mux.HandleFunc("/flags", SetFlagHandler(store))
	mux.HandleFunc("/flags/all", GetAllFlagsHandler(store))
	mux.HandleFunc("/flags/", DeleteFlagHandler(store))

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok"}`))
	})

	return mux
}
