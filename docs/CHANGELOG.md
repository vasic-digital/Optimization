# Changelog

All notable changes to the `digital.vasic.optimization` module will be
documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-01-01

### Added

#### Package `gptcache`
- `Cache` interface with `Get`, `Set`, and `Invalidate` methods.
- `SemanticMatcher` interface for pluggable similarity computation.
- `InMemoryCache` implementation with SHA-256 hash-based exact matching,
  optional semantic matching, TTL expiration, and FIFO eviction.
- `EmbeddingMatcher` implementation using cosine similarity over embedding
  vectors, with fallback to case-insensitive exact matching.
- `Config` struct with `DefaultConfig()` and `Validate()`.
- Functional options: `WithSimilarityThreshold`, `WithMaxEntries`, `WithTTL`.
- Utility functions: `CosineSimilarity`, `NormalizeL2`.
- Sentinel errors: `ErrCacheMiss`, `ErrInvalidQuery`.
- Full test coverage with table-driven tests.

#### Package `prompt`
- `Optimizer` interface for prompt optimization.
- `Compressor` implementation with whitespace normalization, redundant phrase
  removal (8 filler phrases), and word-based token truncation.
- `Template` struct with `{{variable}}` placeholder rendering and unresolved
  variable detection.
- `TemplateRegistry` for thread-safe template management with register, get,
  remove, list, size, and render-by-name operations.
- `Config` struct with `DefaultConfig()`.
- `EstimateTokens` utility function.
- Full test coverage with table-driven tests.

#### Package `streaming`
- `Buffer` interface with `Add`, `Flush`, and `Reset` methods.
- `StreamBuffer` implementation with four flush strategies:
  `FlushOnWord`, `FlushOnSentence`, `FlushOnLine`, `FlushOnSize`.
- `TokenCounter` with configurable tokens-per-word ratio, word counting,
  character counting, and fits-within-limit checking.
- `ChunkMerger` for combining small stream chunks into larger pieces.
- `Config` struct with `DefaultConfig()`.
- `FlushStrategy` typed constants.
- Full test coverage with table-driven tests.

#### Package `outlines`
- `Constrainer` interface for constraining LLM output.
- `Schema` struct supporting full JSON Schema subset: object, array, string,
  number, integer, boolean, enum, pattern, min/max length, min/max value,
  min/max items, additionalProperties, oneOf/anyOf/allOf, description, format.
- `SchemaBuilder` with fluent API for constructing schemas.
- `JSONConstrainer` for extracting and validating JSON from LLM output,
  including embedded JSON extraction from surrounding text.
- `RegexConstrainer` for validating output against compiled regex patterns
  with substring fallback matching.
- `ValidationResult` and `ValidationError` for structured error reporting.
- `Validate` function for direct JSON Schema validation.
- Helper constructors: `StringSchema`, `IntegerSchema`, `NumberSchema`,
  `BooleanSchema`, `ArraySchema`, `ObjectSchema`.
- `ParseSchema` for parsing schemas from JSON bytes.
- Full test coverage with table-driven tests.

#### Package `sglang`
- `Client` interface with `Generate` and `Health` methods.
- `HTTPClient` implementation using OpenAI-compatible `/v1/chat/completions`
  endpoint with context support, configurable timeout, and error handling.
- `Program` struct for specifying generation parameters (system/user prompts,
  temperature, max tokens, top-p, stop sequences).
- `Config` struct with `DefaultConfig()`.
- Full test coverage using `httptest` servers.

#### Package `adapter`
- `LlamaIndexAdapter` interface with `Query`, `Rerank`, and `Health` methods.
- `LlamaIndexHTTPAdapter` implementation using HTTP with JSON serialization.
- `LangChainAdapter` interface with `ExecuteChain`, `Decompose`, and `Health`
  methods.
- `LangChainHTTPAdapter` implementation using HTTP with JSON serialization.
- Data types: `QueryResult`, `Source`, `RankedDocument`, `ChainResult`,
  `ChainStep`, `DecomposeResult`, `Subtask`.
- Config structs with `DefaultLlamaIndexConfig()` and `DefaultLangChainConfig()`.
- Full test coverage using `httptest` servers.

#### Documentation
- `CLAUDE.md` -- Module overview and development standards.
- `README.md` -- Quick start and package summary.
- `AGENTS.md` -- Multi-agent coordination guide.
- `docs/USER_GUIDE.md` -- Comprehensive usage guide with code examples.
- `docs/ARCHITECTURE.md` -- Design decisions and patterns.
- `docs/API_REFERENCE.md` -- Complete API reference for all 6 packages.
- `docs/CONTRIBUTING.md` -- Contribution guide.
- `docs/CHANGELOG.md` -- This file.
- `docs/diagrams/architecture.mmd` -- Mermaid package relationship diagram.
- `docs/diagrams/sequence.mmd` -- Mermaid cache-hit optimization flow.
- `docs/diagrams/class.mmd` -- Mermaid class diagram for interfaces.
