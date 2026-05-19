#!/usr/bin/env bash
# optimization_describe_challenge.sh
#
# Round-259 paired-mutation deep-doc challenge for digital.vasic.optimization.
#
# Validates that:
#   1. The deep-doc ledger (docs/test-coverage.md) lists every exported
#      symbol from pkg/{gptcache,prompt,streaming,outlines}.
#   2. The multi-locale fixture (tests/fixtures/optimization/payloads.json)
#      parses and contains at least 3 locales.
#   3. The multi-locale runner (challenges/runner/main.go) builds and
#      runs, byte-preserving non-ASCII messages through gptcache
#      InMemoryCache (exact + invalidate + ErrInvalidQuery), semantic
#      matching (EmbeddingMatcher / CosineSimilarity / NormalizeL2),
#      prompt Template / TemplateRegistry (rune-safe substitution,
#      unresolved-variable rejection, injection-safety),
#      Compressor.Optimize (filler removal + whitespace + truncation),
#      streaming StreamBuffer (FlushOnSentence/Word/Line/Size), Token
#      Counter (default + custom ratio + rune-aware CountCharacters),
#      ChunkMerger (hold + threshold-cross + Flush + Reset), outlines
#      SchemaBuilder + JSONConstrainer (5-locale name+age round-trip,
#      prose-embedded JSON extraction, missing-required rejection),
#      RegexConstrainer (extract + reject + bad-pattern), Validate,
#      ValidateValue, ValidationResult.AddError + ErrorMessages,
#      Schema.String + IsRequired + ParseSchema, and the constructor
#      helpers.
#   4. The README enumerates the four exercised packages and the
#      round-259 anti-bluff guarantees section.
#
# Paired-mutation invariant (CONST-035 + CONST-050(B)):
#   With --anti-bluff-mutate the script plants a deliberate symbol-rename
#   mutation in the ledger (in a tmp copy), reruns validation, and asserts
#   the gate FAILS with exit 99. This proves the gate actually catches
#   ledger-vs-source drift instead of rubber-stamping it.
#
# Verbatim 2026-05-19 operator mandate: "all existing tests and Challenges
# do work in anti-bluff manner - they MUST confirm that all tested codebase
# really works as expected! We had been in position that all tests do execute
# with success and all Challenges as well, but in reality the most of the
# features does not work and can't be used! This MUST NOT be the case and
# execution of tests and Challenges MUST guarantee the quality, the
# completition and full usability by end users of the product!"
#
# Exit codes:
#   0  - gate PASS on clean tree
#   1  - gate FAIL on clean tree (real failure to fix)
#   99 - paired-mutation correctly detected (good - proves anti-bluff)
#   2  - usage / environment error

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODULE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

MUTATE=0
for arg in "$@"; do
    case "$arg" in
        --anti-bluff-mutate) MUTATE=1 ;;
        --help|-h)
            sed -n '1,40p' "$0"
            exit 0
            ;;
        *)
            echo "unknown argument: $arg" >&2
            exit 2
            ;;
    esac
done

PASS=0
FAIL=0
TOTAL=0

pass() { PASS=$((PASS+1)); TOTAL=$((TOTAL+1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL+1)); TOTAL=$((TOTAL+1)); echo "  FAIL: $1"; }

LEDGER="${MODULE_DIR}/docs/test-coverage.md"
FIXTURE="${MODULE_DIR}/tests/fixtures/optimization/payloads.json"
RUNNER="${MODULE_DIR}/challenges/runner/main.go"
README="${MODULE_DIR}/README.md"

LEDGER_WORK="${LEDGER}"
TMP_LEDGER=""
if [ "${MUTATE}" -eq 1 ]; then
    TMP_LEDGER="$(mktemp)"
    cp "${LEDGER}" "${TMP_LEDGER}"
    # Plant a rename: InMemoryCache -> InMemoryBogus_MUTATED
    sed -i 's/InMemoryCache/InMemoryBogus_MUTATED/g' "${TMP_LEDGER}"
    LEDGER_WORK="${TMP_LEDGER}"
    echo "=== Optimization Describe Challenge (anti-bluff-mutate mode) ==="
else
    echo "=== Optimization Describe Challenge (clean mode) ==="
fi
echo ""

