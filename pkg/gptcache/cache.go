// Package gptcache provides semantic caching for LLM responses
// to reduce redundant API calls using embedding similarity.
package gptcache

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrCacheMiss indicates no matching entry was found.
	ErrCacheMiss = errors.New("cache miss")
	// ErrInvalidQuery indicates the query is invalid.
	ErrInvalidQuery = errors.New("invalid query")
)

// CachedResponse represents a cached LLM response.
type CachedResponse struct {
	// Response is the cached response content.
	Response string `json:"response"`
	// Similarity is the similarity score (0-1) for the match.
	Similarity float64 `json:"similarity"`
	// CachedAt is when the response was cached.
	CachedAt time.Time `json:"cached_at"`
	// TTL is the time-to-live for this entry.
	TTL time.Duration `json:"ttl"`
	// Metadata contains additional metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Cache defines the interface for semantic LLM response caching.
type Cache interface {
	// Get retrieves a cached response for the given query.
	// Returns ErrCacheMiss if no sufficiently similar entry is found.
	Get(ctx context.Context, query string) (*CachedResponse, error)
	// Set stores a query-response pair in the cache.
	Set(ctx context.Context, query string, response string) error
	// Invalidate removes entries matching the given query.
	Invalidate(ctx context.Context, query string) error
}

// SemanticMatcher computes similarity between queries using embeddings.
type SemanticMatcher interface {
	// Similarity computes the similarity score between two queries (0-1).
	Similarity(query1, query2 string) (float64, error)
}
