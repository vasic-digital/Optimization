#!/usr/bin/env bash
# optimization_functionality_challenge.sh - Validates Optimization module core functionality and structure
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODULE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
MODULE_NAME="Optimization"

PASS=0
FAIL=0
TOTAL=0

pass() { PASS=$((PASS+1)); TOTAL=$((TOTAL+1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL+1)); TOTAL=$((TOTAL+1)); echo "  FAIL: $1"; }

echo "=== ${MODULE_NAME} Functionality Challenge ==="
echo ""

# Test 1: Required packages exist
echo "Test: Required packages exist"
pkgs_ok=true
for pkg in gptcache outlines streaming sglang prompt adapter; do
    if [ ! -d "${MODULE_DIR}/pkg/${pkg}" ]; then
        fail "Missing package: pkg/${pkg}"
        pkgs_ok=false
    fi
done
if [ "$pkgs_ok" = true ]; then
    pass "All required packages present (gptcache, outlines, streaming, sglang, prompt, adapter)"
fi

# Test 2: GPT-Cache interface is defined
echo "Test: Cache interface for GPT-Cache is defined"
if grep -rq "type Cache interface" "${MODULE_DIR}/pkg/gptcache/"; then
    pass "Cache interface is defined in pkg/gptcache"
else
    fail "Cache interface not found in pkg/gptcache"
fi

# Test 3: SemanticMatcher interface is defined
echo "Test: SemanticMatcher interface is defined"
if grep -rq "type SemanticMatcher interface" "${MODULE_DIR}/pkg/gptcache/"; then
    pass "SemanticMatcher interface is defined in pkg/gptcache"
else
    fail "SemanticMatcher interface not found in pkg/gptcache"
fi

# Test 4: Outlines structured output support
echo "Test: Outlines structured output support exists"
if grep -rq "Constrainer\|Optimizer\|Schema\|JSON" "${MODULE_DIR}/pkg/outlines/"; then
    pass "Outlines structured output support found in pkg/outlines"
else
    fail "No Outlines support found in pkg/outlines"
fi

# Test 5: Streaming optimization support
echo "Test: Streaming optimization support exists"
if grep -rq "Buffer\|Stream\|Compress\|Optimize" "${MODULE_DIR}/pkg/streaming/"; then
    pass "Streaming optimization support found in pkg/streaming"
else
    fail "No streaming optimization support found"
fi

# Test 6: SGLang support exists
echo "Test: SGLang support exists"
if grep -rq "type\s\+\w\+\s\+struct\|Program\|Execute" "${MODULE_DIR}/pkg/sglang/"; then
    pass "SGLang support found in pkg/sglang"
else
    fail "No SGLang support found"
fi

# Test 7: Prompt optimization support
echo "Test: Prompt optimization support exists"
if grep -rq "Prompt\|Optimize\|optimize\|Compress" "${MODULE_DIR}/pkg/prompt/"; then
    pass "Prompt optimization support found in pkg/prompt"
else
    fail "No prompt optimization support found"
fi

# Test 8: CachedResponse type exists
echo "Test: CachedResponse type exists"
if grep -rq "type CachedResponse struct\|CachedResponse" "${MODULE_DIR}/pkg/gptcache/"; then
    pass "CachedResponse type found"
else
    fail "CachedResponse type not found"
fi

# Test 9: Config support
echo "Test: Config support exists"
if grep -rq "type Config struct" "${MODULE_DIR}/pkg/"; then
    pass "Config struct found"
else
    fail "Config struct not found"
fi

# Test 10: Adapter package for integration
echo "Test: Adapter package exists for integration"
if [ -d "${MODULE_DIR}/pkg/adapter" ] && [ "$(find "${MODULE_DIR}/pkg/adapter" -name "*.go" ! -name "*_test.go" | wc -l)" -gt 0 ]; then
    pass "Adapter package found with Go files"
else
    fail "No adapter package or files found"
fi

echo ""
echo "=== Results: ${PASS}/${TOTAL} passed, ${FAIL} failed ==="
[ "${FAIL}" -eq 0 ] && exit 0 || exit 1