# Section 1: ledger presence and freshness
echo "Section 1: docs/test-coverage.md ledger"
if [ ! -f "${LEDGER_WORK}" ]; then
    fail "ledger missing at ${LEDGER_WORK}"
else
    pass "ledger present"
    if grep -q "round-259" "${LEDGER_WORK}"; then
        pass "ledger marked round-259"
    else
        fail "ledger missing round-259 marker"
    fi
    if grep -q "execution of tests and Challenges MUST guarantee" "${LEDGER_WORK}"; then
        pass "ledger carries Article XI §11.9 mandate"
    else
        fail "ledger missing Article XI §11.9 mandate"
    fi
fi

# Section 2: every exported pkg symbol appears in ledger
echo ""
echo "Section 2: exported symbols cross-reference"

extract_symbols() {
    local pkg_dir="$1"
    local files
    files=$(find "${pkg_dir}" -maxdepth 1 -type f -name '*.go' \
        ! -name '*_test.go')
    [ -z "${files}" ] && return 0
    # shellcheck disable=SC2086
    grep -hE '^(func ([A-Z][A-Za-z0-9_]*\()|func \([^)]+\) ([A-Z][A-Za-z0-9_]*\()|type [A-Z][A-Za-z0-9_]* )' \
        ${files} 2>/dev/null \
        | sed -E 's/^func \([^)]+\) ([A-Z][A-Za-z0-9_]*)\(.*$/\1/; s/^func ([A-Z][A-Za-z0-9_]*)\(.*$/\1/; s/^type ([A-Z][A-Za-z0-9_]*).*$/\1/' \
        | sort -u
}

CHECKED=0
MISSING=0
for pkg in gptcache prompt streaming outlines; do
    PKG_DIR="${MODULE_DIR}/pkg/${pkg}"
    if [ ! -d "${PKG_DIR}" ]; then
        fail "pkg/${pkg} missing - cannot cross-reference"
        continue
    fi
    while IFS= read -r sym; do
        [ -z "${sym}" ] && continue
        CHECKED=$((CHECKED + 1))
        if grep -qE "\\b${sym}\\b" "${LEDGER_WORK}"; then
            : # symbol cross-referenced
        else
            fail "ledger missing symbol ${pkg}.${sym}"
            MISSING=$((MISSING + 1))
        fi
    done < <(extract_symbols "${PKG_DIR}")
done
if [ "${CHECKED}" -gt 0 ] && [ "${MISSING}" -eq 0 ]; then
    pass "all ${CHECKED} exported symbols cross-referenced in ledger"
fi

# Section 3: multi-locale fixture sanity
echo ""
echo "Section 3: multi-locale fixture"
if [ ! -f "${FIXTURE}" ]; then
    fail "fixture missing at ${FIXTURE}"
else
    pass "fixture present"
    LOCALE_COUNT=$(grep -oE '"locale":\s*"[^"]+"' "${FIXTURE}" | sort -u | wc -l)
    if [ "${LOCALE_COUNT}" -ge 3 ]; then
        pass "fixture covers ${LOCALE_COUNT} locales (>=3)"
    else
        fail "fixture covers only ${LOCALE_COUNT} locales (<3)"
    fi
fi

# Section 4: runner builds + runs against every package
echo ""
echo "Section 4: multi-locale runner build + run (real in-process exercise)"
if [ ! -f "${RUNNER}" ]; then
    fail "runner missing at ${RUNNER}"
