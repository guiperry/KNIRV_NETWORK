#!/bin/bash

# Month 12 KNIRV_D-TEN Comprehensive Test Runner
# This script executes all Month 12 integration tests and generates reports

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
GATEWAY_URL="http://localhost:8000"
SERVICES=("knirvchain" "knirvgraph" "knirvserver" "knirvoracle" "knirvrouter")
TEST_TIMEOUT="30m"
REPORT_DIR="./test-reports"

echo -e "${BLUE}=== MONTH 12 KNIRV_D-TEN COMPREHENSIVE TEST RUNNER ===${NC}"
echo -e "${BLUE}Starting comprehensive system validation...${NC}"
echo ""

# Create report directory
mkdir -p "$REPORT_DIR"

# Function to check service health
check_service_health() {
    local service=$1
    local url="${GATEWAY_URL}/${service}/health"
    
    echo -n "Checking $service... "
    
    if curl -s -f "$url" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Healthy${NC}"
        return 0
    else
        echo -e "${RED}✗ Unhealthy${NC}"
        return 1
    fi
}

# Function to wait for services
wait_for_services() {
    echo -e "${YELLOW}Checking service health...${NC}"
    
    local all_healthy=true
    
    for service in "${SERVICES[@]}"; do
        if ! check_service_health "$service"; then
            all_healthy=false
        fi
    done
    
    if [ "$all_healthy" = false ]; then
        echo -e "${RED}Some services are not healthy. Please start all services before running tests.${NC}"
        echo "Required services: ${SERVICES[*]}"
        exit 1
    fi
    
    echo -e "${GREEN}All services are healthy!${NC}"
    echo ""
}

# Function to run a specific test suite
run_test_suite() {
    local test_name=$1
    local test_function=$2
    local description=$3
    
    echo -e "${BLUE}Running $test_name...${NC}"
    echo "Description: $description"
    echo ""
    
    local start_time=$(date +%s)
    local log_file="$REPORT_DIR/${test_name,,}_$(date +%Y%m%d_%H%M%S).log"
    
    if go test -v -timeout "$TEST_TIMEOUT" -run "$test_function" 2>&1 | tee "$log_file"; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        echo -e "${GREEN}✓ $test_name completed successfully in ${duration}s${NC}"
        echo ""
        return 0
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        echo -e "${RED}✗ $test_name failed after ${duration}s${NC}"
        echo "Log file: $log_file"
        echo ""
        return 1
    fi
}

# Function to run comprehensive test suite
run_comprehensive_tests() {
    echo -e "${BLUE}Running Month 12 Comprehensive Test Suite...${NC}"
    echo "This will execute all test suites and generate a comprehensive report."
    echo ""
    
    local start_time=$(date +%s)
    local log_file="$REPORT_DIR/month12_comprehensive_$(date +%Y%m%d_%H%M%S).log"
    
    if go test -v -timeout "$TEST_TIMEOUT" -run TestMonth12ComprehensiveTestSuite 2>&1 | tee "$log_file"; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        echo -e "${GREEN}✓ Month 12 Comprehensive Test Suite completed successfully in ${duration}s${NC}"
        
        # Look for generated report
        local report_file=$(ls -t month12_comprehensive_test_report_*.json 2>/dev/null | head -1)
        if [ -n "$report_file" ]; then
            echo -e "${GREEN}Comprehensive report generated: $report_file${NC}"
            mv "$report_file" "$REPORT_DIR/"
        fi
        
        echo ""
        return 0
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        echo -e "${RED}✗ Month 12 Comprehensive Test Suite failed after ${duration}s${NC}"
        echo "Log file: $log_file"
        echo ""
        return 1
    fi
}

# Function to display usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -h, --help              Show this help message"
    echo "  -c, --comprehensive     Run comprehensive test suite only"
    echo "  -i, --individual        Run individual test suites"
    echo "  -a, --all              Run all tests (individual + comprehensive)"
    echo "  --skip-health-check    Skip service health check"
    echo ""
    echo "Individual Test Suites:"
    echo "  --e2e                  Run E2E Integration Tests"
    echo "  --performance          Run Performance and Load Tests"
    echo "  --security             Run Security Tests"
    echo "  --cross-component      Run Cross-Component Integration Tests"
    echo "  --knirv-router         Run KNIRV-ROUTER Tests"
    echo "  --websocket            Run WebSocket Tests"
    echo ""
    echo "Examples:"
    echo "  $0 --comprehensive     # Run comprehensive test suite"
    echo "  $0 --all              # Run all tests"
    echo "  $0 --e2e --security   # Run specific test suites"
}

# Parse command line arguments
COMPREHENSIVE=false
INDIVIDUAL=false
ALL=false
SKIP_HEALTH_CHECK=false
RUN_E2E=false
RUN_PERFORMANCE=false
RUN_SECURITY=false
RUN_CROSS_COMPONENT=false
RUN_KNIRV_ROUTER=false
RUN_WEBSOCKET=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_usage
            exit 0
            ;;
        -c|--comprehensive)
            COMPREHENSIVE=true
            shift
            ;;
        -i|--individual)
            INDIVIDUAL=true
            shift
            ;;
        -a|--all)
            ALL=true
            shift
            ;;
        --skip-health-check)
            SKIP_HEALTH_CHECK=true
            shift
            ;;
        --e2e)
            RUN_E2E=true
            shift
            ;;
        --performance)
            RUN_PERFORMANCE=true
            shift
            ;;
        --security)
            RUN_SECURITY=true
            shift
            ;;
        --cross-component)
            RUN_CROSS_COMPONENT=true
            shift
            ;;
        --knirv-router)
            RUN_KNIRV_ROUTER=true
            shift
            ;;
        --websocket)
            RUN_WEBSOCKET=true
            shift
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            show_usage
            exit 1
            ;;
    esac
