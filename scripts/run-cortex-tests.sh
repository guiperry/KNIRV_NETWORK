#!/bin/bash

# KNIRV-CORTEX Integration Test Runner
# This script runs comprehensive tests for the KNIRV-CORTEX AI system

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CORTEX_DIR="$PROJECT_ROOT/KNIRVCORTEX"
INTEGRATION_TESTS_DIR="$PROJECT_ROOT/integration-tests"
TEST_REPORTS_DIR="$INTEGRATION_TESTS_DIR/test-reports/knirv-cortex"

# Test configuration
CORTEX_PORT=3001
CORTEX_URL="http://localhost:$CORTEX_PORT"
TEST_TIMEOUT=600  # 10 minutes
CLEANUP_ON_EXIT=true

# Logging
LOG_FILE="$TEST_REPORTS_DIR/cortex_test_$(date +%Y%m%d_%H%M%S).log"

# Function to print colored output
print_status() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1"
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

# Function to log messages
log_message() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" >> "$LOG_FILE"
}

# Function to setup test environment
setup_test_environment() {
    print_status "Setting up KNIRV-CORTEX test environment..."
    
    # Create test reports directory
    mkdir -p "$TEST_REPORTS_DIR"
    
    # Initialize log file
    echo "KNIRV-CORTEX Integration Test Log - $(date)" > "$LOG_FILE"
    log_message "Test environment setup started"
    
    # Check if KNIRV-CORTEX directory exists
    if [ ! -d "$CORTEX_DIR" ]; then
        print_error "KNIRV-CORTEX directory not found: $CORTEX_DIR"
        exit 1
    fi
    
    # Check if integration tests directory exists
    if [ ! -d "$INTEGRATION_TESTS_DIR" ]; then
        print_error "Integration tests directory not found: $INTEGRATION_TESTS_DIR"
        exit 1
    fi
    
    print_success "Test environment setup completed"
    log_message "Test environment setup completed"
}

# Function to check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    # Check Node.js
    if ! command -v node &> /dev/null; then
        print_error "Node.js is not installed"
        exit 1
    fi
    
    # Check npm
    if ! command -v npm &> /dev/null; then
        print_error "npm is not installed"
        exit 1
    fi
    
    # Check Go (for integration tests)
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed"
        exit 1
    fi
    
    # Check Rust (for WASM compilation)
    if ! command -v rustc &> /dev/null; then
        print_warning "Rust is not installed - WASM tests may fail"
    fi
    
    # Check wasm-pack
    if ! command -v wasm-pack &> /dev/null; then
        print_warning "wasm-pack is not installed - WASM tests may fail"
    fi
    
    print_success "Prerequisites check completed"
    log_message "Prerequisites check completed"
}

# Function to build KNIRV-CORTEX
build_cortex() {
    print_status "Building KNIRV-CORTEX..."
    
    cd "$CORTEX_DIR"
    
    # Install dependencies
    print_status "Installing dependencies..."
    npm install >> "$LOG_FILE" 2>&1
    
    # Build WASM modules
    if command -v wasm-pack &> /dev/null; then
        print_status "Building WASM modules..."
        cd rust-wasm
        wasm-pack build --target web >> "$LOG_FILE" 2>&1
        cd ..
    else
        print_warning "Skipping WASM build - wasm-pack not available"
    fi
    
    # Build TypeScript
    print_status "Building TypeScript..."
    npm run build >> "$LOG_FILE" 2>&1
    
    print_success "KNIRV-CORTEX build completed"
    log_message "KNIRV-CORTEX build completed"
}

# Function to start KNIRV-CORTEX server
start_cortex_server() {
    print_status "Starting KNIRV-CORTEX server..."
    
    cd "$CORTEX_DIR"
    
    # Start the development server in background
    npm run dev > "$TEST_REPORTS_DIR/cortex_server.log" 2>&1 &
    CORTEX_PID=$!
    
    # Wait for server to start
    print_status "Waiting for KNIRV-CORTEX server to start..."
    for i in {1..30}; do
        if curl -s "$CORTEX_URL/api/health" > /dev/null 2>&1; then
            print_success "KNIRV-CORTEX server started successfully"
            log_message "KNIRV-CORTEX server started (PID: $CORTEX_PID)"
            return 0
        fi
        sleep 2
    done
    
    print_error "Failed to start KNIRV-CORTEX server"
    log_message "Failed to start KNIRV-CORTEX server"
    return 1
}

# Function to stop KNIRV-CORTEX server
stop_cortex_server() {
    if [ ! -z "$CORTEX_PID" ]; then
        print_status "Stopping KNIRV-CORTEX server..."
        kill $CORTEX_PID 2>/dev/null || true
        wait $CORTEX_PID 2>/dev/null || true
        print_success "KNIRV-CORTEX server stopped"
        log_message "KNIRV-CORTEX server stopped"
    fi
}

