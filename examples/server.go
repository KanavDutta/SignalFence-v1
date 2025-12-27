package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/yourusername/signalfence"
)

func main() {
	// Create rate limiter: 10 requests per second, burst of 20
	limiter := signalfence.NewRateLimiter(signalfence.Config{
		Capacity:     20,
		RefillPerSec: 10,
		// Optional: use API key from header
		KeyFunc: func(r *http.Request) string {
			if key := r.Header.Get("X-API-Key"); key != "" {
				return "api:" + key
			}
			// Fallback to IP
			return r.RemoteAddr
		},
	})

	// Protected endpoint
	http.Handle("/api/data", limiter.Middleware(http.HandlerFunc(dataHandler)))
	
	// Unprotected endpoint for comparison
	http.HandleFunc("/api/public", publicHandler)
	
	// Health check
	http.HandleFunc("/health", healthHandler)

	fmt.Println("🚦 SignalFence demo server starting...")
	fmt.Println("📍 Listening on http://localhost:8080")
	fmt.Println()
	fmt.Println("Try these commands:")
	fmt.Println("  curl http://localhost:8080/api/data")
	fmt.Println("  curl -H 'X-API-Key: test123' http://localhost:8080/api/data")
	fmt.Println()
	fmt.Println("Spam to trigger rate limit:")
	fmt.Println("  for i in {1..25}; do curl http://localhost:8080/api/data; echo; done")
	fmt.Println()

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func dataHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   "Success! This endpoint is rate limited.",
		"timestamp": time.Now().Unix(),
		"path":      r.URL.Path,
	})
}

func publicHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "This endpoint has no rate limit.",
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
}
