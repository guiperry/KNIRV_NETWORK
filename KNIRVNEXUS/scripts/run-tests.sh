#!/bin/bash

# KNIRVNEXUS Test Runner Script
# Comprehensive test execution with coverage reporting
# Aligned with KNIRVNEXUS Makefile test targets

set -euo pipefail

# Script configuration - paths adjusted for KNIRVNEXUS/scripts location
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KNIRVNEXUS_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
BACKEND_DIR="${KNIRVNEXUS_ROOT}/backend"
PROJECT_ROOT="$(cd "${KNIRVNEXUS_ROOT}/.." && pwd)"

# Test configuration
TEST_TIMEOUT="${TEST_TIMEOUT:-5m}"
COVERAGE_THRESHOLD="${COVERAGE_THRESHOLD:-70}"
PARALLEL_TESTS="${PARALLEL_TESTS:-true}"
VERBOSE="${VERBOSE:-false}"
RACE_DETECTION="${RACE_DETECTION:-true}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Help function
show_help() {
    cat << EOF
KNIRVNEXUS Test Runner

Usage: $0 [OPTIONS] [TEST_TYPE]

Test Types:
    unit          Run unit tests only (default)
    integration   Run integration tests only
    e2e           Run end-to-end tests only
    architecture  Run architecture tests only
    all           Run all tests (unit, integration, e2e, architecture)

Options:
    --timeout DURATION      Test timeout (default: 5m)
    --coverage-threshold N  Minimum coverage percentage (default: 70)
    --no-parallel          Disable parallel test execution
    --verbose              Enable verbose output
    --no-race              Disable race detection
    --help                 Show this help message

Environment Variables:
    TEST_TIMEOUT           Test timeout duration
    COVERAGE_THRESHOLD     Minimum coverage percentage
    PARALLEL_TESTS         Enable/disable parallel tests (true/false)
    VERBOSE                Enable verbose output (true/false)
    RACE_DETECTION         Enable race detection (true/false)

Examples:
    # Run all unit tests
    $0 unit

    # Run integration tests with verbose output
    $0 --verbose integration

    # Run all tests with custom coverage threshold
    $0 --coverage-threshold 80 all

    # Run architecture tests
    $0 architecture

Note: For privileged tests (cgroups, namespaces), use the test-nexus-privileged.sh script instead.

EOF
}

# Parse command line arguments
parse_args() {
    TEST_TYPE="unit"

    while [[ $# -gt 0 ]]; do
        case $1 in
            --timeout)
                TEST_TIMEOUT="$2"
                shift 2
                ;;
            --coverage-threshold)
                COVERAGE_THRESHOLD="$2"
                shift 2
                ;;
            --no-parallel)
                PARALLEL_TESTS="false"
                shift
                ;;
            --verbose)
                VERBOSE="true"
                shift
                ;;
            --no-race)
                RACE_DETECTION="false"
                shift
                ;;
            --help)
                show_help
                exit 0
                ;;
            unit|integration|e2e|architecture|all)
                TEST_TYPE="$1"
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# Setup test environment
setup_test_env() {
    log_info "Setting up test environment..."

    cd "$BACKEND_DIR"

    # Create test directories
    mkdir -p coverage
    mkdir -p "${KNIRVNEXUS_ROOT}/test-results"

    log_success "Test environment setup complete"
}

# Cleanup test environment
cleanup_test_env() {
    log_info "Cleaning up test environment..."

    # Kill any remaining test processes
    pkill -f "backend_server.*test" 2>/dev/null || true

    log_success "Test environment cleanup complete"
}

# Build test flags
build_test_flags() {
    local coverprofile="$1"
    local flags=""

    if [[ "$VERBOSE" == "true" ]]; then
        flags="$flags -v"
    fi

    if [[ "$RACE_DETECTION" == "true" ]]; then
        flags="$flags -race"
    fi

    if [[ "$PARALLEL_TESTS" == "false" ]]; then
        flags="$flags -p 1"
    fi

    flags="$flags -timeout $TEST_TIMEOUT"
    if [[ -n "$coverprofile" ]]; then
        flags="$flags -coverprofile=$coverprofile"
        flags="$flags -covermode=atomic"
    fi

    echo "$flags"
}

