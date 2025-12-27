package signalfence

import (
	"fmt"
	"net/http"
	"time"
)

// RateLimiter is the main interface for rate limiting.
type RateLimiter interface {
	// Allow checks if a request with the given key is allowed.
	// Returns a Decision indicating whether the request should be allowed.
	Allow(key string) (*Decision, error)

	// AllowRequest is a convenience method that extracts the key from the request
	// and checks if it's allowed. It uses the configured key extractor and route extractor.
	AllowRequest(r *http.Request) (*Decision, error)

	// Middleware returns an HTTP middleware that applies rate limiting.
	Middleware(next http.Handler) http.Handler

	// StartBackgroundCleanup starts a goroutine that periodically cleans up idle buckets.
	// Returns a function to stop the cleanup goroutine.
	StartBackgroundCleanup() func()
}

// Decision contains the result of a rate limit check.
type Decision struct {
	// Allowed indicates whether the request should be allowed (true) or denied (false)
	Allowed bool

	// Remaining is the number of tokens remaining in the bucket
	Remaining int64

	// Limit is the total capacity of the bucket (max burst)
	Limit int64

	// RetryAfter is how long to wait before the next request would be allowed
	// This is 0 if Allowed is true
	RetryAfter time.Duration

	// Key is the rate limit key that was used
	Key string

	// Route is the route path that was checked
	Route string
}

// rateLimiter is the concrete implementation of RateLimiter.
type rateLimiter struct {
	store           Store
	config          *Config
	keyExtractor    KeyExtractor
	routeExtractor  func(string) string
	cleanupAge      time.Duration
	cleanupInterval time.Duration
}

// NewRateLimiter creates a new RateLimiter with the given options.
// If no options are provided, it uses sensible defaults.
//
// Example:
//
//	limiter, err := NewRateLimiter(
//	    WithDefaults(100, 10.0),  // 100 tokens, 10/sec refill
//	    WithKeyExtractor(ExtractIPWithProxy()),
//	)
func NewRateLimiter(opts ...Option) (RateLimiter, error) {
	// Start with defaults
	rl := &rateLimiter{
		config:          NewConfig(),
		keyExtractor:    nil, // Will be set below
		routeExtractor:  func(path string) string { return path },
		cleanupAge:      1 * time.Hour,
		cleanupInterval: 10 * time.Minute,
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(rl); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// Set key extractor if not explicitly provided via option
	if rl.keyExtractor == nil {
		extractor, err := ParseKeyExtractorConfig(rl.config.KeyExtractor)
		if err != nil {
			return nil, fmt.Errorf("failed to parse key extractor config: %w", err)
		}
		rl.keyExtractor = extractor
	}

	// Create default store if not provided
	if rl.store == nil {
		bucketConfig := rl.config.Defaults.ToBucketConfig()
		store, err := NewInMemoryStore(bucketConfig, rl.cleanupAge)
		if err != nil {
			return nil, fmt.Errorf("failed to create default store: %w", err)
		}
		rl.store = store
	}

	return rl, nil
}

// Allow checks if a request with the given key is allowed.
func (rl *rateLimiter) Allow(key string) (*Decision, error) {
	if key == "" {
		return nil, ErrInvalidKey
	}

	// Get bucket for this key
	bucket, err := rl.store.GetBucket(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}

	// Check if request is allowed
	allowed := bucket.Allow()

	decision := &Decision{
		Allowed:    allowed,
		Remaining:  bucket.Remaining(),
		Limit:      bucket.Capacity(),
		RetryAfter: 0,
		Key:        key,
	}

	if !allowed {
		decision.RetryAfter = bucket.RetryAfter()
	}

	return decision, nil
}

// AllowRequest checks if an HTTP request is allowed based on the configured
// key extractor. For MVP, uses the default policy for all routes.
// Per-route policies will be supported in a future version.
func (rl *rateLimiter) AllowRequest(r *http.Request) (*Decision, error) {
	// Extract key
	key, err := rl.keyExtractor(r)
	if err != nil {
		return nil, fmt.Errorf("key extraction failed: %w", err)
	}

	// Get route
	route := rl.routeExtractor(r.URL.Path)

	// Get policy for this route
	policy := rl.config.GetPolicy(route)

	// Check if rate limiting is enabled for this route
	if !policy.Enabled {
		return &Decision{
			Allowed:    true,
			Remaining:  policy.Capacity,
			Limit:      policy.Capacity,
			RetryAfter: 0,
			Key:        key,
			Route:      route,
		}, nil
	}

	// For MVP: use the default policy from the store
	// In future versions, we'll support per-route policies with separate stores
	decision, err := rl.Allow(key)
	if err != nil {
		return nil, err
	}
	decision.Route = route

	return decision, nil
}

// Middleware returns an HTTP middleware that applies rate limiting.
// It sets standard rate limit headers and returns 429 when limits are exceeded.
//
// Standard Headers (RFC 6585 + draft-ietf-httpapi-ratelimit-headers):
//   - X-RateLimit-Limit: Maximum requests allowed in the window
//   - X-RateLimit-Remaining: Remaining requests in current window
//   - X-RateLimit-Reset: Time when the limit resets (Unix timestamp)
//   - Retry-After: Seconds to wait before retrying (when rate limited)
func (rl *rateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decision, err := rl.AllowRequest(r)
		if err != nil {
			// Log the error if logger is available
			// For now, return generic error
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Set rate limit headers (always, even when allowed)
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", decision.Limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", decision.Remaining))

		// Calculate reset time (approximation based on refill rate)
		// For token bucket: time to fully refill = capacity / refill_rate
		if !decision.Allowed && decision.RetryAfter > 0 {
			resetTime := time.Now().Add(decision.RetryAfter).Unix()
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime))
			w.Header().Set("Retry-After", fmt.Sprintf("%.0f", decision.RetryAfter.Seconds()))

			// Return 429 Too Many Requests
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// Request allowed - proceed to next handler
		next.ServeHTTP(w, r)
	})
}

// StartBackgroundCleanup starts a goroutine that periodically cleans up idle buckets.
// Returns a function to stop the cleanup goroutine.
func (rl *rateLimiter) StartBackgroundCleanup() func() {
	// If store supports background cleanup, use it
	if inMemStore, ok := rl.store.(*InMemoryStore); ok {
		return inMemStore.StartBackgroundCleanup(rl.cleanupInterval)
	}

	// Return no-op function for stores that don't support cleanup
	return func() {}
}
