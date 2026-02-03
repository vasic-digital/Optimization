package gptcache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// cacheEntry is an internal cache entry.
type cacheEntry struct {
	Query      string
	QueryHash  string
	Response   string
	CachedAt   time.Time
	AccessedAt time.Time
	Metadata   map[string]interface{}
}

// InMemoryCache implements Cache with in-memory storage and
// optional semantic matching.
type InMemoryCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry // hash -> entry
	order   []string               // insertion order for eviction
	config  *Config
	matcher SemanticMatcher
}

// NewInMemoryCache creates a new in-memory cache with the given options.
func NewInMemoryCache(opts ...ConfigOption) *InMemoryCache {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(config)
	}
	config.Validate()

	return &InMemoryCache{
		entries: make(map[string]*cacheEntry),
		order:   make([]string, 0),
		config:  config,
	}
}

// NewInMemoryCacheWithConfig creates a new in-memory cache with explicit config.
func NewInMemoryCacheWithConfig(config *Config) *InMemoryCache {
	if config == nil {
		config = DefaultConfig()
	}
	config.Validate()

	return &InMemoryCache{
		entries: make(map[string]*cacheEntry),
		order:   make([]string, 0),
		config:  config,
	}
}

// SetMatcher sets the semantic matcher for similarity-based lookups.
func (c *InMemoryCache) SetMatcher(matcher SemanticMatcher) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.matcher = matcher
}

// Get retrieves a cached response for the given query.
func (c *InMemoryCache) Get(
	ctx context.Context,
	query string,
) (*CachedResponse, error) {
	if query == "" {
		return nil, ErrInvalidQuery
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	// Try exact hash match first.
	hash := hashQuery(query)
	if entry, ok := c.entries[hash]; ok {
		if !c.isExpired(entry) {
			entry.AccessedAt = time.Now()
			return &CachedResponse{
				Response:   entry.Response,
				Similarity: 1.0,
				CachedAt:   entry.CachedAt,
				TTL:        c.config.TTL,
				Metadata:   entry.Metadata,
			}, nil
		}
	}

	// Try semantic matching if a matcher is configured.
	if c.matcher != nil {
		return c.findSemantic(query)
	}

	return nil, ErrCacheMiss
}

// Set stores a query-response pair in the cache.
func (c *InMemoryCache) Set(
	ctx context.Context,
	query string,
	response string,
) error {
	if query == "" {
		return ErrInvalidQuery
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	hash := hashQuery(query)
	now := time.Now()

	c.entries[hash] = &cacheEntry{
		Query:      query,
		QueryHash:  hash,
		Response:   response,
		CachedAt:   now,
		AccessedAt: now,
	}
	c.order = append(c.order, hash)

	// Evict oldest if over capacity.
	c.evictIfNeeded()

	return nil
}

// Invalidate removes entries matching the given query by exact hash.
func (c *InMemoryCache) Invalidate(
	ctx context.Context,
	query string,
) error {
	if query == "" {
		return ErrInvalidQuery
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	hash := hashQuery(query)
	delete(c.entries, hash)

	// Remove from order.
	for i, h := range c.order {
		if h == hash {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}

	return nil
}

// Size returns the current number of entries.
func (c *InMemoryCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Clear removes all entries from the cache.
func (c *InMemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*cacheEntry)
	c.order = make([]string, 0)
}

// Config returns the cache configuration.
func (c *InMemoryCache) Config() *Config {
	return c.config
}

func (c *InMemoryCache) findSemantic(query string) (*CachedResponse, error) {
	var bestEntry *cacheEntry
	var bestSimilarity float64

	for _, entry := range c.entries {
		if c.isExpired(entry) {
			continue
		}

		sim, err := c.matcher.Similarity(query, entry.Query)
		if err != nil {
			continue
		}

		if sim > bestSimilarity && sim >= c.config.SimilarityThreshold {
			bestSimilarity = sim
			bestEntry = entry
		}
	}

	if bestEntry == nil {
		return nil, ErrCacheMiss
	}

	bestEntry.AccessedAt = time.Now()
	return &CachedResponse{
		Response:   bestEntry.Response,
		Similarity: bestSimilarity,
		CachedAt:   bestEntry.CachedAt,
		TTL:        c.config.TTL,
		Metadata:   bestEntry.Metadata,
	}, nil
}

func (c *InMemoryCache) isExpired(entry *cacheEntry) bool {
	return time.Since(entry.CachedAt) > c.config.TTL
}

func (c *InMemoryCache) evictIfNeeded() {
	for len(c.entries) > c.config.MaxEntries && len(c.order) > 0 {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.entries, oldest)
	}
}

func hashQuery(query string) string {
	h := sha256.Sum256([]byte(query))
	return hex.EncodeToString(h[:])
}
