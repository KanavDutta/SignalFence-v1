// Package signalfence provides rate limiting and abuse detection for Go applications.
//
// SignalFence implements the token bucket algorithm for flexible, burst-friendly rate limiting.
// It's designed for both learning and production use, with a clean API and comprehensive test coverage.
//
// # Quick Start
//
// Basic usage with default settings:
//
//	limiter, err := signalfence.NewRateLimiter(
//	    signalfence.WithDefaults(100, 10.0),  // 100 tokens, 10/sec refill
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	decision, err := limiter.Allow("user-123")
//	if !decision.Allowed {
//	    fmt.Printf("Rate limited. Retry after %v\n", decision.RetryAfter)
//	}
//
// # HTTP Middleware
//
// Use as HTTP middleware for automatic rate limiting:
//
//	limiter, _ := signalfence.NewRateLimiter(
//	    signalfence.WithDefaults(100, 10.0),
//	    signalfence.WithKeyExtractor(signalfence.ExtractIPWithProxy()),
//	)
//
//	http.Handle("/api/", limiter.Middleware(yourHandler))
//
// The middleware automatically sets standard rate limit headers:
//   - X-RateLimit-Limit: Maximum requests allowed
//   - X-RateLimit-Remaining: Remaining requests in current window
//   - X-RateLimit-Reset: Unix timestamp when limit resets
//   - Retry-After: Seconds to wait before retrying (when rate limited)
//
// # Configuration
//
// Load configuration from YAML file:
//
//	limiter, err := signalfence.NewRateLimiter(
//	    signalfence.WithConfigFile("config.yaml"),
//	)
//
// Example YAML configuration:
//
//	defaults:
//	  capacity: 100
//	  refill_rate: 10.0
//	  enabled: true
//
//	policies:
//	  "/api/login":
//	    capacity: 5
//	    refill_rate: 0.083  # ~5 req/min
//	    enabled: true
//
//	key_extractor: "ip"
//	cleanup_age: "1h"
//
// # Key Extraction
//
// Multiple strategies for identifying clients:
//
//	// Extract from IP address
//	signalfence.WithKeyExtractor(signalfence.ExtractIP())
//
//	// Extract from IP with proxy support (X-Forwarded-For, X-Real-IP)
//	signalfence.WithKeyExtractor(signalfence.ExtractIPWithProxy())
//
//	// Extract from header
//	signalfence.WithKeyExtractor(signalfence.ExtractHeader("X-API-Key"))
//
//	// Extract from Bearer token
//	signalfence.WithKeyExtractor(signalfence.ExtractBearer())
//
//	// Extract from cookie
//	signalfence.WithKeyExtractor(signalfence.ExtractCookie("session_id"))
//
//	// Composite with fallback
//	signalfence.WithKeyExtractor(signalfence.ExtractComposite(
//	    signalfence.ExtractHeader("X-API-Key"),
//	    signalfence.ExtractIPWithProxy(),  // Fallback
//	))
//
// # Token Bucket Algorithm
//
// SignalFence uses the token bucket algorithm, which:
//   - Allows burst traffic up to capacity
//   - Refills tokens at a steady rate
//   - Is thread-safe and high-performance
//   - Is used by AWS, Stripe, GitHub, and other major services
//
// Each request consumes one token. When tokens are exhausted, requests are denied
// until tokens refill over time.
//
// # Concurrency
//
// All operations are thread-safe:
//   - Uses sync.RWMutex for bucket map
//   - Uses sync.Mutex for individual buckets
//   - Tested with -race flag for data race detection
//   - Supports high concurrent load
//
// # Storage
//
// The library uses an in-memory store by default, which:
//   - Provides nanosecond-level performance
//   - Automatically cleans up idle buckets
//   - Works for single-instance deployments
//
// The Store interface allows future extensions:
//   - Redis for distributed rate limiting
//   - Database for persistent limits
//   - Custom implementations
//
// # Examples
//
// See the examples/ directory and cmd/demo/ for complete working examples.
//
// # Performance
//
// Benchmarks show:
//   - <1μs per Allow() call
//   - Linear scaling with concurrent goroutines
//   - <10MB memory for 10,000 active buckets
//
// Run benchmarks:
//
//	go test -bench=. -benchmem ./benchmarks
//
// # Testing
//
// Run tests with coverage and race detection:
//
//	go test -v -race -cover ./pkg/signalfence
//
package signalfence
