package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/yourusername/signalfence"
	"github.com/yourusername/signalfence/store"
)

func main() {
	// Create Redis store
	redisStore := store.NewRedisStore(store.RedisConfig{
		Addr:     "localhost:6379",
		Password: "", // No password for local dev
		DB:       0,
		TTL:      5 * time.Minute,
	})

	// Test Redis connection
	if err := redisStore.Ping(); err != nil {
		log.Fatal("❌ Failed to connect to Redis:", err)
	}
	fmt.Println("✅ Connected to Redis")

	// Create rate limiter with Redis backend
	limiter := signalfence.NewRateLimiter(signalfence.Config{
		Capacity:     20,
		RefillPerSec: 10,
		Store:        redisStore, // Use Redis instead of in-memory
		KeyFunc: func(r *http.Request) string {
			if key := r.Header.Get("X-API-Key"); key != "" {
				return "api:" + key
			}
			return r.RemoteAddr
		},
	})

	// Protected endpoint
	http.Handle("/api/data", limiter.Middleware(http.HandlerFunc(dataHandler)))
	
	// Health check
	http.HandleFunc("/health", healthHandler)

	fmt.Println("🚦 SignalFence Redis demo server starting...")
	fmt.Println("📍 Listening on http://localhost:8080")
	fmt.Println("🔴 Using Redis for distributed rate limiting")
	fmt.Println()
	fmt.Println("Try these commands:")
	fmt.Println("  curl http://localhost:8080/api/data")
	fmt.Println("  curl -H 'X-API-Key: test123' http://localhost:8080/api/data")
	fmt.Println()
	fmt.Println("Test multi-server scenario:")
	fmt.Println("  1. Run this server on port 8080")
	fmt.Println("  2. Run another instance on port 8081")
	fmt.Println("  3. Hit both servers - they share the same rate limit!")
	fmt.Println()

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func dataHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   "Success! Rate limited via Redis.",
		"timestamp": time.Now().Unix(),
		"server":    "instance-1",
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"store":  "redis",
	})
}
