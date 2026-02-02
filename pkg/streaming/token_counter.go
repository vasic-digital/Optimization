package streaming

import (
	"strings"
)

// TokenCounter estimates token counts for text.
type TokenCounter struct {
	// TokensPerWord is the average tokens per word (default ~1.3 for English).
	TokensPerWord float64
}

// NewTokenCounter creates a new token counter.
func NewTokenCounter() *TokenCounter {
	return &TokenCounter{
		TokensPerWord: 1.3,
	}
}

// NewTokenCounterWithRatio creates a token counter with a custom ratio.
func NewTokenCounterWithRatio(tokensPerWord float64) *TokenCounter {
	if tokensPerWord <= 0 {
		tokensPerWord = 1.3
	}
	return &TokenCounter{
		TokensPerWord: tokensPerWord,
	}
}

// Count estimates the token count for the given text.
func (c *TokenCounter) Count(text string) int {
	if text == "" {
		return 0
	}
	words := len(strings.Fields(text))
	return int(float64(words) * c.TokensPerWord)
}

// CountWords returns the exact word count.
func (c *TokenCounter) CountWords(text string) int {
	return len(strings.Fields(text))
}

// CountCharacters returns the character count.
func (c *TokenCounter) CountCharacters(text string) int {
	return len([]rune(text))
}

// Fits checks if the text fits within the given token limit.
func (c *TokenCounter) Fits(text string, limit int) bool {
	return c.Count(text) <= limit
}
