#!/bin/bash

# KNIRVANA Comprehensive Test Runner
# Runs all tests for both Rust and TypeScript implementations

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test results tracking
RUST_TESTS_PASSED=0
TS_TESTS_PASSED=0
TOTAL_TESTS=0
FAILED_TESTS=()

# Logging function
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
}

warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Function to run a test and track results
run_test() {
    local test_name="$1"
    local test_command="$2"
    local test_dir="$3"
    
    log "Running $test_name..."
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    if [ -n "$test_dir" ]; then
        cd "$test_dir"
    fi
    
    if eval "$test_command" > /dev/null 2>&1; then
        success "$test_name passed"
        return 0
    else
        error "$test_name failed"
        FAILED_TESTS+=("$test_name")
        return 1
    fi
    
    if [ -n "$test_dir" ]; then
        cd - > /dev/null
    fi
}

# Function to check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check Rust
    if ! command -v cargo &> /dev/null; then
        error "Cargo not found. Please install Rust."
        exit 1
    fi
    
    # Check Node.js
    if ! command -v npm &> /dev/null; then
        error "npm not found. Please install Node.js."
        exit 1
    fi
    
    # Check if we're in the right directory
    if [ ! -f "Makefile" ] || [ ! -d "rust-client" ] || [ ! -d "ts-client" ]; then
        error "Please run this script from the KNIRVANA root directory."
        exit 1
    fi
    
    success "Prerequisites check passed"
}

# Function to setup test environment
setup_test_environment() {
    log "Setting up test environment..."
    
    # Setup test data
    if [ -f "scripts/setup-test-data.sh" ]; then
        chmod +x scripts/setup-test-data.sh
        ./scripts/setup-test-data.sh
    fi
    
    # Load test environment variables
    if [ -f "test-data/.env.test" ]; then
        source test-data/.env.test
    fi
    
    success "Test environment setup complete"
}

# Function to run Rust tests
run_rust_tests() {
    log "Starting Rust test suite..."
    
    local rust_passed=0
    
    # Unit tests
    if run_test "Rust Unit Tests" "cargo test --lib" "rust-client"; then
        rust_passed=$((rust_passed + 1))
    fi
    
    # Integration tests
    if run_test "Rust Integration Tests" "cargo test --test '*'" "rust-client"; then
        rust_passed=$((rust_passed + 1))
    fi
    
    # Performance tests
    if run_test "Rust Performance Tests" "cargo test --release --test performance_tests" "rust-client"; then
        rust_passed=$((rust_passed + 1))
    fi
    
    # Mobile tests
    if run_test "Rust Mobile Tests" "cargo test --features mobile --test mobile_tests" "rust-client"; then
        rust_passed=$((rust_passed + 1))
    fi
    
    # Benchmark tests
    if run_test "Rust Benchmark Tests" "cargo test --release --test benchmark_tests" "rust-client"; then
        rust_passed=$((rust_passed + 1))
    fi
    
    RUST_TESTS_PASSED=$rust_passed
    log "Rust tests completed: $rust_passed/5 passed"
}

# Function to run TypeScript tests
run_typescript_tests() {
    log "Starting TypeScript test suite..."
    
    local ts_passed=0
    
    # Install dependencies if needed
    if [ ! -d "ts-client/node_modules" ]; then
        log "Installing TypeScript dependencies..."
        run_test "TypeScript Dependencies" "npm install" "ts-client"
    fi
    
    # Unit tests
    if run_test "TypeScript Unit Tests" "npm run test:unit" "ts-client"; then
        ts_passed=$((ts_passed + 1))
    fi
    
    # Integration tests
    if run_test "TypeScript Integration Tests" "npm run test:integration" "ts-client"; then
        ts_passed=$((ts_passed + 1))
    fi
    
    # Component tests
    if run_test "TypeScript Component Tests" "npm run test:components" "ts-client"; then
        ts_passed=$((ts_passed + 1))
    fi
    
    # E2E tests
    if run_test "TypeScript E2E Tests" "npm run test:e2e" "ts-client"; then
        ts_passed=$((ts_passed + 1))
    fi
    
    # Performance tests
    if run_test "TypeScript Performance Tests" "npm run test:performance" "ts-client"; then
        ts_passed=$((ts_passed + 1))
    fi
    
    TS_TESTS_PASSED=$ts_passed
    log "TypeScript tests completed: $ts_passed/5 passed"
}

