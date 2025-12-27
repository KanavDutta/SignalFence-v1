package core

import (
	"math"
	"time"
)

// TokenBucket implements the token bucket rate limiting algorithm
type TokenBucket struct {
	config Config
}

// NewTokenBucket creates a new token bucket with the given configuration
func NewTokenBucket(config Config) *TokenBucket {
	return &TokenBucket{config: config}
}

// Check determines if a request should be allowed based on the current bucket state
// It returns the updated state and check result
func (tb *TokenBucket) Check(state *BucketState, now time.Time) (*BucketState, CheckResult) {
	// Initialize new bucket if needed
	if state == nil {
		state = &BucketState{
			Tokens:       tb.config.Capacity,
			LastRefillAt: now,
		}
	}

	// Calculate tokens to add based on time elapsed
	elapsed := now.Sub(state.LastRefillAt).Seconds()
	tokensToAdd := elapsed * tb.config.RefillPerSec
	
	// Refill tokens (capped at capacity)
	newTokens := math.Min(state.Tokens+tokensToAdd, tb.config.Capacity)
	
	// Create new state with refilled tokens
	newState := &BucketState{
		Tokens:       newTokens,
		LastRefillAt: now,
	}

	// Check if request can be allowed (need at least 1 token)
	if newState.Tokens >= 1.0 {
		// Allow request and consume 1 token
		newState.Tokens -= 1.0
		return newState, CheckResult{
			Allowed:      true,
			Remaining:    newState.Tokens,
			RetryAfterMs: 0,
			Limit:        tb.config.Capacity,
		}
	}

	// Request blocked - calculate retry time
	tokensNeeded := 1.0 - newState.Tokens
	retryAfterSec := tokensNeeded / tb.config.RefillPerSec
	retryAfterMs := int64(math.Ceil(retryAfterSec * 1000))

	return newState, CheckResult{
		Allowed:      false,
		Remaining:    0,
		RetryAfterMs: retryAfterMs,
		Limit:        tb.config.Capacity,
	}
}
