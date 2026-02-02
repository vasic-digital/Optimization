# CLAUDE.md - Optimization Module

## Overview

`digital.vasic.optimization` is a generic, reusable Go module providing LLM optimization
capabilities including semantic caching, prompt optimization, streaming enhancements,
structured output constraints, SGLang integration, and LLM framework adapters.

**Module**: `digital.vasic.optimization` (Go 1.24+)
**Dependencies**: `github.com/stretchr/testify` (testing only)

## Build & Test

```bash
go build ./...
go test ./... -count=1 -race
go test ./... -short              # Unit tests only
go test -bench=. ./...            # Benchmarks
```

## Code Style

- Standard Go conventions, `gofmt` formatting
- Imports grouped: stdlib, third-party, internal (blank line separated)
- Line length <= 100 chars
- Naming: `camelCase` private, `PascalCase` exported, acronyms all-caps
- Errors: always check, wrap with `fmt.Errorf("...: %w", err)`
- Tests: table-driven, `testify`, naming `Test<Struct>_<Method>_<Scenario>`

## Package Structure

| Package | Purpose |
|---------|---------|
| `pkg/gptcache` | Semantic caching for LLM responses with embedding similarity |
| `pkg/prompt` | Prompt optimization: compression, templates, registry |
| `pkg/streaming` | Streaming optimizations: buffers, token counting, chunk merging |
| `pkg/outlines` | Structured output constraints: JSON Schema, regex constrainers |
| `pkg/sglang` | SGLang integration: client, programs, prefix caching |
| `pkg/adapter` | LLM framework adapters: LlamaIndex, LangChain |

## Key Interfaces

- `gptcache.Cache` -- Semantic cache (Get, Set, Invalidate)
- `prompt.Optimizer` -- Prompt optimization (Optimize)
- `outlines.Constrainer` -- Output constraint (Constrain)
- `sglang.Client` -- SGLang generation (Generate)
- `adapter.LlamaIndexAdapter` -- LlamaIndex integration
- `adapter.LangChainAdapter` -- LangChain integration

## Design Patterns

- **Strategy**: Eviction policies, similarity metrics, flush strategies
- **Functional Options**: ConfigOption for cache configuration
- **Builder**: SchemaBuilder for JSON Schema construction
- **Interface Segregation**: Small focused interfaces throughout
- **Template Method**: BaseStreamHandler for stream event handling

## Commit Style

Conventional Commits: `feat(gptcache): add semantic similarity caching`
