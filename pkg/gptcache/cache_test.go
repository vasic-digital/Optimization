package gptcache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryCache_Get_ExactMatch(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		response string
		lookup   string
		wantHit  bool
	}{
		{
			name:     "exact match returns cached response",
			query:    "What is Go?",
			response: "Go is a programming language.",
			lookup:   "What is Go?",
			wantHit:  true,
		},
		{
			name:     "different query returns cache miss",
			query:    "What is Go?",
			response: "Go is a programming language.",
			lookup:   "What is Rust?",
			wantHit:  false,
		},
		{
			name:     "empty cache returns miss",
			query:    "",
			response: "",
			lookup:   "anything",
			wantHit:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewInMemoryCache()
			ctx := context.Background()

			if tt.query != "" {
				err := cache.Set(ctx, tt.query, tt.response)
				require.NoError(t, err)
			}

			result, err := cache.Get(ctx, tt.lookup)
			if tt.wantHit {
				require.NoError(t, err)
				assert.Equal(t, tt.response, result.Response)
				assert.Equal(t, 1.0, result.Similarity)
				assert.False(t, result.CachedAt.IsZero())
			} else {
				assert.ErrorIs(t, err, ErrCacheMiss)
				assert.Nil(t, result)
			}
		})
	}
}

func TestInMemoryCache_Get_EmptyQuery(t *testing.T) {
	cache := NewInMemoryCache()
	ctx := context.Background()

	result, err := cache.Get(ctx, "")
	assert.ErrorIs(t, err, ErrInvalidQuery)
	assert.Nil(t, result)
}

func TestInMemoryCache_Set_EmptyQuery(t *testing.T) {
	cache := NewInMemoryCache()
	ctx := context.Background()

	err := cache.Set(ctx, "", "response")
	assert.ErrorIs(t, err, ErrInvalidQuery)
}

func TestInMemoryCache_Set_Eviction(t *testing.T) {
	cache := NewInMemoryCache(WithMaxEntries(3))
	ctx := context.Background()

	require.NoError(t, cache.Set(ctx, "q1", "r1"))
	require.NoError(t, cache.Set(ctx, "q2", "r2"))
	require.NoError(t, cache.Set(ctx, "q3", "r3"))
	assert.Equal(t, 3, cache.Size())

	// Adding a 4th should evict the oldest.
	require.NoError(t, cache.Set(ctx, "q4", "r4"))
	assert.Equal(t, 3, cache.Size())

	// q1 should be evicted.
	_, err := cache.Get(ctx, "q1")
	assert.ErrorIs(t, err, ErrCacheMiss)

	// q4 should exist.
	result, err := cache.Get(ctx, "q4")
	require.NoError(t, err)
	assert.Equal(t, "r4", result.Response)
}

func TestInMemoryCache_Invalidate(t *testing.T) {
	cache := NewInMemoryCache()
	ctx := context.Background()

	require.NoError(t, cache.Set(ctx, "query", "response"))

	result, err := cache.Get(ctx, "query")
	require.NoError(t, err)
	assert.Equal(t, "response", result.Response)

	require.NoError(t, cache.Invalidate(ctx, "query"))

	_, err = cache.Get(ctx, "query")
	assert.ErrorIs(t, err, ErrCacheMiss)
	assert.Equal(t, 0, cache.Size())
}

func TestInMemoryCache_Invalidate_EmptyQuery(t *testing.T) {
	cache := NewInMemoryCache()
	ctx := context.Background()

	err := cache.Invalidate(ctx, "")
	assert.ErrorIs(t, err, ErrInvalidQuery)
}

func TestInMemoryCache_Clear(t *testing.T) {
	cache := NewInMemoryCache()
	ctx := context.Background()

	require.NoError(t, cache.Set(ctx, "q1", "r1"))
	require.NoError(t, cache.Set(ctx, "q2", "r2"))
	assert.Equal(t, 2, cache.Size())

	cache.Clear()
	assert.Equal(t, 0, cache.Size())
}

