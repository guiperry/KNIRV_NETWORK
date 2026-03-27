#!/bin/bash

# KNIRV-SERVER Test Runner Script
# Comprehensive test execution with coverage reporting

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
PROJECT_ROOT="$(cd "${BACKEND_DIR}/.." && pwd)"

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
KNIRV-SERVER Test Runner

Usage: $0 [OPTIONS] [TEST_TYPE]

Test Types:
    unit        Run unit tests only (default)
    integration Run integration tests only
    e2e         Run end-to-end tests only
    all         Run all tests

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
            unit|integration|e2e|all)
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
    mkdir -p test-results
    mkdir -p coverage
    mkdir -p /tmp/test-data
    mkdir -p /tmp/test-workspaces
    mkdir -p /tmp/test-projects
    
    # Set test environment variables
    export NEXUS_ENV=test
    export NEXUS_CONFIG="$BACKEND_DIR/config/test.yaml"
    export TEST_MODE=true
    
    log_success "Test environment setup complete"
}

# Cleanup test environment
cleanup_test_env() {
    log_info "Cleaning up test environment..."
    
    # Clean up test data
    rm -rf /tmp/test-data
    rm -rf /tmp/test-workspaces
    rm -rf /tmp/test-projects
    
    # Kill any remaining test processes
    pkill -f "backend_server.*test" || true
    
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

# Run unit tests
run_unit_tests() {
    log_info "Running unit tests..."

    local flags
    flags=$(build_test_flags "coverage/unit.out")

    # Run tests with coverage
    if go test $flags ./...; then
        log_success "Unit tests passed"
        return 0
    else
        log_error "Unit tests failed"
        return 1
    fi
}

# Run integration tests
run_integration_tests() {
    log_info "Running integration tests..."

    local flags
    flags=$(build_test_flags "coverage/integration.out")

    # Run integration tests
    if go test $flags -tags=integration ./tests/integration/...; then
        log_success "Integration tests passed"
        return 0
    else
        log_error "Integration tests failed"
        return 1
    fi
}

# Run end-to-end tests
run_e2e_tests() {
    log_info "Running end-to-end tests..."

    local flags
    flags=$(build_test_flags "coverage/e2e.out")

    # Start test server
    log_info "Starting test server..."
    ./bin/backend_server --config config/test.yaml &
    local server_pid=$!

    # Wait for server to start
    sleep 5

    # Run e2e tests
    local result=0
    if go test $flags -tags=e2e ./tests/e2e/...; then
        log_success "End-to-end tests passed"
    else
        log_error "End-to-end tests failed"
        result=1
    fi

    # Stop test server
    kill $server_pid || true
    wait $server_pid 2>/dev/null || true

    return $result
}

# Generate coverage report
generate_coverage_report() {
    log_info "Generating coverage report..."

    # Merge coverage profiles if they exist
    if [[ -f "coverage/unit.out" ]] || [[ -f "coverage/integration.out" ]] || [[ -f "coverage/e2e.out" ]]; then
        log_info "Merging coverage profiles..."
        gocovmerge coverage/unit.out coverage/integration.out coverage/e2e.out > coverage/coverage.out 2>/dev/null || {
            # If gocovmerge fails, try to use the first available profile
            if [[ -f "coverage/unit.out" ]]; then
                cp coverage/unit.out coverage/coverage.out
            elif [[ -f "coverage/integration.out" ]]; then
                cp coverage/integration.out coverage/coverage.out
            elif [[ -f "coverage/e2e.out" ]]; then
                cp coverage/e2e.out coverage/coverage.out
            fi
        }
    fi

    if [[ ! -f "coverage/coverage.out" ]]; then
        log_warning "No coverage data found"
        return 0
    fi

    # Generate HTML coverage report
    go tool cover -html=coverage/coverage.out -o coverage/coverage.html

    # Calculate coverage percentage
    local coverage_percent
    coverage_percent=$(go tool cover -func=coverage/coverage.out | grep total | awk '{print $3}' | sed 's/%//')

    log_info "Coverage: ${coverage_percent}%"

    # Check coverage threshold
    if (( $(echo "$coverage_percent >= $COVERAGE_THRESHOLD" | bc -l) )); then
        log_success "Coverage threshold met (${coverage_percent}% >= ${COVERAGE_THRESHOLD}%)"
        return 0
    else
        log_error "Coverage threshold not met (${coverage_percent}% < ${COVERAGE_THRESHOLD}%)"
        return 1
    fi
}

# Run benchmarks
run_benchmarks() {
    log_info "Running benchmarks..."
    
    go test -bench=. -benchmem ./... > test-results/benchmarks.txt
    
    log_success "Benchmarks completed"
}

# Generate test report
generate_test_report() {
    log_info "Generating test report..."
    
    local report_file="test-results/test-report-$(date +%Y%m%d-%H%M%S).md"
    
    cat > "$report_file" << EOF
# KNIRV-SERVER Test Report

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
    echo "See [coverage.html](../coverage/coverage.html) for detailed coverage report." >> "$report_file"
    
    if [[ -f "test-results/benchmarks.txt" ]]; then
        echo "" >> "$report_file"
        echo "## Benchmarks" >> "$report_file"
        echo '```' >> "$report_file"
        cat test-results/benchmarks.txt >> "$report_file"
        echo '```' >> "$report_file"
    fi
    
    log_success "Test report generated: $report_file"
}

# Main test execution
main() {
    log_info "Starting KNIRV-SERVER test execution..."
    log_info "Test type: $TEST_TYPE"
    
    # Setup
    setup_test_env
    trap cleanup_test_env EXIT
    
    # Build application first
    log_info "Building application..."
    if ! go build -o bin/nexus-server ./cmd/backend_server; then
        log_error "Build failed"
        exit 1
    fi
    
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
        all)
            run_unit_tests || test_result=1
            run_integration_tests || test_result=1
            run_e2e_tests || test_result=1
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