# Function to run unit tests
run_unit_tests() {
    print_status "Running KNIRV-CORTEX unit tests..."
    
    cd "$CORTEX_DIR"
    
    # Run Jest tests if available
    if [ -f "package.json" ] && grep -q "jest" package.json; then
        npm test >> "$LOG_FILE" 2>&1
        if [ $? -eq 0 ]; then
            print_success "Unit tests passed"
            log_message "Unit tests passed"
        else
            print_error "Unit tests failed"
            log_message "Unit tests failed"
            return 1
        fi
    else
        print_warning "No unit tests found"
        log_message "No unit tests found"
    fi
}

# Function to run integration tests
run_integration_tests() {
    print_status "Running KNIRV-CORTEX integration tests..."

    cd "$INTEGRATION_TESTS_DIR"

    # Run comprehensive integration test suite
    local test_files=(
        "knirvcortex_integration_test.go"
        "hrm_cognitive_tests.go"
        "neural_network_tests.go"
        "ecosystem_integration_tests.go"
    )

    local all_passed=true

    for test_file in "${test_files[@]}"; do
        if [ -f "$test_file" ]; then
            print_status "Running $test_file..."
            go test -v -timeout ${TEST_TIMEOUT}s "./$test_file" >> "$LOG_FILE" 2>&1
            if [ $? -eq 0 ]; then
                print_success "$test_file passed"
                log_message "$test_file passed"
            else
                print_error "$test_file failed"
                log_message "$test_file failed"
                all_passed=false
            fi
        else
            print_warning "$test_file not found"
            log_message "$test_file not found"
        fi
    done

    if [ "$all_passed" = true ]; then
        print_success "All integration tests passed"
        log_message "All integration tests passed"
        return 0
    else
        print_error "Some integration tests failed"
        log_message "Some integration tests failed"
        return 1
    fi
}

# Function to run performance tests
run_performance_tests() {
    print_status "Running KNIRV-CORTEX performance tests..."

    cd "$INTEGRATION_TESTS_DIR"

    # Run performance benchmarks
    if [ -f "performance_benchmarks.go" ]; then
        print_status "Running comprehensive performance benchmarks..."
        go test -v -timeout 600s "./performance_benchmarks.go" >> "$LOG_FILE" 2>&1
        if [ $? -eq 0 ]; then
            print_success "Performance benchmarks completed"
            log_message "Performance benchmarks completed"
        else
            print_error "Performance benchmarks failed"
            log_message "Performance benchmarks failed"
            return 1
        fi
    else
        print_warning "Performance benchmark file not found, running basic tests..."

        # Test cognitive processing performance
        print_status "Testing cognitive processing performance..."
        for i in {1..10}; do
            start_time=$(date +%s%N)
            curl -s -X POST "$CORTEX_URL/api/cognitive/process" \
                -H "Content-Type: application/json" \
                -d '{"type":"text","data":"Test cognitive processing performance"}' \
                > /dev/null 2>&1
            end_time=$(date +%s%N)
            duration=$(( (end_time - start_time) / 1000000 ))  # Convert to milliseconds
            echo "Request $i: ${duration}ms" >> "$LOG_FILE"
        done

        # Test neural network operations performance
        print_status "Testing neural network performance..."
        curl -s -X POST "$CORTEX_URL/api/neural/test" \
            -H "Content-Type: application/json" \
            -d '{"operation":"performance_test","iterations":100}' \
            >> "$LOG_FILE" 2>&1

        print_success "Basic performance tests completed"
        log_message "Basic performance tests completed"
    fi
}

# Function to run ecosystem integration tests
run_ecosystem_tests() {
    print_status "Running ecosystem integration tests..."
    
    # Test wallet integration
    print_status "Testing wallet integration..."
    curl -s -X POST "$CORTEX_URL/api/wallet/test" \
        -H "Content-Type: application/json" \
        -d '{"type":"balance_check"}' \
        >> "$LOG_FILE" 2>&1
    
    # Test blockchain integration
    print_status "Testing blockchain integration..."
    curl -s -X POST "$CORTEX_URL/api/blockchain/test" \
        -H "Content-Type: application/json" \
        -d '{"type":"network_consensus"}' \
        >> "$LOG_FILE" 2>&1
    
    # Test visual processing
    print_status "Testing visual processing..."
    curl -s -X POST "$CORTEX_URL/api/visual/test" \
        -H "Content-Type: application/json" \
        -d '{"type":"object_detection","image_size":[640,480]}' \
        >> "$LOG_FILE" 2>&1
    
    print_success "Ecosystem integration tests completed"
    log_message "Ecosystem integration tests completed"
}

