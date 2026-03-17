# Optimization

Generic, reusable LLM optimization module for Go applications. Provides semantic caching for LLM responses, prompt compression and template management, streaming buffer optimization, structured output constraints with JSON Schema validation, SGLang integration, and LLM framework adapters for LlamaIndex and LangChain.

**Module**: `digital.vasic.optimization` (Go 1.24+)

## Architecture

The module is organized into six independent packages, each addressing a specific LLM optimization concern. The gptcache package reduces redundant API calls through semantic similarity matching. The prompt package compresses and templates prompts. The streaming package optimizes real-time token delivery. The outlines package constrains and validates structured output. The sglang package integrates with SGLang serving. The adapter package bridges LlamaIndex and LangChain frameworks.

```
pkg/
  gptcache/    Semantic caching with embedding similarity and TTL eviction
  prompt/      Prompt compression, template registry, variable substitution
  streaming/   Configurable stream buffers, token counting, chunk merging
  outlines/    JSON Schema validation, regex constrainers, schema builder
  sglang/      SGLang HTTP client for efficient LLM serving
  adapter/     LlamaIndex and LangChain HTTP adapters
```

## Package Reference

### pkg/gptcache -- Semantic LLM Cache

Caches LLM responses using embedding similarity to serve semantically equivalent queries from cache.

**Types:**
- `Cache` -- Interface with `Get(ctx, query)`, `Set(ctx, query, response)`, `Invalidate(ctx, query)`.
- `CachedResponse` -- Response, Similarity score, CachedAt, TTL, and Metadata.
- `SemanticMatcher` -- Interface for computing similarity between queries (0-1).
- `Config` -- SimilarityThreshold (0.85), MaxEntries (10000), TTL (24h).
- `ConfigOption` -- Functional options: `WithSimilarityThreshold`, `WithMaxEntries`, `WithTTL`.
- `InMemoryCache` -- In-memory Cache implementation with optional semantic matching.
- `EmbeddingMatcher` -- SemanticMatcher using cosine similarity on embedding vectors.

**Key Functions:**
- `NewInMemoryCache(opts ...ConfigOption) *InMemoryCache` -- Creates a cache with functional options.
- `NewInMemoryCacheWithConfig(config *Config) *InMemoryCache` -- Creates a cache with explicit config.
- `InMemoryCache.SetMatcher(matcher SemanticMatcher)` -- Enables semantic matching.
- `InMemoryCache.Get(ctx, query) (*CachedResponse, error)` -- Exact hash match first, then semantic.
- `InMemoryCache.Set(ctx, query, response) error` -- Stores with automatic eviction.
- `InMemoryCache.Invalidate(ctx, query) error` -- Removes by exact hash.
- `InMemoryCache.Size() int` / `InMemoryCache.Clear()`
- `CosineSimilarity(vec1, vec2 []float64) float64` -- Computes cosine similarity.
- `NormalizeL2(vec []float64) []float64` -- L2 normalization.

**Errors:**
- `ErrCacheMiss` -- No matching entry found.
- `ErrInvalidQuery` -- Empty or invalid query.

### pkg/prompt -- Prompt Optimization

Compresses prompts by removing redundancy and manages reusable templates with variable substitution.

**Types:**
- `Optimizer` -- Interface with `Optimize(ctx, prompt string) (string, error)`.
- `Config` -- MaxTokens (4096), PreserveInstructions, RemoveRedundancy.
- `Compressor` -- Implements Optimizer; normalizes whitespace, removes filler phrases, truncates.
- `Template` -- Named template with `{{variable}}` placeholders, Description, and Variables list.
- `TemplateRegistry` -- Thread-safe registry for prompt templates.

**Key Functions:**
- `NewCompressor(config *Config) *Compressor` -- Creates a prompt compressor.
- `Compressor.Optimize(ctx, prompt) (string, error)` -- Compresses a prompt.
- `EstimateTokens(s string) int` -- Estimates token count (word-based approximation).
- `NewTemplateRegistry() *TemplateRegistry` -- Creates a template registry.
- `TemplateRegistry.Register(template *Template) error` -- Registers a template.
- `TemplateRegistry.Get(name) (*Template, error)` / `TemplateRegistry.Remove(name)`
- `TemplateRegistry.RenderTemplate(name, vars map[string]string) (string, error)` -- Renders with variables.
- `Template.Render(vars map[string]string) (string, error)` -- Substitutes `{{key}}` placeholders.

