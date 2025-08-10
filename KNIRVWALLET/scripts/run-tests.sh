#!/bin/bash
# KNIRVWALLET Comprehensive Test Runner
# This script runs all unit and integration tests for KNIRVWALLET

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WALLET_DIR="$(dirname "$SCRIPT_DIR")"
PROJECT_ROOT="$(dirname "$WALLET_DIR")"
INTEGRATION_TESTS_DIR="$PROJECT_ROOT/integration-tests"

# Test configuration
RUN_UNIT_TESTS=true
RUN_INTEGRATION_TESTS=true
RUN_E2E_TESTS=false
GENERATE_COVERAGE=true
VERBOSE=false
WATCH_MODE=false
BAIL_ON_FAILURE=false

# Service URLs for integration tests
GATEWAY_URL="http://localhost:8000"
WALLET_URL="http://localhost:8083"
XION_RPC="https://rpc.xion-testnet-1.burnt.com:443"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --unit-only)
            RUN_INTEGRATION_TESTS=false
            RUN_E2E_TESTS=false
            shift
            ;;
        --integration-only)
            RUN_UNIT_TESTS=false
            RUN_E2E_TESTS=false
            shift
            ;;
        --e2e-only)
            RUN_UNIT_TESTS=false
            RUN_INTEGRATION_TESTS=false
            RUN_E2E_TESTS=true
            shift
            ;;
        --all)
            RUN_UNIT_TESTS=true
            RUN_INTEGRATION_TESTS=true
            RUN_E2E_TESTS=true
            shift
            ;;
        --no-coverage)
            GENERATE_COVERAGE=false
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --watch)
            WATCH_MODE=true
            shift
            ;;
        --bail)
            BAIL_ON_FAILURE=true
            shift
            ;;
        --help)
            echo "KNIRVWALLET Test Runner"
            echo ""
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --unit-only        Run only unit tests"
            echo "  --integration-only Run only integration tests"
            echo "  --e2e-only         Run only end-to-end tests"
            echo "  --all              Run all tests (default)"
            echo "  --no-coverage      Skip coverage generation"
            echo "  --verbose          Verbose output"
            echo "  --watch            Run in watch mode"
            echo "  --bail             Bail on first test failure"
            echo "  --help             Show this help message"
            echo ""
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

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

# Check if services are running for integration tests
check_services() {
    log_info "Checking if required services are running..."
    
    local services_ok=true
    
    if ! curl -s "$GATEWAY_URL/health" > /dev/null 2>&1; then
        log_warning "KNIRVGATEWAY not running at $GATEWAY_URL"
        services_ok=false
    fi
    
    if ! curl -s "$WALLET_URL/health" > /dev/null 2>&1; then
        log_warning "KNIRVWALLET backend not running at $WALLET_URL"
        services_ok=false
    fi
    
    if [ "$services_ok" = false ]; then
        log_warning "Some services are not running. Integration tests may fail."
        log_info "To start services, run: $PROJECT_ROOT/scripts/manage-knirv.sh start"
        read -p "Continue anyway? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    else
        log_success "All required services are running"
    fi
}

# Setup test environment
setup_test_env() {
    log_info "Setting up test environment..."
    
    # Set environment variables
    export NODE_ENV=test
    export JEST_WORKER_ID=1
    export MOCK_SERVICES=true
    
    # Create test directories
    mkdir -p "$WALLET_DIR/test-results"
    mkdir -p "$WALLET_DIR/coverage"
    
    # Install dependencies if needed
    if [ ! -d "$WALLET_DIR/node_modules" ]; then
        log_info "Installing dependencies..."
        cd "$WALLET_DIR"
        npm install
    fi
    
    log_success "Test environment setup complete"
}

# Run unit tests
run_unit_tests() {
    log_info "Running unit tests..."
    
    cd "$WALLET_DIR"
    
    local jest_args=""
    
    if [ "$GENERATE_COVERAGE" = true ]; then
        jest_args="$jest_args --coverage"
    fi
    
    if [ "$VERBOSE" = true ]; then
        jest_args="$jest_args --verbose"
    fi
    
    if [ "$WATCH_MODE" = true ]; then
        jest_args="$jest_args --watch"
    fi
    
    if [ "$BAIL_ON_FAILURE" = true ]; then
        jest_args="$jest_args --bail"
    fi
    
    # Run browser wallet tests
    log_info "Running browser wallet unit tests..."
    cd "$WALLET_DIR/browser-wallet"
    if ! npm test $jest_args; then
        log_error "Browser wallet unit tests failed"
        return 1
    fi
    
    # Run agentic wallet tests
    log_info "Running agentic wallet unit tests..."
    cd "$WALLET_DIR/agentic-wallet"
    if ! npm test $jest_args; then
        log_error "Agentic wallet unit tests failed"
        return 1
    fi
    
    # Run Go backend tests
    log_info "Running Go backend unit tests..."
    cd "$WALLET_DIR/agentic-wallet/go-backend"
    if ! make test; then
        log_error "Go backend unit tests failed"
        return 1
    fi
    
    log_success "Unit tests completed successfully"
    return 0
}

