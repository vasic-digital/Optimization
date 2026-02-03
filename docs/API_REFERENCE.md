# API Reference - Optimization Module

Module: `digital.vasic.optimization`

---

## Package `gptcache`

`import "digital.vasic.optimization/pkg/gptcache"`

Semantic caching for LLM responses using embedding similarity.

### Sentinel Errors

```go
var ErrCacheMiss = errors.New("cache miss")
var ErrInvalidQuery = errors.New("invalid query")
```

### Types

#### CachedResponse

```go
type CachedResponse struct {
    Response   string                 `json:"response"`
    Similarity float64                `json:"similarity"`
    CachedAt   time.Time              `json:"cached_at"`
    TTL        time.Duration          `json:"ttl"`
    Metadata   map[string]interface{} `json:"metadata,omitempty"`
}
```

Represents a cached LLM response returned by `Cache.Get`.

#### Config

```go
type Config struct {
    SimilarityThreshold float64       `json:"similarity_threshold"`
    MaxEntries          int           `json:"max_entries"`
    TTL                 time.Duration `json:"ttl"`
}
```

Cache configuration. Fields:
- `SimilarityThreshold`: Minimum similarity score (0-1) for a cache hit. Default: 0.85.
- `MaxEntries`: Maximum number of entries. Default: 10,000.
- `TTL`: Time-to-live for entries. Default: 24h.

**Methods:**

| Method | Signature | Description |
|--------|-----------|-------------|
| `Validate` | `(c *Config) Validate()` | Validates config and applies defaults for invalid values. |

#### ConfigOption

```go
type ConfigOption func(*Config)
```

Functional option for configuring the cache.

#### InMemoryCache

```go
type InMemoryCache struct { /* unexported fields */ }
```

In-memory `Cache` implementation with optional semantic matching.

**Constructors:**

| Function | Signature | Description |
|----------|-----------|-------------|
| `NewInMemoryCache` | `NewInMemoryCache(opts ...ConfigOption) *InMemoryCache` | Creates cache with functional options. |
| `NewInMemoryCacheWithConfig` | `NewInMemoryCacheWithConfig(config *Config) *InMemoryCache` | Creates cache with explicit config. |

**Methods:**

| Method | Signature | Description |
|--------|-----------|-------------|
| `Get` | `(c *InMemoryCache) Get(ctx context.Context, query string) (*CachedResponse, error)` | Retrieves cached response. Returns `ErrCacheMiss` on miss. |
| `Set` | `(c *InMemoryCache) Set(ctx context.Context, query string, response string) error` | Stores a query-response pair. Evicts oldest if over capacity. |
| `Invalidate` | `(c *InMemoryCache) Invalidate(ctx context.Context, query string) error` | Removes entry by exact query hash. |
| `SetMatcher` | `(c *InMemoryCache) SetMatcher(matcher SemanticMatcher)` | Sets semantic matcher for similarity-based lookups. |
| `Size` | `(c *InMemoryCache) Size() int` | Returns current entry count. |
| `Clear` | `(c *InMemoryCache) Clear()` | Removes all entries. |
| `Config` | `(c *InMemoryCache) Config() *Config` | Returns the cache configuration. |

#### EmbeddingMatcher

```go
type EmbeddingMatcher struct {
    EmbedFunc func(query string) ([]float64, error)
}
```

Implements `SemanticMatcher` using embedding vectors. If `EmbedFunc` is nil,
falls back to exact (case-insensitive) matching.

**Methods:**

| Method | Signature | Description |
|--------|-----------|-------------|
| `Similarity` | `(m *EmbeddingMatcher) Similarity(query1, query2 string) (float64, error)` | Computes cosine similarity between embeddings, normalized to [0,1]. |

### Interfaces

#### Cache

```go
type Cache interface {
    Get(ctx context.Context, query string) (*CachedResponse, error)
    Set(ctx context.Context, query string, response string) error
    Invalidate(ctx context.Context, query string) error
}
```

#### SemanticMatcher

