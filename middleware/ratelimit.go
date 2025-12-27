package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/yourusername/signalfence/core"
	"github.com/yourusername/signalfence/store"
)

// KeyFunc extracts a unique identifier from the request
type KeyFunc func(*http.Request) string

// RateLimiter provides HTTP middleware for rate limiting
type RateLimiter struct {
	bucket  *core.TokenBucket
	store   store.Store
	keyFunc KeyFunc
}

// Config for creating a rate limiter
type Config struct {
	Capacity     float64     // Maximum tokens (burst size)
	RefillPerSec float64     // Tokens added per second
	KeyFunc      KeyFunc     // Optional: custom key extraction
	Store        store.Store // Optional: custom store (defaults to in-memory)
}

// NewRateLimiter creates a new rate limiting middleware
func NewRateLimiter(config Config) *RateLimiter {
	if config.KeyFunc == nil {
		config.KeyFunc = defaultKeyFunc
	}

	if config.Store == nil {
		config.Store = store.NewMemoryStore()
	}

	return &RateLimiter{
		bucket: core.NewTokenBucket(core.Config{
			Capacity:     config.Capacity,
			RefillPerSec: config.RefillPerSec,
		}),
		store:   config.Store,
		keyFunc: config.KeyFunc,
	}
}

// defaultKeyFunc extracts client identifier from IP address
func defaultKeyFunc(r *http.Request) string {
	// Try X-Forwarded-For first (for proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	
	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	// Remove port if present
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// Middleware wraps an http.Handler with rate limiting
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract client key
		key := rl.keyFunc(r)
		
		// Get current state
		state := rl.store.Get(key)
		
		// Check rate limit
		newState, result := rl.bucket.Check(state, time.Now())
		
		// Update state
		rl.store.Set(key, newState)
		
		// Add rate limit headers
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%.0f", result.Limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%.0f", result.Remaining))
		
		if !result.Allowed {
			// Request blocked
			retryAfterSec := result.RetryAfterMs / 1000
			if retryAfterSec == 0 {
				retryAfterSec = 1
			}
			w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfterSec))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":        "rate_limit_exceeded",
				"message":      "Too many requests. Please try again later.",
				"retryAfterMs": result.RetryAfterMs,
			})
			return
		}
		
		// Request allowed
		next.ServeHTTP(w, r)
	})
}
