#!/bin/bash

# Gateway Migration Integration Tests Runner
# This script runs all gateway migration tests from the integration-tests module
# Run from project root: ./integration-tests/run_gateway_migration_tests.sh

set -e

echo "🎯 KNIRV Gateway Migration - Integration Tests"
echo "=============================================="

# Get script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Function to run a test script
run_test_script() {
    local script_name="$1"
    local description="$2"
    local script_path="$PROJECT_ROOT/scripts/$script_name"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo ""
    echo -e "${BLUE}🧪 Running: $description${NC}"
    echo "Script: $script_path"
    echo "----------------------------------------"
    
    if [ ! -f "$script_path" ]; then
        echo -e "${RED}❌ FAIL: Script not found${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
    
    # Make script executable
    chmod +x "$script_path"
    
    # Run the script
    if bash "$script_path"; then
        echo -e "${GREEN}✅ PASS: $description${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        echo -e "${RED}❌ FAIL: $description${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# Function to run Go integration tests
run_go_tests() {
    local test_name="$1"
    local description="$2"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo ""
    echo -e "${BLUE}🧪 Running: $description${NC}"
    echo "Test: $test_name"
    echo "----------------------------------------"
    
    cd "$SCRIPT_DIR"
    
    if go test -v -run "$test_name" -timeout 60s; then
        echo -e "${GREEN}✅ PASS: $description${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        echo -e "${RED}❌ FAIL: $description${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

echo ""
echo "🔍 Phase 1: Script-based Tests"
echo "==============================="

# Test 1: Gateway Migration Test
run_test_script "test-gateway-migration.sh" "Gateway Migration Functionality"

# Test 2: Economics Integration Test
run_test_script "test-economics-integration.sh" "Economics Integration"

# Test 3: Complete Migration Validation
run_test_script "validate-complete-migration.sh" "Complete Migration Validation"

echo ""
echo "🔍 Phase 2: Go Integration Tests"
echo "================================="

# Test 4: Gateway Migration Go Tests
run_go_tests "TestGatewayMigrationComplete" "Gateway Migration Complete Test"

# Test 5: Gateway Migration Scripts Test
run_go_tests "TestGatewayMigrationScripts" "Gateway Migration Scripts Test"

echo ""
echo "🔍 Phase 3: Service Integration Tests"
echo "====================================="

# Test 6: Economics Integration (if available)
if [ -f "$SCRIPT_DIR/economics_integration_test.go" ]; then
    run_go_tests "TestEconomicsIntegration" "Economics Service Integration"
else
    echo "⚠️  Economics integration test not available"
fi

echo ""
echo "🧹 Cleanup and Summary"
echo "======================"

# Calculate success rate
if [ $TOTAL_TESTS -gt 0 ]; then
    SUCCESS_RATE=$((PASSED_TESTS * 100 / TOTAL_TESTS))
else
    SUCCESS_RATE=0
fi

echo ""
echo "📊 Gateway Migration Integration Test Results"
echo "============================================="
echo -e "Total Tests: ${BLUE}$TOTAL_TESTS${NC}"
echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed: ${RED}$FAILED_TESTS${NC}"
echo -e "Success Rate: ${PURPLE}$SUCCESS_RATE%${NC}"

echo ""
if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "🎉 ${GREEN}ALL GATEWAY MIGRATION TESTS PASSED!${NC}"
    echo ""
    echo "✅ Gateway migration is fully functional:"
    echo "   • All scripts working correctly"
    echo "   • Integration tests passing"
    echo "   • Service connectivity verified"
    echo "   • Migration validation complete"
    echo ""
    echo "🚀 The gateway migration is ready for production!"
    
    exit 0
else
    echo -e "⚠️  ${YELLOW}SOME TESTS FAILED${NC}"
    echo ""
    echo "Some gateway migration tests failed. This may be due to:"
    echo "• Services not running (KNIRVORACLE, Netlify dev, etc.)"
    echo "• Network connectivity issues"
    echo "• Configuration problems"
    echo ""
    echo "Review the failed tests and ensure all services are running."
    echo ""
    echo "To start required services:"
    echo "1. Start KNIRVORACLE: ./scripts/start-with-economics.sh"
    echo "2. Start Netlify dev: cd KNIRVGATEWAY && netlify dev"
    echo ""
    echo "Then re-run this test suite."
    
    exit 1
fi