```go
type SemanticMatcher interface {
    Similarity(query1, query2 string) (float64, error)
}
```

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `DefaultConfig` | `DefaultConfig() *Config` | Returns default configuration (0.85 threshold, 10k entries, 24h TTL). |
| `WithSimilarityThreshold` | `WithSimilarityThreshold(threshold float64) ConfigOption` | Sets similarity threshold. |
| `WithMaxEntries` | `WithMaxEntries(n int) ConfigOption` | Sets max entries. |
| `WithTTL` | `WithTTL(ttl time.Duration) ConfigOption` | Sets TTL. |
| `CosineSimilarity` | `CosineSimilarity(vec1, vec2 []float64) float64` | Computes cosine similarity between vectors. Returns [-1, 1]. Returns 0 for mismatched/empty vectors. |
| `NormalizeL2` | `NormalizeL2(vec []float64) []float64` | L2-normalizes a vector to unit length. Returns input if empty or zero-norm. |

---

## Package `prompt`

`import "digital.vasic.optimization/pkg/prompt"`

Prompt optimization with compression and template management.

### Types

#### Config

```go
type Config struct {
    MaxTokens            int  `json:"max_tokens"`
    PreserveInstructions bool `json:"preserve_instructions"`
    RemoveRedundancy     bool `json:"remove_redundancy"`
}
```

- `MaxTokens`: Max token count for optimized prompt. Default: 4096.
- `PreserveInstructions`: Keep system instructions intact. Default: true.
- `RemoveRedundancy`: Remove filler phrases. Default: true.

#### Compressor

```go
type Compressor struct { /* unexported fields */ }
```

Reduces prompt length while preserving meaning. Implements `Optimizer`.

**Constructor:**

| Function | Signature | Description |
|----------|-----------|-------------|
| `NewCompressor` | `NewCompressor(config *Config) *Compressor` | Creates compressor. Uses `DefaultConfig()` if nil. |

**Methods:**

| Method | Signature | Description |
|--------|-----------|-------------|
| `Optimize` | `(c *Compressor) Optimize(ctx context.Context, prompt string) (string, error)` | Compresses prompt: normalizes whitespace, removes redundant phrases, truncates to max tokens. |

#### Template

```go
type Template struct {
    Name        string   `json:"name"`
    Content     string   `json:"content"`
    Description string   `json:"description,omitempty"`
    Variables   []string `json:"variables,omitempty"`
}
```

Prompt template with `{{variable}}` placeholder substitution.

**Methods:**

| Method | Signature | Description |
|--------|-----------|-------------|
| `Render` | `(t *Template) Render(vars map[string]string) (string, error)` | Renders template. Returns error if unresolved placeholders remain. |

#### TemplateRegistry

```go
type TemplateRegistry struct { /* unexported fields */ }
```

Thread-safe template manager.

**Constructor:**

| Function | Signature | Description |
|----------|-----------|-------------|
| `NewTemplateRegistry` | `NewTemplateRegistry() *TemplateRegistry` | Creates empty registry. |

**Methods:**

| Method | Signature | Description |
|--------|-----------|-------------|
| `Register` | `(r *TemplateRegistry) Register(template *Template) error` | Registers a template. Errors on nil or empty name. |
| `Get` | `(r *TemplateRegistry) Get(name string) (*Template, error)` | Retrieves template by name. |
| `Remove` | `(r *TemplateRegistry) Remove(name string)` | Removes template by name. |
| `List` | `(r *TemplateRegistry) List() []string` | Returns all template names. |
| `Size` | `(r *TemplateRegistry) Size() int` | Returns template count. |
| `RenderTemplate` | `(r *TemplateRegistry) RenderTemplate(name string, vars map[string]string) (string, error)` | Gets and renders a template in one call. |

### Interfaces

#### Optimizer

```go
type Optimizer interface {
    Optimize(ctx context.Context, prompt string) (string, error)
}
```

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `DefaultConfig` | `DefaultConfig() *Config` | Returns default config (4096 tokens, preserve instructions, remove redundancy). |
| `EstimateTokens` | `EstimateTokens(s string) int` | Estimates token count using word count approximation. |

---

## Package `streaming`

`import "digital.vasic.optimization/pkg/streaming"`

Streaming optimizations with configurable buffers and chunk merging.

### Constants

```go
type FlushStrategy string

const (
    FlushOnWord     FlushStrategy = "word"
    FlushOnSentence FlushStrategy = "sentence"
    FlushOnLine     FlushStrategy = "line"
    FlushOnSize     FlushStrategy = "size"
)
```

### Types

#### Config

```go
type Config struct {
    BufferSize    int           `json:"buffer_size"`
    FlushInterval time.Duration `json:"flush_interval"`
    MinChunkSize  int           `json:"min_chunk_size"`
    Strategy      FlushStrategy `json:"strategy"`
}
```

