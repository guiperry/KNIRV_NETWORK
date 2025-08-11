#!/bin/bash

# KNIRVGRAPH Integration Test Script for Unified Test Runner
# This script integrates KNIRVGRAPH economics tests with the unified test suite
# Run from project root: ./scripts/test-knirvgraph-integration.sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
REPORT_DIR="$PROJECT_ROOT/integration-tests/test-reports"
KNIRVGRAPH_URL=${KNIRVGRAPH_URL:-http://localhost:8081}
KNIRVROOT_URL=${KNIRVROOT_URL:-http://localhost:1317}

# Helper functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Create report directory
mkdir -p "$REPORT_DIR"

# Test report file
REPORT_FILE="$REPORT_DIR/knirvgraph_integration_test_$TIMESTAMP.json"

echo "🧪 KNIRVGRAPH Integration Tests for Unified Test Suite"
echo "📊 Configuration:"
echo "  - KNIRVGRAPH URL: $KNIRVGRAPH_URL"
echo "  - KNIRVROOT URL: $KNIRVROOT_URL"
echo "  - Report File: $REPORT_FILE"
echo ""

# Initialize test report
cat > "$REPORT_FILE" << EOF
{
  "test_suite": "KNIRVGRAPH Integration Tests",
  "timestamp": "$TIMESTAMP",
  "configuration": {
    "knirvgraph_url": "$KNIRVGRAPH_URL",
    "knirvroot_url": "$KNIRVROOT_URL"
  },
  "tests": [],
  "summary": {
    "total": 0,
    "passed": 0,
    "failed": 0,
    "skipped": 0
  }
}
EOF

# Function to add test result to report
add_test_result() {
    local test_name="$1"
    local status="$2"
    local message="$3"
    local duration="$4"
    
    # Create temporary file with test result
    local temp_result=$(mktemp)
    cat > "$temp_result" << EOF
{
  "name": "$test_name",
  "status": "$status",
  "message": "$message",
  "duration": "$duration",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
    
    # Add to report using jq if available, otherwise append manually
    if command -v jq >/dev/null 2>&1; then
        local temp_report=$(mktemp)
        jq --argjson test "$(cat "$temp_result")" '.tests += [$test] | .summary.total += 1 | .summary.'$status' += 1' "$REPORT_FILE" > "$temp_report"
        mv "$temp_report" "$REPORT_FILE"
    else
        # Fallback: manual JSON manipulation
        log_warning "jq not available, using fallback JSON handling"
    fi
    
    rm -f "$temp_result"
}

# Function to run a test with timing
run_timed_test() {
    local test_name="$1"
    local test_command="$2"
    
    log_info "Running: $test_name"
    
    local start_time=$(date +%s.%N)
    
    if eval "$test_command" >/dev/null 2>&1; then
        local end_time=$(date +%s.%N)
        local duration=$(echo "$end_time - $start_time" | bc -l 2>/dev/null || echo "0")
        
        log_success "$test_name passed (${duration}s)"
        add_test_result "$test_name" "passed" "Test completed successfully" "$duration"
        return 0
    else
        local end_time=$(date +%s.%N)
        local duration=$(echo "$end_time - $start_time" | bc -l 2>/dev/null || echo "0")
        
        log_error "$test_name failed (${duration}s)"
        add_test_result "$test_name" "failed" "Test execution failed" "$duration"
        return 1
    fi
}

# Test functions
test_knirvgraph_health() {
    curl -s --max-time 10 "$KNIRVGRAPH_URL/health" | grep -q '"status":"healthy"'
}

test_knirvgraph_economics_metrics() {
    curl -s --max-time 10 "$KNIRVGRAPH_URL/economics/metrics" | grep -q 'total_nrvs_created'
}

test_nrv_system_endpoints() {
    # Test NRV vectors endpoint
    curl -s --max-time 10 "$KNIRVGRAPH_URL/nrv/vectors" >/dev/null
}

test_skill_confirmation_endpoint() {
    # Test skill confirmation endpoint with mock data
    curl -s --max-time 10 -X POST "$KNIRVGRAPH_URL/economics/skill/confirm" \
        -H "Content-Type: application/json" \
        -d '{"skill_id":"test","nrv_id":"test","creator_id":"test"}' | grep -q 'success'
}

test_proof_of_solution_endpoint() {
    # Test proof of solution endpoint
    curl -s --max-time 10 -X POST "$KNIRVGRAPH_URL/economics/proof/solution" \
        -H "Content-Type: application/json" \
        -d '{"error_node_id":"test","skill_node_id":"test","solver_id":"test","efficiency_score":0.9,"quality_score":0.8}' | grep -q 'success'
}

test_reward_distribution_endpoint() {
    # Test reward distribution endpoint
    curl -s --max-time 10 -X POST "$KNIRVGRAPH_URL/economics/rewards/distribute" \
        -H "Content-Type: application/json" \
        -d '{"recipient_id":"test","amount":"1000","reason":"test"}' | grep -q 'success'
}

# Main test execution
main() {
    log_info "Starting KNIRVGRAPH Integration Tests..."
    
    local failed_tests=0
    local total_tests=0
    
    # Core functionality tests
    ((total_tests++))
    if ! run_timed_test "KNIRVGRAPH Health Check" "test_knirvgraph_health"; then
        ((failed_tests++))
        log_error "KNIRVGRAPH is not accessible. Skipping remaining tests."
        
        # Add skipped tests to report
        local skipped_tests=("Economics Metrics" "NRV System Endpoints" "Skill Confirmation" "Proof of Solution" "Reward Distribution")
        for test in "${skipped_tests[@]}"; do
            add_test_result "$test" "skipped" "KNIRVGRAPH not accessible" "0"
        done
        
        # Finalize report
        finalize_report $failed_tests $total_tests
        return 1
    fi
    
    # Economics integration tests
    ((total_tests++))
    if ! run_timed_test "Economics Metrics Endpoint" "test_knirvgraph_economics_metrics"; then
        ((failed_tests++))
    fi
    
    ((total_tests++))
    if ! run_timed_test "NRV System Endpoints" "test_nrv_system_endpoints"; then
        ((failed_tests++))
    fi
    
    ((total_tests++))
    if ! run_timed_test "Skill Confirmation Endpoint" "test_skill_confirmation_endpoint"; then
        ((failed_tests++))
    fi
    
    ((total_tests++))
    if ! run_timed_test "Proof of Solution Endpoint" "test_proof_of_solution_endpoint"; then
        ((failed_tests++))
    fi
    
    ((total_tests++))
    if ! run_timed_test "Reward Distribution Endpoint" "test_reward_distribution_endpoint"; then
        ((failed_tests++))
    fi
    
    # Finalize report
    finalize_report $failed_tests $total_tests
    
    # Test summary
    echo ""
    echo "📊 KNIRVGRAPH Integration Test Summary:"
    echo "  - Total Tests: $total_tests"
    echo "  - Passed: $((total_tests - failed_tests))"
    echo "  - Failed: $failed_tests"
    echo "  - Report: $REPORT_FILE"
    
    if [ $failed_tests -eq 0 ]; then
        log_success "All KNIRVGRAPH integration tests passed!"
        return 0
    else
        log_error "Some KNIRVGRAPH integration tests failed."
        return 1
    fi
}

# Function to finalize the test report
finalize_report() {
    local failed_tests=$1
    local total_tests=$2
    
    if command -v jq >/dev/null 2>&1; then
        local temp_report=$(mktemp)
        jq --arg end_time "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
           --argjson total "$total_tests" \
           --argjson failed "$failed_tests" \
           --argjson passed "$((total_tests - failed_tests))" \
           '.end_time = $end_time | .summary.total = $total | .summary.failed = $failed | .summary.passed = $passed' \
           "$REPORT_FILE" > "$temp_report"
        mv "$temp_report" "$REPORT_FILE"
    fi
}

# Integration with unified test runner
if [ "$1" = "--unified" ]; then
    # Called from unified test runner
    echo "KNIRVGRAPH Integration Tests" >&2
    main
    exit $?
else
    # Standalone execution
    main "$@"
fi
