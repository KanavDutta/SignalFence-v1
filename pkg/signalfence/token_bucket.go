package signalfence

import (
	"sync"
	"time"
)

// Bucket represents a token bucket for rate limiting a single client.
// It implements the token bucket algorithm with lazy refill.
type Bucket struct {
	capacity       int64      // Maximum number of tokens (burst size)
	tokens         float64    // Current available tokens
	refillRate     float64    // Tokens added per second
	lastRefillTime time.Time  // Last time tokens were refilled
	mu             sync.Mutex // Protects bucket state
}

// NewBucket creates a new token bucket with the specified capacity and refill rate.
// Capacity determines the maximum burst size (max tokens available at once).
// RefillRate determines how many tokens are added per second.
//
// Example: NewBucket(100, 10.0) creates a bucket that:
// - Allows bursts up to 100 requests
// - Refills at 10 tokens/second (600 requests/minute sustained)
func NewBucket(capacity int64, refillRate float64) (*Bucket, error) {
	if capacity <= 0 {
		return nil, ErrNegativeCapacity
	}
	if refillRate <= 0 {
		return nil, ErrNegativeRefillRate
	}

	return &Bucket{
		capacity:       capacity,
		tokens:         float64(capacity), // Start with full bucket
		refillRate:     refillRate,
		lastRefillTime: time.Now(),
	}, nil
}

// Allow attempts to consume one token from the bucket.
// Returns true if the request is allowed (token available), false otherwise.
// This method is thread-safe and can be called concurrently.
func (b *Bucket) Allow() bool {
	return b.AllowN(1)
}

// AllowN attempts to consume n tokens from the bucket.
// Returns true if the request is allowed (n tokens available), false otherwise.
// This method is thread-safe and can be called concurrently.
func (b *Bucket) AllowN(n int64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.refill()

	if b.tokens >= float64(n) {
		b.tokens -= float64(n)
		return true
	}

	return false
}

// refill adds tokens based on elapsed time since last refill.
// Uses lazy refill: tokens are only computed when needed.
// MUST be called with b.mu locked.
func (b *Bucket) refill() {
	now := time.Now()
	elapsed := now.Sub(b.lastRefillTime).Seconds()

	// Add tokens based on elapsed time
	tokensToAdd := elapsed * b.refillRate
	b.tokens += tokensToAdd

	// Cap at capacity (prevent overflow)
	if b.tokens > float64(b.capacity) {
		b.tokens = float64(b.capacity)
	}

	b.lastRefillTime = now
}

// Remaining returns the number of tokens currently available in the bucket.
// This is a snapshot and may change immediately due to concurrent access.
func (b *Bucket) Remaining() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.refill()
	return int64(b.tokens)
}

// Capacity returns the maximum capacity of the bucket.
func (b *Bucket) Capacity() int64 {
	return b.capacity
}

// RefillRate returns the refill rate (tokens per second).
func (b *Bucket) RefillRate() float64 {
	return b.refillRate
}

// RetryAfter calculates how long to wait before the next request would be allowed.
// Returns 0 if a request can be made immediately.
func (b *Bucket) RetryAfter() time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.refill()

	if b.tokens >= 1 {
		return 0
	}

	// Calculate time needed to refill 1 token
	tokensNeeded := 1.0 - b.tokens
	secondsNeeded := tokensNeeded / b.refillRate

	return time.Duration(secondsNeeded * float64(time.Second))
}
