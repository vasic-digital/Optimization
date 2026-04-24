package stress

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"digital.vasic.optimization/pkg/gptcache"
	"digital.vasic.optimization/pkg/prompt"
	"digital.vasic.optimization/pkg/streaming"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheConcurrentAccess_Stress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")  // SKIP-OK: #short-mode
	}

	cache := gptcache.NewInMemoryCache(
		gptcache.WithMaxEntries(1000),
		gptcache.WithTTL(10*time.Minute),
	)
	ctx := context.Background()

	const goroutines = 100
	var wg sync.WaitGroup

	// Concurrent writes
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("query-%d", idx)
			value := fmt.Sprintf("response-%d", idx)
			err := cache.Set(ctx, key, value)
			assert.NoError(t, err)
		}(i)
	}
	wg.Wait()

	// Concurrent reads
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("query-%d", idx)
			resp, err := cache.Get(ctx, key)
			if err == nil {
				assert.NotEmpty(t, resp.Response)
			}
		}(i)
	}
	wg.Wait()

	// Concurrent mixed operations
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("mixed-query-%d", idx)
			_ = cache.Set(ctx, key, "value")
			_, _ = cache.Get(ctx, key)
			_ = cache.Invalidate(ctx, key)
			_ = cache.Size()
		}(i)
	}
	wg.Wait()
}

func TestTemplateRegistryConcurrent_Stress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")  // SKIP-OK: #short-mode
	}

	registry := prompt.NewTemplateRegistry()
	const goroutines = 80
	var wg sync.WaitGroup

	// Register templates concurrently
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			tmpl := &prompt.Template{
				Name:    fmt.Sprintf("tmpl-%d", idx),
				Content: fmt.Sprintf("Hello {{name}}, template %d", idx),
			}
			err := registry.Register(tmpl)
			assert.NoError(t, err)
		}(i)
	}
	wg.Wait()

	assert.Equal(t, goroutines, registry.Size())

	// Concurrent reads and renders
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			name := fmt.Sprintf("tmpl-%d", idx)
			rendered, err := registry.RenderTemplate(name, map[string]string{
				"name": "World",
			})
			require.NoError(t, err)
			assert.Contains(t, rendered, "World")
		}(i)
	}
	wg.Wait()

	// Concurrent list and remove
	wg.Add(goroutines / 2)
	for i := 0; i < goroutines/2; i++ {
		go func(idx int) {
			defer wg.Done()
			_ = registry.List()
			if idx%2 == 0 {
				registry.Remove(fmt.Sprintf("tmpl-%d", idx))
			}
		}(i)
	}
	wg.Wait()
}

func TestStreamBufferConcurrent_Stress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")  // SKIP-OK: #short-mode
	}

	const goroutines = 50
	var wg sync.WaitGroup

	// Each goroutine creates its own buffer and processes data
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			buf := streaming.NewStreamBuffer(streaming.FlushOnWord, 5)
			counter := streaming.NewTokenCounter()
			merger := streaming.NewChunkMerger(3)

			text := fmt.Sprintf("Goroutine %d is processing text with multiple words. ", idx)
			for j := 0; j < 100; j++ {
				flushed := buf.Add(text)
				for _, f := range flushed {
					merged := merger.Add(f)
					if merged != "" {
						tokens := counter.Count(merged)
						assert.GreaterOrEqual(t, tokens, 0)
					}
				}
			}
			buf.Flush()
			merger.Flush()
		}(i)
	}
	wg.Wait()
}

func TestCacheHighThroughput_Stress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")  // SKIP-OK: #short-mode
	}

	cache := gptcache.NewInMemoryCache(
		gptcache.WithMaxEntries(5000),
		gptcache.WithTTL(5*time.Minute),
	)
	ctx := context.Background()

	const operations = 10000
	start := time.Now()

	for i := 0; i < operations; i++ {
		key := fmt.Sprintf("key-%d", i%500) // Reuse keys to test overwrites
		_ = cache.Set(ctx, key, fmt.Sprintf("value-%d", i))
		_, _ = cache.Get(ctx, key)
	}

	duration := time.Since(start)
	opsPerSec := float64(operations*2) / duration.Seconds()

	t.Logf("Completed %d operations in %v (%.0f ops/sec)", operations*2, duration, opsPerSec)
	assert.Greater(t, opsPerSec, 1000.0, "should handle at least 1000 ops/sec")
}

func TestSimilarityComputation_Stress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")  // SKIP-OK: #short-mode
	}

	const goroutines = 50
	var wg sync.WaitGroup

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			// Create random vectors
			size := 128
			vec1 := make([]float64, size)
			vec2 := make([]float64, size)
			for j := 0; j < size; j++ {
				vec1[j] = float64(j+idx) / float64(size)
				vec2[j] = float64(size-j+idx) / float64(size)
			}

			sim := gptcache.CosineSimilarity(vec1, vec2)
			assert.GreaterOrEqual(t, sim, -1.0)
			assert.LessOrEqual(t, sim, 1.0)

			normalized := gptcache.NormalizeL2(vec1)
			assert.Equal(t, len(vec1), len(normalized))
		}(i)
	}
	wg.Wait()
}
