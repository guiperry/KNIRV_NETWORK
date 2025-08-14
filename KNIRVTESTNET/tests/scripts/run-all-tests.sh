#!/bin/bash

# KNIRV TESTNET - Complete Test Suite Execution Script
# Implements the full test suite as defined in TESTNET_TEST_SUITE_IMPLEMENTATION_PLAN.md

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_ROOT="$(dirname "$SCRIPT_DIR")"
TESTNET_ROOT="$(dirname "$TEST_ROOT")"
PROJECT_ROOT="$(dirname "$TESTNET_ROOT")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging
LOG_DIR="$TEST_ROOT/logs"
REPORT_DIR="$TEST_ROOT/reports"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
LOG_FILE="$LOG_DIR/test_execution_$TIMESTAMP.log"

# Test configuration
DEFAULT_TIMEOUT="30m"
PARALLEL_EXECUTION=true
CLEANUP_ON_EXIT=true
GENERATE_REPORTS=true

# Test categories
CATEGORIES=("e2e" "performance" "security" "cortex-demos")

# Print functions
print_header() {
    echo -e "${BLUE}================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}================================${NC}"
}

print_step() {
    echo -e "${YELLOW}[STEP]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

# Logging function
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1" >> "$LOG_FILE"
    echo "$1"
}

# Initialize test environment
initialize_test_environment() {
    print_step "Initializing test environment..."
    
    # Create necessary directories
    mkdir -p "$LOG_DIR" "$REPORT_DIR"
    
    # Initialize log file
    echo "KNIRV TESTNET Test Suite Execution Log" > "$LOG_FILE"
    echo "Started: $(date)" >> "$LOG_FILE"
    echo "========================================" >> "$LOG_FILE"
    
    # Validate testnet environment
    if ! validate_testnet_environment; then
        print_error "Testnet environment validation failed"
        exit 1
    fi
    
    print_success "Test environment initialized"
}

# Validate testnet environment
validate_testnet_environment() {
    print_step "Validating testnet environment..."
    
    # Check if testnet scripts exist
    local required_scripts=(
        "$TESTNET_ROOT/start-testnet.sh"
        "$TESTNET_ROOT/stop-testnet.sh"
        "$TESTNET_ROOT/health-check.sh"
    )
    
    for script in "${required_scripts[@]}"; do
        if [[ ! -f "$script" ]]; then
            print_error "Required script not found: $script"
            return 1
        fi
    done
    
    # Check if test binaries exist
    if [[ ! -f "$TEST_ROOT/automation/orchestrator" ]]; then
        print_step "Building test orchestrator..."
        cd "$TEST_ROOT/automation"
        go build -o orchestrator .
        cd - > /dev/null
    fi
    
    print_success "Testnet environment validated"
    return 0
}

# Start testnet services
start_testnet() {
    print_step "Starting KNIRV testnet services..."
    
    cd "$TESTNET_ROOT"
    
    # Start testnet with timeout
    timeout 300 ./start-testnet.sh || {
        print_error "Failed to start testnet within timeout"
        return 1
    }
    
    # Wait for all services to be healthy
    print_step "Waiting for services to be healthy..."
    local max_attempts=30
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        if ./health-check.sh --quiet; then
            print_success "All services are healthy"
            return 0
        fi
        
        print_info "Health check attempt $attempt/$max_attempts..."
        sleep 10
        ((attempt++))
    done
    
    print_error "Services failed to become healthy within timeout"
    return 1
}

# Stop testnet services
stop_testnet() {
    print_step "Stopping KNIRV testnet services..."
    
    cd "$TESTNET_ROOT"
    
    if [[ -f "./stop-testnet.sh" ]]; then
        ./stop-testnet.sh
        print_success "Testnet services stopped"
    else
        print_error "Stop script not found"
    fi
}

# Execute test category
execute_test_category() {
    local category="$1"
    local start_time=$(date +%s)
    
    print_header "Executing $category tests"
    
    case "$category" in
        "e2e")
            execute_e2e_tests
            ;;
        "performance")
            execute_performance_tests
            ;;
        "security")
            execute_security_tests
            ;;
        "cortex-demos")
            execute_cortex_demos
            ;;
        *)
            print_error "Unknown test category: $category"
            return 1
            ;;
    esac
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    print_success "$category tests completed in ${duration}s"
    log "Category $category completed in ${duration}s"
}

# Execute E2E tests
execute_e2e_tests() {
    print_step "Running end-to-end tests..."
    
    # User journey tests
    print_step "Running user journey tests..."
    cd "$TEST_ROOT/e2e/user-journey-tests"
    if [[ -f "run-tests.sh" ]]; then
        ./run-tests.sh
    else
        print_info "User journey tests not yet implemented"
    fi
    
    # Economic loop tests
    print_step "Running economic loop tests..."
    cd "$TEST_ROOT/e2e/economic-loop-tests"
    if [[ -f "run-tests.sh" ]]; then
        ./run-tests.sh
    else
        print_info "Economic loop tests not yet implemented"
    fi
    
    # Cross-service integration tests
    print_step "Running cross-service integration tests..."
    cd "$TEST_ROOT/e2e/cross-service-integration"
    if [[ -f "run-tests.sh" ]]; then
        ./run-tests.sh
    else
        print_info "Cross-service integration tests not yet implemented"
    fi
}