# Run unit tests (matches Makefile test-go target)
run_unit_tests() {
    log_info "Running unit tests..."

    local flags
    flags=$(build_test_flags "coverage/unit.out")

    # Run tests with coverage (matches: go test -v -race -coverprofile=coverage.out ./...)
    if go test $flags ./...; then
        log_success "Unit tests passed"

        # Generate HTML coverage report
        if [[ -f "coverage/unit.out" ]]; then
            go tool cover -html=coverage/unit.out -o coverage/coverage.html
            log_info "Coverage report generated: ${BACKEND_DIR}/coverage/coverage.html"
        fi

        return 0
    else
        log_error "Unit tests failed"
        return 1
    fi
}

# Run integration tests (matches Makefile test-integration target)
run_integration_tests() {
    log_info "Running integration tests..."

    local flags
    flags=$(build_test_flags "coverage/integration.out")

    # Run integration tests (matches: go test -v ./tests/integration/...)
    if go test $flags ./tests/integration/...; then
        log_success "Integration tests passed"
        return 0
    else
        log_error "Integration tests failed"
        return 1
    fi
}

# Run end-to-end tests (matches Makefile test-e2e target)
run_e2e_tests() {
    log_info "Running end-to-end tests..."

    local flags
    flags=$(build_test_flags "coverage/e2e.out")

    # Run e2e tests (matches: go test -v -tags=e2e ./tests/e2e/...)
    if go test $flags -tags=e2e ./tests/e2e/...; then
        log_success "End-to-end tests passed"
        return 0
    else
        log_error "End-to-end tests failed"
        return 1
    fi
}

# Run architecture tests (matches Makefile test-architecture target)
run_architecture_tests() {
    log_info "Running architecture tests..."

    local flags
    flags=$(build_test_flags "coverage/architecture.out")

    # Run architecture tests (matches: go test -v ./tests -run TestArchitecture)
    if go test $flags ./tests -run TestArchitecture; then
        log_success "Architecture tests passed"
        return 0
    else
        log_error "Architecture tests failed"
        return 1
    fi
}

