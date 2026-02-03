# AGENTS.md - Optimization Module

## Multi-Agent Coordination Guide

This document describes how AI agents should coordinate when working on the
`digital.vasic.optimization` module, which provides generic LLM optimization
capabilities across 6 packages.

## Module Scope

- **Module path**: `digital.vasic.optimization`
- **Language**: Go 1.24+
- **External dependencies**: `github.com/stretchr/testify` (testing only)
- **Packages**: gptcache, outlines, streaming, sglang, adapter, prompt

## Agent Roles

### Cache Agent
- **Scope**: `pkg/gptcache/`
- **Responsibilities**: Semantic caching, similarity computation, eviction,
  cache configuration, embedding-based matching.
- **Key interfaces**: `Cache`, `SemanticMatcher`
- **Key types**: `InMemoryCache`, `EmbeddingMatcher`, `CachedResponse`, `Config`
- **Coordination**: When modifying the `Cache` interface, notify the
  Integration Agent -- any consumer of the cache contract must be updated.
  Changes to `SemanticMatcher` may affect embedding provider integrations in
  the parent HelixAgent project.

### Prompt Agent
- **Scope**: `pkg/prompt/`
- **Responsibilities**: Prompt compression, template management, variable
  substitution, token estimation.
- **Key interfaces**: `Optimizer`
- **Key types**: `Compressor`, `Template`, `TemplateRegistry`, `Config`
- **Coordination**: Changes to the `Optimizer` interface affect any pipeline
  that optimizes prompts before sending to an LLM. The Prompt Agent must
  coordinate with the Cache Agent when prompt normalization affects cache keys.

### Streaming Agent
- **Scope**: `pkg/streaming/`
- **Responsibilities**: Stream buffer strategies, token counting, chunk
  merging, flush configuration.
- **Key interfaces**: `Buffer`
- **Key types**: `StreamBuffer`, `TokenCounter`, `ChunkMerger`, `Config`,
  `FlushStrategy`
- **Coordination**: Flush strategies affect downstream consumers of streamed
  tokens. The Streaming Agent must coordinate with the SGLang Agent when
  streaming responses from SGLang servers.

### Outlines Agent
- **Scope**: `pkg/outlines/`
- **Responsibilities**: JSON Schema construction/validation, regex constraining,
  structured output enforcement.
- **Key interfaces**: `Constrainer`
- **Key types**: `Schema`, `SchemaBuilder`, `JSONConstrainer`,
  `RegexConstrainer`, `ValidationResult`, `ValidationError`
- **Coordination**: Schema validation changes affect all consumers that enforce
  structured LLM output. Coordinate with the Prompt Agent when schemas are
  injected into prompts.

### SGLang Agent
- **Scope**: `pkg/sglang/`
- **Responsibilities**: SGLang HTTP client, program execution, health checks,
  prefix caching integration.
- **Key interfaces**: `Client`
- **Key types**: `HTTPClient`, `Program`, `Config`
- **Coordination**: The SGLang Agent must coordinate with the Streaming Agent
  for streaming generation, and with the Cache Agent when prefix caching
  overlaps with semantic caching.

### Adapter Agent
- **Scope**: `pkg/adapter/`
- **Responsibilities**: LlamaIndex and LangChain HTTP adapter integration,
  document querying, chain execution, task decomposition.
- **Key interfaces**: `LlamaIndexAdapter`, `LangChainAdapter`
- **Key types**: `LlamaIndexHTTPAdapter`, `LangChainHTTPAdapter`,
  `QueryResult`, `ChainResult`, `DecomposeResult`, `RankedDocument`,
  `Source`, `Subtask`, `ChainStep`
- **Coordination**: Adapter changes affect HelixAgent's optimization layer.
  Coordinate with the Cache Agent for caching adapter responses, and with
  the Prompt Agent for template-based prompts sent through adapters.

## Cross-Package Dependencies

```
gptcache  <--  (independent, core caching)
prompt    <--  (independent, core prompt optimization)
streaming <--  (independent, core streaming)
outlines  <--  (independent, core structured output)
sglang    <--  (independent, SGLang client)
adapter   <--  (independent, framework adapters)
```

All 6 packages are independent with no cross-package imports within this
module. They are designed to be composed at the integration layer (e.g.,
HelixAgent's `internal/optimization/`).

## Coordination Rules

1. **Interface changes require cross-agent review.** Any modification to an
   exported interface (`Cache`, `SemanticMatcher`, `Optimizer`, `Buffer`,
   `Constrainer`, `Client`, `LlamaIndexAdapter`, `LangChainAdapter`) must
   be reviewed by all agents whose packages may be affected.

2. **No cross-package imports.** This module maintains strict package
   independence. Composition happens at the consumer level.

3. **Test isolation.** Each package must have self-contained tests. Unit tests
   use mocks/stubs; only integration tests in the parent project use live
   services.

4. **Backward compatibility.** Interface additions must be additive. Breaking
   changes require a major version bump coordinated across all agents.

5. **Configuration defaults.** Each package provides a `DefaultConfig()`
   function. Changes to defaults must be coordinated with any agent whose
   package depends on those defaults for sensible behavior.

## Development Workflow

1. **Before starting**: Read this file and `CLAUDE.md` for context.
2. **Making changes**: Run `go build ./...` and `go test ./... -count=1 -race`
   after every change.
3. **Adding exported types**: Update `docs/API_REFERENCE.md`.
4. **Changing interfaces**: Update `docs/ARCHITECTURE.md` and the class diagram
   in `docs/diagrams/class.mmd`.
5. **Before committing**: Ensure `go fmt ./...` and `go vet ./...` pass.
   Use Conventional Commits: `feat(gptcache): add LRU eviction policy`.
