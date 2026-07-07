#!/bin/bash
# run-modp-tests.sh - Run modp formal verification from the integration-tests context
#
# Usage:
#   ./run-modp-tests.sh                  # run all modp tests
#   ./run-modp-tests.sh compile          # compile P model only
#   ./run-modp-tests.sh network          # network-wide tests only
#   ./run-modp-tests.sh component        # component tests only
#   ./run-modp-tests.sh test <name>      # run a single named test
#
# Environment variables:
#   MODP_DIR      Override the path to the modp directory (default: ../modp)
#   MODP_ENABLED  Set to "false" to skip without error (useful in CI)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

# Resolve modp directory: env var > repo-relative default
MODP_DIR="${MODP_DIR:-$REPO_ROOT/modp}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

print_status()  { echo -e "${BLUE}[MODP]${NC} $1"; }
print_success() { echo -e "${GREEN}[PASS]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARN]${NC} $1"; }
print_error()   { echo -e "${RED}[FAIL]${NC} $1"; }
print_header()  { echo -e "${CYAN}=== $1 ===${NC}"; }

# Allow CI pipelines to disable modp without failure
if [ "${MODP_ENABLED}" = "false" ]; then
    print_warning "MODP_ENABLED=false - skipping formal verification"
    exit 0
fi

print_header "KNIRV Network modp Formal Verification"
print_status "modp directory: $MODP_DIR"

# Verify modp directory exists
if [ ! -d "$MODP_DIR" ]; then
    print_error "modp directory not found: $MODP_DIR"
    print_status "Set MODP_DIR to the correct path, or set MODP_ENABLED=false to skip"
    exit 1
fi

# Verify the run-tests.sh entry point exists
MODP_TESTS_SCRIPT="$MODP_DIR/scripts/run-tests.sh"
if [ ! -f "$MODP_TESTS_SCRIPT" ]; then
    print_error "modp test script not found: $MODP_TESTS_SCRIPT"
    exit 1
fi

# Forward all arguments to modp's own run-tests.sh
# Default to "all" if no arguments provided
MODP_CMD="${1:-all}"

print_status "Running: modp/scripts/run-tests.sh $MODP_CMD ${*:2}"
echo ""

# Create results directory in integration-tests so reports land alongside other test reports
RESULTS_DIR="$SCRIPT_DIR/reports"
mkdir -p "$RESULTS_DIR"

TIMESTAMP=$(date +%Y%m%d-%H%M%S)
LOG_FILE="$RESULTS_DIR/modp-verification-$TIMESTAMP.log"

print_status "Output log: $LOG_FILE"
echo ""

# Run modp tests; tee output to both console and log file
set +e
bash "$MODP_TESTS_SCRIPT" "$MODP_CMD" "${@:2}" 2>&1 | tee "$LOG_FILE"
EXIT_CODE=${PIPESTATUS[0]}
set -e

echo ""
if [ $EXIT_CODE -eq 0 ]; then
    print_success "modp formal verification completed successfully"
    print_status "Full log: $LOG_FILE"
else
    print_error "modp formal verification finished with failures (exit code $EXIT_CODE)"
    print_status "Full log: $LOG_FILE"
    exit $EXIT_CODE
fi
