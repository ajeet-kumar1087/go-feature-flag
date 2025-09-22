package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	featureflag "github.com/ajeet-kumar1087/go-feature-flag"
)

type WebService struct {
	client featureflag.Client
}

func main() {
	// Configure for web service
	config := featureflag.Config{
		Storage: featureflag.StorageConfig{
			Type: "memory", // Use Redis in production
		},
		Cache: featureflag.CacheConfig{
			Enabled: true,
			TTL:     featureflag.Duration(5 * time.Minute),
			MaxSize: 1000,
		},
		Observability: featureflag.ObservabilityConfig{
			Logging: featureflag.LoggingConfig{
				Enabled: true,
				Level:   "info",
			},
			Metrics: featureflag.MetricsConfig{
				Enabled: true,
			},
		},
	}

	client, err := featureflag.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	service := &WebService{client: client}

	// Set up some demo flags
	service.setupDemoFlags()

	// HTTP routes
	http.HandleFunc("/", service.homeHandler)
	http.HandleFunc("/api/features", service.featuresHandler)
	http.HandleFunc("/health", service.healthHandler)
	http.HandleFunc("/metrics", service.metricsHandler)

	log.Println("🌐 Web service starting on http://localhost:8080")
	log.Println("   Try: curl http://localhost:8080")
	log.Println("   Try: curl http://localhost:8080/api/features")
	log.Println("   Try: curl http://localhost:8080/health")
	log.Println("   Try: curl http://localhost:8080/metrics")

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (s *WebService) setupDemoFlags() {
	ctx := context.Background()
	flags := []featureflag.FeatureFlag{
		{
			Key:         "new-homepage",
			Enabled:     true,
			Description: "Enable new homepage design",
		},
		{
			Key:         "dark-mode",
			Enabled:     false,
			Description: "Enable dark mode theme",
		},
		{
			Key:         "premium-features",
			Enabled:     true,
			Description: "Enable premium features",
		},
	}

	for _, flag := range flags {
		if err := s.client.SetFlag(ctx, flag); err != nil {
			log.Printf("Failed to set flag %s: %v", flag.Key, err)
		}
	}
}

func (s *WebService) homeHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<h1>Feature Flag Demo</h1>")

	// Check new homepage
	if enabled, _ := s.client.IsEnabled(ctx, "new-homepage"); enabled {
		fmt.Fprintf(w, "<p>🎨 <strong>New Homepage Design</strong> - You're seeing the latest design!</p>")
	} else {
		fmt.Fprintf(w, "<p>📝 Legacy Homepage - Classic design</p>")
	}

	// Check dark mode
	if enabled, _ := s.client.IsEnabled(ctx, "dark-mode"); enabled {
		fmt.Fprintf(w, "<p>🌙 Dark mode is available</p>")
	} else {
		fmt.Fprintf(w, "<p>☀️ Light mode only</p>")
	}

	// Check premium features
	if enabled, _ := s.client.IsEnabled(ctx, "premium-features"); enabled {
		fmt.Fprintf(w, "<p>⭐ Premium features unlocked!</p>")
	}

	fmt.Fprintf(w, "<p><a href='/api/features'>View all features (JSON)</a></p>")
	fmt.Fprintf(w, "<p><a href='/health'>Health check</a></p>")
	fmt.Fprintf(w, "<p><a href='/metrics'>Metrics</a></p>")
}

func (s *WebService) featuresHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	flags, err := s.client.GetAllFlags(ctx)
	if err != nil {
		http.Error(w, "Failed to get flags", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"flags": flags,
		"count": len(flags),
	})
}

func (s *WebService) healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
	}

	if err := s.client.HealthCheck(ctx); err != nil {
		health["status"] = "unhealthy"
		health["error"] = err.Error()
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (s *WebService) metricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := s.client.GetMetrics()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}
