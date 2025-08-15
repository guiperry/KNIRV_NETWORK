#!/bin/bash

# KNIRV Network Integration Test Suite
# This script runs comprehensive integration tests across all KNIRV components

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
TEST_TIMEOUT=300  # 5 minutes timeout for each test suite
STARTUP_WAIT=30   # Wait time for services to start
HEALTH_CHECK_RETRIES=10
HEALTH_CHECK_INTERVAL=3

# Directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
REPORTS_DIR="$ROOT_DIR/test-reports"
LOGS_DIR="$ROOT_DIR/test-logs"

# Create directories
mkdir -p "$REPORTS_DIR" "$LOGS_DIR"

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$LOGS_DIR/integration-tests.log"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOGS_DIR/integration-tests.log"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOGS_DIR/integration-tests.log"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOGS_DIR/integration-tests.log"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up test environment..."
    
    # Stop all KNIRV services
    if [ -f "$SCRIPT_DIR/kill_knirv.sh" ]; then
        log_info "Stopping KNIRV services..."
        bash "$SCRIPT_DIR/kill_knirv.sh" || true
    fi
    
    # Kill any remaining processes
    pkill -f "KNIRVROOT" || true
    pkill -f "knirv" || true
    
    # Wait for processes to stop
    sleep 5
    
    log_info "Cleanup completed"
}

# Trap cleanup on exit
trap cleanup EXIT

# Health check function
check_service_health() {
    local service_name="$1"
    local health_url="$2"
    local retries="$3"
    
    log_info "Checking health of $service_name..."
    
    for i in $(seq 1 $retries); do
        if curl -s -f "$health_url" > /dev/null 2>&1; then
            log_success "$service_name is healthy"
            return 0
        fi
        
        log_info "Health check $i/$retries failed for $service_name, retrying in ${HEALTH_CHECK_INTERVAL}s..."
        sleep $HEALTH_CHECK_INTERVAL
    done
    
    log_error "$service_name health check failed after $retries attempts"
    return 1
}

# Start services function
start_services() {
    log_info "Starting KNIRV services for integration testing..."
    
    # Cleanup any existing processes first
    cleanup
    
    # Start bootnode
    log_info "Starting bootnode..."
    cd "$ROOT_DIR"
    timeout $TEST_TIMEOUT go run . -role=bootnode -config=config/bootnode_config.go > "$LOGS_DIR/bootnode.log" 2>&1 &
    BOOTNODE_PID=$!
    
    # Wait for bootnode to start
    sleep $STARTUP_WAIT
    
    # Start root node
    log_info "Starting root node..."
    timeout $TEST_TIMEOUT go run . -role=root -config=config/root_config.go > "$LOGS_DIR/root.log" 2>&1 &
    ROOT_PID=$!
    
    # Wait for root node to start
    sleep $STARTUP_WAIT
    
    # Start tunnel registry
    log_info "Starting tunnel registry..."
    cd "$ROOT_DIR/agent-tunnel-registry"
    timeout $TEST_TIMEOUT npm start > "$LOGS_DIR/tunnel-registry.log" 2>&1 &
    TUNNEL_PID=$!
    
    # Wait for tunnel registry to start
    sleep $STARTUP_WAIT
    
    # Health checks
    check_service_health "Tunnel Registry" "http://localhost:3000/health" $HEALTH_CHECK_RETRIES
    
    log_success "All services started successfully"
}

# Run Go tests
run_go_tests() {
    log_info "Running Go unit and integration tests..."
    
    cd "$ROOT_DIR"
    
    # Run tests with coverage
    if go test -v -race -buildvcs -coverprofile="$REPORTS_DIR/coverage.out" -timeout=${TEST_TIMEOUT}s ./... > "$REPORTS_DIR/go-tests.log" 2>&1; then
        log_success "Go tests passed"
        
        # Generate coverage report
        go tool cover -html="$REPORTS_DIR/coverage.out" -o "$REPORTS_DIR/coverage.html"
        go tool cover -func="$REPORTS_DIR/coverage.out" > "$REPORTS_DIR/coverage-summary.txt"
        
        return 0
    else
        log_error "Go tests failed"
        cat "$REPORTS_DIR/go-tests.log"
        return 1
    fi
}

# Run tunnel registry tests
run_tunnel_registry_tests() {
    log_info "Running tunnel registry integration tests..."
    
    cd "$ROOT_DIR"
    
    if go test -v -run "TestTunnelRegistryIntegration" ./tests > "$REPORTS_DIR/tunnel-registry-tests.log" 2>&1; then
        log_success "Tunnel registry integration tests passed"
        return 0
    else
        log_error "Tunnel registry integration tests failed"
        cat "$REPORTS_DIR/tunnel-registry-tests.log"
        return 1
    fi
}

