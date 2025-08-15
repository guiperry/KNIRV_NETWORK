#!/bin/bash

# KNIRV Integration Test Runner and Teardown Script
# This script provides a complete integration test lifecycle management

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
INTEGRATION_TEST_DIR="$PROJECT_ROOT/integration-tests"
CONFIG_DIR="$INTEGRATION_TEST_DIR/config"

# Default values
AUTO_TEARDOWN=true
SKIP_SETUP=false
VERBOSE=false
TEST_PATTERN=".*"
TIMEOUT="600s"
PARALLEL=false
GENERATE_REPORT=true
PRESERVE_LOGS=true
FORCE_CLEANUP=false

# Function to print colored output
print_header() {
    echo -e "${PURPLE}╔══════════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${PURPLE}║${NC} $1 ${PURPLE}║${NC}"
    echo -e "${PURPLE}╚══════════════════════════════════════════════════════════════════════════════╝${NC}"
}

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_step() {
    echo -e "${CYAN}[STEP]${NC} $1"
}

# Function to check prerequisites
check_prerequisites() {
    print_step "Checking prerequisites..."
    
    # Check if integration test directory exists
    if [ ! -d "$INTEGRATION_TEST_DIR" ]; then
        print_error "Integration test directory not found: $INTEGRATION_TEST_DIR"
        exit 1
    fi
    
    # Check if config scripts exist
    local required_scripts=("setup.sh" "run-tests.sh" "teardown.sh")
    for script in "${required_scripts[@]}"; do
        if [ ! -f "$CONFIG_DIR/$script" ]; then
            print_error "Required script not found: $CONFIG_DIR/$script"
            exit 1
        fi
        
        if [ ! -x "$CONFIG_DIR/$script" ]; then
            print_error "Script not executable: $CONFIG_DIR/$script"
            print_status "Making script executable..."
            chmod +x "$CONFIG_DIR/$script"
        fi
    done
    
    # Check Go installation
    if ! command -v go >/dev/null 2>&1; then
        print_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    print_success "Prerequisites check passed"
}

# Function to setup test environment
setup_test_environment() {
    if [ "$SKIP_SETUP" = true ]; then
        print_status "Skipping test environment setup"
        return
    fi
    
    print_step "Setting up test environment..."
    
    local setup_args=""
    if [ "$VERBOSE" = true ]; then
        setup_args="$setup_args --verbose"
    fi
    if [ "$FORCE_CLEANUP" = true ]; then
        setup_args="$setup_args --clean-start"
    fi
    
    if "$CONFIG_DIR/setup.sh" $setup_args; then
        print_success "Test environment setup completed"
    else
        print_error "Test environment setup failed"
        return 1
    fi
}

# Function to run integration tests
run_integration_tests() {
    print_step "Running integration tests..."
    
    local test_args=""
    if [ "$VERBOSE" = true ]; then
        test_args="$test_args --verbose"
    fi
    if [ "$PARALLEL" = true ]; then
        test_args="$test_args --parallel"
    fi
    if [ "$GENERATE_REPORT" = false ]; then
        test_args="$test_args --no-report"
    fi
    if [ "$AUTO_TEARDOWN" = false ]; then
        test_args="$test_args --no-teardown"
    fi
    
    test_args="$test_args --pattern $TEST_PATTERN"
    test_args="$test_args --timeout $TIMEOUT"
    test_args="$test_args --no-setup"  # We handle setup ourselves
    
    print_status "Executing: $CONFIG_DIR/run-tests.sh $test_args"
    
    if "$CONFIG_DIR/run-tests.sh" $test_args; then
        print_success "Integration tests completed successfully"
        return 0
    else
        print_error "Integration tests failed"
        return 1
    fi
}

# Function to teardown test environment
teardown_test_environment() {
    if [ "$AUTO_TEARDOWN" = false ]; then
        print_status "Skipping automatic teardown"
        return
    fi
    
    print_step "Tearing down test environment..."
    
    local teardown_args=""
    if [ "$VERBOSE" = true ]; then
        teardown_args="$teardown_args --verbose"
    fi
    if [ "$PRESERVE_LOGS" = false ]; then
        teardown_args="$teardown_args --no-preserve-logs"
    fi
    if [ "$FORCE_CLEANUP" = true ]; then
        teardown_args="$teardown_args --force-kill"
    fi
    
    if "$CONFIG_DIR/teardown.sh" $teardown_args; then
        print_success "Test environment teardown completed"
    else
        print_warning "Test environment teardown had issues"
    fi
}

# Function to display test summary
display_test_summary() {
    print_header "KNIRV Integration Test Summary"
    
    echo -e "${CYAN}Test Configuration:${NC}"
    echo "  • Test Pattern: $TEST_PATTERN"
    echo "  • Timeout: $TIMEOUT"
    echo "  • Parallel Execution: $PARALLEL"
    echo "  • Verbose Output: $VERBOSE"
    echo "  • Auto Teardown: $AUTO_TEARDOWN"
    echo "  • Generate Report: $GENERATE_REPORT"
    echo "  • Preserve Logs: $PRESERVE_LOGS"
    echo ""
    
    if [ -d "$INTEGRATION_TEST_DIR/reports" ]; then
        echo -e "${CYAN}Generated Reports:${NC}"
        find "$INTEGRATION_TEST_DIR/reports" -name "*.json" -o -name "*.html" -o -name "*.txt" | head -5 | while read -r report; do
            echo "  • $(basename "$report")"
        done
        echo ""
    fi
    
    if [ "$PRESERVE_LOGS" = true ] && [ -d "$INTEGRATION_TEST_DIR/logs" ]; then
        echo -e "${CYAN}Available Logs:${NC}"
        find "$INTEGRATION_TEST_DIR/logs" -name "*.log" | head -5 | while read -r log; do
            echo "  • $(basename "$log")"
        done
        echo ""
    fi
    
    echo -e "${GREEN}Integration test lifecycle completed!${NC}"
}

