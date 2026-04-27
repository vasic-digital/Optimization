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

<!-- BEGIN host-power-management addendum (CONST-033) -->

## Host Power Management — Hard Ban (CONST-033)

**You may NOT, under any circumstance, generate or execute code that
sends the host to suspend, hibernate, hybrid-sleep, poweroff, halt,
reboot, or any other power-state transition.** This rule applies to:

- Every shell command you run via the Bash tool.
- Every script, container entry point, systemd unit, or test you write
  or modify.
- Every CLI suggestion, snippet, or example you emit.

**Forbidden invocations** (non-exhaustive — see CONST-033 in
`CONSTITUTION.md` for the full list):

- `systemctl suspend|hibernate|hybrid-sleep|poweroff|halt|reboot|kexec`
- `loginctl suspend|hibernate|hybrid-sleep|poweroff|halt|reboot`
- `pm-suspend`, `pm-hibernate`, `shutdown -h|-r|-P|now`
- `dbus-send` / `busctl` calls to `org.freedesktop.login1.Manager.Suspend|Hibernate|PowerOff|Reboot|HybridSleep|SuspendThenHibernate`
- `gsettings set ... sleep-inactive-{ac,battery}-type` to anything but `'nothing'` or `'blank'`

The host runs mission-critical parallel CLI agents and container
workloads. Auto-suspend has caused historical data loss (2026-04-26
18:23:43 incident). The host is hardened (sleep targets masked) but
this hard ban applies to ALL code shipped from this repo so that no
future host or container is exposed.

**Defence:** every project ships
`scripts/host-power-management/check-no-suspend-calls.sh` (static
scanner) and
`challenges/scripts/no_suspend_calls_challenge.sh` (challenge wrapper).
Both MUST be wired into the project's CI / `run_all_challenges.sh`.

**Full background:** `docs/HOST_POWER_MANAGEMENT.md` and `CONSTITUTION.md` (CONST-033).

<!-- END host-power-management addendum (CONST-033) -->



<!-- CONST-035 anti-bluff addendum (cascaded) -->

## CONST-035 — Anti-Bluff Tests & Challenges (mandatory; inherits from root)

Tests and Challenges in this submodule MUST verify the product, not
the LLM's mental model of the product. A test that passes when the
feature is broken is worse than a missing test — it gives false
confidence and lets defects ship to users. Functional probes at the
protocol layer are mandatory:

- TCP-open is the FLOOR, not the ceiling. Postgres → execute
  `SELECT 1`. Redis → `PING` returns `PONG`. ChromaDB → `GET
  /api/v1/heartbeat` returns 200. MCP server → TCP connect + valid
  JSON-RPC handshake. HTTP gateway → real request, real response,
  non-empty body.
- Container `Up` is NOT application healthy. A `docker/podman ps`
  `Up` status only means PID 1 is running; the application may be
  crash-looping internally.
- No mocks/fakes outside unit tests (already CONST-030; CONST-035
  raises the cost of a mock-driven false pass to the same severity
  as a regression).
- Re-verify after every change. Don't assume a previously-passing
  test still verifies the same scope after a refactor.
- Verification of CONST-035 itself: deliberately break the feature
  (e.g. `kill <service>`, swap a password). The test MUST fail. If
  it still passes, the test is non-conformant and MUST be tightened.

## CONST-033 clarification — distinguishing host events from sluggishness

Heavy container builds (BuildKit pulling many GB of layers, parallel
podman/docker compose-up across many services) can make the host
**appear** unresponsive — high load average, slow SSH, watchers
timing out. **This is NOT a CONST-033 violation.** Suspend / hibernate
/ logout are categorically different events. Distinguish via:

- `uptime` — recent boot? if so, the host actually rebooted.
- `loginctl list-sessions` — session(s) still active? if yes, no logout.
- `journalctl ... | grep -i 'will suspend\|hibernate'` — zero broadcasts
  since the CONST-033 fix means no suspend ever happened.
- `dmesg | grep -i 'killed process\|out of memory'` — OOM kills are
  also NOT host-power events; they're memory-pressure-induced and
  require their own separate fix (lower per-container memory limits,
  reduce parallelism).

A sluggish host under build pressure recovers when the build finishes;
a suspended host requires explicit unsuspend (and CONST-033 should
make that impossible by hardening `IdleAction=ignore` +
`HandleSuspendKey=ignore` + masked `sleep.target`,
`suspend.target`, `hibernate.target`, `hybrid-sleep.target`).

If you observe what looks like a suspend during heavy builds, the
correct first action is **not** "edit CONST-033" but `bash
challenges/scripts/host_no_auto_suspend_challenge.sh` to confirm the
hardening is intact. If hardening is intact AND no suspend
broadcast appears in journal, the perceived event was build-pressure
sluggishness, not a power transition.