func TestInMemoryCache_TTL_Expiry(t *testing.T) {
	cache := NewInMemoryCache(WithTTL(50 * time.Millisecond))
	ctx := context.Background()

	require.NoError(t, cache.Set(ctx, "query", "response"))

	// Should be available immediately.
	result, err := cache.Get(ctx, "query")
	require.NoError(t, err)
	assert.Equal(t, "response", result.Response)

	// Wait for expiry.
	time.Sleep(60 * time.Millisecond)

	_, err = cache.Get(ctx, "query")
	assert.ErrorIs(t, err, ErrCacheMiss)
}

func TestInMemoryCache_SemanticMatcher(t *testing.T) {
	cache := NewInMemoryCache(WithSimilarityThreshold(0.5))

	// Set up a matcher that returns high similarity for similar prefixes.
	cache.SetMatcher(&EmbeddingMatcher{
		EmbedFunc: func(query string) ([]float64, error) {
			// Simple embedding: first 3 chars as float64 values.
			emb := make([]float64, 3)
			for i := 0; i < 3 && i < len(query); i++ {
				emb[i] = float64(query[i])
			}
			return emb, nil
		},
	})

	ctx := context.Background()

	require.NoError(t, cache.Set(ctx, "What is Go?", "Go is great."))

	// Same prefix should match semantically.
	result, err := cache.Get(ctx, "What is Go programming?")
	require.NoError(t, err)
	assert.Equal(t, "Go is great.", result.Response)
	assert.Greater(t, result.Similarity, 0.5)
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		wantThreshold float64
		wantEntries   int
	}{
		{
			name: "valid config unchanged",
			config: &Config{
				SimilarityThreshold: 0.9,
				MaxEntries:          500,
				TTL:                 time.Hour,
			},
			wantThreshold: 0.9,
			wantEntries:   500,
		},
		{
			name: "invalid threshold gets default",
			config: &Config{
				SimilarityThreshold: -1,
				MaxEntries:          100,
				TTL:                 time.Hour,
			},
			wantThreshold: 0.85,
			wantEntries:   100,
		},
		{
			name: "zero max entries gets default",
			config: &Config{
				SimilarityThreshold: 0.8,
				MaxEntries:          0,
				TTL:                 time.Hour,
			},
			wantThreshold: 0.8,
			wantEntries:   10000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.Validate()
			assert.Equal(t, tt.wantThreshold, tt.config.SimilarityThreshold)
			assert.Equal(t, tt.wantEntries, tt.config.MaxEntries)
		})
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		vec1     []float64
		vec2     []float64
		expected float64
	}{
		{
			name:     "identical vectors",
			vec1:     []float64{1, 0, 0},
			vec2:     []float64{1, 0, 0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			vec1:     []float64{1, 0, 0},
			vec2:     []float64{0, 1, 0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			vec1:     []float64{1, 0, 0},
			vec2:     []float64{-1, 0, 0},
			expected: -1.0,
		},
		{
			name:     "empty vectors",
			vec1:     []float64{},
			vec2:     []float64{},
			expected: 0.0,
		},
		{
			name:     "different lengths",
			vec1:     []float64{1, 2},
			vec2:     []float64{1, 2, 3},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CosineSimilarity(tt.vec1, tt.vec2)
			assert.InDelta(t, tt.expected, result, 1e-10)
		})
	}
}

func TestNormalizeL2(t *testing.T) {
	tests := []struct {
		name string
		vec  []float64
	}{
		{
			name: "unit vector unchanged",
			vec:  []float64{1, 0, 0},
		},
		{
			name: "normalizes non-unit vector",
			vec:  []float64{3, 4, 0},
		},
		{
			name: "empty vector",
			vec:  []float64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeL2(tt.vec)
			if len(tt.vec) > 0 {
				// Compute L2 norm of result.
				var norm float64
				for _, v := range result {
					norm += v * v
				}
				if norm > 0 {
					assert.InDelta(t, 1.0, norm, 1e-10)
				}
			}
		})
	}
}

func TestInMemoryCache_ImplementsCacheInterface(t *testing.T) {
	var _ Cache = (*InMemoryCache)(nil)
}
