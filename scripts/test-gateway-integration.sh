#!/bin/bash

# KNIRV Gateway Integration Testing Script
# This script integrates gateway testing with the existing integration test suite

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
GATEWAY_DIR="$PROJECT_ROOT/KNIRVGATEWAY"
ECONOMICS_DIR="$GATEWAY_DIR/economics"
INTEGRATION_TESTS_DIR="$PROJECT_ROOT/integration-tests"

# Test configuration
ECONOMICS_PORT=${ECONOMICS_PORT:-8090}
GATEWAY_PORT=${GATEWAY_PORT:-8000}
TEST_TIMEOUT=300  # 5 minutes

# Function to print colored output
print_info() {
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

print_header() {
    echo -e "${PURPLE}[HEADER]${NC} $1"
}

# Function to wait for service to be ready
wait_for_service() {
    local url=$1
    local service_name=$2
    local max_attempts=30
    local attempt=1
    
    print_info "Waiting for $service_name to be ready..."
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s -f "$url" > /dev/null 2>&1; then
            print_success "$service_name is ready!"
            return 0
        fi
        
        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    print_error "$service_name failed to start within $((max_attempts * 2)) seconds"
    return 1
}

# Function to run economics tests
run_economics_tests() {
    print_header "Running Economics Service Tests"
    
    if [ ! -d "$ECONOMICS_DIR" ]; then
        print_error "Economics directory not found: $ECONOMICS_DIR"
        return 1
    fi
    
    cd "$ECONOMICS_DIR"
    
    # Run economics-specific tests
    if [ -x "./test-economics.sh" ]; then
        print_info "Running economics API tests..."
        if ./test-economics.sh; then
            print_success "Economics API tests passed"
        else
            print_error "Economics API tests failed"
            return 1
        fi
    else
        print_warning "Economics test script not found"
    fi
    
    # Run verification
    if [ -x "./verify-month11.sh" ]; then
        print_info "Running Month 11 verification..."
        if ./verify-month11.sh; then
            print_success "Month 11 verification passed"
        else
            print_error "Month 11 verification failed"
            return 1
        fi
    else
        print_warning "Month 11 verification script not found"
    fi
    
    cd "$PROJECT_ROOT"
}

# Function to run integration tests
run_integration_tests() {
    print_header "Running Integration Tests"
    
    if [ ! -d "$INTEGRATION_TESTS_DIR" ]; then
        print_error "Integration tests directory not found: $INTEGRATION_TESTS_DIR"
        return 1
    fi
    
    cd "$INTEGRATION_TESTS_DIR"
    
    # Set environment variables for integration tests
    export ECONOMICS_SERVICE_URL="http://localhost:$ECONOMICS_PORT"
    export GATEWAY_SERVICE_URL="http://localhost:$GATEWAY_PORT"
    
    # Run existing integration tests
    if [ -x "./config/run-tests.sh" ]; then
        print_info "Running existing integration tests..."
        if ./config/run-tests.sh; then
            print_success "Integration tests passed"
        else
            print_error "Integration tests failed"
            return 1
        fi
    else
        print_warning "Integration test runner not found"
    fi
    
    # Run Go-based integration tests
    if [ -f "go.mod" ]; then
        print_info "Running Go integration tests..."
        if go test -v -timeout ${TEST_TIMEOUT}s ./...; then
            print_success "Go integration tests passed"
        else
            print_error "Go integration tests failed"
            return 1
        fi
    fi
    
    cd "$PROJECT_ROOT"
}

# Function to run gateway-specific integration tests
run_gateway_integration_tests() {
    print_header "Running Gateway Integration Tests"
    
    # Test economics service integration
    print_info "Testing economics service endpoints..."
    
    # Test health endpoint
    if curl -s -f "http://localhost:$ECONOMICS_PORT/economics/health" > /dev/null; then
        print_success "Economics health endpoint accessible"
    else
        print_error "Economics health endpoint not accessible"
        return 1
    fi
    
    # Test metrics endpoint
    if curl -s -f "http://localhost:$ECONOMICS_PORT/economics/metrics" > /dev/null; then
        print_success "Economics metrics endpoint accessible"
    else
        print_error "Economics metrics endpoint not accessible"
        return 1
    fi
    
    # Test skill invocation
    skill_data='{"user_id":"test_user","skill_id":"test_skill","amount":"100000"}'
    if curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$skill_data" \
        "http://localhost:$ECONOMICS_PORT/economics/skill/invoke" > /dev/null; then
        print_success "Skill invocation endpoint working"
    else
        print_error "Skill invocation endpoint failed"
        return 1
    fi
    
    # Test gateway endpoints if available
    if curl -s -f "http://localhost:$GATEWAY_PORT/health" > /dev/null 2>&1; then
        print_success "API Gateway health endpoint accessible"
        
        # Test gateway routing to economics
        if curl -s -f "http://localhost:$GATEWAY_PORT/economics/health" > /dev/null 2>&1; then
            print_success "Gateway routing to economics working"
        else
            print_warning "Gateway routing to economics not working"
        fi
    else
        print_warning "API Gateway not accessible"
    fi
}

# Function to generate test report
generate_test_report() {
    local test_status=$1
    local report_file="$INTEGRATION_TESTS_DIR/reports/gateway-integration-report-$(date +%Y%m%d-%H%M%S).html"
    
    print_info "Generating test report: $report_file"
    
    mkdir -p "$(dirname "$report_file")"
    
    cat > "$report_file" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>KNIRV Gateway Integration Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 20px; border-radius: 5px; }
        .success { color: green; }
        .error { color: red; }
        .warning { color: orange; }
        .section { margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
        .timestamp { font-size: 0.9em; color: #666; }
    </style>
</head>
<body>
    <div class="header">
        <h1>KNIRV Gateway Integration Test Report</h1>
        <p class="timestamp">Generated: $(date)</p>
        <p class="$([ "$test_status" = "0" ] && echo "success" || echo "error")">
            Overall Status: $([ "$test_status" = "0" ] && echo "PASSED" || echo "FAILED")
        </p>
    </div>
    
    <div class="section">
        <h2>Test Configuration</h2>
        <ul>
            <li>Economics Service Port: $ECONOMICS_PORT</li>
            <li>Gateway Service Port: $GATEWAY_PORT</li>
            <li>Test Timeout: ${TEST_TIMEOUT}s</li>
            <li>Project Root: $PROJECT_ROOT</li>
        </ul>
    </div>
    
    <div class="section">
        <h2>Services Tested</h2>
        <ul>
            <li>Economics Service (Month 11 Implementation)</li>
            <li>API Gateway</li>
            <li>Integration Test Suite</li>
            <li>Cross-Component Validation</li>
        </ul>
    </div>
    
    <div class="section">
        <h2>Test Results Summary</h2>
        <p>Detailed test results are available in the console output and individual test logs.</p>
        <p>For more information, check:</p>
        <ul>
            <li>Economics tests: $ECONOMICS_DIR/test-economics.sh</li>
            <li>Integration tests: $INTEGRATION_TESTS_DIR/config/run-tests.sh</li>
            <li>Gateway verification: $ECONOMICS_DIR/verify-month11.sh</li>
        </ul>
    </div>
    
    <div class="section">
        <h2>Next Steps</h2>
        $(if [ "$test_status" = "0" ]; then
            echo "<p class='success'>All tests passed! The gateway integration is working correctly.</p>"
            echo "<ul>"
            echo "<li>Deploy to production environment</li>"
            echo "<li>Set up monitoring and alerting</li>"
            echo "<li>Configure load balancing</li>"
            echo "</ul>"
        else
            echo "<p class='error'>Some tests failed. Please review the console output and fix issues.</p>"
            echo "<ul>"
            echo "<li>Check service logs for errors</li>"
            echo "<li>Verify service configurations</li>"
            echo "<li>Ensure all dependencies are running</li>"
            echo "</ul>"
        fi)
    </div>
</body>
</html>
EOF
    
    print_success "Test report generated: $report_file"
}

# Function to cleanup test environment
cleanup_test_environment() {
    print_info "Cleaning up test environment..."
    
    # Stop any test services that might be running
    if [ -f "/tmp/economics.pid" ]; then
        local pid=$(cat /tmp/economics.pid)
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null || true
            rm -f /tmp/economics.pid
        fi
    fi
    
    if [ -f "/tmp/gateway.pid" ]; then
        local pid=$(cat /tmp/gateway.pid)
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null || true
            rm -f /tmp/gateway.pid
        fi
    fi
    
    print_success "Test environment cleaned up"
}

# Function to show usage
show_usage() {
    echo "KNIRV Gateway Integration Testing Script"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -e, --economics-only     Run only economics tests"
    echo "  -i, --integration-only   Run only integration tests"
    echo "  -g, --gateway-only       Run only gateway tests"
    echo "  -s, --start-services     Start services before testing"
    echo "  -c, --cleanup            Cleanup test environment after testing"
    echo "  -r, --report             Generate HTML test report"
    echo "  -t, --timeout SECONDS    Set test timeout (default: $TEST_TIMEOUT)"
    echo "  -h, --help               Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  ECONOMICS_PORT           Economics service port (default: $ECONOMICS_PORT)"
    echo "  GATEWAY_PORT             Gateway service port (default: $GATEWAY_PORT)"
}

# Parse command line arguments
ECONOMICS_ONLY=false
INTEGRATION_ONLY=false
GATEWAY_ONLY=false
START_SERVICES=false
CLEANUP=false
GENERATE_REPORT=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -e|--economics-only)
            ECONOMICS_ONLY=true
            shift
            ;;
        -i|--integration-only)
            INTEGRATION_ONLY=true
            shift
            ;;
        -g|--gateway-only)
            GATEWAY_ONLY=true
            shift
            ;;
        -s|--start-services)
            START_SERVICES=true
            shift
            ;;
        -c|--cleanup)
            CLEANUP=true
            shift
            ;;
        -r|--report)
            GENERATE_REPORT=true
            shift
            ;;
        -t|--timeout)
            TEST_TIMEOUT="$2"
            shift 2
            ;;
        -h|--help)
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