# Run integration tests
run_integration_tests() {
    log_info "Running integration tests..."
    
    # Check if services are running
    if [ "$RUN_INTEGRATION_TESTS" = true ]; then
        check_services
    fi
    
    cd "$INTEGRATION_TESTS_DIR"
    
    # Run KNIRVWALLET integration tests
    log_info "Running KNIRVWALLET integration tests..."
    if ! go test -v -run "TestKNIRVWallet" ./...; then
        log_error "KNIRVWALLET integration tests failed"
        return 1
    fi
    
    # Run cross-platform wallet tests
    log_info "Running cross-platform wallet tests..."
    if ! go test -v -run "TestCrossPlatformWallet" ./...; then
        log_error "Cross-platform wallet tests failed"
        return 1
    fi
    
    # Run XION integration tests
    log_info "Running XION integration tests..."
    if ! go test -v -run "TestXionIntegration" ./...; then
        log_error "XION integration tests failed"
        return 1
    fi
    
    log_success "Integration tests completed successfully"
    return 0
}

# Run end-to-end tests
run_e2e_tests() {
    log_info "Running end-to-end tests..."
    
    cd "$WALLET_DIR"
    
    # Run comprehensive E2E workflow tests
    log_info "Running E2E wallet workflow tests..."
    if ! npm run test:e2e; then
        log_error "E2E tests failed"
        return 1
    fi
    
    log_success "E2E tests completed successfully"
    return 0
}

# Generate test report
generate_report() {
    log_info "Generating test report..."
    
    local report_file="$WALLET_DIR/test-results/test-report-$(date +%Y%m%d-%H%M%S).html"
    
    cat > "$report_file" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>KNIRVWALLET Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #f0f0f0; padding: 20px; border-radius: 5px; }
        .section { margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
        .success { background: #d4edda; border-color: #c3e6cb; }
        .error { background: #f8d7da; border-color: #f5c6cb; }
        .warning { background: #fff3cd; border-color: #ffeaa7; }
        pre { background: #f8f9fa; padding: 10px; border-radius: 3px; overflow-x: auto; }
    </style>
</head>
<body>
    <div class="header">
        <h1>KNIRVWALLET Test Report</h1>
        <p>Generated on: $(date)</p>
        <p>Test Configuration:</p>
        <ul>
            <li>Unit Tests: $RUN_UNIT_TESTS</li>
            <li>Integration Tests: $RUN_INTEGRATION_TESTS</li>
            <li>E2E Tests: $RUN_E2E_TESTS</li>
            <li>Coverage: $GENERATE_COVERAGE</li>
        </ul>
    </div>
EOF

    if [ -f "$WALLET_DIR/coverage/lcov-report/index.html" ]; then
        echo '<div class="section success"><h2>Coverage Report</h2><p>Coverage report available at: <a href="./coverage/lcov-report/index.html">coverage/lcov-report/index.html</a></p></div>' >> "$report_file"
    fi

    echo '</body></html>' >> "$report_file"
    
    log_success "Test report generated: $report_file"
}

# Main execution
main() {
    log_info "Starting KNIRVWALLET test suite..."
    log_info "Configuration: Unit=$RUN_UNIT_TESTS, Integration=$RUN_INTEGRATION_TESTS, E2E=$RUN_E2E_TESTS"
    
    # Setup
    setup_test_env
    
    local exit_code=0
    
    # Run tests
    if [ "$RUN_UNIT_TESTS" = true ]; then
        if ! run_unit_tests; then
            exit_code=1
            if [ "$BAIL_ON_FAILURE" = true ]; then
                log_error "Bailing on unit test failure"
                exit $exit_code
            fi
        fi
    fi
    
    if [ "$RUN_INTEGRATION_TESTS" = true ]; then
        if ! run_integration_tests; then
            exit_code=1
            if [ "$BAIL_ON_FAILURE" = true ]; then
                log_error "Bailing on integration test failure"
                exit $exit_code
            fi
        fi
    fi
    
    if [ "$RUN_E2E_TESTS" = true ]; then
        if ! run_e2e_tests; then
            exit_code=1
            if [ "$BAIL_ON_FAILURE" = true ]; then
                log_error "Bailing on E2E test failure"
                exit $exit_code
            fi
        fi
    fi
    
    # Generate report
    generate_report
    
    # Summary
    if [ $exit_code -eq 0 ]; then
        log_success "All tests completed successfully!"
    else
        log_error "Some tests failed. Check the output above for details."
    fi
    
    exit $exit_code
}

# Run main function
main "$@"