done

# Default to comprehensive if no options specified
if [ "$COMPREHENSIVE" = false ] && [ "$INDIVIDUAL" = false ] && [ "$ALL" = false ] && \
   [ "$RUN_E2E" = false ] && [ "$RUN_PERFORMANCE" = false ] && [ "$RUN_SECURITY" = false ] && \
   [ "$RUN_CROSS_COMPONENT" = false ] && [ "$RUN_KNIRV_ROUTER" = false ] && [ "$RUN_WEBSOCKET" = false ]; then
    COMPREHENSIVE=true
fi

# Check service health unless skipped
if [ "$SKIP_HEALTH_CHECK" = false ]; then
    wait_for_services
fi

# Track test results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Run individual test suites if requested
if [ "$INDIVIDUAL" = true ] || [ "$ALL" = true ] || [ "$RUN_E2E" = true ] || [ "$RUN_PERFORMANCE" = true ] || \
   [ "$RUN_SECURITY" = true ] || [ "$RUN_CROSS_COMPONENT" = true ] || [ "$RUN_KNIRV_ROUTER" = true ] || [ "$RUN_WEBSOCKET" = true ]; then
    
    echo -e "${YELLOW}Running Individual Test Suites...${NC}"
    echo ""
    
    if [ "$INDIVIDUAL" = true ] || [ "$ALL" = true ] || [ "$RUN_E2E" = true ]; then
        TOTAL_TESTS=$((TOTAL_TESTS + 1))
        if run_test_suite "E2E Integration Tests" "TestE2ETestSuite" "Complete workflow testing across all KNIRV components"; then
            PASSED_TESTS=$((PASSED_TESTS + 1))
        else
            FAILED_TESTS=$((FAILED_TESTS + 1))
        fi
    fi
    
    if [ "$INDIVIDUAL" = true ] || [ "$ALL" = true ] || [ "$RUN_PERFORMANCE" = true ]; then
        TOTAL_TESTS=$((TOTAL_TESTS + 1))
        if run_test_suite "Performance and Load Tests" "TestPerformanceAndLoad" "System performance validation under load"; then
            PASSED_TESTS=$((PASSED_TESTS + 1))
        else
            FAILED_TESTS=$((FAILED_TESTS + 1))
        fi
    fi
    
    if [ "$INDIVIDUAL" = true ] || [ "$ALL" = true ] || [ "$RUN_SECURITY" = true ]; then
        TOTAL_TESTS=$((TOTAL_TESTS + 1))
        if run_test_suite "Security Tests" "TestSecurityTestSuite" "Comprehensive security validation"; then
            PASSED_TESTS=$((PASSED_TESTS + 1))
        else
            FAILED_TESTS=$((FAILED_TESTS + 1))
        fi
    fi
    
    if [ "$INDIVIDUAL" = true ] || [ "$ALL" = true ] || [ "$RUN_CROSS_COMPONENT" = true ]; then
        TOTAL_TESTS=$((TOTAL_TESTS + 1))
        if run_test_suite "Cross-Component Integration Tests" "TestCrossComponentTestSuite" "Integration validation between KNIRV components"; then
            PASSED_TESTS=$((PASSED_TESTS + 1))
        else
            FAILED_TESTS=$((FAILED_TESTS + 1))
        fi
    fi
    
    if [ "$INDIVIDUAL" = true ] || [ "$ALL" = true ] || [ "$RUN_KNIRV_ROUTER" = true ]; then
        TOTAL_TESTS=$((TOTAL_TESTS + 1))
        if run_test_suite "KNIRV-ROUTER Tests" "TestKNIRVROUTERTestSuite" "KNIRV-ROUTER connectivity and functionality testing"; then
            PASSED_TESTS=$((PASSED_TESTS + 1))
        else
            FAILED_TESTS=$((FAILED_TESTS + 1))
        fi
    fi
    
    if [ "$INDIVIDUAL" = true ] || [ "$ALL" = true ] || [ "$RUN_WEBSOCKET" = true ]; then
        TOTAL_TESTS=$((TOTAL_TESTS + 1))
        if run_test_suite "WebSocket Tests" "TestWebSocketTestSuite" "Real-time communication and WebSocket functionality"; then
            PASSED_TESTS=$((PASSED_TESTS + 1))
        else
            FAILED_TESTS=$((FAILED_TESTS + 1))
        fi
    fi
fi

# Run comprehensive test suite if requested
if [ "$COMPREHENSIVE" = true ] || [ "$ALL" = true ]; then
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    if run_comprehensive_tests; then
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
fi

# Print final summary
echo -e "${BLUE}=== TEST EXECUTION SUMMARY ===${NC}"
echo "Total Test Suites: $TOTAL_TESTS"
echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed: ${RED}$FAILED_TESTS${NC}"
echo "Report Directory: $REPORT_DIR"
echo ""

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed! System is ready for production deployment.${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed. Please review the logs and fix issues before deployment.${NC}"
    exit 1
fi
