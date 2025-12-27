package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/KanavDutta/SignalFence-v1/cmd/demo/handlers"
	"github.com/KanavDutta/SignalFence-v1/pkg/signalfence"
)

func main() {
	// Command-line flags
	port := flag.String("port", "8080", "Port to run the server on")
	configFile := flag.String("config", "cmd/demo/config.yaml", "Path to configuration file")
	flag.Parse()

	// Print banner
	printBanner()

	// Initialize rate limiter
	log.Println("Loading configuration from:", *configFile)
	limiter, err := signalfence.NewRateLimiter(
		signalfence.WithConfigFile(*configFile),
	)
	if err != nil {
		log.Fatalf("Failed to create rate limiter: %v", err)
	}

	// Start background cleanup
	stopCleanup := limiter.StartBackgroundCleanup()
	defer stopCleanup()

	log.Println("Rate limiter initialized successfully")
	log.Println("Background cleanup started")

	// Create HTTP mux
	mux := http.NewServeMux()

	// Health check endpoint (no rate limiting)
	mux.HandleFunc("/health", handlers.Health)

	// API endpoints with rate limiting
	mux.Handle("/api/search", limiter.Middleware(http.HandlerFunc(handlers.Search)))
	mux.Handle("/api/create", limiter.Middleware(http.HandlerFunc(handlers.Create)))
	mux.Handle("/api/login", limiter.Middleware(http.HandlerFunc(handlers.Login)))
	mux.Handle("/api/update", limiter.Middleware(http.HandlerFunc(handlers.Update)))

	// Root endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, `SignalFence Demo Server

Available endpoints:
  GET  /health       - Health check (no rate limit)
  GET  /api/search   - Search endpoint (100 req/min)
  POST /api/create   - Create resource (20 req/min)
  POST /api/login    - Login endpoint (5 req/min - anti brute-force)
  PUT  /api/update   - Update resource (30 req/min)

Try it:
  curl http://localhost:%s/health
  curl http://localhost:%s/api/search?q=test
  curl -X POST http://localhost:%s/api/login

Rate limit headers:
  X-RateLimit-Limit     - Maximum requests allowed
  X-RateLimit-Remaining - Remaining requests in current window
  X-RateLimit-Reset     - Unix timestamp when limit resets
  Retry-After           - Seconds to wait (when rate limited)
`, *port, *port, *port)
	})

	// Start server
	addr := ":" + *port
	log.Printf("Starting server on http://localhost%s", addr)
	log.Println("Press Ctrl+C to stop")
	log.Println("")
	log.Println("Try these commands:")
	log.Printf("  curl http://localhost%s/health\n", *port)
	log.Printf("  curl http://localhost%s/api/search?q=golang\n", *port)
	log.Printf("  curl -X POST http://localhost%s/api/login\n", *port)
	log.Println("")

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func printBanner() {
	banner := `
╔═══════════════════════════════════════════════════════╗
║                                                       ║
║   ███████╗██╗ ██████╗ ███╗   ██╗ █████╗ ██╗          ║
║   ██╔════╝██║██╔════╝ ████╗  ██║██╔══██╗██║          ║
║   ███████╗██║██║  ███╗██╔██╗ ██║███████║██║          ║
║   ╚════██║██║██║   ██║██║╚██╗██║██╔══██║██║          ║
║   ███████║██║╚██████╔╝██║ ╚████║██║  ██║███████╗     ║
║   ╚══════╝╚═╝ ╚═════╝ ╚═╝  ╚═══╝╚═╝  ╚═╝╚══════╝     ║
║                                                       ║
║             FENCE - Demo Server                       ║
║                                                       ║
║   Rate Limiting & Abuse Detection Service            ║
║   Token Bucket Algorithm | Go Implementation         ║
║                                                       ║
╚═══════════════════════════════════════════════════════╝
`
	fmt.Println(banner)
}
