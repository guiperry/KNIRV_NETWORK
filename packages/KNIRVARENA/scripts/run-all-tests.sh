#!/bin/bash

# KNIRVANA Comprehensive Test Runner
# Runs all tests for all language implementations in packages/

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test results tracking
RUST_TESTS_PASSED=0
TS1_TESTS_PASSED=0
TS2_TESTS_PASSED=0
ELIXIR_TESTS_PASSED=0
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
        warning "Cargo not found. Rust tests will be skipped."
    fi
    
    # Check Node.js
    if ! command -v npm &> /dev/null; then
        error "npm not found. Please install Node.js."
        exit 1
    fi
    
    # Check Elixir
    if ! command -v mix &> /dev/null; then
        warning "Mix not found. Elixir tests will be skipped."
    fi
    
    # Check if we're in the right directory
    if [ ! -f "Makefile" ] || [ ! -d "packages" ]; then
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
    
    success "Test environment setup complete"
}

# Function to run Rust tests
run_rust_tests() {
    if ! command -v cargo &> /dev/null; then
        warning "Rust not available, skipping Rust tests"
        return
    fi
    
    log "Starting Rust test suite..."
    
    local rust_passed=0
    
    # Unit tests
    if run_test "Rust Unit Tests" "cargo test --lib" "packages/rust-client"; then
        rust_passed=$((rust_passed + 1))
    fi
    
    # Integration tests
    if run_test "Rust Integration Tests" "cargo test --test '*'" "packages/rust-client"; then
        rust_passed=$((rust_passed + 1))
    fi
    
    RUST_TESTS_PASSED=$rust_passed
    log "Rust tests completed: $rust_passed/2 passed"
}

# Function to run TypeScript client 1 tests
run_ts1_tests() {
    log "Starting TypeScript client 1 test suite..."
    
    local ts1_passed=0
    
    # Install dependencies if needed
    if [ ! -d "packages/ts-client_1/node_modules" ]; then
        log "Installing TypeScript client 1 dependencies..."
        run_test "TypeScript Client 1 Dependencies" "npm install" "packages/ts-client_1"
    fi
    
    # Unit tests
    if run_test "TypeScript Client 1 Unit Tests" "npm run test:unit" "packages/ts-client_1"; then
        ts1_passed=$((ts1_passed + 1))
    fi
    
    # Integration tests
    if run_test "TypeScript Client 1 Integration Tests" "npm run test:integration" "packages/ts-client_1"; then
        ts1_passed=$((ts1_passed + 1))
    fi
    
    # E2E tests
    if run_test "TypeScript Client 1 E2E Tests" "npm run test:e2e" "packages/ts-client_1"; then
        ts1_passed=$((ts1_passed + 1))
    fi
    
    TS1_TESTS_PASSED=$ts1_passed
    log "TypeScript client 1 tests completed: $ts1_passed/3 passed"
}

# Function to run TypeScript client 2 tests
run_ts2_tests() {
    log "Starting TypeScript client 2 test suite..."
    
    local ts2_passed=0
    
    # Install dependencies if needed
    if [ ! -d "packages/ts_client_2/node_modules" ]; then
        log "Installing TypeScript client 2 dependencies..."
        run_test "TypeScript Client 2 Dependencies" "npm install" "packages/ts_client_2"
    fi
    
    # Unit tests
    if run_test "TypeScript Client 2 Unit Tests" "npm run test:unit" "packages/ts_client_2"; then
        ts2_passed=$((ts2_passed + 1))
    fi
    
    TS2_TESTS_PASSED=$ts2_passed
    log "TypeScript client 2 tests completed: $ts2_passed/1 passed"
}

# Function to run Elixir tests
run_elixir_tests() {
    if ! command -v mix &> /dev/null; then
        warning "Elixir not available, skipping Elixir tests"
        return
    fi
    
    log "Starting Elixir test suite..."
    
    local elixir_passed=0
    
    # Install dependencies if needed
    if [ ! -d "packages/elixer_client/backend/deps" ]; then
        log "Installing Elixir dependencies..."
        run_test "Elixir Dependencies" "mix deps.get" "packages/elixer_client/backend"
    fi
    
    # Test suite
    if run_test "Elixir Tests" "mix test" "packages/elixer_client/backend"; then
        elixir_passed=$((elixir_passed + 1))
    fi
    
    ELIXIR_TESTS_PASSED=$elixir_passed
    log "Elixir tests completed: $elixir_passed/1 passed"
}

# Function to run code quality checks
run_code_quality_checks() {
    log "Running code quality checks..."
    
    if command -v cargo &> /dev/null; then
        # Rust linting
        run_test "Rust Linting (Clippy)" "cargo clippy -- -D warnings" "packages/rust-client" || true
        
        # Rust formatting check
        run_test "Rust Formatting Check" "cargo fmt -- --check" "packages/rust-client" || true
    fi
    
    # TypeScript linting
    run_test "TypeScript Client 1 Linting" "npm run lint" "packages/ts-client_1" || true
    run_test "TypeScript Client 2 Linting" "npm run lint" "packages/ts_client_2" || true
    
    # TypeScript type checking
    run_test "TypeScript Client 1 Type Check" "npm run type-check" "packages/ts-client_1" || true
}

# Function to generate coverage reports
generate_coverage() {
    log "Generating coverage reports..."
    
    mkdir -p coverage
    
    # Rust coverage
    if command -v cargo-tarpaulin &> /dev/null; then
        log "Generating Rust coverage..."
        cd packages/rust-client
        cargo tarpaulin --out Html --output-dir ../../coverage/rust || warning "Rust coverage generation failed"
        cd ../..
    else
        warning "cargo-tarpaulin not installed, skipping Rust coverage"
    fi
    
    # TypeScript coverage
    log "Generating TypeScript client 1 coverage..."
    cd packages/ts-client_1
    npm run test:coverage || warning "TypeScript client 1 coverage generation failed"
    cd ../..
}

# Function to print final report
print_final_report() {
    echo ""
    echo "========================================"
    echo "         KNIRVANA TEST REPORT"
    echo "========================================"
    echo ""
    
    echo "Rust Tests:        $RUST_TESTS_PASSED passed"
    echo "TypeScript 1 Tests: $TS1_TESTS_PASSED passed"
    echo "TypeScript 2 Tests: $TS2_TESTS_PASSED passed"
    echo "Elixir Tests:      $ELIXIR_TESTS_PASSED passed"
    echo ""
    
    if [ ${#FAILED_TESTS[@]} -eq 0 ]; then
        success "All tests passed!"
        echo ""
        echo "Coverage reports available in coverage/ directory"
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
    run_ts1_tests
    run_ts2_tests
    run_elixir_tests
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
    --ts1-only)
        check_prerequisites
        setup_test_environment
        run_ts1_tests
        ;;
    --ts2-only)
        check_prerequisites
        setup_test_environment
        run_ts2_tests
        ;;
    --elixir-only)
        check_prerequisites
        setup_test_environment
        run_elixir_tests
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
        echo "  --ts1-only       Run only TypeScript client 1 tests"
        echo "  --ts2-only       Run only TypeScript client 2 tests"
        echo "  --elixir-only    Run only Elixir tests"
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
