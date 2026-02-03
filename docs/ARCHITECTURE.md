# Architecture - Optimization Module

## Overview

`digital.vasic.optimization` is a generic, dependency-minimal Go module that
provides LLM optimization primitives. It is designed to be consumed by larger
systems (such as HelixAgent) as a library, not as a standalone service.

**Module path**: `digital.vasic.optimization`
**Go version**: 1.24+
**External dependencies**: `github.com/stretchr/testify` (testing only)

## Design Principles

1. **Zero runtime dependencies.** The module uses only the Go standard library
   at runtime. The single external dependency (`testify`) is for testing only.

2. **Package independence.** All 6 packages are fully independent with no
   cross-package imports. They are composed at the consumer level.

3. **Interface-first design.** Each package defines small, focused interfaces
   that enable multiple implementations and easy testing.

4. **Functional options and configuration.** Packages use functional options
   (`ConfigOption`) or explicit config structs with `DefaultConfig()` +
   `Validate()` patterns for ergonomic initialization.

5. **Concurrency safety.** Thread-safe implementations use `sync.RWMutex` for
   concurrent read/write access (e.g., `InMemoryCache`, `TemplateRegistry`).

## Package Architecture

```
digital.vasic.optimization/
    pkg/
        gptcache/      Semantic caching with embedding similarity
        prompt/        Prompt optimization, compression, templates
        streaming/     Stream buffering, token counting, chunk merging
        outlines/      JSON Schema validation, regex constraining
        sglang/        SGLang HTTP client for efficient LLM serving
        adapter/       LlamaIndex and LangChain HTTP adapters
```

## Design Patterns

### Proxy Pattern

**Used in**: `sglang`, `adapter`

The SGLang `HTTPClient` and both adapter types (`LlamaIndexHTTPAdapter`,
`LangChainHTTPAdapter`) act as proxies to remote services. They encapsulate
HTTP communication, request/response serialization, error handling, and health
checking behind clean interfaces.

```
Client code --> HTTPClient (proxy) --> SGLang server
Client code --> LlamaIndexHTTPAdapter (proxy) --> LlamaIndex service
Client code --> LangChainHTTPAdapter (proxy) --> LangChain service
```

Each proxy:
- Accepts `context.Context` for cancellation and timeout propagation.
- Handles JSON marshaling/unmarshaling internally.
- Returns typed Go structs, not raw HTTP responses.
- Provides a `Health()` method for availability checks.

### Decorator Pattern

**Used in**: `gptcache`

The `InMemoryCache` can be decorated with a `SemanticMatcher` via
`SetMatcher()`. Without a matcher, it performs exact-hash lookups only. With
a matcher, it adds semantic similarity search on top of the base caching
behavior. This is a runtime decorator -- the matching behavior is layered
onto the existing cache without modifying its core storage logic.

```
Get(query) --> exact hash lookup
           |
           +--> (if matcher set) semantic scan over all entries
```

### Strategy Pattern

**Used in**: `streaming`, `gptcache`

The `StreamBuffer` uses the Strategy pattern for flush behavior. The
`FlushStrategy` type (`word`, `sentence`, `line`, `size`) selects the
algorithm used to determine when buffered content is emitted. The strategy
is chosen at construction time and dispatch happens in the `Add()` method:

| Strategy         | Behavior                                    |
|------------------|---------------------------------------------|
| `FlushOnWord`    | Emits on space boundaries                   |
| `FlushOnSentence`| Emits on sentence-ending punctuation (. ! ?)|
| `FlushOnLine`    | Emits on newline characters                 |
| `FlushOnSize`    | Emits when word count reaches threshold     |

In `gptcache`, the eviction strategy (oldest-first FIFO) is built into
`InMemoryCache`. The similarity metric is pluggable via `SemanticMatcher`,
which is itself a strategy for computing query similarity.

### Builder Pattern

**Used in**: `outlines`

`SchemaBuilder` provides a fluent API for constructing JSON Schema objects:

```go
schema := outlines.NewSchemaBuilder().
    Object().
    Property("name", outlines.StringSchema()).
    RequiredProps("name").
    Build()
```

The builder accumulates schema properties through method chaining and produces
a `*Schema` via `Build()`. This avoids complex struct literal construction for
nested schemas.

### Functional Options Pattern

**Used in**: `gptcache`

`NewInMemoryCache` accepts variadic `ConfigOption` functions:

```go
cache := gptcache.NewInMemoryCache(
    gptcache.WithSimilarityThreshold(0.9),
    gptcache.WithMaxEntries(5000),
    gptcache.WithTTL(12 * time.Hour),
)
```

This pattern provides a clean API with sensible defaults while allowing
selective configuration.

### Template Method Pattern

**Used in**: `streaming`

The `Buffer` interface defines the template for stream processing (Add, Flush,
Reset). `StreamBuffer` implements the template with pluggable flush logic,
while `ChunkMerger` provides an alternative implementation focused on merging
small chunks into larger ones.