# Execute performance tests
execute_performance_tests() {
    print_step "Running performance tests..."
    
    # Load testing
    print_step "Running load tests..."
    cd "$TEST_ROOT/performance/load-testing"
    if [[ -f "run-load-tests.sh" ]]; then
        ./run-load-tests.sh
    else
        print_info "Load tests not yet implemented"
    fi
    
    # Stress testing
    print_step "Running stress tests..."
    cd "$TEST_ROOT/performance/stress-testing"
    if [[ -f "run-stress-tests.sh" ]]; then
        ./run-stress-tests.sh
    else
        print_info "Stress tests not yet implemented"
    fi
    
    # Benchmarking
    print_step "Running benchmarks..."
    cd "$TEST_ROOT/performance/benchmarking"
    if [[ -f "run-benchmarks.sh" ]]; then
        ./run-benchmarks.sh
    else
        print_info "Benchmarks not yet implemented"
    fi
}

# Execute security tests
execute_security_tests() {
    print_step "Running security tests..."
    
    # Authentication testing
    print_step "Running authentication tests..."
    cd "$TEST_ROOT/security/auth-testing"
    if [[ -f "run-auth-tests.sh" ]]; then
        ./run-auth-tests.sh
    else
        print_info "Authentication tests not yet implemented"
    fi
    
    # Permission testing
    print_step "Running permission tests..."
    cd "$TEST_ROOT/security/permission-testing"
    if [[ -f "run-permission-tests.sh" ]]; then
        ./run-permission-tests.sh
    else
        print_info "Permission tests not yet implemented"
    fi
    
    # Vulnerability scanning
    print_step "Running vulnerability scans..."
    cd "$TEST_ROOT/security/vulnerability-scanning"
    if [[ -f "run-vuln-scans.sh" ]]; then
        ./run-vuln-scans.sh
    else
        print_info "Vulnerability scans not yet implemented"
    fi
}

# Execute CORTEX demos
execute_cortex_demos() {
    print_step "Running CORTEX automated demos..."
    
    cd "$TEST_ROOT/e2e/cortex-demo-suite"
    
    # Run skill development demo
    print_step "Running skill development demo..."
    if [[ -f "skill-development-demo.sh" ]]; then
        ./skill-development-demo.sh
    else
        print_info "Skill development demo not yet implemented"
    fi
    
    # Run multi-agent collaboration demo
    print_step "Running multi-agent collaboration demo..."
    if [[ -f "collaboration-demo.sh" ]]; then
        ./collaboration-demo.sh
    else
        print_info "Multi-agent collaboration demo not yet implemented"
    fi
    
    # Run learning adaptation demo
    print_step "Running learning adaptation demo..."
    if [[ -f "learning-demo.sh" ]]; then
        ./learning-demo.sh
    else
        print_info "Learning adaptation demo not yet implemented"
    fi
}

# Generate test reports
generate_reports() {
    if [[ "$GENERATE_REPORTS" != "true" ]]; then
        return 0
    fi
    
    print_step "Generating test reports..."
    
    local report_file="$REPORT_DIR/test_suite_report_$TIMESTAMP.html"
    
    # Create HTML report
    cat > "$report_file" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>KNIRV TESTNET Test Suite Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 20px; border-radius: 5px; }
        .success { color: green; }
        .error { color: red; }
        .info { color: blue; }
        .section { margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>KNIRV TESTNET Test Suite Report</h1>
        <p>Generated: $(date)</p>
        <p>Execution ID: $TIMESTAMP</p>
    </div>
    
    <div class="section">
        <h2>Test Execution Summary</h2>
        <p>Test suite execution completed.</p>
        <p>Log file: $LOG_FILE</p>
    </div>
    
    <div class="section">
        <h2>Test Categories</h2>
        <ul>
            <li>End-to-End Tests</li>
            <li>Performance Tests</li>
            <li>Security Tests</li>
            <li>CORTEX Demos</li>
        </ul>
    </div>
</body>
</html>
EOF
    
    print_success "Test report generated: $report_file"
}

# Cleanup function
cleanup() {
    if [[ "$CLEANUP_ON_EXIT" == "true" ]]; then
        print_step "Cleaning up test environment..."
        stop_testnet
        print_success "Cleanup completed"
    fi
}

# Main execution function
main() {
    local categories_to_run=("${CATEGORIES[@]}")
    local start_testnet_flag=true
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --category)
                categories_to_run=("$2")
                shift 2
                ;;
            --no-start)
                start_testnet_flag=false
                shift
                ;;
            --no-cleanup)
                CLEANUP_ON_EXIT=false
                shift
                ;;
            --no-reports)
                GENERATE_REPORTS=false
                shift
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo "Options:"
                echo "  --category CATEGORY    Run specific test category (e2e, performance, security, cortex-demos)"
                echo "  --no-start            Don't start testnet (assume already running)"
                echo "  --no-cleanup          Don't cleanup on exit"
                echo "  --no-reports          Don't generate reports"
                echo "  --help                Show this help message"
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    # Set up trap for cleanup
    trap cleanup EXIT
    
    print_header "KNIRV TESTNET Test Suite Execution"
    print_info "Execution ID: $TIMESTAMP"
    print_info "Categories to run: ${categories_to_run[*]}"
    
    # Initialize environment
    initialize_test_environment
    
    # Start testnet if requested
    if [[ "$start_testnet_flag" == "true" ]]; then
        start_testnet
    fi
    
    # Execute test categories
    local overall_success=true
    for category in "${categories_to_run[@]}"; do
        if ! execute_test_category "$category"; then
            overall_success=false
            print_error "Category $category failed"
        fi
    done
    
    # Generate reports
    generate_reports
    
    # Final status
    if [[ "$overall_success" == "true" ]]; then
        print_success "All test categories completed successfully"
        log "Test suite execution completed successfully"
        exit 0
    else
        print_error "Some test categories failed"
        log "Test suite execution completed with failures"
        exit 1
    fi
}

# Run main function
main "$@"
