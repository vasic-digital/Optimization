# User Guide - Optimization Module

## Overview

The `digital.vasic.optimization` module provides reusable LLM optimization
capabilities for Go applications. It includes semantic caching, prompt
optimization, streaming enhancements, structured output constraints, SGLang
integration, and LLM framework adapters.

## Installation

```bash
go get digital.vasic.optimization
```

**Requirements**: Go 1.24+

---

## 1. GPT-Cache: Semantic Caching

The `gptcache` package provides semantic caching for LLM responses, reducing
redundant API calls by matching similar queries using embedding similarity.

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "time"

    "digital.vasic.optimization/pkg/gptcache"
)

func main() {
    ctx := context.Background()

    // Create a cache with functional options.
    cache := gptcache.NewInMemoryCache(
        gptcache.WithSimilarityThreshold(0.85),
        gptcache.WithMaxEntries(10000),
        gptcache.WithTTL(24 * time.Hour),
    )

    // Store an LLM response.
    err := cache.Set(ctx, "What is Go?", "Go is a compiled programming language.")
    if err != nil {
        panic(err)
    }

    // Retrieve a cached response (exact match).
    result, err := cache.Get(ctx, "What is Go?")
    if err != nil {
        panic(err)
    }
    fmt.Printf("Response: %s (similarity: %.2f)\n", result.Response, result.Similarity)
    // Output: Response: Go is a compiled programming language. (similarity: 1.00)
}
```

### Semantic Matching with Embeddings

To enable fuzzy matching based on semantic similarity, provide an
`EmbeddingMatcher` with a custom embedding function:

```go
cache := gptcache.NewInMemoryCache(
    gptcache.WithSimilarityThreshold(0.80),
)

// Set an embedding function (integrate with your embedding provider).
cache.SetMatcher(&gptcache.EmbeddingMatcher{
    EmbedFunc: func(query string) ([]float64, error) {
        // Call your embedding API (OpenAI, Cohere, Voyage, etc.)
        return myEmbeddingProvider.Embed(query)
    },
})

ctx := context.Background()
_ = cache.Set(ctx, "What is Go?", "Go is a programming language by Google.")

// This query is semantically similar and will match if similarity >= 0.80.
result, err := cache.Get(ctx, "Tell me about the Go programming language")
if err == nil {
    fmt.Printf("Cache hit! Similarity: %.2f\n", result.Similarity)
}
```

### Using Config Directly

```go
config := &gptcache.Config{
    SimilarityThreshold: 0.9,
    MaxEntries:          5000,
    TTL:                 12 * time.Hour,
}
config.Validate() // Applies defaults for any invalid values.

cache := gptcache.NewInMemoryCacheWithConfig(config)
```

### Similarity Utilities

```go
// Compute cosine similarity between two vectors.
sim := gptcache.CosineSimilarity(
    []float64{1.0, 0.5, 0.3},
    []float64{0.9, 0.6, 0.2},
)
fmt.Printf("Cosine similarity: %.4f\n", sim)

// Normalize a vector to unit length.
normalized := gptcache.NormalizeL2([]float64{3.0, 4.0, 0.0})
// Result: [0.6, 0.8, 0.0]
```

### Cache Management

```go
cache := gptcache.NewInMemoryCache()
ctx := context.Background()

_ = cache.Set(ctx, "q1", "r1")
_ = cache.Set(ctx, "q2", "r2")

fmt.Println(cache.Size()) // 2

_ = cache.Invalidate(ctx, "q1")
fmt.Println(cache.Size()) // 1

cache.Clear()
fmt.Println(cache.Size()) // 0
```

---

## 2. Outlines: Structured Output

The `outlines` package enforces structured output from LLMs using JSON Schema
validation and regex pattern matching.

### Building Schemas

```go
import "digital.vasic.optimization/pkg/outlines"

// Build a schema using the fluent builder.
schema := outlines.NewSchemaBuilder().
    Object().
    Property("name", outlines.StringSchema()).
    Property("age", outlines.IntegerSchema()).
    Property("active", outlines.BooleanSchema()).
    RequiredProps("name", "age").
    SetDescription("A user profile").
    Build()

fmt.Println(schema.String()) // Pretty-printed JSON Schema
```

### Helper Schema Constructors

```go
// Quick schema constructors for common types.
strSchema := outlines.StringSchema()
intSchema := outlines.IntegerSchema()
numSchema := outlines.NumberSchema()
boolSchema := outlines.BooleanSchema()
arrSchema := outlines.ArraySchema(outlines.StringSchema())
objSchema := outlines.ObjectSchema(
    map[string]*outlines.Schema{
        "name": outlines.StringSchema(),
    },
    "name", // required properties
)
```

### JSON Constraining

```go
constrainer := outlines.NewJSONConstrainer()

schema := outlines.ObjectSchema(
    map[string]*outlines.Schema{
        "name": outlines.StringSchema(),
        "age":  outlines.IntegerSchema(),
    },
    "name",
)

