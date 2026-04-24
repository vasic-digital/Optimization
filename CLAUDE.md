# CLAUDE.md - Optimization Module


## Definition of Done

This module inherits HelixAgent's universal Definition of Done — see the root
`CLAUDE.md` and `docs/development/definition-of-done.md`. In one line: **no
task is done without pasted output from a real run of the real system in the
same session as the change.** Coverage and green suites are not evidence.

### Acceptance demo for this module

```bash
# Semantic cache hits+eviction + prompt compression end-to-end
cd Optimization && GOMAXPROCS=2 nice -n 19 go test -count=1 -race -v \
  -run 'TestCacheWithSemanticMatcher_Integration|TestCacheEviction_Integration|TestPromptCompressorWithTemplates_Integration' \
  ./tests/integration/...
```
Expect: three tests PASS; gptcache-style similarity matching + compression on real embeddings.


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

## Integration Seams

| Direction | Sibling modules |
|-----------|-----------------|
| Upstream (this module imports) | none |
| Downstream (these import this module) | HelixLLM |

*Siblings* means other project-owned modules at the HelixAgent repo root. The root HelixAgent app and external systems are not listed here — the list above is intentionally scoped to module-to-module seams, because drift *between* sibling modules is where the "tests pass, product broken" class of bug most often lives. See root `CLAUDE.md` for the rules that keep these seams contract-tested.
