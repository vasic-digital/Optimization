package prompt

import (
	"context"
	"strings"
	"unicode"
)

// Compressor reduces prompt length while preserving meaning.
type Compressor struct {
	config *Config
}

// NewCompressor creates a new prompt compressor.
func NewCompressor(config *Config) *Compressor {
	if config == nil {
		config = DefaultConfig()
	}
	return &Compressor{config: config}
}

// Optimize compresses a prompt by removing redundancy and trimming.
func (c *Compressor) Optimize(
	ctx context.Context,
	prompt string,
) (string, error) {
	if prompt == "" {
		return "", nil
	}

	result := prompt

	// Remove excessive whitespace.
	result = normalizeWhitespace(result)

	// Remove redundant phrases if configured.
	if c.config.RemoveRedundancy {
		result = removeRedundantPhrases(result)
	}

	// Truncate to max tokens if configured.
	if c.config.MaxTokens > 0 {
		result = truncateToTokens(result, c.config.MaxTokens)
	}

	return result, nil
}

// normalizeWhitespace collapses multiple spaces and trims.
func normalizeWhitespace(s string) string {
	var b strings.Builder
	prevSpace := false

	for _, r := range strings.TrimSpace(s) {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteRune(' ')
			}
			prevSpace = true
		} else {
			b.WriteRune(r)
			prevSpace = false
		}
	}

	return b.String()
}

// removeRedundantPhrases removes common filler phrases.
var redundantPhrases = []string{
	"please note that",
	"it is important to note that",
	"as mentioned earlier",
	"in other words",
	"to put it simply",
	"basically",
	"essentially",
	"in order to",
}

func removeRedundantPhrases(s string) string {
	lower := strings.ToLower(s)
	result := s
	for _, phrase := range redundantPhrases {
		idx := strings.Index(strings.ToLower(result), phrase)
		for idx != -1 {
			// Remove the phrase, preserving case of surrounding text.
			result = result[:idx] + result[idx+len(phrase):]
			lower = strings.ToLower(result)
			idx = strings.Index(lower, phrase)
		}
	}
	// Normalize whitespace after removal.
	return normalizeWhitespace(result)
}

// truncateToTokens truncates text to approximately maxTokens tokens.
// Uses word count as a simple token approximation.
func truncateToTokens(s string, maxTokens int) string {
	words := strings.Fields(s)
	if len(words) <= maxTokens {
		return s
	}
	return strings.Join(words[:maxTokens], " ")
}

// EstimateTokens estimates the token count for a string.
// Uses word count as a simple approximation.
func EstimateTokens(s string) int {
	return len(strings.Fields(s))
}
