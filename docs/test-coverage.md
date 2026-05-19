# Test Coverage — digital.vasic.optimization (round-259)

> Verbatim 2026-05-19 operator mandate: *"all existing tests and Challenges do work in anti-bluff manner - they MUST confirm that all tested codebase really works as expected! We had been in position that all tests do execute with success and all Challenges as well, but in reality the most of the features does not work and can't be used! This MUST NOT be the case and execution of tests and Challenges MUST guarantee the quality, the completition and full usability by end users of the product!"*

CONST-050(B) symbol-to-test ledger. Every exported symbol in
`pkg/{gptcache,prompt,streaming,outlines}` is cross-referenced to the
unit-test name(s) that exercise it AND to the round-259 Challenge runner
section that exercises it against the real Go runtime (real
`gptcache.InMemoryCache` in-process, real `prompt.Template` substitution,
real `streaming.StreamBuffer` flush logic, real `encoding/json`-backed
`outlines.JSONConstrainer`). No metadata-only PASS — every entry below
names the production code path and the runtime evidence channel that
proves it works.

## Anti-bluff posture (round-259)

- **Multi-locale cache round-trip.** `challenges/runner/main.go` Section 1
  builds a real `gptcache.InMemoryCache`, stores 5 locale-distinct
  query/response pairs (en, sr, ja, ar, zh-CN), and asserts every
  `Get(query)` returns byte-exact bytes with `Similarity == 1.0`. PASS
  lines carry `utf8.RuneCountInString` so byte preservation is
  observable, not assumed. Invalidate-then-Get verifies `ErrCacheMiss`.
  Empty-query verifies `ErrInvalidQuery`.
- **Semantic match with deterministic embedder.** Section 2 wires an
  `EmbeddingMatcher` whose `EmbedFunc` is sha256-derived (no external
  service), so identical / paraphrased queries land on identical
  vectors and `CosineSimilarity` returns ~1.0. Exercises `Config`,
  `DefaultConfig`, `Validate`, `WithSimilarityThreshold`,
  `WithMaxEntries`, `NormalizeL2`, `Clear`.
- **Rune-safe template substitution.** Section 3 registers one template
  per locale through `TemplateRegistry`, renders with the non-ASCII
  `Name` placeholder, and asserts byte-exact match for Cyrillic,
  Japanese, Arabic, Han. Includes the unresolved-variable rejection
  contract and the injection-safety contract (a value containing
  `{{Other}}` MUST be preserved verbatim, never re-scanned).
- **Real compression path.** Section 4 runs `Compressor.Optimize` on
  fixture filler prompts and asserts a minimum byte reduction per
  locale; verifies whitespace normalization and token-budget
  truncation; exercises `EstimateTokens` and `DefaultConfig`.
- **Real streaming flush logic.** Section 5 feeds multi-sentence
  payloads into a `StreamBuffer` with `FlushOnSentence`, asserts the
  flushed count meets a per-locale minimum, and exercises `FlushOnWord`,
  `FlushOnLine`, `FlushOnSize`, `Reset`, `Flush`. Section 6 covers
  `TokenCounter` (default + custom ratio + rune-aware `CountCharacters`
  on Cyrillic) and `ChunkMerger` (below-threshold hold, threshold-cross
  emission, `Flush`, `Reset`).
- **Real JSON-schema validation.** Section 7 builds a
  `{name:string,age:integer}` object schema via `SchemaBuilder`, feeds
  every locale's JSON payload through `JSONConstrainer`, asserts
  non-ASCII names survive `encoding/json` unmarshalling, exercises
  prose-embedded JSON extraction, missing-required-field rejection,
  enum validation, array `MinItems`, `ParseSchema`, `IsRequired`,
  `RegexConstrainer` (extract + reject + bad-pattern), the constructor
  helpers, `Schema.String`, `ValidationError.Error`, `ValidateValue`,
  and `ValidationResult.AddError` + `ErrorMessages`.
- **Paired mutation.** Running the gate with `--anti-bluff-mutate`
  plants a deliberate `InMemoryCache -> InMemoryBogus_MUTATED` rename
  in a tmp copy of this ledger, reruns the cross-reference check, and
  asserts the gate exits 99. Proves the symbol-to-test ledger actually
  catches drift instead of rubber-stamping it.

## pkg/gptcache

