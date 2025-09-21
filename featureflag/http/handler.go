package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ajeet-kumar1087/go-feature-flag/featureflag/core"
)

func GetFlagHandler(store core.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		key := strings.TrimPrefix(r.URL.Path, "/flags/")
		key = strings.TrimSuffix(key, "/")
		if key == "" {
			http.Error(w, "Missing flag key in URL path", http.StatusBadRequest)
			return
		}

		flag, err := store.Get(r.Context(), key)
		if err != nil {
			http.Error(w, "Error retrieving flag: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(flag); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}

func SetFlagHandler(store core.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var flag core.FeatureFlag
		if err := json.NewDecoder(r.Body).Decode(&flag); err != nil {
			http.Error(w, "invalid input", http.StatusBadRequest)
			return
		}
		if err := store.Set(r.Context(), flag); err != nil {
			http.Error(w, "error setting flag", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"message": "flag added"})
	}
}

func EnableFlagHandler(store core.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var flag core.FeatureFlag
		if err := json.NewDecoder(r.Body).Decode(&flag); err != nil {
			http.Error(w, "invalid input", http.StatusBadRequest)
			return
		}
		if err := store.Set(r.Context(), flag); err != nil {
			http.Error(w, "error enabling flag", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"message": "flag enabled"})
	}
}

func GetAllFlagsHandler(store core.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		flags, err := store.GetAll(r.Context())
		if err != nil {
			http.Error(w, "Error retrieving flags", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(flags)
	}
}

func DeleteFlagHandler(store core.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		key := strings.TrimPrefix(r.URL.Path, "/flags/")
		key = strings.TrimSuffix(key, "/")
		if key == "" {
			http.Error(w, "Missing flag key in URL", http.StatusBadRequest)
			return
		}

		if err := store.Delete(r.Context(), key); err != nil {
			http.Error(w, "Error deleting flag: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "Flag deleted"})
	}
}
