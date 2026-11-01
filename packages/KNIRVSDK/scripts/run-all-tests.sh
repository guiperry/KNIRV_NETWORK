#!/usr/bin/env bash
#
# KNIRVSDK test runner.
#
# The SDK is built from one Rust core:
#   core/                Rust core (signing, URI, authority, secrets, HTTP client)
#   bindings/wasi/       the core as a WASI module, embedded by the Go package
#   go-package/          Go bindings: types plus a wazero runner over the WASI module
#   core/wasm-modules/   the crypto-core, controller-relay and proof-verifier modules
#   npm-package/         TypeScript bindings over the same module
#
# A change is only verified when the Rust core, the module the Go/TS bindings
# actually run, and both language packages agree. This script runs all of them
# and exits non-zero if any suite fails.
#
# Usage: ./scripts/run-all-tests.sh

set -uo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'
if [ ! -t 1 ]; then
    RED=''; GREEN=''; YELLOW=''; BLUE=''; NC=''
fi

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPORTS_DIR="$PROJECT_ROOT/test-reports"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
SUMMARY="$REPORTS_DIR/sdk_test_summary_$TIMESTAMP.txt"

mkdir -p "$REPORTS_DIR"

PASSED=()
FAILED=()
SKIPPED=()

print_section() {
    echo
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}$(printf '=%.0s' $(seq 1 ${#1}))${NC}"
}

# run_suite <label> <workdir> <command...>
run_suite() {
    local label="$1" dir="$2"
    shift 2
    local log="$REPORTS_DIR/$(echo "$label" | tr '[:upper:] ' '[:lower:]-').log"

    print_section "$label"
    echo "  cd $dir && $*"
    if ( cd "$PROJECT_ROOT/$dir" && "$@" ) 2>&1 | tee "$log"; then
        PASSED+=("$label")
        echo -e "${GREEN}✓ $label${NC}"
    else
        FAILED+=("$label")
        echo -e "${RED}✗ $label (log: $log)${NC}"
    fi
}

skip_suite() {
    local label="$1" reason="$2"
    print_section "$label"
    echo -e "${YELLOW}⚠ skipped: $reason${NC}"
    SKIPPED+=("$label — $reason")
}

print_section "KNIRVSDK multi-language test suite"
echo "Project root: $PROJECT_ROOT"
echo "Reports:      $REPORTS_DIR"
echo "Timestamp:    $TIMESTAMP"

run_suite "rust-core" "core" cargo test --all-features

# `make stamp` builds both cores, writes dist/<version>/ and pins Go and npm to it.
run_suite "build-and-stamp" "." make stamp
run_suite "go-sdk" "." make test-go
run_suite "typescript-sdk" "npm-package" npm test
run_suite "stamp-consistency" "." make check-stamp

print_section "Test summary"
{
    echo "KNIRVSDK test summary $TIMESTAMP"
    echo
    for item in ${PASSED[@]+"${PASSED[@]}"}; do echo "PASS  $item"; done
    for item in ${SKIPPED[@]+"${SKIPPED[@]}"}; do echo "SKIP  $item"; done
    for item in ${FAILED[@]+"${FAILED[@]}"}; do echo "FAIL  $item"; done
    echo
    if [ "${#FAILED[@]}" -eq 0 ]; then
        echo "RESULT: PASS (${#PASSED[@]} suites)"
    else
        echo "RESULT: FAIL (${#FAILED[@]} of $(( ${#PASSED[@]} + ${#FAILED[@]} )) suites)"
    fi
} | tee "$SUMMARY"

if [ "${#FAILED[@]}" -ne 0 ]; then
    exit 1
fi