| Exported symbol | Unit-test coverage | Runner section |
|-----------------|--------------------|----------------|
| `type Cache interface` | every `Test*` in `cache_test.go` | Section 1 (consumed via `InMemoryCache`) |
| `type CachedResponse` | every `Test*` returning a response | Section 1 (Similarity, Response, CachedAt) |
| `type SemanticMatcher interface` | `TestEmbeddingMatcher_*` | Section 2 (deterministic embedder) |
| `type Config` | `TestConfig_Validate`, `TestDefaultConfig` | Section 2 (DefaultConfig + Validate paths) |
| `type ConfigOption` | `TestWithSimilarityThreshold`, `TestWithMaxEntries`, `TestWithTTL` | Section 1 + 2 (functional-options construction) |
| `func DefaultConfig() *Config` | `TestDefaultConfig` | Section 2 |
| `func (*Config) Validate()` | `TestConfig_Validate` | Section 2 |
| `func WithSimilarityThreshold(float64) ConfigOption` | `TestWithSimilarityThreshold` | Section 1, 2 |
| `func WithMaxEntries(int) ConfigOption` | `TestWithMaxEntries` | Section 1, 2 |
| `func WithTTL(time.Duration) ConfigOption` | `TestWithTTL` | Section 1 |
| `var ErrCacheMiss` | `TestCache_GetMiss` | Section 1 (invalidate-then-Get) |
| `var ErrInvalidQuery` | `TestCache_EmptyQuery` | Section 1 (empty-query check) |
| `type InMemoryCache` | every memory_cache `Test*` | Sections 1, 2 |
| `func NewInMemoryCache(opts ...ConfigOption) *InMemoryCache` | `TestNewInMemoryCache_*` | Sections 1, 2 |
| `func NewInMemoryCacheWithConfig(*Config) *InMemoryCache` | `TestNewInMemoryCache_WithConfig` | (unit only) |
| `func (*InMemoryCache) SetMatcher(SemanticMatcher)` | `TestInMemoryCache_Semantic` | Section 2 |
| `func (*InMemoryCache) Get(ctx, query) (*CachedResponse, error)` | `TestInMemoryCache_Get*` | Sections 1, 2 |
| `func (*InMemoryCache) Set(ctx, query, response) error` | `TestInMemoryCache_Set*` | Sections 1, 2 |
| `func (*InMemoryCache) Invalidate(ctx, query) error` | `TestInMemoryCache_Invalidate` | Section 1 |
| `func (*InMemoryCache) Size() int` | `TestInMemoryCache_Size` | Section 1 |
| `func (*InMemoryCache) Clear()` | `TestInMemoryCache_Clear` | Section 2 |
| `func (*InMemoryCache) Config() *Config` | `TestInMemoryCache_Config` | (unit only) |
| `func CosineSimilarity([]float64, []float64) float64` | `TestCosineSimilarity` | Section 2 |
| `func NormalizeL2([]float64) []float64` | `TestNormalizeL2` | Section 2 |
| `type EmbeddingMatcher` | `TestEmbeddingMatcher_*` | Section 2 |
| `func (*EmbeddingMatcher) Similarity(string, string) (float64, error)` | `TestEmbeddingMatcher_Similarity` | Section 2 |

## pkg/prompt

| Exported symbol | Unit-test coverage | Runner section |
|-----------------|--------------------|----------------|
| `type Optimizer interface` | every `Test*Optimize*` | Section 4 (via Compressor) |
| `type Config` | `TestDefaultConfig` | Section 4 |
| `func DefaultConfig() *Config` | `TestDefaultConfig` | Section 4 |
| `type Compressor` | `TestCompressor_*` | Section 4 |
| `func NewCompressor(*Config) *Compressor` | `TestCompressor_*` | Section 4 |
| `func (*Compressor) Optimize(ctx, prompt) (string, error)` | `TestCompressor_Optimize*` | Section 4 (5 locales + whitespace + truncate) |
| `func EstimateTokens(string) int` | `TestEstimateTokens` | Section 4 |
| `type Template` | `TestTemplate_*`, security `TestTemplate_InjectionResistance` | Section 3 |
| `func (*Template) Render(map[string]string) (string, error)` | `TestTemplate_Render*` | Section 3 (rune-safety + unresolved + injection-safety) |
| `type TemplateRegistry` | `TestTemplateRegistry_*` | Section 3 |
| `func NewTemplateRegistry() *TemplateRegistry` | `TestTemplateRegistry_*` | Section 3 |
| `func (*TemplateRegistry) Register(*Template) error` | `TestTemplateRegistry_Register*` | Section 3 |
| `func (*TemplateRegistry) Get(name) (*Template, error)` | `TestTemplateRegistry_Get*` | Section 3 |
| `func (*TemplateRegistry) Remove(name)` | `TestTemplateRegistry_Remove` | Section 3 |
| `func (*TemplateRegistry) List() []string` | `TestTemplateRegistry_List` | Section 3 |
| `func (*TemplateRegistry) Size() int` | `TestTemplateRegistry_Size` | Section 3 |
| `func (*TemplateRegistry) RenderTemplate(name, vars) (string, error)` | `TestTemplateRegistry_RenderTemplate*` | Section 3 (5 locales) |