# Function to run code quality checks
run_code_quality_checks() {
    log "Running code quality checks..."
    
    # Rust linting
    run_test "Rust Linting (Clippy)" "cargo clippy -- -D warnings" "rust-client"
    
    # Rust formatting check
    run_test "Rust Formatting Check" "cargo fmt -- --check" "rust-client"
    
    # TypeScript linting
    run_test "TypeScript Linting" "npm run lint" "ts-client"
    
    # TypeScript type checking
    run_test "TypeScript Type Check" "npm run type-check" "ts-client"
}

# Function to generate coverage reports
generate_coverage() {
    log "Generating coverage reports..."
    
    mkdir -p coverage
    
    # Rust coverage
    if command -v cargo-tarpaulin &> /dev/null; then
        log "Generating Rust coverage..."
        cd rust-client
        cargo tarpaulin --out Html --output-dir ../coverage/rust || warning "Rust coverage generation failed"
        cd ..
    else
        warning "cargo-tarpaulin not installed, skipping Rust coverage"
    fi
    
    # TypeScript coverage
    log "Generating TypeScript coverage..."
    cd ts-client
    npm run test:coverage || warning "TypeScript coverage generation failed"
    cd ..
}

# Function to print final report
print_final_report() {
    echo ""
    echo "========================================"
    echo "         KNIRVANA TEST REPORT"
    echo "========================================"
    echo ""
    
    echo "Rust Tests:       $RUST_TESTS_PASSED/5 passed"
    echo "TypeScript Tests: $TS_TESTS_PASSED/5 passed"
    echo "Total Tests:      $((RUST_TESTS_PASSED + TS_TESTS_PASSED))/10 passed"
    echo ""
    
    if [ ${#FAILED_TESTS[@]} -eq 0 ]; then
        success "All tests passed! 🎉"
        echo ""
        echo "Coverage reports available in coverage/ directory"
        echo "Rust docs: rust-client/target/doc/"
        echo "TypeScript docs: ts-client/docs/"
    else
        echo "Failed tests:"
        for test in "${FAILED_TESTS[@]}"; do
            error "  - $test"
        done
        echo ""
        error "Some tests failed. Please check the output above."
        exit 1
    fi
}

# Main execution
main() {
    echo "========================================"
    echo "    KNIRVANA COMPREHENSIVE TEST SUITE"
    echo "========================================"
    echo ""
    
    check_prerequisites
    setup_test_environment
    
    # Run tests
    run_rust_tests
    run_typescript_tests
    run_code_quality_checks
    
    # Generate reports
    generate_coverage
    
    # Print final report
    print_final_report
}

# Handle script arguments
case "${1:-}" in
    --rust-only)
        check_prerequisites
        setup_test_environment
        run_rust_tests
        ;;
    --ts-only)
        check_prerequisites
        setup_test_environment
        run_typescript_tests
        ;;
    --quality-only)
        check_prerequisites
        run_code_quality_checks
        ;;
    --coverage-only)
        generate_coverage
        ;;
    --help)
        echo "Usage: $0 [option]"
        echo ""
        echo "Options:"
        echo "  --rust-only      Run only Rust tests"
        echo "  --ts-only        Run only TypeScript tests"
        echo "  --quality-only   Run only code quality checks"
        echo "  --coverage-only  Generate only coverage reports"
        echo "  --help           Show this help message"
        echo ""
        echo "Run without arguments to execute the full test suite."
        ;;
    *)
        main
        ;;
esac