#### StreamBuffer

```go
type StreamBuffer struct { /* unexported fields */ }
```

Implements `Buffer` with configurable flush strategies.

**Constructor:**

| Function | Signature | Description |
|----------|-----------|-------------|
| `NewStreamBuffer` | `NewStreamBuffer(strategy FlushStrategy, threshold int) *StreamBuffer` | Creates buffer. Threshold used for `FlushOnSize`; defaults to 5 if <= 0. |

**Methods:**

| Method | Signature | Description |
|--------|-----------|-------------|
| `Add` | `(b *StreamBuffer) Add(text string) []string` | Adds text, returns flushed content per strategy. |
| `Flush` | `(b *StreamBuffer) Flush() string` | Returns remaining buffered content. |
| `Reset` | `(b *StreamBuffer) Reset()` | Clears the buffer. |

#### TokenCounter

```go
type TokenCounter struct {
    TokensPerWord float64
}
```

Estimates token counts. Default ratio: 1.3 tokens/word.

**Constructors:**

| Function | Signature | Description |
|----------|-----------|-------------|
| `NewTokenCounter` | `NewTokenCounter() *TokenCounter` | Creates counter with default 1.3 ratio. |
| `NewTokenCounterWithRatio` | `NewTokenCounterWithRatio(tokensPerWord float64) *TokenCounter` | Creates counter with custom ratio. Defaults to 1.3 if <= 0. |

**Methods:**

| Method | Signature | Description |
|--------|-----------|-------------|
| `Count` | `(c *TokenCounter) Count(text string) int` | Estimates token count (words * ratio). |
| `CountWords` | `(c *TokenCounter) CountWords(text string) int` | Returns exact word count. |
| `CountCharacters` | `(c *TokenCounter) CountCharacters(text string) int` | Returns rune count. |
| `Fits` | `(c *TokenCounter) Fits(text string, limit int) bool` | Checks if estimated tokens fit within limit. |

#### ChunkMerger

```go
type ChunkMerger struct { /* unexported fields */ }
```

Merges small stream chunks into larger ones.

**Constructor:**

| Function | Signature | Description |
|----------|-----------|-------------|
| `NewChunkMerger` | `NewChunkMerger(minChunkSize int) *ChunkMerger` | Creates merger. Defaults to 3 if <= 0. |

**Methods:**

| Method | Signature | Description |
|--------|-----------|-------------|
| `Add` | `(m *ChunkMerger) Add(chunk string) string` | Adds chunk. Returns merged content when >= minChunkSize words, empty string otherwise. |
| `Flush` | `(m *ChunkMerger) Flush() string` | Returns remaining buffered content. |
| `Reset` | `(m *ChunkMerger) Reset()` | Clears merger state. |

### Interfaces

#### Buffer

```go
type Buffer interface {
    Add(text string) []string
    Flush() string
    Reset()
}
```

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `DefaultConfig` | `DefaultConfig() *Config` | Returns default config (5 buffer, 100ms interval, 3 min chunk, word strategy). |

---

## Package `outlines`

`import "digital.vasic.optimization/pkg/outlines"`

Structured output constraints with JSON Schema and regex validation.

### Types

#### Schema

```go
type Schema struct {
    Type                 string             `json:"type,omitempty"`
    Properties           map[string]*Schema `json:"properties,omitempty"`
    Required             []string           `json:"required,omitempty"`
    Items                *Schema            `json:"items,omitempty"`
    Enum                 []interface{}      `json:"enum,omitempty"`
    MinLength            *int               `json:"minLength,omitempty"`
    MaxLength            *int               `json:"maxLength,omitempty"`
    Minimum              *float64           `json:"minimum,omitempty"`
    Maximum              *float64           `json:"maximum,omitempty"`
    Pattern              string             `json:"pattern,omitempty"`
    MinItems             *int               `json:"minItems,omitempty"`
    MaxItems             *int               `json:"maxItems,omitempty"`
    UniqueItems          bool               `json:"uniqueItems,omitempty"`
    AdditionalProperties *bool              `json:"additionalProperties,omitempty"`
    Description          string             `json:"description,omitempty"`
    Default              interface{}        `json:"default,omitempty"`
    Format               string             `json:"format,omitempty"`
    OneOf                []*Schema          `json:"oneOf,omitempty"`
    AnyOf                []*Schema          `json:"anyOf,omitempty"`
    AllOf                []*Schema          `json:"allOf,omitempty"`
}
```

