# Contributing to the Optimization Module

Thank you for your interest in contributing to `digital.vasic.optimization`.
This document outlines the process and standards for contributing.

## Prerequisites

- Go 1.24 or later
- Git with SSH configured
- `gofmt` and `goimports` (included with Go)
- `golangci-lint` (recommended)

## Getting Started

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd Optimization
   ```

2. Verify the build:
   ```bash
   go build ./...
   ```

3. Run all tests:
   ```bash
   go test ./... -count=1 -race
   ```

## Development Workflow

### Branch Naming

Use conventional branch prefixes:
- `feat/<description>` -- New features
- `fix/<description>` -- Bug fixes
- `refactor/<description>` -- Refactoring without behavior change
- `test/<description>` -- Test additions or improvements
- `docs/<description>` -- Documentation changes
- `chore/<description>` -- Tooling, CI, or dependency updates

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>
```

**Types**: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `perf`

**Scopes**: `gptcache`, `prompt`, `streaming`, `outlines`, `sglang`, `adapter`

**Examples**:
```
feat(gptcache): add LRU eviction policy
fix(outlines): handle nested array validation
refactor(streaming): simplify sentence detection logic
test(adapter): add LangChain decompose timeout test
docs(prompt): update template rendering examples
```

## Code Standards

### Style

- Standard Go conventions per [Effective Go](https://go.dev/doc/effective_go).
- Format with `gofmt`. Use `goimports` for import grouping.
- Line length: 100 characters maximum.
- Imports grouped: stdlib, third-party, internal (blank line separated).

### Naming

- Private: `camelCase`
- Exported: `PascalCase`
- Constants: `UPPER_SNAKE_CASE` (or `PascalCase` for typed constants)
- Acronyms: all caps (`HTTP`, `URL`, `ID`, `JSON`, `TTL`)
- Receivers: 1-2 letters (`c` for cache, `b` for buffer, `m` for merger)

### Error Handling

- Always check errors.
- Wrap errors with context: `fmt.Errorf("failed to X: %w", err)`.
- Use sentinel errors for expected conditions (e.g., `ErrCacheMiss`).
- Use `defer` for cleanup.

### Interfaces

- Keep interfaces small and focused (1-3 methods).
- Accept interfaces, return concrete structs.
- Verify implementations at compile time:
  ```go
  var _ Cache = (*InMemoryCache)(nil)
  ```

### Concurrency

- Pass `context.Context` as the first parameter.
- Use `sync.RWMutex` for shared state.
- Document whether types are safe for concurrent use.

## Testing Standards

### Test Organization

- Tests live alongside source files (`*_test.go`).
- Use table-driven tests with `testify`.
- Name tests: `Test<Struct>_<Method>_<Scenario>`.

### Test Requirements

- All exported types and functions must have tests.
- Interface compliance tests:
  ```go
  func TestMyType_ImplementsInterface(t *testing.T) {
      var _ MyInterface = (*MyType)(nil)
  }
  ```
- Edge cases: empty inputs, nil parameters, boundary values, error paths.

### Running Tests

```bash
# All tests with race detection.
go test ./... -count=1 -race

# Specific package.
go test -v ./pkg/gptcache/...

# Specific test.
go test -v -run TestInMemoryCache_Get_ExactMatch ./pkg/gptcache/...

# Benchmarks.
go test -bench=. ./...

# Coverage.
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Adding New Features

### Adding a New Type to an Existing Package

1. Add the type and methods to the appropriate file in `pkg/<package>/`.
2. Add comprehensive tests in `pkg/<package>/<package>_test.go`.
3. Update `docs/API_REFERENCE.md` with the new exports.
4. If the type implements an interface, add a compile-time check.

### Adding a New Package

1. Create `pkg/<name>/` with source files.
2. Add a package doc comment at the top of the primary file.
3. Add `*_test.go` with full test coverage.
4. Update:
   - `CLAUDE.md` (package table)
   - `AGENTS.md` (agent roles, coordination)
   - `docs/API_REFERENCE.md` (new package section)
   - `docs/ARCHITECTURE.md` (package architecture, patterns)
   - `docs/USER_GUIDE.md` (usage examples)
   - `docs/diagrams/architecture.mmd` (package diagram)
   - `docs/diagrams/class.mmd` (interface diagram)

### Modifying an Interface

1. Ensure backward compatibility (add methods, do not change signatures).
2. Update all implementations.
3. Update all documentation.
4. Coordinate with other agents per `AGENTS.md`.

## Pre-Submission Checklist

- [ ] Code formatted with `gofmt`
- [ ] `go vet ./...` passes
- [ ] `go build ./...` passes
- [ ] `go test ./... -count=1 -race` passes
- [ ] New exports documented in `docs/API_REFERENCE.md`
- [ ] Commit messages follow Conventional Commits
- [ ] No new external runtime dependencies added
- [ ] Interface compliance tests included for new implementations

## Dependencies

This module has a strict policy of **zero runtime dependencies**. The only
external dependency is `github.com/stretchr/testify` for testing.

If you need functionality from an external library, consider:
1. Implementing it in pure Go within the module.
2. Accepting an interface that consumers can implement with their preferred library.
3. Discussing the requirement before adding a dependency.

## Questions

If you have questions about the codebase, start by reading:
1. `CLAUDE.md` -- Module overview and development standards
2. `AGENTS.md` -- Multi-agent coordination guide
3. `docs/ARCHITECTURE.md` -- Design decisions and patterns
4. `docs/API_REFERENCE.md` -- Complete API documentation