## pkg/streaming

| Exported symbol | Unit-test coverage | Runner section |
|-----------------|--------------------|----------------|
| `type Buffer interface` | every `Test*Buffer*` | Section 5 |
| `type FlushStrategy` | `TestFlushStrategy_Constants` | Section 5 |
| `const FlushOnWord/FlushOnSentence/FlushOnLine/FlushOnSize` | every strategy `Test*` | Section 5 |
| `type StreamBuffer` | `TestStreamBuffer_*` | Section 5 |
| `func NewStreamBuffer(FlushStrategy, int) *StreamBuffer` | `TestStreamBuffer_*` | Section 5 |
| `func (*StreamBuffer) Add(string) []string` | `TestStreamBuffer_Add*` | Section 5 |
| `func (*StreamBuffer) Flush() string` | `TestStreamBuffer_Flush` | Section 5 |
| `func (*StreamBuffer) Reset()` | `TestStreamBuffer_Reset` | Section 5 |
| `type TokenCounter` | `TestTokenCounter_*` | Section 6 |
| `func NewTokenCounter() *TokenCounter` | `TestNewTokenCounter` | Section 6 |
| `func NewTokenCounterWithRatio(float64) *TokenCounter` | `TestNewTokenCounterWithRatio` | Section 6 |
| `func (*TokenCounter) Count(string) int` | `TestTokenCounter_Count` | Section 6 |
| `func (*TokenCounter) CountWords(string) int` | `TestTokenCounter_CountWords` | Section 6 |
| `func (*TokenCounter) CountCharacters(string) int` | `TestTokenCounter_CountCharacters` | Section 6 (rune-aware Cyrillic) |
| `func (*TokenCounter) Fits(string, int) bool` | `TestTokenCounter_Fits` | Section 6 |
| `type ChunkMerger` | `TestChunkMerger_*` | Section 6 |
| `func NewChunkMerger(int) *ChunkMerger` | `TestNewChunkMerger` | Section 6 |
| `func (*ChunkMerger) Add(string) string` | `TestChunkMerger_Add` | Section 6 |
| `func (*ChunkMerger) Flush() string` | `TestChunkMerger_Flush` | Section 6 |
| `func (*ChunkMerger) Reset()` | `TestChunkMerger_Reset` | Section 6 |
| `type Config` | `TestStreamingConfig_*` | Section 5 |
| `func DefaultConfig() *Config` | `TestStreamingDefaultConfig` | Section 5 |

## pkg/outlines

