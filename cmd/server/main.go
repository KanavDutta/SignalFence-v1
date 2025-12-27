package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/yourusername/signalfence/api"
	"github.com/yourusername/signalfence/core"
	"github.com/yourusername/signalfence/metrics"
	"github.com/yourusername/signalfence/store"
)

func main() {
	// Configuration
	port := getEnv("PORT", "8080")
	redisAddr := getEnv("REDIS_ADDR", "")
	
	// Choose storage backend
	var storage store.Store
	if redisAddr != "" {
		redisStore := store.NewRedisStore(store.RedisConfig{
			Addr:     redisAddr,
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       0,
			TTL:      5 * time.Minute,
		})
		
		if err := redisStore.Ping(); err != nil {
			log.Fatal("❌ Failed to connect to Redis:", err)
		}
		fmt.Println("✅ Connected to Redis at", redisAddr)
		storage = redisStore
	} else {
		fmt.Println("⚠️  Using in-memory storage (not suitable for production)")
		storage = store.NewMemoryStore()
	}

	// Default rate limit policy
	defaultPolicy := core.Config{
		Capacity:     100,
		RefillPerSec: 10,
	}

	// Create metrics tracker
	metricsTracker := metrics.NewMetrics()

	// Create API handler
	handler := api.NewHandler(storage, defaultPolicy, metricsTracker)
	metricsHandler := api.NewMetricsHandler(metricsTracker)

	// Routes
	http.HandleFunc("/check", handler.CheckRateLimit)
	http.HandleFunc("/metrics", metricsHandler.ServeHTTP)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/dashboard", dashboardHandler)
	http.HandleFunc("/", rootHandler)

	// Start server
	addr := ":" + port
	fmt.Println("🚦 SignalFence Rate Limiting Service")
	fmt.Println("📍 Listening on http://localhost" + addr)
	fmt.Println()
	fmt.Println("Endpoints:")
	fmt.Println("  POST /check       - Check if request is allowed")
	fmt.Println("  GET  /metrics     - View metrics (JSON)")
	fmt.Println("  GET  /dashboard   - View dashboard (HTML)")
	fmt.Println("  GET  /health      - Health check")
	fmt.Println()
	fmt.Println("📊 Dashboard: http://localhost" + addr + "/dashboard")
	fmt.Println()

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "signalfence",
		"version": "1.0.0",
	})
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service": "SignalFence Rate Limiting Service",
		"version": "1.0.0",
		"endpoints": map[string]string{
			"POST /check":  "Check if a request is allowed",
			"GET /health": "Health check",
		},
		"docs": "https://github.com/yourusername/signalfence",
	})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
