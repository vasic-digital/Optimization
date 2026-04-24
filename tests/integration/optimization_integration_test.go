package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"digital.vasic.optimization/pkg/gptcache"
	"digital.vasic.optimization/pkg/outlines"
	"digital.vasic.optimization/pkg/prompt"
	"digital.vasic.optimization/pkg/streaming"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheWithSemanticMatcher_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")  // SKIP-OK: #short-mode
	}

	cache := gptcache.NewInMemoryCache(
		gptcache.WithSimilarityThreshold(0.9),
		gptcache.WithMaxEntries(100),
		gptcache.WithTTL(10*time.Minute),
	)

	matcher := &gptcache.EmbeddingMatcher{
		EmbedFunc: nil, // Uses exact match fallback
	}
	cache.SetMatcher(matcher)

	ctx := context.Background()

	// Set a response
	err := cache.Set(ctx, "What is the capital of France?", "Paris")
	require.NoError(t, err)

	// Exact match retrieval
	resp, err := cache.Get(ctx, "What is the capital of France?")
	require.NoError(t, err)
	assert.Equal(t, "Paris", resp.Response)
	assert.Equal(t, 1.0, resp.Similarity)

	// Cache miss for different query
	_, err = cache.Get(ctx, "What is the capital of Germany?")
	assert.ErrorIs(t, err, gptcache.ErrCacheMiss)
}

func TestCacheEviction_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")  // SKIP-OK: #short-mode
	}

	cache := gptcache.NewInMemoryCache(
		gptcache.WithMaxEntries(3),
	)
	ctx := context.Background()

	// Fill beyond capacity
	for i := 0; i < 5; i++ {
		err := cache.Set(ctx, fmt.Sprintf("query-%d", i), fmt.Sprintf("response-%d", i))
		require.NoError(t, err)
	}

	// Should have evicted oldest entries
	assert.LessOrEqual(t, cache.Size(), 3)
}

func TestPromptCompressorWithTemplates_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")  // SKIP-OK: #short-mode
	}

	// Create template registry
	registry := prompt.NewTemplateRegistry()
	tmpl := &prompt.Template{
		Name:      "summarize",
		Content:   "Please note that {{topic}} is important. In order to understand it, basically we need to study {{subject}}.",
		Variables: []string{"topic", "subject"},
	}
	err := registry.Register(tmpl)
	require.NoError(t, err)

	// Render template
	rendered, err := registry.RenderTemplate("summarize", map[string]string{
		"topic":   "machine learning",
		"subject": "neural networks",
	})
	require.NoError(t, err)

	// Compress the rendered prompt
	compressor := prompt.NewCompressor(&prompt.Config{
		MaxTokens:            4096,
		PreserveInstructions: true,
		RemoveRedundancy:     true,
	})

	compressed, err := compressor.Optimize(context.Background(), rendered)
	require.NoError(t, err)

	// Compressed should be shorter (redundant phrases removed)
	assert.Less(t, len(compressed), len(rendered))
	// Key content should be preserved
	assert.Contains(t, compressed, "machine learning")
	assert.Contains(t, compressed, "neural networks")
}

func TestStreamBufferWithChunkMerger_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")  // SKIP-OK: #short-mode
	}

	// Create a stream buffer with sentence flushing
	buf := streaming.NewStreamBuffer(streaming.FlushOnSentence, 0)

	// Feed text incrementally
	var allFlushed []string
	chunks := []string{"Hello ", "world. ", "This is ", "a test. ", "End."}
	for _, chunk := range chunks {
		flushed := buf.Add(chunk)
		allFlushed = append(allFlushed, flushed...)
	}

	// Get remaining
	remaining := buf.Flush()
	if remaining != "" {
		allFlushed = append(allFlushed, remaining)
	}

	// Should have split on sentence boundaries
	assert.NotEmpty(t, allFlushed)

	// Merge small chunks
	merger := streaming.NewChunkMerger(3)
	var merged []string
	for _, piece := range allFlushed {
		result := merger.Add(piece)
		if result != "" {
			merged = append(merged, result)
		}
	}
	if rem := merger.Flush(); rem != "" {
		merged = append(merged, rem)
	}
	assert.NotEmpty(t, merged)
}

func TestSchemaValidation_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")  // SKIP-OK: #short-mode
	}

	// Build a complex schema
	schema := outlines.NewSchemaBuilder().
		Object().
		Property("name", outlines.StringSchema()).
		Property("age", outlines.IntegerSchema()).
		Property("tags", outlines.ArraySchema(outlines.StringSchema())).
		RequiredProps("name", "age").
		Build()

	// Validate valid JSON
	validJSON := `{"name": "Alice", "age": 30, "tags": ["dev", "go"]}`
	result := outlines.Validate(validJSON, schema)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)

	// Validate invalid JSON (missing required field)
	invalidJSON := `{"name": "Bob"}`
	result = outlines.Validate(invalidJSON, schema)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
}

func TestJSONConstrainer_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")  // SKIP-OK: #short-mode
	}

	constrainer := outlines.NewJSONConstrainer()
	schema := outlines.ObjectSchema(
		map[string]*outlines.Schema{
			"result": outlines.StringSchema(),
		},
		"result",
	)

	// Valid JSON embedded in text
	output := `Here is the result: {"result": "success"} as you can see.`
	constrained, err := constrainer.Constrain(output, schema)
	require.NoError(t, err)
	assert.Contains(t, constrained, "success")
}
