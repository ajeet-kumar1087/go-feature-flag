package featureflag

import (
	"encoding/json"
	"net/http"
)

func GetFlagHandler(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var input struct {
			Key string `json:"key"`
		}

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid input", http.StatusBadRequest)
			return
		}

		if input.Key == "" {
			http.Error(w, "missing key", http.StatusBadRequest)
			return
		}

		flag, err := store.IsEnabled(input.Key)
		if err != nil {
			http.Error(w, "Error retrieving flag", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(flag)
	}
}

func AddFlagHandler(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var flag FeatureFlag
		if err := json.NewDecoder(r.Body).Decode(&flag); err != nil {
			http.Error(w, "invalid input", http.StatusBadRequest)
			return
		}
		if err := store.Create(flag); err != nil {
			http.Error(w, "error setting flag", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"message": "flag added"})
	}
}

func EnableFlagHandler(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var flag FeatureFlag
		if err := json.NewDecoder(r.Body).Decode(&flag); err != nil {
			http.Error(w, "invalid input", http.StatusBadRequest)
			return
		}
		if err := store.Enable(flag); err != nil {
			http.Error(w, "error enabling flag", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"message": "flag enabled"})
	}
}

func GetAllFlagsHandler(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		flags, err := store.GetAll()
		if err != nil {
			http.Error(w, "Error retrieving flags", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(flags)
	}
}

func DeleteFlagHandler(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var input struct {
			Key string `json:"key"`
		}

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid input", http.StatusBadRequest)
			return
		}

		if input.Key == "" {
			http.Error(w, "missing key", http.StatusBadRequest)
			return
		}

		if err := store.Delete(input.Key); err != nil {
			http.Error(w, "Error deleting flag", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"message": "flag deleted"})
	}
}

func ResetFlagsHandler(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := store.Reset(); err != nil {
			http.Error(w, "Error resetting flags", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"message": "flags reset"})
	}
}
