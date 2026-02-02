package gptcache

import (
	"time"
)

// Config holds configuration for the semantic cache.
type Config struct {
	// SimilarityThreshold is the minimum similarity score (0-1) for a cache hit.
	SimilarityThreshold float64 `json:"similarity_threshold"`
	// MaxEntries is the maximum number of entries to cache.
	MaxEntries int `json:"max_entries"`
	// TTL is the time-to-live for cache entries.
	TTL time.Duration `json:"ttl"`
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *Config {
	return &Config{
		SimilarityThreshold: 0.85,
		MaxEntries:          10000,
		TTL:                 24 * time.Hour,
	}
}

// Validate validates the configuration and applies defaults for invalid values.
func (c *Config) Validate() {
	if c.SimilarityThreshold < 0 || c.SimilarityThreshold > 1 {
		c.SimilarityThreshold = 0.85
	}
	if c.MaxEntries <= 0 {
		c.MaxEntries = 10000
	}
	if c.TTL <= 0 {
		c.TTL = 24 * time.Hour
	}
}

// ConfigOption is a functional option for configuring the cache.
type ConfigOption func(*Config)

// WithSimilarityThreshold sets the similarity threshold.
func WithSimilarityThreshold(threshold float64) ConfigOption {
	return func(c *Config) {
		c.SimilarityThreshold = threshold
	}
}

// WithMaxEntries sets the maximum number of entries.
func WithMaxEntries(n int) ConfigOption {
	return func(c *Config) {
		c.MaxEntries = n
	}
}

// WithTTL sets the time-to-live.
func WithTTL(ttl time.Duration) ConfigOption {
	return func(c *Config) {
		c.TTL = ttl
	}
}