**Removed Filler Phrases:** "please note that", "it is important to note that", "as mentioned earlier", "in other words", "to put it simply", "basically", "essentially", "in order to".

### pkg/streaming -- Stream Optimization

Configurable buffers for streaming LLM responses with multiple flush strategies, token counting, and chunk merging.

**Types:**
- `Buffer` -- Interface with `Add(text) []string`, `Flush() string`, `Reset()`.
- `FlushStrategy` -- FlushOnWord, FlushOnSentence, FlushOnLine, FlushOnSize.
- `StreamBuffer` -- Implements Buffer with configurable strategy.
- `TokenCounter` -- Estimates token counts with configurable tokens-per-word ratio (default 1.3).
- `ChunkMerger` -- Combines small chunks into larger ones to reduce processing overhead.
- `Config` -- BufferSize (5), FlushInterval (100ms), MinChunkSize (3), Strategy (FlushOnWord).

**Key Functions:**
- `NewStreamBuffer(strategy FlushStrategy, threshold int) *StreamBuffer`
- `StreamBuffer.Add(text string) []string` -- Adds text, returns flushed content.
- `StreamBuffer.Flush() string` -- Returns remaining buffered content.
- `NewTokenCounter() *TokenCounter` -- Creates a counter (1.3 tokens/word).
- `NewTokenCounterWithRatio(tokensPerWord float64) *TokenCounter`
- `TokenCounter.Count(text) int` / `TokenCounter.CountWords(text) int` / `TokenCounter.Fits(text, limit) bool`
- `NewChunkMerger(minChunkSize int) *ChunkMerger`
- `ChunkMerger.Add(chunk) string` -- Returns merged content when threshold reached, empty otherwise.

### pkg/outlines -- Structured Output Constraints

JSON Schema validation and regex constraining for LLM output. Includes a fluent schema builder and JSON extraction from mixed text.

**Types:**
- `Schema` -- Full JSON Schema representation with Type, Properties, Required, Items, Enum, Min/MaxLength, Min/Maximum, Pattern, Format, OneOf/AnyOf/AllOf.
- `SchemaBuilder` -- Fluent API: `Object()`, `Array()`, `StringType()`, `Property(name, schema)`, `RequiredProps(...)`, `Build()`.
- `Constrainer` -- Interface with `Constrain(output, schema) (string, error)`.
- `JSONConstrainer` -- Extracts JSON from text and validates against schema.
- `RegexConstrainer` -- Validates output against a regex pattern.
- `ValidationResult` -- Valid flag, Errors list, and parsed Data.
- `ValidationError` -- Path and Message for each validation failure.

**Key Functions:**
- `NewSchemaBuilder() *SchemaBuilder` -- Creates a fluent schema builder.
- `StringSchema()` / `IntegerSchema()` / `NumberSchema()` / `BooleanSchema()` / `ArraySchema(items)` / `ObjectSchema(props, required...)`
- `ParseSchema(data []byte) (*Schema, error)` -- Parses JSON Schema from bytes.
- `Validate(jsonStr string, schema *Schema) *ValidationResult` -- Validates JSON against a schema.
- `ValidateValue(data, schema, path) *ValidationResult` -- Validates a single Go value.
- `NewJSONConstrainer() *JSONConstrainer` -- Creates a JSON constrainer.
- `NewRegexConstrainer(pattern string) (*RegexConstrainer, error)` -- Creates a regex constrainer.

### pkg/sglang -- SGLang Integration

HTTP client for SGLang, an efficient LLM serving framework with RadixAttention prefix caching.

**Types:**
- `Client` -- Interface with `Generate(ctx, program) (string, error)` and `Health(ctx) error`.
- `Program` -- SystemPrompt, UserPrompt, Temperature, MaxTokens, TopP, Stop sequences.
- `Config` -- Endpoint (default `http://localhost:30000`), Model, Timeout (120s).
- `HTTPClient` -- Implements Client via HTTP requests to SGLang's OpenAI-compatible API.

**Key Functions:**
- `NewHTTPClient(config *Config) *HTTPClient` -- Creates an SGLang client.
- `HTTPClient.Generate(ctx, program) (string, error)` -- Executes a program via `/v1/chat/completions`.
- `HTTPClient.Health(ctx) error` -- Checks service availability via `/health`.