# Function to generate test report
generate_test_report() {
    print_status "Generating test report..."
    
    local report_file="$TEST_REPORTS_DIR/cortex_test_summary_$(date +%Y%m%d_%H%M%S).html"
    
    cat > "$report_file" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>KNIRV-CORTEX Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 20px; border-radius: 5px; }
        .success { color: green; }
        .error { color: red; }
        .warning { color: orange; }
        .section { margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
        pre { background-color: #f5f5f5; padding: 10px; border-radius: 3px; overflow-x: auto; }
    </style>
</head>
<body>
    <div class="header">
        <h1>KNIRV-CORTEX Integration Test Report</h1>
        <p>Generated on: $(date)</p>
        <p>Test Duration: $(( $(date +%s) - START_TIME )) seconds</p>
    </div>
    
    <div class="section">
        <h2>Test Summary</h2>
        <p>All KNIRV-CORTEX integration tests have been executed.</p>
        <p>Detailed logs are available in: <code>$LOG_FILE</code></p>
    </div>
    
    <div class="section">
        <h2>Test Components</h2>
        <ul>
            <li>HRM Cognitive Core</li>
            <li>Neural Network Operations (TensorFlow.js)</li>
            <li>Enhanced LoRA Adapters</li>
            <li>Adaptive Learning Pipeline</li>
            <li>Ecosystem Communication Layer</li>
            <li>Wallet Integration</li>
            <li>Blockchain Integration</li>
            <li>Visual Processing AI</li>
        </ul>
    </div>
    
    <div class="section">
        <h2>Performance Metrics</h2>
        <p>Performance test results are logged in the detailed log file.</p>
    </div>
</body>
</html>
EOF
    
    print_success "Test report generated: $report_file"
    log_message "Test report generated: $report_file"
}

# Function to cleanup
cleanup() {
    if [ "$CLEANUP_ON_EXIT" = true ]; then
        print_status "Cleaning up..."
        stop_cortex_server
        log_message "Cleanup completed"
    fi
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --no-build          Skip building KNIRV-CORTEX"
    echo "  --no-server         Skip starting KNIRV-CORTEX server"
    echo "  --unit-only         Run only unit tests"
    echo "  --integration-only  Run only integration tests"
    echo "  --performance-only  Run only performance tests"
    echo "  --ecosystem-only    Run only ecosystem tests"
    echo "  --no-cleanup        Don't cleanup on exit"
    echo "  --help              Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                  Run all tests"
    echo "  $0 --unit-only      Run only unit tests"
    echo "  $0 --no-build       Skip build and run all tests"
}

# Main execution function
main() {
    local skip_build=false
    local skip_server=false
    local unit_only=false
    local integration_only=false
    local performance_only=false
    local ecosystem_only=false
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --no-build)
                skip_build=true
                shift
                ;;
            --no-server)
                skip_server=true
                shift
                ;;
            --unit-only)
                unit_only=true
                shift
                ;;
            --integration-only)
                integration_only=true
                shift
                ;;
            --performance-only)
                performance_only=true
                shift
                ;;
            --ecosystem-only)
                ecosystem_only=true
                shift
                ;;
            --no-cleanup)
                CLEANUP_ON_EXIT=false
                shift
                ;;
            --help)
                show_usage
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    # Record start time
    START_TIME=$(date +%s)
    
    # Setup trap for cleanup
    trap cleanup EXIT
    
    print_status "Starting KNIRV-CORTEX Integration Tests"
    print_status "========================================"
    
    # Setup test environment
    setup_test_environment
    
    # Check prerequisites
    check_prerequisites
    
    # Build KNIRV-CORTEX
    if [ "$skip_build" = false ]; then
        build_cortex
    fi
    
    # Start KNIRV-CORTEX server
    if [ "$skip_server" = false ]; then
        start_cortex_server
    fi
    
    # Run tests based on options
    local test_failed=false
    
    if [ "$unit_only" = true ]; then
        run_unit_tests || test_failed=true
    elif [ "$integration_only" = true ]; then
        run_integration_tests || test_failed=true
    elif [ "$performance_only" = true ]; then
        run_performance_tests || test_failed=true
    elif [ "$ecosystem_only" = true ]; then
        run_ecosystem_tests || test_failed=true
    else
        # Run all tests
        run_unit_tests || test_failed=true
        run_integration_tests || test_failed=true
        run_performance_tests || test_failed=true
        run_ecosystem_tests || test_failed=true
    fi
    
    # Generate test report
    generate_test_report
    
    # Final status
    local end_time=$(date +%s)
    local duration=$((end_time - START_TIME))
    
    print_status "========================================"
    if [ "$test_failed" = true ]; then
        print_error "KNIRV-CORTEX tests completed with failures (Duration: ${duration}s)"
        log_message "Tests completed with failures (Duration: ${duration}s)"
        exit 1
    else
        print_success "KNIRV-CORTEX tests completed successfully (Duration: ${duration}s)"
        log_message "Tests completed successfully (Duration: ${duration}s)"
        exit 0
    fi
}

# Run main function with all arguments
main "$@"
