package benchmark

import (
	"context"
	"fmt"
	"testing"
	"time"

	"digital.vasic.optimization/pkg/gptcache"
	"digital.vasic.optimization/pkg/outlines"
	"digital.vasic.optimization/pkg/prompt"
	"digital.vasic.optimization/pkg/streaming"
)

func BenchmarkCacheSet(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	cache := gptcache.NewInMemoryCache(
		gptcache.WithMaxEntries(100000),
		gptcache.WithTTL(1*time.Hour),
	)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("query-%d", i)
		_ = cache.Set(ctx, key, "response")
	}
}

func BenchmarkCacheGet(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	cache := gptcache.NewInMemoryCache(
		gptcache.WithMaxEntries(100000),
		gptcache.WithTTL(1*time.Hour),
	)
	ctx := context.Background()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		_ = cache.Set(ctx, fmt.Sprintf("query-%d", i), "response")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("query-%d", i%1000)
		_, _ = cache.Get(ctx, key)
	}
}

func BenchmarkCosineSimilarity(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	vec1 := make([]float64, 768)
	vec2 := make([]float64, 768)
	for i := range vec1 {
		vec1[i] = float64(i) / 768.0
		vec2[i] = float64(768-i) / 768.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gptcache.CosineSimilarity(vec1, vec2)
	}
}

func BenchmarkNormalizeL2(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	vec := make([]float64, 768)
	for i := range vec {
		vec[i] = float64(i) / 768.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gptcache.NormalizeL2(vec)
	}
}

func BenchmarkPromptCompression(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	compressor := prompt.NewCompressor(prompt.DefaultConfig())
	ctx := context.Background()
	text := "Please note that this is a test. In order to understand " +
		"the system, basically we need to review the documentation. " +
		"It is important to note that performance matters."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compressor.Optimize(ctx, text)
	}
}

func BenchmarkTemplateRender(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	tmpl := &prompt.Template{
		Name:    "bench",
		Content: "Hello {{name}}, you are working on {{project}} version {{version}}.",
	}
	vars := map[string]string{
		"name":    "Alice",
		"project": "HelixAgent",
		"version": "1.0.0",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tmpl.Render(vars)
	}
}

func BenchmarkStreamBufferWordFlush(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := streaming.NewStreamBuffer(streaming.FlushOnWord, 5)
		buf.Add("The quick brown fox jumps over the lazy dog. ")
		buf.Flush()
	}
}

func BenchmarkTokenCounter(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	counter := streaming.NewTokenCounter()
	text := "The quick brown fox jumps over the lazy dog repeatedly in this benchmark test."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		counter.Count(text)
	}
}

func BenchmarkSchemaValidation(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	schema := outlines.ObjectSchema(
		map[string]*outlines.Schema{
			"name":  outlines.StringSchema(),
			"age":   outlines.IntegerSchema(),
			"email": outlines.StringSchema(),
		},
		"name", "age",
	)
	jsonStr := `{"name": "Alice", "age": 30, "email": "alice@example.com"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		outlines.Validate(jsonStr, schema)
	}
}