**Methods:**

| Method | Signature | Description |
|--------|-----------|-------------|
| `String` | `(s *Schema) String() string` | Returns pretty-printed JSON representation. |
| `IsRequired` | `(s *Schema) IsRequired(property string) bool` | Checks if a property is in the Required list. |

#### SchemaBuilder

```go
type SchemaBuilder struct { /* unexported fields */ }
```

Fluent API for building schemas.

**Constructor:**

| Function | Signature | Description |
|----------|-----------|-------------|
| `NewSchemaBuilder` | `NewSchemaBuilder() *SchemaBuilder` | Creates empty builder. |

**Methods:**

| Method | Signature | Description |
|--------|-----------|-------------|
| `Object` | `(b *SchemaBuilder) Object() *SchemaBuilder` | Sets type to "object". |
| `Array` | `(b *SchemaBuilder) Array() *SchemaBuilder` | Sets type to "array". |
| `StringType` | `(b *SchemaBuilder) StringType() *SchemaBuilder` | Sets type to "string". |
| `NumberType` | `(b *SchemaBuilder) NumberType() *SchemaBuilder` | Sets type to "number". |
| `IntegerType` | `(b *SchemaBuilder) IntegerType() *SchemaBuilder` | Sets type to "integer". |
| `BooleanType` | `(b *SchemaBuilder) BooleanType() *SchemaBuilder` | Sets type to "boolean". |
| `Property` | `(b *SchemaBuilder) Property(name string, schema *Schema) *SchemaBuilder` | Adds a property to an object schema. |
| `RequiredProps` | `(b *SchemaBuilder) RequiredProps(properties ...string) *SchemaBuilder` | Marks properties as required. |
| `Items` | `(b *SchemaBuilder) Items(schema *Schema) *SchemaBuilder` | Sets items schema for arrays. |
| `EnumValues` | `(b *SchemaBuilder) EnumValues(values ...interface{}) *SchemaBuilder` | Sets allowed enum values. |
| `SetPattern` | `(b *SchemaBuilder) SetPattern(pattern string) *SchemaBuilder` | Sets regex pattern for strings. |
| `SetDescription` | `(b *SchemaBuilder) SetDescription(desc string) *SchemaBuilder` | Sets schema description. |
| `Build` | `(b *SchemaBuilder) Build() *Schema` | Returns the constructed schema. |

#### ValidationError

```go
type ValidationError struct {
    Path    string `json:"path"`
    Message string `json:"message"`
}
```

**Methods:**

| Method | Signature | Description |
|--------|-----------|-------------|
| `Error` | `(e *ValidationError) Error() string` | Returns "path: message" or just "message" if path is empty. |

#### ValidationResult

```go
type ValidationResult struct {
    Valid  bool               `json:"valid"`
    Errors []*ValidationError `json:"errors,omitempty"`
    Data   interface{}        `json:"data,omitempty"`
}
```

**Methods:**

| Method | Signature | Description |
|--------|-----------|-------------|
| `AddError` | `(r *ValidationResult) AddError(path, message string)` | Adds a validation error and sets Valid to false. |
| `ErrorMessages` | `(r *ValidationResult) ErrorMessages() []string` | Returns all error messages as strings. |

#### JSONConstrainer

```go
type JSONConstrainer struct{}
```

Validates and extracts JSON output against a schema. Implements `Constrainer`.

**Constructor:**

| Function | Signature | Description |
|----------|-----------|-------------|
| `NewJSONConstrainer` | `NewJSONConstrainer() *JSONConstrainer` | Creates a JSON constrainer. |

**Methods:**

| Method | Signature | Description |
|--------|-----------|-------------|
| `Constrain` | `(c *JSONConstrainer) Constrain(output string, schema *Schema) (string, error)` | Extracts JSON from output, validates against schema. |

#### RegexConstrainer

```go
type RegexConstrainer struct { /* unexported fields */ }
```

Validates output against a compiled regex pattern. Implements `Constrainer`.

**Constructor:**

| Function | Signature | Description |
|----------|-----------|-------------|
| `NewRegexConstrainer` | `NewRegexConstrainer(pattern string) (*RegexConstrainer, error)` | Creates regex constrainer. Returns error for invalid patterns. |

