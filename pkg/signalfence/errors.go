package signalfence

import "errors"

var (
	// ErrInvalidConfig is returned when configuration is invalid
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrNegativeCapacity is returned when bucket capacity is negative
	ErrNegativeCapacity = errors.New("bucket capacity must be positive")

	// ErrNegativeRefillRate is returned when refill rate is negative
	ErrNegativeRefillRate = errors.New("refill rate must be positive")

	// ErrInvalidKey is returned when the rate limit key is invalid or empty
	ErrInvalidKey = errors.New("rate limit key cannot be empty")

	// ErrStoreFailed is returned when store operations fail
	ErrStoreFailed = errors.New("store operation failed")

	// ErrKeyExtractionFailed is returned when key extraction from request fails
	ErrKeyExtractionFailed = errors.New("failed to extract key from request")
)
