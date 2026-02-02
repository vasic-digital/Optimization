// Package prompt provides prompt optimization capabilities including
// compression, template management, and variable substitution.
package prompt

import (
	"context"
)

// Optimizer defines the interface for prompt optimization.
type Optimizer interface {
	// Optimize optimizes a prompt, reducing length while preserving meaning.
	Optimize(ctx context.Context, prompt string) (string, error)
}

// Config holds configuration for prompt optimization.
type Config struct {
	// MaxTokens is the maximum token count for the optimized prompt.
	MaxTokens int `json:"max_tokens"`
	// PreserveInstructions keeps system instructions intact during compression.
	PreserveInstructions bool `json:"preserve_instructions"`
	// RemoveRedundancy removes redundant phrases and sentences.
	RemoveRedundancy bool `json:"remove_redundancy"`
}

// DefaultConfig returns a default prompt optimization configuration.
func DefaultConfig() *Config {
	return &Config{
		MaxTokens:            4096,
		PreserveInstructions: true,
		RemoveRedundancy:     true,
	}
}