### Interface Segregation

All interfaces in this module are small and focused:

| Interface          | Methods                                  | Package    |
|--------------------|------------------------------------------|------------|
| `Cache`            | Get, Set, Invalidate                     | gptcache   |
| `SemanticMatcher`  | Similarity                               | gptcache   |
| `Optimizer`        | Optimize                                 | prompt     |
| `Buffer`           | Add, Flush, Reset                        | streaming  |
| `Constrainer`      | Constrain                                | outlines   |
| `Client`           | Generate, Health                         | sglang     |
| `LlamaIndexAdapter`| Query, Rerank, Health                    | adapter    |
| `LangChainAdapter` | ExecuteChain, Decompose, Health          | adapter    |

No interface has more than 3 methods. This makes implementations easy to write
and test, and allows consumers to depend on exactly the behavior they need.

## Data Flow

### Cache-Hit Optimization Flow

```
1. User query arrives
2. (Optional) Prompt optimization -- compress, remove redundancy
3. Cache lookup:
   a. Hash query -> check exact match in entries map
   b. If miss and SemanticMatcher is set -> scan all entries for similarity
   c. If similarity >= threshold -> return cached response
   d. If miss -> proceed to LLM
4. LLM generates response
5. (Optional) Constrain output via JSON Schema or regex
6. Store in cache with TTL
7. Return response
```

### Streaming Optimization Flow

```
1. LLM produces token stream
2. Tokens fed to StreamBuffer.Add()
3. Buffer accumulates until flush condition (word/sentence/line/size)
4. Flushed chunks optionally pass through ChunkMerger
5. Merged chunks delivered to consumer
6. On stream end: Buffer.Flush() + ChunkMerger.Flush() for remaining content
```

## Concurrency Model

- **InMemoryCache**: `sync.RWMutex` protects `entries` and `order`. Read
  operations (Get, Size) acquire `RLock`; write operations (Set, Invalidate,
  Clear) acquire full `Lock`.

- **TemplateRegistry**: `sync.RWMutex` protects the `templates` map. Same
  read/write locking pattern as InMemoryCache.

- **StreamBuffer, ChunkMerger, TokenCounter**: Not thread-safe. Designed
  for single-goroutine use within a streaming pipeline. If concurrent access
  is needed, the consumer must provide external synchronization.

- **HTTP Clients** (SGLang, LlamaIndex, LangChain): Thread-safe via
  `http.Client` (which is safe for concurrent use). Each request uses
  `context.Context` for cancellation.

## Error Handling

- Sentinel errors: `gptcache.ErrCacheMiss`, `gptcache.ErrInvalidQuery`
- Wrapped errors: All packages wrap errors with `fmt.Errorf("...: %w", err)`
  for chain inspection via `errors.Is()` and `errors.As()`.
- Validation errors: `outlines.ValidationError` and `ValidationResult`
  provide structured error reporting with JSON path information.

## Configuration Defaults

| Package    | Parameter             | Default          |
|------------|-----------------------|------------------|
| gptcache   | SimilarityThreshold   | 0.85             |
| gptcache   | MaxEntries            | 10,000           |
| gptcache   | TTL                   | 24 hours         |
| prompt     | MaxTokens             | 4,096            |
| prompt     | PreserveInstructions  | true             |
| prompt     | RemoveRedundancy      | true             |
| streaming  | BufferSize            | 5                |
| streaming  | FlushInterval         | 100ms            |
| streaming  | MinChunkSize          | 3                |
| streaming  | Strategy              | FlushOnWord      |
| sglang     | Endpoint              | localhost:30000  |
| sglang     | Timeout               | 120s             |
| sglang     | Default Temperature   | 0.7              |
| sglang     | Default MaxTokens     | 500              |
| adapter    | LlamaIndex BaseURL    | localhost:8012   |
| adapter    | LangChain BaseURL     | localhost:8011   |
| adapter    | Timeout               | 120s             |

## Extensibility

### Adding a New Cache Backend

Implement the `gptcache.Cache` interface:

```go
type MyRedisCache struct { ... }
func (c *MyRedisCache) Get(ctx context.Context, query string) (*CachedResponse, error) { ... }
func (c *MyRedisCache) Set(ctx context.Context, query string, response string) error { ... }
func (c *MyRedisCache) Invalidate(ctx context.Context, query string) error { ... }
```

### Adding a New Constrainer

Implement the `outlines.Constrainer` interface:

```go
type YAMLConstrainer struct { ... }
func (c *YAMLConstrainer) Constrain(output string, schema *Schema) (string, error) { ... }
```

### Adding a New Flush Strategy

Add a new `FlushStrategy` constant and implement the flush method in
`StreamBuffer`, then add a case to the `Add()` switch.

### Adding a New Framework Adapter

Define an interface and HTTP adapter following the pattern in
`pkg/adapter/llamaindex.go` or `pkg/adapter/langchain.go`.
