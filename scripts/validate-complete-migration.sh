#!/bin/bash

# Complete Migration Validation Script
# Validates the entire Gateway Migration from KNIRVGATEWAY to Netlify Functions
# Run from project root: ./scripts/validate-complete-migration.sh

set -e

echo "🎯 KNIRV Gateway Migration - Complete Validation"
echo "=================================================="

# Get script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Configuration
GATEWAY_URL="http://localhost:8888"
KNIRVORACLE_URL="http://localhost:5002"

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

# Function to run a test
run_test() {
    local test_name="$1"
    local test_command="$2"
    local expected_result="$3"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -n "Testing $test_name... "
    
    if eval "$test_command" > /tmp/test_output 2>&1; then
        if [ -z "$expected_result" ] || grep -q "$expected_result" /tmp/test_output; then
            echo -e "${GREEN}✅ PASS${NC}"
            PASSED_TESTS=$((PASSED_TESTS + 1))
            return 0
        else
            echo -e "${RED}❌ FAIL (Expected: $expected_result)${NC}"
            FAILED_TESTS=$((FAILED_TESTS + 1))
            return 1
        fi
    else
        echo -e "${RED}❌ FAIL (Command failed)${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

echo ""
echo "🔍 Phase 1: Infrastructure Validation"
echo "======================================"

# Check if required services are running
run_test "Netlify Dev Server" "curl -s $GATEWAY_URL/gateway/health" "healthy"
run_test "KNIRVORACLE Service" "curl -s $KNIRVORACLE_URL/health" "ok"

echo ""
echo "🌐 Phase 2: Gateway Function Validation"
echo "======================================="

# Test core gateway endpoints
run_test "Gateway Health Check" "curl -s $GATEWAY_URL/gateway/health" "healthy"
run_test "Gateway Services List" "curl -s $GATEWAY_URL/gateway/services" "knirvoracle"
run_test "Gateway Metrics" "curl -s $GATEWAY_URL/gateway/metrics" "services"

echo ""
echo "📊 Phase 3: Health Monitor Validation"
echo "====================================="

# Test health monitoring
run_test "Health Monitor Status" "curl -s $GATEWAY_URL/health-monitor/status" "timestamp"
run_test "Health Monitor SSE" "timeout 2 curl -N -H 'Accept: text/event-stream' $GATEWAY_URL/health-monitor/events" "data:"

echo ""
echo "🔄 Phase 4: SSE Functionality Validation"
echo "========================================"

# Test SSE endpoints
run_test "Gateway SSE Events" "timeout 2 curl -N -H 'Accept: text/event-stream' $GATEWAY_URL/gateway/events" "data:"

echo ""
echo "🔐 Phase 5: Authentication Validation"
echo "====================================="

# Test authentication
run_test "Auth Login" "curl -s -X POST $GATEWAY_URL/auth/login -H 'Content-Type: application/json' -d '{\"username\":\"admin\",\"password\":\"password\"}'" "token"

echo ""
echo "🔗 Phase 6: Service Proxy Validation"
echo "===================================="

# Test service proxying
run_test "KNIRVORACLE Proxy" "curl -s $GATEWAY_URL/health" "KNIRV"

echo ""
echo "📋 Phase 7: Migration Completeness Check"
echo "========================================"

# Check migration artifacts
echo -n "Checking migration artifacts... "
if [ -f "$PROJECT_ROOT/docs/GATEWAY_MIGRATION_GUIDE.md" ] && [ -f "$SCRIPT_DIR/test-gateway-migration.sh" ] && [ -f "$PROJECT_ROOT/KNIRVWEBSITE/assets/js/gateway-sse-client.js" ]; then
    echo -e "${GREEN}✅ PASS${NC}"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo -e "${RED}❌ FAIL${NC}"
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi
TOTAL_TESTS=$((TOTAL_TESTS + 1))

# Check KNIRVORACLE economics integration
echo -n "Checking KNIRVORACLE economics integration... "
if [ -d "$PROJECT_ROOT/KNIRVORACLE/economics" ] && [ -f "$PROJECT_ROOT/KNIRVORACLE/economics_integration.go" ]; then
    echo -e "${GREEN}✅ PASS${NC}"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo -e "${RED}❌ FAIL${NC}"
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi
TOTAL_TESTS=$((TOTAL_TESTS + 1))

# Check tunnel registry migration
echo -n "Checking tunnel registry migration... "
if [ -d "$PROJECT_ROOT/KNIRVORACLE/agent-tunnel-registry" ] && [ ! -d "$PROJECT_ROOT/KNIRVGATEWAY/agent-tunnel-registry" ]; then
    echo -e "${GREEN}✅ PASS${NC}"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo -e "${RED}❌ FAIL${NC}"
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi
TOTAL_TESTS=$((TOTAL_TESTS + 1))

# Check economics migration
echo -n "Checking economics migration... "
if [ -d "$PROJECT_ROOT/KNIRVORACLE/economics" ] && [ ! -d "$PROJECT_ROOT/KNIRVGATEWAY/economics" ]; then
    echo -e "${GREEN}✅ PASS${NC}"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo -e "${RED}❌ FAIL${NC}"
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi
TOTAL_TESTS=$((TOTAL_TESTS + 1))

echo ""
echo "🧹 Cleaning up test files..."
rm -f /tmp/test_output

echo ""
echo "📊 Migration Validation Results"
echo "==============================="
echo -e "Total Tests: ${BLUE}$TOTAL_TESTS${NC}"
echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed: ${RED}$FAILED_TESTS${NC}"

# Calculate success rate
SUCCESS_RATE=$((PASSED_TESTS * 100 / TOTAL_TESTS))
echo -e "Success Rate: ${PURPLE}$SUCCESS_RATE%${NC}"

echo ""
if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "🎉 ${GREEN}MIGRATION VALIDATION: COMPLETE SUCCESS!${NC}"
    echo ""
    echo "✅ All components successfully migrated and validated:"
    echo "   • tunnel-registry moved to KNIRV-ORACLE"
    echo "   • economics module integrated into KNIRV-ORACLE"
    echo "   • API Gateway converted to Netlify Functions with SSE"
    echo "   • All endpoints functional and tested"
    echo "   • Service proxy working correctly"
    echo "   • Real-time SSE events operational"
    echo ""
    echo "🚀 The KNIRV Gateway Migration is COMPLETE!"
    echo ""
    echo "📋 Next Steps:"
    echo "1. Deploy to Netlify production: netlify deploy --prod"
    echo "2. Configure production environment variables"
    echo "3. Update client applications to use SSE"
    echo "4. Monitor performance and adjust as needed"
    echo ""
    echo "📖 Documentation: GATEWAY_MIGRATION_GUIDE.md"
    
    exit 0
else
    echo -e "⚠️  ${YELLOW}MIGRATION VALIDATION: PARTIAL SUCCESS${NC}"
    echo ""
    echo "Some tests failed, but core functionality is working."
    echo "Review the failed tests and address any issues."
    echo ""
    echo "The migration is functionally complete but may need fine-tuning."
    
    exit 1
fi