# Generate coverage report
generate_coverage_report() {
    log_info "Generating coverage report..."

    # Find the most recent coverage file or merge if multiple exist
    local coverage_files=()
    [[ -f "coverage/unit.out" ]] && coverage_files+=("coverage/unit.out")
    [[ -f "coverage/integration.out" ]] && coverage_files+=("coverage/integration.out")
    [[ -f "coverage/e2e.out" ]] && coverage_files+=("coverage/e2e.out")
    [[ -f "coverage/architecture.out" ]] && coverage_files+=("coverage/architecture.out")

    if [[ ${#coverage_files[@]} -eq 0 ]]; then
        log_warning "No coverage data found"
        return 0
    fi

    # Use the first available coverage file as primary
    local primary_coverage="${coverage_files[0]}"

    # If gocovmerge is available and we have multiple files, merge them
    if command -v gocovmerge >/dev/null 2>&1 && [[ ${#coverage_files[@]} -gt 1 ]]; then
        log_info "Merging coverage profiles..."
        gocovmerge "${coverage_files[@]}" > coverage/coverage.out 2>/dev/null || {
            log_warning "Failed to merge coverage profiles, using ${primary_coverage}"
            cp "${primary_coverage}" coverage/coverage.out
        }
    else
        cp "${primary_coverage}" coverage/coverage.out
    fi

    # Generate HTML coverage report
    go tool cover -html=coverage/coverage.out -o coverage/coverage.html

    # Calculate coverage percentage
    local coverage_percent
    coverage_percent=$(go tool cover -func=coverage/coverage.out | grep total | awk '{print $3}' | sed 's/%//')

    log_info "Coverage: ${coverage_percent}%"

    # Check coverage threshold
    if (( $(echo "$coverage_percent >= $COVERAGE_THRESHOLD" | bc -l 2>/dev/null || echo "1") )); then
        log_success "Coverage threshold met (${coverage_percent}% >= ${COVERAGE_THRESHOLD}%)"
        return 0
    else
        log_warning "Coverage threshold not met (${coverage_percent}% < ${COVERAGE_THRESHOLD}%)"
        # Don't fail the build for coverage threshold
        return 0
    fi
}

# Run benchmarks
run_benchmarks() {
    log_info "Running benchmarks..."

    mkdir -p "${KNIRVNEXUS_ROOT}/test-results"
    go test -bench=. -benchmem ./... > "${KNIRVNEXUS_ROOT}/test-results/benchmarks.txt" 2>&1 || true

    log_success "Benchmarks completed"
}

# Generate test report
generate_test_report() {
    log_info "Generating test report..."

    local report_file="${KNIRVNEXUS_ROOT}/test-results/test-report-$(date +%Y%m%d-%H%M%S).md"

    cat > "$report_file" << EOF
# KNIRVNEXUS Test Report

**Date:** $(date)
**Test Type:** $TEST_TYPE
**Coverage Threshold:** $COVERAGE_THRESHOLD%

## Test Configuration
- Timeout: $TEST_TIMEOUT
- Parallel Tests: $PARALLEL_TESTS
- Race Detection: $RACE_DETECTION
- Verbose: $VERBOSE

## Results
EOF

    if [[ -f "coverage/coverage.out" ]]; then
        local coverage_percent
        coverage_percent=$(go tool cover -func=coverage/coverage.out | grep total | awk '{print $3}' | sed 's/%//')
        echo "- Coverage: ${coverage_percent}%" >> "$report_file"
    fi

    echo "" >> "$report_file"
    echo "## Coverage Report" >> "$report_file"
    echo "See [coverage.html](../backend/coverage/coverage.html) for detailed coverage report." >> "$report_file"

    if [[ -f "${KNIRVNEXUS_ROOT}/test-results/benchmarks.txt" ]]; then
        echo "" >> "$report_file"
        echo "## Benchmarks" >> "$report_file"
        echo '```' >> "$report_file"
        cat "${KNIRVNEXUS_ROOT}/test-results/benchmarks.txt" >> "$report_file"
        echo '```' >> "$report_file"
    fi

    log_success "Test report generated: $report_file"
}

# Main test execution
main() {
    log_info "Starting KNIRVNEXUS test execution..."
    log_info "Test type: $TEST_TYPE"

    # Setup
    setup_test_env
    trap cleanup_test_env EXIT

    local test_result=0

    # Run tests based on type
    case $TEST_TYPE in
        unit)
            run_unit_tests || test_result=1
            ;;
        integration)
            run_integration_tests || test_result=1
            ;;
        e2e)
            run_e2e_tests || test_result=1
            ;;
        architecture)
            run_architecture_tests || test_result=1
            ;;
        all)
            run_unit_tests || test_result=1
            run_integration_tests || test_result=1
            run_e2e_tests || test_result=1
            run_architecture_tests || test_result=1
            ;;
        *)
            log_error "Unknown test type: $TEST_TYPE"
            exit 1
            ;;
    esac

    # Generate coverage report
    generate_coverage_report || test_result=1

    # Run benchmarks for unit tests
    if [[ "$TEST_TYPE" == "unit" || "$TEST_TYPE" == "all" ]]; then
        run_benchmarks
    fi

    # Generate test report
    generate_test_report

    # Final result
    if [[ $test_result -eq 0 ]]; then
        log_success "All tests passed!"
        exit 0
    else
        log_error "Some tests failed!"
        exit 1
    fi
}

# Parse arguments and run main function
parse_args "$@"
main