// Constrain LLM output -- extracts and validates JSON.
llmOutput := `Here is the result: {"name": "Alice", "age": 30}`
constrained, err := constrainer.Constrain(llmOutput, schema)
if err != nil {
    fmt.Println("Validation error:", err)
} else {
    fmt.Println(constrained) // {"name": "Alice", "age": 30}
}
```

### Regex Constraining

```go
constrainer, err := outlines.NewRegexConstrainer(`^\d{3}-\d{4}$`)
if err != nil {
    panic(err)
}

result, err := constrainer.Constrain("123-4567", nil)
fmt.Println(result) // 123-4567
```

### Direct Validation

```go
result := outlines.Validate(
    `{"name": "Bob", "age": 25}`,
    outlines.ObjectSchema(
        map[string]*outlines.Schema{
            "name": outlines.StringSchema(),
            "age":  outlines.IntegerSchema(),
        },
        "name",
    ),
)

if result.Valid {
    fmt.Println("JSON is valid")
} else {
    for _, err := range result.Errors {
        fmt.Printf("Error at %s: %s\n", err.Path, err.Message)
    }
}
```

---

## 3. Streaming Optimization

The `streaming` package provides configurable stream buffers, token counting,
and chunk merging to optimize LLM streaming responses.

### Stream Buffers

```go
import "digital.vasic.optimization/pkg/streaming"

// Create a buffer that flushes on word boundaries.
buf := streaming.NewStreamBuffer(streaming.FlushOnWord, 0)

// Simulate streaming tokens.
flushed := buf.Add("Hello ")
// flushed: ["Hello "]

flushed = buf.Add("wor")
// flushed: nil (incomplete word)

flushed = buf.Add("ld ")
// flushed: ["world "]

remaining := buf.Flush()
// remaining: "" (nothing left)
```

### Flush Strategies

Four strategies are available:

```go
// FlushOnWord - flushes complete words (space-delimited).
buf := streaming.NewStreamBuffer(streaming.FlushOnWord, 0)

// FlushOnSentence - flushes complete sentences (., !, ?).
buf := streaming.NewStreamBuffer(streaming.FlushOnSentence, 0)

// FlushOnLine - flushes on newline characters.
buf := streaming.NewStreamBuffer(streaming.FlushOnLine, 0)

// FlushOnSize - flushes when word count reaches threshold.
buf := streaming.NewStreamBuffer(streaming.FlushOnSize, 5) // 5 words
```

### Token Counting

```go
counter := streaming.NewTokenCounter()
tokens := counter.Count("Hello world, how are you?")
// Approximate: 5 words * 1.3 ratio = 6

words := counter.CountWords("Hello world foo")   // 3
chars := counter.CountCharacters("Hello")         // 5

// Check if text fits within a token limit.
fits := counter.Fits("short text", 100) // true

// Custom tokens-per-word ratio.
counter = streaming.NewTokenCounterWithRatio(1.5)
```

### Chunk Merging

```go
merger := streaming.NewChunkMerger(3) // Minimum 3 words per merged chunk.

result := merger.Add("Hi")     // "" (accumulating)
result = merger.Add(" there")  // "" (still < 3 words)
result = merger.Add(" friend") // "Hi there friend" (>= 3 words, flushed)

remaining := merger.Flush() // Any leftover content
merger.Reset()              // Clear state
```

---

## 4. SGLang Integration

The `sglang` package provides a client for interacting with
[SGLang](https://github.com/sgl-project/sglang) servers, which offer
efficient LLM serving with RadixAttention for prefix caching.

### Basic Generation

```go
import "digital.vasic.optimization/pkg/sglang"

// Create a client pointing to your SGLang server.
client := sglang.NewHTTPClient(&sglang.Config{
    Endpoint: "http://localhost:30000",
    Model:    "meta-llama/Llama-2-7b-chat-hf",
    Timeout:  120 * time.Second,
})

// Check service health.
if err := client.Health(ctx); err != nil {
    log.Fatal("SGLang server not available:", err)
}

// Generate text using a program.
result, err := client.Generate(ctx, &sglang.Program{
    SystemPrompt: "You are a helpful assistant.",
    UserPrompt:   "Explain prefix caching in 2 sentences.",
    Temperature:  0.7,
    MaxTokens:    200,
    TopP:         0.9,
    Stop:         []string{"\n\n"},
})
if err != nil {
    log.Fatal(err)
}
fmt.Println(result)
```

### Default Configuration

```go
// Uses http://localhost:30000 with 120s timeout.
client := sglang.NewHTTPClient(nil)
```

---

## 5. Prompt Optimization

The `prompt` package reduces prompt length while preserving meaning, and
manages reusable prompt templates with variable substitution.

### Prompt Compression

```go
import "digital.vasic.optimization/pkg/prompt"

compressor := prompt.NewCompressor(&prompt.Config{
    MaxTokens:            4096,
    PreserveInstructions: true,
    RemoveRedundancy:     true,
})

