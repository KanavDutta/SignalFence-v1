package signalfence

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the rate limiting configuration.
// It supports both global defaults and per-route policy overrides.
type Config struct {
	// Defaults are applied to all routes unless overridden
	Defaults PolicyConfig `yaml:"defaults"`

	// Policies is a map of route paths to their specific rate limit policies
	// Example: "/api/login" -> strict policy, "/api/search" -> lenient policy
	Policies map[string]PolicyConfig `yaml:"policies,omitempty"`

	// KeyExtractor specifies how to identify clients
	// Examples: "ip", "header:X-API-Key", "header:Authorization"
	KeyExtractor string `yaml:"key_extractor,omitempty"`

	// CleanupAge specifies how long idle buckets are kept before cleanup
	// Format: "1h", "30m", "0" to disable
	CleanupAge string `yaml:"cleanup_age,omitempty"`
}

// PolicyConfig defines rate limiting parameters for a route or default.
type PolicyConfig struct {
	// Capacity is the maximum number of tokens (burst size)
	Capacity int64 `yaml:"capacity"`

	// RefillRate is the number of tokens added per second
	// Example: 10.0 = 10 tokens/sec = 600 requests/minute
	RefillRate float64 `yaml:"refill_rate"`

	// Enabled allows disabling rate limiting for specific routes
	Enabled bool `yaml:"enabled"`
}

// NewConfig creates a new Config with sensible defaults.
func NewConfig() *Config {
	return &Config{
		Defaults: PolicyConfig{
			Capacity:   100,
			RefillRate: 10.0, // 600 req/min
			Enabled:    true,
		},
		Policies:     make(map[string]PolicyConfig),
		KeyExtractor: "ip", // Default to IP-based rate limiting
		CleanupAge:   "1h", // Clean up buckets idle for > 1 hour
	}
}

// LoadConfigFromFile loads configuration from a YAML file.
func LoadConfigFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read config file: %v", ErrInvalidConfig, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("%w: failed to parse YAML: %v", ErrInvalidConfig, err)
	}

	// Apply defaults if not set
	if config.KeyExtractor == "" {
		config.KeyExtractor = "ip"
	}
	if config.CleanupAge == "" {
		config.CleanupAge = "1h"
	}
	if config.Policies == nil {
		config.Policies = make(map[string]PolicyConfig)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	// Validate defaults
	if err := c.Defaults.Validate(); err != nil {
		return fmt.Errorf("%w: invalid defaults: %v", ErrInvalidConfig, err)
	}

	// Validate per-route policies
	for route, policy := range c.Policies {
		if err := policy.Validate(); err != nil {
			return fmt.Errorf("%w: invalid policy for route %s: %v", ErrInvalidConfig, route, err)
		}
	}

	return nil
}

// Validate checks if a PolicyConfig is valid.
func (p *PolicyConfig) Validate() error {
	if p.Capacity <= 0 {
		return ErrNegativeCapacity
	}
	if p.RefillRate <= 0 {
		return ErrNegativeRefillRate
	}
	return nil
}

// GetPolicy returns the rate limit policy for a given route.
// If no specific policy exists for the route, returns the default policy.
func (c *Config) GetPolicy(route string) PolicyConfig {
	if policy, exists := c.Policies[route]; exists {
		return policy
	}
	return c.Defaults
}

// SetPolicy sets a rate limit policy for a specific route.
func (c *Config) SetPolicy(route string, policy PolicyConfig) error {
	if err := policy.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	if c.Policies == nil {
		c.Policies = make(map[string]PolicyConfig)
	}
	c.Policies[route] = policy
	return nil
}

// ToBucketConfig converts a PolicyConfig to a BucketConfig for store initialization.
func (p *PolicyConfig) ToBucketConfig() BucketConfig {
	return BucketConfig{
		Capacity:   p.Capacity,
		RefillRate: p.RefillRate,
	}
}