# Run Python SDK tests
run_python_tests() {
    log_info "Running Python SDK tests..."
    
    cd "$ROOT_DIR/../KNIRVSDK/py"
    
    # Activate virtual environment if it exists
    if [ -d "venv" ]; then
        source venv/bin/activate
    fi
    
    # Run tests for each module
    local python_test_results=0
    
    for module in gateway transaction unified; do
        log_info "Testing Python $module module..."
        
        cd "$module"
        if pytest --verbose --tb=short > "$REPORTS_DIR/python-$module-tests.log" 2>&1; then
            log_success "Python $module tests passed"
        else
            log_error "Python $module tests failed"
            python_test_results=1
        fi
        cd ..
    done
    
    return $python_test_results
}

# Run KNIRVCORTEX tests
run_cortex_tests() {
    log_info "Running KNIRVCORTEX tests..."
    
    cd "$ROOT_DIR/../KNIRVCORTEX"
    
    if npm test > "$REPORTS_DIR/cortex-tests.log" 2>&1; then
        log_success "KNIRVCORTEX tests passed"
        return 0
    else
        log_error "KNIRVCORTEX tests failed"
        cat "$REPORTS_DIR/cortex-tests.log"
        return 1
    fi
}

# Run KNIRVGATEWAY build tests
run_gateway_tests() {
    log_info "Running KNIRVGATEWAY build tests..."
    
    cd "$ROOT_DIR/../KNIRVGATEWAY"
    
    if npm run build > "$REPORTS_DIR/gateway-build.log" 2>&1; then
        log_success "KNIRVGATEWAY build tests passed"
        return 0
    else
        log_error "KNIRVGATEWAY build tests failed"
        cat "$REPORTS_DIR/gateway-build.log"
        return 1
    fi
}

# Generate final report
generate_report() {
    local total_tests="$1"
    local passed_tests="$2"
    local failed_tests="$3"
    
    log_info "Generating final test report..."
    
    cat > "$REPORTS_DIR/final-test-report.md" << EOF
# KNIRV Network Integration Test Report

**Generated:** $(date)
**Test Suite:** Complete Integration Tests
**Total Test Suites:** $total_tests
**Passed:** $passed_tests
**Failed:** $failed_tests
**Success Rate:** $(( passed_tests * 100 / total_tests ))%

## Test Results Summary

EOF
    
    # Add individual test results
    for log_file in "$REPORTS_DIR"/*.log; do
        if [ -f "$log_file" ]; then
            test_name=$(basename "$log_file" .log)
            echo "### $test_name" >> "$REPORTS_DIR/final-test-report.md"
            echo '```' >> "$REPORTS_DIR/final-test-report.md"
            tail -20 "$log_file" >> "$REPORTS_DIR/final-test-report.md"
            echo '```' >> "$REPORTS_DIR/final-test-report.md"
            echo "" >> "$REPORTS_DIR/final-test-report.md"
        fi
    done
    
    log_success "Final report generated: $REPORTS_DIR/final-test-report.md"
}

# Main execution
main() {
    log_info "Starting KNIRV Network Integration Test Suite..."
    log_info "Logs directory: $LOGS_DIR"
    log_info "Reports directory: $REPORTS_DIR"
    
    local total_tests=0
    local passed_tests=0
    local failed_tests=0
    
    # Start services
    start_services
    
    # Run test suites
    test_suites=(
        "run_go_tests"
        "run_tunnel_registry_tests" 
        "run_python_tests"
        "run_cortex_tests"
        "run_gateway_tests"
    )
    
    for test_suite in "${test_suites[@]}"; do
        total_tests=$((total_tests + 1))
        
        log_info "Running test suite: $test_suite"
        
        if $test_suite; then
            passed_tests=$((passed_tests + 1))
        else
            failed_tests=$((failed_tests + 1))
        fi
    done
    
    # Generate final report
    generate_report $total_tests $passed_tests $failed_tests
    
    # Final summary
    log_info "Integration test suite completed"
    log_info "Total: $total_tests, Passed: $passed_tests, Failed: $failed_tests"
    
    if [ $failed_tests -eq 0 ]; then
        log_success "All integration tests passed! 🎉"
        exit 0
    else
        log_error "Some integration tests failed. Check reports for details."
        exit 1
    fi
}

# Run main function
main "$@"