| Exported symbol | Unit-test coverage | Runner section |
|-----------------|--------------------|----------------|
| `type Schema` | every `Test*Schema*` | Section 7 |
| `func ParseSchema([]byte) (*Schema, error)` | `TestParseSchema*` | Section 7 |
| `func (*Schema) String() string` | `TestSchema_String` | Section 7 |
| `func (*Schema) IsRequired(string) bool` | `TestSchema_IsRequired` | Section 7 |
| `type SchemaBuilder` | every builder `Test*` | Section 7 |
| `func NewSchemaBuilder() *SchemaBuilder` | every builder `Test*` | Section 7 |
| `func (*SchemaBuilder) Object/Array/StringType/NumberType/IntegerType/BooleanType *SchemaBuilder` | per-type `Test*` | Section 7 |
| `func (*SchemaBuilder) Property(string, *Schema) *SchemaBuilder` | `TestSchemaBuilder_Property` | Section 7 |
| `func (*SchemaBuilder) RequiredProps(...string) *SchemaBuilder` | `TestSchemaBuilder_Required` | Section 7 |
| `func (*SchemaBuilder) Items(*Schema) *SchemaBuilder` | `TestSchemaBuilder_Items` | Section 7 |
| `func (*SchemaBuilder) EnumValues(...interface{}) *SchemaBuilder` | `TestSchemaBuilder_Enum` | Section 7 |
| `func (*SchemaBuilder) SetPattern(string) *SchemaBuilder` | `TestSchemaBuilder_Pattern` | Section 7 |
| `func (*SchemaBuilder) SetDescription(string) *SchemaBuilder` | `TestSchemaBuilder_Description` | Section 7 |
| `func (*SchemaBuilder) Build() *Schema` | every builder `Test*` | Section 7 |
| `func StringSchema() *Schema` | `TestHelperSchemas` | Section 7 |
| `func IntegerSchema() *Schema` | `TestHelperSchemas` | Section 7 |
| `func NumberSchema() *Schema` | `TestHelperSchemas` | Section 7 |
| `func BooleanSchema() *Schema` | `TestHelperSchemas` | Section 7 |
| `func ArraySchema(*Schema) *Schema` | `TestHelperSchemas` | Section 7 |
| `func ObjectSchema(map, ...string) *Schema` | `TestHelperSchemas` | Section 7 |
| `type Constrainer interface` | every `Test*Constrainer*` | Section 7 |
| `type ValidationError` | `TestValidationError_Error` | Section 7 |
| `func (*ValidationError) Error() string` | `TestValidationError_Error` | Section 7 |
| `type ValidationResult` | `TestValidate_*` | Section 7 |
| `func (*ValidationResult) AddError(string, string)` | `TestValidationResult_AddError` | Section 7 |
| `func (*ValidationResult) ErrorMessages() []string` | `TestValidationResult_ErrorMessages` | Section 7 |
| `type JSONConstrainer` | `TestJSONConstrainer_*` | Section 7 |
| `func NewJSONConstrainer() *JSONConstrainer` | `TestJSONConstrainer_*` | Section 7 |
| `func (*JSONConstrainer) Constrain(string, *Schema) (string, error)` | `TestJSONConstrainer_Constrain*` | Section 7 (5 locales + prose + missing-required) |
| `type RegexConstrainer` | `TestRegexConstrainer_*` | Section 7 |
| `func NewRegexConstrainer(string) (*RegexConstrainer, error)` | `TestRegexConstrainer_*` | Section 7 |
| `func (*RegexConstrainer) Constrain(string, *Schema) (string, error)` | `TestRegexConstrainer_Constrain*` | Section 7 |
| `func Validate(string, *Schema) *ValidationResult` | `TestValidate_*` | Section 7 |
| `func ValidateValue(interface{}, *Schema, string) *ValidationResult` | `TestValidateValue_*` | Section 7 |

## Round-259 artefacts inventory

| Artefact | Path | Purpose |
|----------|------|---------|
| Runner | `challenges/runner/main.go` | Real multi-locale exerciser (7 sections, 85 PASS lines) |
| Mutation gate | `challenges/scripts/optimization_describe_challenge.sh` | Cross-reference + paired-mutation enforcement |
| Multi-locale fixture | `tests/fixtures/optimization/payloads.json` | 5 locales: en, sr, ja, ar, zh-CN |
| README guarantees | `README.md` | Anti-bluff section + quick start |
| Ledger | `docs/test-coverage.md` (this file) | Symbol -> test cross-reference |

## Inherited governance challenges (still in scope)

| Script | Purpose |
|--------|---------|
| `challenges/scripts/no_suspend_calls_challenge.sh` | CONST-033 host-power scan |
| `challenges/scripts/host_no_auto_suspend_challenge.sh` | CONST-033 host hardening |
| `challenges/scripts/chaos_failure_injection_challenge.sh` | CONST-050(B) chaos type |
| `challenges/scripts/ddos_health_flood_challenge.sh` | CONST-050(B) DDoS type |
| `challenges/scripts/scaling_horizontal_challenge.sh` | CONST-050(B) scaling type |
| `challenges/scripts/stress_sustained_load_challenge.sh` | CONST-050(B) stress type |
| `challenges/scripts/ui_terminal_interaction_challenge.sh` | CONST-050(B) UI type |
| `challenges/scripts/ux_end_to_end_flow_challenge.sh` | CONST-050(B) UX type |
| `challenges/scripts/optimization_compile_challenge.sh` | Compile sanity for the module |
| `challenges/scripts/optimization_functionality_challenge.sh` | Package-presence + interface presence |
| `challenges/scripts/optimization_unit_challenge.sh` | Unit-suite runner |