### pkg/adapter -- LLM Framework Adapters

HTTP adapters for LlamaIndex and LangChain integration.

**LlamaIndex Types:**
- `LlamaIndexAdapter` -- Interface with Query, Rerank, Health methods.
- `QueryResult` -- Answer, Sources with scores, Confidence.
- `RankedDocument` -- Content, Score, Rank.
- `LlamaIndexConfig` -- BaseURL (default `http://localhost:8012`), Timeout (120s).

**LangChain Types:**
- `LangChainAdapter` -- Interface with ExecuteChain, Decompose, Health methods.
- `ChainResult` -- Result string and execution Steps.
- `DecomposeResult` -- Subtasks with dependencies and Reasoning.
- `LangChainConfig` -- BaseURL (default `http://localhost:8011`), Timeout (120s).

**Key Functions:**
- `NewLlamaIndexHTTPAdapter(config *LlamaIndexConfig) *LlamaIndexHTTPAdapter`
- `LlamaIndexHTTPAdapter.Query(ctx, query, topK) (*QueryResult, error)`
- `LlamaIndexHTTPAdapter.Rerank(ctx, query, documents, topK) ([]RankedDocument, error)`
- `NewLangChainHTTPAdapter(config *LangChainConfig) *LangChainHTTPAdapter`
- `LangChainHTTPAdapter.ExecuteChain(ctx, chainType, prompt, variables) (*ChainResult, error)`
- `LangChainHTTPAdapter.Decompose(ctx, task, maxSteps) (*DecomposeResult, error)`

## Usage Examples

### Semantic Caching

```go
cache := gptcache.NewInMemoryCache(
    gptcache.WithSimilarityThreshold(0.85),
    gptcache.WithMaxEntries(10000),
    gptcache.WithTTL(24 * time.Hour),
)

// Optional: enable semantic matching
cache.SetMatcher(&gptcache.EmbeddingMatcher{
    EmbedFunc: myEmbeddingFunction,
})

cache.Set(ctx, "What is Go?", "Go is a programming language...")
resp, err := cache.Get(ctx, "Tell me about Go") // semantic match
```

### Prompt Compression

```go
compressor := prompt.NewCompressor(&prompt.Config{
    MaxTokens:        2048,
    RemoveRedundancy: true,
})
optimized, _ := compressor.Optimize(ctx, longPrompt)
```

### Prompt Templates

```go
registry := prompt.NewTemplateRegistry()
registry.Register(&prompt.Template{
    Name:    "code-review",
    Content: "Review this {{language}} code:\n{{code}}",
})
rendered, _ := registry.RenderTemplate("code-review", map[string]string{
    "language": "Go",
    "code":     "func main() {}",
})
```

### Structured Output Validation

```go
schema := outlines.NewSchemaBuilder().
    Object().
    Property("name", outlines.StringSchema()).
    Property("age", outlines.IntegerSchema()).
    RequiredProps("name", "age").
    Build()

constrainer := outlines.NewJSONConstrainer()
json, err := constrainer.Constrain(llmOutput, schema)
```

### Stream Buffer

```go
buf := streaming.NewStreamBuffer(streaming.FlushOnSentence, 0)
for chunk := range llmStream {
    flushed := buf.Add(chunk)
    for _, s := range flushed {
        sendToClient(s)
    }
}
sendToClient(buf.Flush()) // remaining content
```

## Configuration

All packages use Config structs with `DefaultConfig()` constructors. The gptcache package also supports functional options via `ConfigOption`.

## Testing

```bash
go test ./... -count=1 -race    # All tests with race detection
go test ./... -short             # Unit tests only
go test -bench=. ./...           # Benchmarks
```

## Integration with HelixAgent

The Optimization module is used throughout HelixAgent:
- GPT-Cache reduces redundant LLM API calls during debate sessions
- Prompt compression optimizes token usage across all 22 providers
- Streaming buffers smooth LLM response delivery via SSE endpoints
- Outlines validation enforces structured output in tool calls and debate phases
- SGLang integration provides high-performance local model serving
- LlamaIndex/LangChain adapters connect to external RAG and chain services

The internal adapter at `internal/adapters/optimization/` bridges these generic types to HelixAgent-specific interfaces.

## License

Proprietary.