**Methods:**

| Method | Signature | Description |
|--------|-----------|-------------|
| `Constrain` | `(c *RegexConstrainer) Constrain(output string, _ *Schema) (string, error)` | Validates output against pattern. Falls back to substring match. Schema param is ignored. |

### Interfaces

#### Constrainer

```go
type Constrainer interface {
    Constrain(output string, schema *Schema) (string, error)
}
```

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `ParseSchema` | `ParseSchema(data []byte) (*Schema, error)` | Parses JSON Schema from bytes. |
| `Validate` | `Validate(jsonStr string, schema *Schema) *ValidationResult` | Validates JSON string against schema. Supports object, array, string, number, integer, boolean, enum. |
| `StringSchema` | `StringSchema() *Schema` | Creates a string type schema. |
| `IntegerSchema` | `IntegerSchema() *Schema` | Creates an integer type schema. |
| `NumberSchema` | `NumberSchema() *Schema` | Creates a number type schema. |
| `BooleanSchema` | `BooleanSchema() *Schema` | Creates a boolean type schema. |
| `ArraySchema` | `ArraySchema(items *Schema) *Schema` | Creates an array schema with given item type. |
| `ObjectSchema` | `ObjectSchema(properties map[string]*Schema, required ...string) *Schema` | Creates an object schema with properties and required fields. |

---

## Package `sglang`

`import "digital.vasic.optimization/pkg/sglang"`

SGLang integration for efficient LLM serving with RadixAttention prefix caching.

### Types

#### Config

```go
type Config struct {
    Endpoint string        `json:"endpoint"`
    Model    string        `json:"model,omitempty"`
    Timeout  time.Duration `json:"timeout"`
}
```

- `Endpoint`: SGLang server URL. Default: `http://localhost:30000`.
- `Model`: Model identifier (optional).
- `Timeout`: Request timeout. Default: 120s.

#### Program

```go
type Program struct {
    SystemPrompt string   `json:"system_prompt,omitempty"`
    UserPrompt   string   `json:"user_prompt"`
    Temperature  float64  `json:"temperature,omitempty"`
    MaxTokens    int      `json:"max_tokens,omitempty"`
    TopP         float64  `json:"top_p,omitempty"`
    Stop         []string `json:"stop,omitempty"`
}
```

Represents an SGLang generation program. Defaults applied during generation:
Temperature defaults to 0.7, MaxTokens defaults to 500.

#### HTTPClient

```go
type HTTPClient struct { /* unexported fields */ }
```

Implements `Client` using HTTP requests.

**Constructor:**

| Function | Signature | Description |
|----------|-----------|-------------|
| `NewHTTPClient` | `NewHTTPClient(config *Config) *HTTPClient` | Creates SGLang HTTP client. Uses `DefaultConfig()` if nil. |

**Methods:**

| Method | Signature | Description |
|--------|-----------|-------------|
| `Generate` | `(c *HTTPClient) Generate(ctx context.Context, program *Program) (string, error)` | Executes program via `/v1/chat/completions`. Returns error for nil program. |
| `Health` | `(c *HTTPClient) Health(ctx context.Context) error` | Checks SGLang service health via `/health`. |

### Interfaces

#### Client

```go
type Client interface {
    Generate(ctx context.Context, program *Program) (string, error)
    Health(ctx context.Context) error
}
```

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `DefaultConfig` | `DefaultConfig() *Config` | Returns default config (localhost:30000, 120s timeout). |

---

## Package `adapter`

`import "digital.vasic.optimization/pkg/adapter"`

LLM framework adapters for LlamaIndex and LangChain integration.

### Types

#### QueryResult

```go
type QueryResult struct {
    Answer     string   `json:"answer"`
    Sources    []Source `json:"sources"`
    Confidence float64  `json:"confidence"`
}
```

#### Source

```go
type Source struct {
    Content  string                 `json:"content"`
    Score    float64                `json:"score"`
    Metadata map[string]interface{} `json:"metadata,omitempty"`
}
```

#### RankedDocument

```go
type RankedDocument struct {
    Content string  `json:"content"`
    Score   float64 `json:"score"`
    Rank    int     `json:"rank"`
}
```

#### LlamaIndexConfig

```go
type LlamaIndexConfig struct {
    BaseURL string        `json:"base_url"`
    Timeout time.Duration `json:"timeout"`
}
```

