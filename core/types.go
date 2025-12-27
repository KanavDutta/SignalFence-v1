package core

import "time"

// Config defines the rate limiting policy
type Config struct {
	Capacity     float64 // Maximum tokens (burst size)
	RefillPerSec float64 // Tokens added per second
}

// BucketState represents the current state of a token bucket
type BucketState struct {
	Tokens       float64   // Current tokens available
	LastRefillAt time.Time // Last time tokens were refilled
}

// CheckResult contains the result of a rate limit check
type CheckResult struct {
	Allowed      bool          // Whether the request is allowed
	Remaining    float64       // Tokens remaining after this request
	RetryAfterMs int64         // Milliseconds until retry (if blocked)
	Limit        float64       // Total capacity
}
