package signalfence

import (
	"github.com/yourusername/signalfence/middleware"
)

// Re-export main types for convenience
type (
	Config      = middleware.Config
	RateLimiter = middleware.RateLimiter
	KeyFunc     = middleware.KeyFunc
)

// NewRateLimiter creates a new rate limiter
var NewRateLimiter = middleware.NewRateLimiter