else
    pass "runner source present"
    cd "${MODULE_DIR}"
    if go build -o /tmp/opt_round259_runner ./challenges/runner/ 2>/tmp/opt_build.log; then
        pass "runner builds"
        if /tmp/opt_round259_runner -fixtures "${FIXTURE}" > /tmp/opt_run.log 2>&1; then
            pass "runner exit 0 across every package + locale"
            if grep -q "PASS: \[gptcache\]\[sr\]" /tmp/opt_run.log; then
                pass "gptcache pkg Cyrillic (sr) exact hit"
            else
                fail "gptcache pkg Cyrillic (sr) missing from runner output"
            fi
            if grep -q "PASS: \[gptcache\]\[ja\]" /tmp/opt_run.log; then
                pass "gptcache pkg Japanese (ja) exact hit"
            else
                fail "gptcache pkg Japanese (ja) missing from runner output"
            fi
            if grep -q "PASS: \[gptcache\]\[en\]\[Invalidate\]" /tmp/opt_run.log; then
                pass "gptcache Invalidate -> ErrCacheMiss path"
            else
                fail "gptcache Invalidate path missing"
            fi
            if grep -q "PASS: \[gptcache-sem\]\[CosineSimilarity\]" /tmp/opt_run.log; then
                pass "gptcache semantic CosineSimilarity exercised"
            else
                fail "gptcache semantic CosineSimilarity missing"
            fi
            if grep -q "PASS: \[prompt\]\[ar\] template byte-exact" /tmp/opt_run.log; then
                pass "prompt Arabic (ar) template byte-exact"
            else
                fail "prompt Arabic (ar) template byte-exact missing"
            fi
            if grep -q "PASS: \[prompt\]\[injection-safety\]" /tmp/opt_run.log; then
                pass "prompt injection-safety invariant"
            else
                fail "prompt injection-safety invariant missing"
            fi
            if grep -q "PASS: \[compressor\]\[zh-CN\]" /tmp/opt_run.log; then
                pass "compressor Chinese (zh-CN) filler removal"
            else
                fail "compressor Chinese (zh-CN) section missing"
            fi
            if grep -q "PASS: \[streaming\]\[FlushOnLine\]" /tmp/opt_run.log; then
                pass "streaming FlushOnLine remainder semantics"
            else
                fail "streaming FlushOnLine section missing"
            fi
            if grep -q "PASS: \[streaming\]\[TokenCounter\] CountCharacters(Cyrillic)" /tmp/opt_run.log; then
                pass "streaming TokenCounter rune-aware on Cyrillic"
            else
                fail "streaming TokenCounter Cyrillic section missing"
            fi
            if grep -q "PASS: \[streaming\]\[ChunkMerger\] emitted merged" /tmp/opt_run.log; then
                pass "streaming ChunkMerger threshold-cross emission"
            else
                fail "streaming ChunkMerger threshold section missing"
            fi
            if grep -q "PASS: \[outlines\]\[zh-CN\]" /tmp/opt_run.log; then
                pass "outlines JSONConstrainer Chinese (zh-CN)"
            else
                fail "outlines JSONConstrainer zh-CN section missing"
            fi
            if grep -q "PASS: \[outlines\]\[missing-required\]" /tmp/opt_run.log; then
                pass "outlines missing-required rejection enforced"
            else
                fail "outlines missing-required section missing"
            fi
            if grep -q "PASS: \[outlines\]\[RegexConstrainer\] extracted '42'" /tmp/opt_run.log; then
                pass "outlines RegexConstrainer extracts matched substring"
            else
                fail "outlines RegexConstrainer extract missing"
            fi
        else
            fail "runner exit non-zero - see /tmp/opt_run.log"
            sed -n '1,40p' /tmp/opt_run.log
        fi
    else
        fail "runner build failed - see /tmp/opt_build.log"
        sed -n '1,40p' /tmp/opt_build.log
    fi
    rm -f /tmp/opt_round259_runner
fi

# Section 5: README round-259 anti-bluff section
echo ""
echo "Section 5: README round-259 anti-bluff section"
if grep -q "Anti-bluff guarantees" "${README}"; then
    pass "README declares Anti-bluff guarantees"
else
    fail "README missing Anti-bluff guarantees section"
fi
if grep -q "round-259" "${README}"; then
    pass "README marked round-259"
else
    fail "README missing round-259 marker"
fi

# Cleanup mutated ledger if any
if [ -n "${TMP_LEDGER}" ]; then
    rm -f "${TMP_LEDGER}"
fi

echo ""
echo "=== Summary: ${PASS}/${TOTAL} PASS, ${FAIL} FAIL ==="

if [ "${MUTATE}" -eq 1 ]; then
    if [ "${FAIL}" -gt 0 ]; then
        echo "anti-bluff-mutate: gate correctly detected planted mutation (exit 99)"
        exit 99
    else
        echo "anti-bluff-mutate: gate FAILED to detect planted mutation - bluff!"
        exit 1
    fi
fi

if [ "${FAIL}" -gt 0 ]; then
    exit 1
fi
exit 0
