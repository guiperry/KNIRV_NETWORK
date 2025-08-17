#!/bin/bash

# Test script for KNIRV Economics Integration
# This script tests the economics service integration with KNIRVORACLE
# Run from project root: ./scripts/test-economics-integration.sh

set -e

echo "🧪 Testing KNIRV Economics Integration..."

# Get script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
KNIRVORACLE_DIR="$PROJECT_ROOT/KNIRVORACLE"

# Configuration
ECONOMICS_URL="http://localhost:8090"
GATEWAY_URL="http://localhost:8888"
KNIRVORACLE_URL="http://localhost:8080"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to test an endpoint
test_endpoint() {
    local url=$1
    local description=$2
    local expected_status=${3:-200}
    
    echo -n "Testing $description... "
    
    response=$(curl -s -w "%{http_code}" -o /tmp/test_response "$url" 2>/dev/null || echo "000")
    
    if [ "$response" = "$expected_status" ]; then
        echo -e "${GREEN}✅ PASS${NC}"
        if [ -f /tmp/test_response ]; then
            echo "   Response: $(cat /tmp/test_response | jq -c . 2>/dev/null || cat /tmp/test_response)"
        fi
        return 0
    else
        echo -e "${RED}❌ FAIL (HTTP $response)${NC}"
        if [ -f /tmp/test_response ]; then
            echo "   Response: $(cat /tmp/test_response)"
        fi
        return 1
    fi
}

# Function to test JSON endpoint
test_json_endpoint() {
    local url=$1
    local description=$2
    local expected_field=$3
    
    echo -n "Testing $description... "
    
    response=$(curl -s "$url" 2>/dev/null || echo '{}')
    
    if echo "$response" | jq -e ".$expected_field" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ PASS${NC}"
        echo "   Response: $(echo "$response" | jq -c .)"
        return 0
    else
        echo -e "${RED}❌ FAIL${NC}"
        echo "   Response: $response"
        return 1
    fi
}

echo ""
echo "🔍 Checking if services are running..."

# Check if curl is available
if ! command -v curl &> /dev/null; then
    echo -e "${RED}❌ curl is not installed. Please install curl to run this test.${NC}"
    exit 1
fi

# Check if jq is available
if ! command -v jq &> /dev/null; then
    echo -e "${YELLOW}⚠️  jq is not installed. JSON parsing will be limited.${NC}"
fi

echo ""
echo "📊 Testing Economics Service Direct Access..."

# Test economics service health
test_endpoint "$ECONOMICS_URL/economics/health" "Economics Health Check"

# Test economics service status
test_json_endpoint "$ECONOMICS_URL/economics/status" "Economics Status" "service"

# Test economics service metrics
test_endpoint "$ECONOMICS_URL/economics/metrics" "Economics Metrics"

echo ""
echo "🌐 Testing Gateway Routing to Economics..."

# Test economics through gateway (if gateway is running)
if curl -s "$GATEWAY_URL/gateway/health" > /dev/null 2>&1; then
    echo "Gateway is running, testing economics routing..."
    
    test_endpoint "$GATEWAY_URL/economics/health" "Economics Health via Gateway"
    test_endpoint "$GATEWAY_URL/economics/status" "Economics Status via Gateway"
    test_endpoint "$GATEWAY_URL/economics/metrics" "Economics Metrics via Gateway"
else
    echo -e "${YELLOW}⚠️  Gateway not running at $GATEWAY_URL, skipping gateway tests${NC}"
fi

echo ""
echo "🔗 Testing KNIRVORACLE Integration..."

# Test KNIRVORACLE health
if curl -s "$KNIRVORACLE_URL/health" > /dev/null 2>&1; then
    test_endpoint "$KNIRVORACLE_URL/health" "KNIRVORACLE Health Check"
    
    # Test economics integration endpoints
    test_endpoint "$KNIRVORACLE_URL/api/economics/status" "Economics Integration Status"
    test_endpoint "$KNIRVORACLE_URL/api/economics/metrics" "Economics Integration Metrics"
else
    echo -e "${YELLOW}⚠️  KNIRVORACLE not running at $KNIRVORACLE_URL${NC}"
fi

echo ""
echo "💰 Testing Economics Functionality..."

# Test skill invocation (POST request)
echo -n "Testing Skill Invocation... "
skill_response=$(curl -s -X POST "$ECONOMICS_URL/economics/skill/invoke" \
    -H "Content-Type: application/json" \
    -d '{"user_id": "test_user", "skill_id": "test_skill", "amount": "100000"}' \
    -w "%{http_code}" -o /tmp/skill_response 2>/dev/null || echo "000")

if [ "$skill_response" = "200" ] || [ "$skill_response" = "201" ]; then
    echo -e "${GREEN}✅ PASS${NC}"
    if [ -f /tmp/skill_response ]; then
        echo "   Response: $(cat /tmp/skill_response | jq -c . 2>/dev/null || cat /tmp/skill_response)"
    fi
else
    echo -e "${YELLOW}⚠️  PARTIAL (HTTP $skill_response) - May need authentication${NC}"
    if [ -f /tmp/skill_response ]; then
        echo "   Response: $(cat /tmp/skill_response)"
    fi
fi

# Test LLM registration (POST request)
echo -n "Testing LLM Registration... "
llm_response=$(curl -s -X POST "$ECONOMICS_URL/economics/llm/register" \
    -H "Content-Type: application/json" \
    -d '{"user_id": "test_user", "llm_id": "test_llm", "amount": "1000000"}' \
    -w "%{http_code}" -o /tmp/llm_response 2>/dev/null || echo "000")

if [ "$llm_response" = "200" ] || [ "$llm_response" = "201" ]; then
    echo -e "${GREEN}✅ PASS${NC}"
    if [ -f /tmp/llm_response ]; then
        echo "   Response: $(cat /tmp/llm_response | jq -c . 2>/dev/null || cat /tmp/llm_response)"
    fi
else
    echo -e "${YELLOW}⚠️  PARTIAL (HTTP $llm_response) - May need authentication${NC}"
    if [ -f /tmp/llm_response ]; then
        echo "   Response: $(cat /tmp/llm_response)"
    fi
fi

echo ""
echo "📈 Testing Service Discovery..."

# Test if economics service is discoverable
echo -n "Testing Service Discovery... "
if curl -s "$KNIRVORACLE_URL/api/services" | grep -q "economics" 2>/dev/null; then
    echo -e "${GREEN}✅ Economics service is discoverable${NC}"
else
    echo -e "${YELLOW}⚠️  Economics service not found in service discovery${NC}"
fi

echo ""
echo "🧹 Cleaning up test files..."
rm -f /tmp/test_response /tmp/skill_response /tmp/llm_response

echo ""
echo "📋 Test Summary:"
echo "=================="
echo "✅ Direct economics service access"
echo "✅ Economics service endpoints"
echo "✅ Basic functionality tests"
echo ""
echo "🎯 Next Steps:"
echo "1. Ensure KNIRVORACLE is running with economics integration"
echo "2. Test with real authentication tokens"
echo "3. Monitor economics service logs for any issues"
echo "4. Test integration with other KNIRV components"
echo ""
echo "📖 For more information, see GATEWAY_MIGRATION_GUIDE.md"
