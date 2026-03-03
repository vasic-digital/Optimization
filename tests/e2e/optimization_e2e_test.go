package e2e

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

func TestFullCachingPipeline_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	cache := gptcache.NewInMemoryCache(
		gptcache.WithMaxEntries(1000),
		gptcache.WithTTL(1*time.Hour),
	)

	ctx := context.Background()

	// Simulate a full caching workflow
	queries := []struct {
		query    string
		response string
	}{
		{"What is Go?", "Go is a programming language"},
		{"What is Rust?", "Rust is a systems programming language"},
		{"What is Python?", "Python is an interpreted language"},
	}

	// Store all
	for _, q := range queries {
		err := cache.Set(ctx, q.query, q.response)
		require.NoError(t, err)
	}
	assert.Equal(t, 3, cache.Size())

	// Retrieve all
	for _, q := range queries {
		resp, err := cache.Get(ctx, q.query)
		require.NoError(t, err)
		assert.Equal(t, q.response, resp.Response)
	}

	// Invalidate one
	err := cache.Invalidate(ctx, "What is Rust?")
	require.NoError(t, err)
	assert.Equal(t, 2, cache.Size())

	// Verify invalidated entry is gone
	_, err = cache.Get(ctx, "What is Rust?")
	assert.ErrorIs(t, err, gptcache.ErrCacheMiss)

	// Clear all
	cache.Clear()
	assert.Equal(t, 0, cache.Size())
}

func TestPromptOptimizationPipeline_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	// Step 1: Create and register templates
	registry := prompt.NewTemplateRegistry()
	templates := []*prompt.Template{
		{Name: "code-review", Content: "Review this {{language}} code: {{code}}"},
		{Name: "explain", Content: "Please explain {{concept}} in simple terms"},
		{Name: "translate", Content: "Translate '{{text}}' from {{source}} to {{target}}"},
	}

	for _, tmpl := range templates {
		err := registry.Register(tmpl)
		require.NoError(t, err)
	}
	assert.Equal(t, 3, registry.Size())

	// Step 2: Render a template
	rendered, err := registry.RenderTemplate("code-review", map[string]string{
		"language": "Go",
		"code":     "func main() { fmt.Println(\"hello\") }",
	})
	require.NoError(t, err)
	assert.Contains(t, rendered, "Go")

	// Step 3: Compress the rendered prompt
	compressor := prompt.NewCompressor(prompt.DefaultConfig())
	compressed, err := compressor.Optimize(context.Background(), rendered)
	require.NoError(t, err)
	assert.NotEmpty(t, compressed)

	// Step 4: Estimate tokens
	tokens := prompt.EstimateTokens(compressed)
	assert.Greater(t, tokens, 0)
}

func TestStreamingPipeline_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	// Simulate a streaming response pipeline
	buffer := streaming.NewStreamBuffer(streaming.FlushOnWord, 0)
	counter := streaming.NewTokenCounter()
	merger := streaming.NewChunkMerger(5)

	response := "The quick brown fox jumps over the lazy dog. " +
		"This is a test of the streaming optimization pipeline."

	// Feed character by character (simulating streaming)
	var totalFlushed string
	for _, ch := range response {
		flushed := buffer.Add(string(ch))
		for _, f := range flushed {
			merged := merger.Add(f)
			if merged != "" {
				totalFlushed += merged
			}
		}
	}

	// Flush remaining
	remaining := buffer.Flush()
	if remaining != "" {
		merged := merger.Add(remaining)
		if merged != "" {
			totalFlushed += merged
		}
	}
	if rem := merger.Flush(); rem != "" {
		totalFlushed += rem
	}

	// Verify all content was processed
	assert.NotEmpty(t, totalFlushed)
	tokenCount := counter.Count(totalFlushed)
	assert.Greater(t, tokenCount, 0)
	assert.True(t, counter.Fits(totalFlushed, 1000))
}

func TestSchemaConstrainingPipeline_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	// Build schema for expected API response
	schema := outlines.NewSchemaBuilder().
		Object().
		Property("status", outlines.StringSchema()).
		Property("code", outlines.IntegerSchema()).
		Property("data", outlines.NewSchemaBuilder().
			Object().
			Property("items", outlines.ArraySchema(outlines.StringSchema())).
			Build()).
		RequiredProps("status", "code").
		Build()

	// Test constrainer with various LLM-like outputs
	constrainer := outlines.NewJSONConstrainer()

	validOutput := `{"status": "ok", "code": 200, "data": {"items": ["a", "b"]}}`
	result, err := constrainer.Constrain(validOutput, schema)
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Test with markdown-wrapped JSON (common LLM output)
	wrappedOutput := "```json\n" + validOutput + "\n```"
	// Extract JSON should handle this
	result2, err := constrainer.Constrain(wrappedOutput, schema)
	require.NoError(t, err)
	assert.NotEmpty(t, result2)
}

func TestTokenCounting_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	counter := streaming.NewTokenCounter()

	texts := map[string]int{
		"":                    0,
		"hello":               1,
		"hello world":         2,
		"The quick brown fox": 5, // ~4 words * 1.3 rounded
	}

	for text, expectedWords := range texts {
		wordCount := counter.CountWords(text)
		assert.Equal(t, expectedWords, wordCount, "word count mismatch for %q", text)

		tokenCount := counter.Count(text)
		if text == "" {
			assert.Equal(t, 0, tokenCount)
		} else {
			assert.GreaterOrEqual(t, tokenCount, wordCount,
				fmt.Sprintf("tokens should be >= words for %q", text))
		}
	}
}
