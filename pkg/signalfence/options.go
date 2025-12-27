package signalfence

import (
	"fmt"
	"time"
)

// Option is a functional option for configuring a RateLimiter.
type Option func(*rateLimiter) error

// WithStore sets a custom store for the rate limiter.
// If not provided, an in-memory store with default config will be used.
func WithStore(store Store) Option {
	return func(rl *rateLimiter) error {
		if store == nil {
			return fmt.Errorf("%w: store cannot be nil", ErrInvalidConfig)
		}
		rl.store = store
		return nil
	}
}

// WithConfig sets the configuration for the rate limiter.
func WithConfig(config *Config) Option {
	return func(rl *rateLimiter) error {
		if config == nil {
			return fmt.Errorf("%w: config cannot be nil", ErrInvalidConfig)
		}
		if err := config.Validate(); err != nil {
			return err
		}
		rl.config = config
		return nil
	}
}

// WithConfigFile loads configuration from a YAML file.
func WithConfigFile(path string) Option {
	return func(rl *rateLimiter) error {
		config, err := LoadConfigFromFile(path)
		if err != nil {
			return err
		}
		rl.config = config
		return nil
	}
}

// WithKeyExtractor sets a custom key extractor function.
func WithKeyExtractor(extractor KeyExtractor) Option {
	return func(rl *rateLimiter) error {
		if extractor == nil {
			return fmt.Errorf("%w: key extractor cannot be nil", ErrInvalidConfig)
		}
		rl.keyExtractor = extractor
		return nil
	}
}

// WithDefaults sets simple default rate limiting parameters.
// This is a convenience option for basic use cases.
func WithDefaults(capacity int64, refillRate float64) Option {
	return func(rl *rateLimiter) error {
		if capacity <= 0 {
			return ErrNegativeCapacity
		}
		if refillRate <= 0 {
			return ErrNegativeRefillRate
		}

		rl.config = &Config{
			Defaults: PolicyConfig{
				Capacity:   capacity,
				RefillRate: refillRate,
				Enabled:    true,
			},
			Policies:     make(map[string]PolicyConfig),
			KeyExtractor: "ip",
			CleanupAge:   "1h",
		}
		return nil
	}
}

// WithCleanupAge sets the age after which idle buckets are cleaned up.
// Examples: "1h", "30m", "0" (disabled)
func WithCleanupAge(age time.Duration) Option {
	return func(rl *rateLimiter) error {
		rl.cleanupAge = age
		return nil
	}
}

// WithCleanupInterval sets how often the cleanup goroutine runs.
// Only used when StartBackgroundCleanup is called.
// Default: 10 minutes
func WithCleanupInterval(interval time.Duration) Option {
	return func(rl *rateLimiter) error {
		if interval < 0 {
			return fmt.Errorf("%w: cleanup interval cannot be negative", ErrInvalidConfig)
		}
		rl.cleanupInterval = interval
		return nil
	}
}

// WithRouteExtractor sets a function to extract the route from a request.
// By default, r.URL.Path is used. This allows customization for path parameters, etc.
type RouteExtractorFunc func(path string) string

func WithRouteExtractor(fn RouteExtractorFunc) Option {
	return func(rl *rateLimiter) error {
		if fn == nil {
			return fmt.Errorf("%w: route extractor cannot be nil", ErrInvalidConfig)
		}
		rl.routeExtractor = fn
		return nil
	}
}
