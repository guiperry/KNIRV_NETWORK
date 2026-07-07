#!/bin/bash

# KNIRVCHAIN Comprehensive Test Suite
# This script runs all tests for the KNIRVCHAIN multi-model blockchain system

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
KNIRVCHAIN_DIR="KNIRVCHAIN"
TEST_RESULTS_DIR="integration-tests/test-reports"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
REPORT_FILE="$TEST_RESULTS_DIR/knirvchain_test_report_$TIMESTAMP.json"

# Ensure directories exist
mkdir -p "$TEST_RESULTS_DIR"

echo -e "${BLUE}🚀 KNIRVCHAIN Comprehensive Test Suite${NC}"
echo -e "${BLUE}=======================================${NC}"
echo "Timestamp: $(date)"
echo "Report will be saved to: $REPORT_FILE"
echo ""

# Initialize test results
cat > "$REPORT_FILE" << EOF
{
  "test_suite": "KNIRVCHAIN Comprehensive Tests",
  "timestamp": "$(date -Iseconds)",
  "results": {
EOF

# Function to log test results
log_test_result() {
    local test_name="$1"
    local status="$2"
    local duration="$3"
    local details="$4"
    
    echo "    \"$test_name\": {" >> "$REPORT_FILE"
    echo "      \"status\": \"$status\"," >> "$REPORT_FILE"
    echo "      \"duration\": \"$duration\"," >> "$REPORT_FILE"
    echo "      \"details\": \"$details\"" >> "$REPORT_FILE"
    echo "    }," >> "$REPORT_FILE"
}

# Function to run a test with timing
run_test() {
    local test_name="$1"
    local test_command="$2"
    local test_description="$3"
    
    echo -e "${YELLOW}🧪 Running: $test_description${NC}"
    
    local start_time=$(date +%s)
    
    if eval "$test_command" > /tmp/test_output_$$ 2>&1; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        echo -e "${GREEN}✅ PASSED: $test_name (${duration}s)${NC}"
        log_test_result "$test_name" "PASSED" "${duration}s" "$(cat /tmp/test_output_$$)"
        return 0
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        echo -e "${RED}❌ FAILED: $test_name (${duration}s)${NC}"
        echo -e "${RED}Error output:${NC}"
        cat /tmp/test_output_$$
        log_test_result "$test_name" "FAILED" "${duration}s" "$(cat /tmp/test_output_$$)"
        return 1
    fi
    
    rm -f /tmp/test_output_$$
}

# Change to KNIRVCHAIN directory
cd "$KNIRVCHAIN_DIR"

echo -e "${BLUE}📋 Pre-test Setup${NC}"
echo "Working directory: $(pwd)"
echo "Rust version: $(rustc --version)"
echo "Cargo version: $(cargo --version)"
echo ""

# Test 1: Code Quality Checks
echo -e "${BLUE}🔍 Code Quality Checks${NC}"
run_test "code_format_check" "cargo fmt -- --check" "Code formatting check"
run_test "clippy_lints" "cargo clippy -- -D warnings" "Clippy linting"
run_test "compilation_check" "cargo check" "Compilation check"

# Test 2: Unit Tests
echo -e "${BLUE}🧪 Unit Tests${NC}"
run_test "unit_tests" "cargo test --lib" "Unit tests"

# Test 3: Integration Tests
echo -e "${BLUE}🔗 Integration Tests${NC}"
run_test "integration_tests" "cargo test --test integration_tests" "Integration tests"

# Test 4: Performance Tests
echo -e "${BLUE}⚡ Performance Tests${NC}"
run_test "performance_tests" "cargo test --test performance_tests --release" "Performance tests"

# Test 5: Documentation Tests
echo -e "${BLUE}📚 Documentation Tests${NC}"
run_test "doc_tests" "cargo test --doc" "Documentation tests"

# Test 6: Build Tests
echo -e "${BLUE}🏗️ Build Tests${NC}"
run_test "debug_build" "cargo build" "Debug build"
run_test "release_build" "cargo build --release" "Release build"

# Test 7: Feature Tests
echo -e "${BLUE}🎯 Feature Tests${NC}"
run_test "testnet_features" "cargo test --features testnet" "Testnet features"
run_test "production_features" "cargo test --features production" "Production features"

# Test 8: Environment Variable Tests
echo -e "${BLUE}🌍 Environment Tests${NC}"
if [ -f ".env" ]; then
    run_test "env_file_test" "test -f .env && echo 'Environment file exists'" "Environment file check"
    run_test "env_vars_test" "source .env && test -n \"\$GEMINI_API_KEY\" && echo 'API keys configured'" "Environment variables check"
else
    echo -e "${YELLOW}⚠️  No .env file found - skipping environment tests${NC}"
    log_test_result "env_file_test" "SKIPPED" "0s" "No .env file found"
    log_test_result "env_vars_test" "SKIPPED" "0s" "No .env file found"
fi

# Test 9: API Endpoint Tests (if server is running)
echo -e "${BLUE}🌐 API Tests${NC}"
if curl -s http://localhost:8080/health > /dev/null 2>&1; then
    run_test "health_endpoint" "curl -s http://localhost:8080/health | grep -q 'ok'" "Health endpoint test"
    run_test "models_endpoint" "curl -s http://localhost:8080/v3/models/list" "Models endpoint test"
else
    echo -e "${YELLOW}⚠️  Server not running - skipping API tests${NC}"
    log_test_result "health_endpoint" "SKIPPED" "0s" "Server not running"
    log_test_result "models_endpoint" "SKIPPED" "0s" "Server not running"
fi

# Test 10: Memory and Resource Tests
echo -e "${BLUE}💾 Resource Tests${NC}"
run_test "memory_test" "cargo test test_memory_usage_under_load --release" "Memory usage test"

# Finalize test results JSON
sed -i '$ s/,$//' "$REPORT_FILE"  # Remove last comma
cat >> "$REPORT_FILE" << EOF
  },
  "summary": {
    "total_tests": $(grep -c '"status":' "$REPORT_FILE"),
    "passed": $(grep -c '"status": "PASSED"' "$REPORT_FILE"),
    "failed": $(grep -c '"status": "FAILED"' "$REPORT_FILE"),
    "skipped": $(grep -c '"status": "SKIPPED"' "$REPORT_FILE")
  }
}
EOF

# Generate summary
echo ""
echo -e "${BLUE}📊 Test Summary${NC}"
echo -e "${BLUE}===============${NC}"

TOTAL_TESTS=$(grep -c '"status":' "$REPORT_FILE")
PASSED_TESTS=$(grep -c '"status": "PASSED"' "$REPORT_FILE")
FAILED_TESTS=$(grep -c '"status": "FAILED"' "$REPORT_FILE")
SKIPPED_TESTS=$(grep -c '"status": "SKIPPED"' "$REPORT_FILE")

echo "Total Tests: $TOTAL_TESTS"
echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed: ${RED}$FAILED_TESTS${NC}"
echo -e "Skipped: ${YELLOW}$SKIPPED_TESTS${NC}"
echo ""
echo "Detailed report saved to: $REPORT_FILE"

# Return to original directory
cd ..

# Exit with appropriate code
if [ "$FAILED_TESTS" -eq 0 ]; then
    echo -e "${GREEN}🎉 All tests passed successfully!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed. Check the report for details.${NC}"
    exit 1
fi
