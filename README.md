# Optimization

Generic, reusable LLM optimization module for Go applications.

## Packages

- **gptcache** - Semantic caching for LLM responses using embedding similarity
- **prompt** - Prompt optimization with compression and template management
- **streaming** - Streaming optimizations with configurable buffers and chunk merging
- **outlines** - Structured output constraints with JSON Schema and regex validation
- **sglang** - SGLang integration interface for efficient LLM serving
- **adapter** - LLM framework adapters for LlamaIndex and LangChain

## Installation

```bash
go get digital.vasic.optimization
```

## Usage

```go
import (
    "digital.vasic.optimization/pkg/gptcache"
    "digital.vasic.optimization/pkg/prompt"
    "digital.vasic.optimization/pkg/streaming"
    "digital.vasic.optimization/pkg/outlines"
)
```

## Testing

```bash
go test ./... -count=1 -race
```
