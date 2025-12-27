package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/KanavDutta/SignalFence-v1/pkg/signalfence"
)

func main() {
	// Load configuration from YAML file
	limiter, err := signalfence.NewRateLimiter(
		signalfence.WithConfigFile("examples/with-config/config.yaml"),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("SignalFence Config File Example")
	fmt.Println("================================")
	fmt.Println("Configuration loaded from: examples/with-config/config.yaml")
	fmt.Println()

	// Create a simple HTTP handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Request successful!\n")
	})

	// Wrap with rate limiting middleware
	http.Handle("/", limiter.Middleware(handler))

	fmt.Println("Server starting on http://localhost:8081")
	fmt.Println("Try these commands:")
	fmt.Println("  curl -i http://localhost:8081")
	fmt.Println("  for i in {1..20}; do curl http://localhost:8081; done")
	fmt.Println()

	log.Fatal(http.ListenAndServe(":8081", nil))
}