# Function to run KNIRVTestnet integration tests
run_testnet_tests() {
    print_step "Running KNIRVTestnet integration tests..."
    
    local testnet_dir="$PROJECT_ROOT/KNIRVGATEWAY/knirvtestnet"
    local test_script="$testnet_dir/test-integration.sh"
    
    if [ ! -f "$test_script" ]; then
        print_warning "KNIRVTestnet test script not found: $test_script"
        return 1
    fi
    
    if [ ! -x "$test_script" ]; then
        print_status "Making test script executable..."
        chmod +x "$test_script"
    fi
    
    print_status "Executing: $test_script"
    cd "$testnet_dir" || return 1
    
    if "$test_script"; then
        print_success "KNIRVTestnet integration tests completed successfully"
        return 0
    else
        print_error "KNIRVTestnet integration tests failed"
        return 1
    fi
}

# Function to handle cleanup on script exit
cleanup_on_exit() {
    local exit_code=$?
    
    if [ $exit_code -ne 0 ]; then
        print_error "Script exited with error code $exit_code"
        
        if [ "$AUTO_TEARDOWN" = true ]; then
            print_status "Running emergency teardown..."
            teardown_test_environment
        fi
    fi
    
    exit $exit_code
}

# Function to display usage
usage() {
    echo "Usage: $0 [OPTIONS] [TEST_PATTERN]"
    echo ""
    echo "This script provides complete integration test lifecycle management for KNIRV."
    echo "It will setup the test environment, run tests, and teardown automatically."
    echo ""
    echo "Arguments:"
    echo "  TEST_PATTERN         Regex pattern for tests to run (default: '.*')"
    echo ""
    echo "Options:"
    echo "  --no-teardown        Skip automatic teardown after tests"
    echo "  --skip-setup         Skip test environment setup"
    echo "  --timeout DURATION   Test timeout (default: 600s)"
    echo "  --parallel           Run tests in parallel"
    echo "  --verbose            Enable verbose output"
    echo "  --no-report          Skip generating test reports"
    echo "  --no-preserve-logs   Remove logs during teardown"
    echo "  --force-cleanup      Force kill existing processes"
    echo "  --help               Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Run all tests with default settings"
    echo "  $0 --verbose --parallel              # Run all tests with verbose output in parallel"
    echo "  $0 --no-teardown TestLLM             # Run LLM tests without teardown"
    echo "  $0 --timeout 900s --force-cleanup    # Run with extended timeout and force cleanup"
    echo "  $0 TestPerformance                   # Run only performance tests"
    echo ""
    echo "Test Suites Available:"
    echo "  • TestIntegrationSuite               # Basic integration tests"
    echo "  • TestCrossComponentValidation       # Cross-component validation"
    echo "  • TestPerformanceAndLoad            # Performance and load tests"
    echo "  • TestE2EWorkflows                  # End-to-end workflow tests"
    echo ""
    echo "Environment:"
    echo "  Project Root: $PROJECT_ROOT"
    echo "  Integration Tests: $INTEGRATION_TEST_DIR"
    echo "  Config Directory: $CONFIG_DIR"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --no-teardown)
            AUTO_TEARDOWN=false
            shift
            ;;
        --skip-setup)
            SKIP_SETUP=true
            shift
            ;;
        --timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        --parallel)
            PARALLEL=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --no-report)
            GENERATE_REPORT=false
            shift
            ;;
        --no-preserve-logs)
            PRESERVE_LOGS=false
            shift
            ;;
        --force-cleanup)
            FORCE_CLEANUP=true
            shift
            ;;
        --help)
            usage
            exit 0
            ;;
        -*)
            print_error "Unknown option: $1"
            usage
            exit 1
            ;;
        *)
            TEST_PATTERN="$1"
            shift
            ;;
    esac
done

# Set up trap for cleanup on exit
trap cleanup_on_exit EXIT

# Main execution
main() {
    print_header "KNIRV Integration Test Lifecycle Manager"
    
    print_status "Starting integration test lifecycle..."
    print_status "Project Root: $PROJECT_ROOT"
    print_status "Integration Test Directory: $INTEGRATION_TEST_DIR"
    print_status "Test Pattern: $TEST_PATTERN"
    print_status "Auto Teardown: $AUTO_TEARDOWN"
    
    # Execute test lifecycle
    check_prerequisites
    setup_test_environment
    
    local test_result=0
    run_integration_tests || test_result=$?
    
    # Run testnet tests only if main tests succeeded
    if [ $test_result -eq 0 ]; then
        run_testnet_tests || test_result=$?
    fi
    
    teardown_test_environment
    display_test_summary
    
    if [ $test_result -eq 0 ]; then
        print_success "All integration tests completed successfully!"
        exit 0
    else
        print_error "Integration tests failed with exit code $test_result"
        exit $test_result
    fi
}

# Run main function
main "$@"