# Trap to ensure cleanup on exit
trap 'cleanup_test_environment' EXIT

print_header "KNIRV Gateway Integration Testing"
print_info "Starting comprehensive gateway integration tests..."

# Start services if requested
if [ "$START_SERVICES" = true ]; then
    print_info "Starting gateway services..."
    "$SCRIPT_DIR/run-gateway.sh" start
    
    # Wait for services to be ready
    wait_for_service "http://localhost:$ECONOMICS_PORT/economics/health" "Economics Service"
fi

# Run tests based on options
TEST_STATUS=0

if [ "$ECONOMICS_ONLY" = true ]; then
    run_economics_tests || TEST_STATUS=1
elif [ "$INTEGRATION_ONLY" = true ]; then
    run_integration_tests || TEST_STATUS=1
elif [ "$GATEWAY_ONLY" = true ]; then
    run_gateway_integration_tests || TEST_STATUS=1
else
    # Run all tests
    run_economics_tests || TEST_STATUS=1
    run_gateway_integration_tests || TEST_STATUS=1
    run_integration_tests || TEST_STATUS=1
fi

# Generate report if requested
if [ "$GENERATE_REPORT" = true ]; then
    generate_test_report "$TEST_STATUS"
fi

# Cleanup if requested
if [ "$CLEANUP" = true ]; then
    "$SCRIPT_DIR/run-gateway.sh" stop
fi

# Final status
if [ "$TEST_STATUS" = "0" ]; then
    print_success "All gateway integration tests passed!"
    exit 0
else
    print_error "Some gateway integration tests failed!"
    exit 1
fi