original := "Please note that Go is a fast language. Basically it compiles quickly."
optimized, err := compressor.Optimize(ctx, original)
// Result: "Go is a fast language. it compiles quickly."
// Removed: "Please note that" and "Basically"
```

### Removed Filler Phrases

The compressor removes these common redundant phrases:
- "please note that"
- "it is important to note that"
- "as mentioned earlier"
- "in other words"
- "to put it simply"
- "basically"
- "essentially"
- "in order to"

### Token Estimation

```go
tokens := prompt.EstimateTokens("Hello world, how are you?")
// Returns: 5 (word-based approximation)
```

### Prompt Templates

```go
// Create and register templates.
registry := prompt.NewTemplateRegistry()

err := registry.Register(&prompt.Template{
    Name:        "classification",
    Content:     "Classify the following {{type}}: {{input}}",
    Description: "Generic classification prompt",
    Variables:   []string{"type", "input"},
})

// Render a template with variables.
result, err := registry.RenderTemplate("classification", map[string]string{
    "type":  "sentiment",
    "input": "I love this product!",
})
// Result: "Classify the following sentiment: I love this product!"
```

### Template Management

```go
registry := prompt.NewTemplateRegistry()

_ = registry.Register(&prompt.Template{Name: "a", Content: "..."})
_ = registry.Register(&prompt.Template{Name: "b", Content: "..."})

names := registry.List()   // ["a", "b"]
size := registry.Size()    // 2

tmpl, _ := registry.Get("a")
registry.Remove("a")
```

---

## 6. Framework Adapters

The `adapter` package provides HTTP-based adapters for integrating with
LlamaIndex and LangChain services.

### LlamaIndex Adapter

```go
import "digital.vasic.optimization/pkg/adapter"

// Create a LlamaIndex adapter.
lla := adapter.NewLlamaIndexHTTPAdapter(&adapter.LlamaIndexConfig{
    BaseURL: "http://localhost:8012",
    Timeout: 120 * time.Second,
})

// Check health.
if err := lla.Health(ctx); err != nil {
    log.Fatal("LlamaIndex service not available:", err)
}

// Query documents.
result, err := lla.Query(ctx, "What is Go?", 5)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Answer: %s (confidence: %.2f)\n", result.Answer, result.Confidence)
for _, src := range result.Sources {
    fmt.Printf("  Source: %s (score: %.2f)\n", src.Content, src.Score)
}

// Rerank documents by relevance.
ranked, err := lla.Rerank(ctx, "Go programming", []string{
    "Go is a language by Google",
    "Python is popular for ML",
    "Go compiles to native code",
}, 2)
for _, doc := range ranked {
    fmt.Printf("Rank %d: %s (%.2f)\n", doc.Rank, doc.Content, doc.Score)
}
```

### LangChain Adapter

```go
// Create a LangChain adapter.
lca := adapter.NewLangChainHTTPAdapter(&adapter.LangChainConfig{
    BaseURL: "http://localhost:8011",
    Timeout: 120 * time.Second,
})

// Execute a chain.
result, err := lca.ExecuteChain(ctx, "summary", "Summarize this text.", nil)
if err != nil {
    log.Fatal(err)
}
fmt.Println("Result:", result.Result)
for _, step := range result.Steps {
    fmt.Printf("  Step: %s -> %s\n", step.Step, step.Output)
}

// Decompose a task into subtasks.
decomp, err := lca.Decompose(ctx, "Build a web application", 5)
if err != nil {
    log.Fatal(err)
}
fmt.Println("Reasoning:", decomp.Reasoning)
for _, task := range decomp.Subtasks {
    fmt.Printf("  [%d] %s (complexity: %s, deps: %v)\n",
        task.ID, task.Description, task.Complexity, task.Dependencies)
}
```

---

## Composing Packages

These packages are designed to be composed. Here is an example that combines
caching with prompt optimization and structured output:

```go
func processQuery(ctx context.Context, query string) (string, error) {
    // 1. Optimize the prompt.
    compressor := prompt.NewCompressor(nil)
    optimized, err := compressor.Optimize(ctx, query)
    if err != nil {
        return "", fmt.Errorf("prompt optimization failed: %w", err)
    }

    // 2. Check the cache.
    cache := gptcache.NewInMemoryCache(
        gptcache.WithSimilarityThreshold(0.85),
    )
    if cached, err := cache.Get(ctx, optimized); err == nil {
        return cached.Response, nil // Cache hit
    }

    // 3. Call the LLM (your provider).
    response := callLLM(optimized)

    // 4. Constrain the output to valid JSON.
    schema := outlines.ObjectSchema(
        map[string]*outlines.Schema{"answer": outlines.StringSchema()},
        "answer",
    )
    constrainer := outlines.NewJSONConstrainer()
    constrained, err := constrainer.Constrain(response, schema)
    if err != nil {
        return "", fmt.Errorf("output constraint failed: %w", err)
    }

    // 5. Cache the result.
    _ = cache.Set(ctx, optimized, constrained)

    return constrained, nil
}
```

---

## Testing

Run all tests:

```bash
go test ./... -count=1 -race
```

Run tests for a specific package:

```bash
go test -v ./pkg/gptcache/...
go test -v ./pkg/outlines/...
go test -v ./pkg/streaming/...
go test -v ./pkg/sglang/...
go test -v ./pkg/adapter/...
go test -v ./pkg/prompt/...
```

Run benchmarks:

```bash
go test -bench=. ./...
```