Default: `http://localhost:8012`, 120s timeout.

#### LlamaIndexHTTPAdapter

```go
type LlamaIndexHTTPAdapter struct { /* unexported fields */ }
```

Implements `LlamaIndexAdapter` using HTTP.

**Constructor:**

| Function | Signature | Description |
|----------|-----------|-------------|
| `NewLlamaIndexHTTPAdapter` | `NewLlamaIndexHTTPAdapter(config *LlamaIndexConfig) *LlamaIndexHTTPAdapter` | Creates adapter. Uses `DefaultLlamaIndexConfig()` if nil. |

**Methods:**

| Method | Signature | Description |
|--------|-----------|-------------|
| `Query` | `(a *LlamaIndexHTTPAdapter) Query(ctx context.Context, query string, topK int) (*QueryResult, error)` | Queries documents via POST `/query`. Defaults topK to 5 if <= 0. |
| `Rerank` | `(a *LlamaIndexHTTPAdapter) Rerank(ctx context.Context, query string, documents []string, topK int) ([]RankedDocument, error)` | Reranks documents via POST `/rerank`. Defaults topK to 5 if <= 0. |
| `Health` | `(a *LlamaIndexHTTPAdapter) Health(ctx context.Context) error` | Checks health via GET `/health`. |

#### ChainResult

```go
type ChainResult struct {
    Result string      `json:"result"`
    Steps  []ChainStep `json:"steps"`
}
```

#### ChainStep

```go
type ChainStep struct {
    Step   string `json:"step"`
    Input  string `json:"input,omitempty"`
    Output string `json:"output,omitempty"`
}
```

#### DecomposeResult

```go
type DecomposeResult struct {
    Subtasks  []Subtask `json:"subtasks"`
    Reasoning string    `json:"reasoning"`
}
```

#### Subtask

```go
type Subtask struct {
    ID           int    `json:"id"`
    Description  string `json:"description"`
    Dependencies []int  `json:"dependencies"`
    Complexity   string `json:"complexity"`
}
```

#### LangChainConfig

```go
type LangChainConfig struct {
    BaseURL string        `json:"base_url"`
    Timeout time.Duration `json:"timeout"`
}
```

Default: `http://localhost:8011`, 120s timeout.

#### LangChainHTTPAdapter

```go
type LangChainHTTPAdapter struct { /* unexported fields */ }
```

Implements `LangChainAdapter` using HTTP.

**Constructor:**

| Function | Signature | Description |
|----------|-----------|-------------|
| `NewLangChainHTTPAdapter` | `NewLangChainHTTPAdapter(config *LangChainConfig) *LangChainHTTPAdapter` | Creates adapter. Uses `DefaultLangChainConfig()` if nil. |

**Methods:**

| Method | Signature | Description |
|--------|-----------|-------------|
| `ExecuteChain` | `(a *LangChainHTTPAdapter) ExecuteChain(ctx context.Context, chainType string, prompt string, variables map[string]interface{}) (*ChainResult, error)` | Executes chain via POST `/chain` with temperature 0.7. |
| `Decompose` | `(a *LangChainHTTPAdapter) Decompose(ctx context.Context, task string, maxSteps int) (*DecomposeResult, error)` | Decomposes task via POST `/decompose`. Defaults maxSteps to 5 if <= 0. |
| `Health` | `(a *LangChainHTTPAdapter) Health(ctx context.Context) error` | Checks health via GET `/health`. |

### Interfaces

#### LlamaIndexAdapter

```go
type LlamaIndexAdapter interface {
    Query(ctx context.Context, query string, topK int) (*QueryResult, error)
    Rerank(ctx context.Context, query string, documents []string, topK int) ([]RankedDocument, error)
    Health(ctx context.Context) error
}
```

#### LangChainAdapter

```go
type LangChainAdapter interface {
    ExecuteChain(ctx context.Context, chainType string, prompt string, variables map[string]interface{}) (*ChainResult, error)
    Decompose(ctx context.Context, task string, maxSteps int) (*DecomposeResult, error)
    Health(ctx context.Context) error
}
```

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `DefaultLlamaIndexConfig` | `DefaultLlamaIndexConfig() *LlamaIndexConfig` | Returns default config (localhost:8012, 120s). |
| `DefaultLangChainConfig` | `DefaultLangChainConfig() *LangChainConfig` | Returns default config (localhost:8011, 120s). |
