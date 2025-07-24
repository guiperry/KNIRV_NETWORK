#!/bin/bash

# API Endpoint Testing Script for Phase 1.3 Standardization
# Tests all standardized endpoints with proper response format validation

set -e

API_BASE="http://localhost:8081/api/v1"
TEMP_DIR="/tmp/api_test_$$"
mkdir -p "$TEMP_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Function to print test results
print_result() {
    local test_name="$1"
    local status="$2"
    local details="$3"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    if [ "$status" = "PASS" ]; then
        echo -e "${GREEN}✓ PASS${NC} $test_name"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}✗ FAIL${NC} $test_name"
        echo -e "  ${YELLOW}Details:${NC} $details"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
}

# Function to validate standard response format
validate_response() {
    local response="$1"
    local test_name="$2"
    
    # Check if response is valid JSON
    if ! echo "$response" | jq . >/dev/null 2>&1; then
        print_result "$test_name - JSON Format" "FAIL" "Response is not valid JSON"
        return 1
    fi
    
    # Check for required fields
    local success=$(echo "$response" | jq -r '.success // "null"')
    local message=$(echo "$response" | jq -r '.message // "null"')
    
    if [ "$success" = "null" ]; then
        print_result "$test_name - Standard Format" "FAIL" "Missing 'success' field"
        return 1
    fi
    
    if [ "$message" = "null" ]; then
        print_result "$test_name - Standard Format" "FAIL" "Missing 'message' field"
        return 1
    fi
    
    print_result "$test_name - Standard Format" "PASS" "Response follows standard format"
    return 0
}

# Function to test an endpoint
test_endpoint() {
    local method="$1"
    local endpoint="$2"
    local test_name="$3"
    local data="$4"
    local expected_status="$5"
    
    echo -e "\n${BLUE}Testing:${NC} $test_name"
    echo -e "${BLUE}Endpoint:${NC} $method $endpoint"
    
    local curl_cmd="curl -s -w '%{http_code}' -X $method"
    
    if [ -n "$data" ]; then
        curl_cmd="$curl_cmd -H 'Content-Type: application/json' -d '$data'"
    fi
    
    local response=$(eval "$curl_cmd '$API_BASE$endpoint'")
    local http_code="${response: -3}"
    local body="${response%???}"
    
    # Check HTTP status code
    if [ "$http_code" != "$expected_status" ]; then
        print_result "$test_name - HTTP Status" "FAIL" "Expected $expected_status, got $http_code"
        echo -e "  ${YELLOW}Response:${NC} $body"
        return 1
    fi
    
    print_result "$test_name - HTTP Status" "PASS" "Status code $http_code"
    
    # Validate response format for successful responses
    if [ "$http_code" -ge 200 ] && [ "$http_code" -lt 300 ]; then
        validate_response "$body" "$test_name"
        echo -e "  ${YELLOW}Response:${NC} $(echo "$body" | jq -c .)"
    fi
    
    return 0
}

echo -e "${BLUE}=== API Endpoint Testing Suite ===${NC}"
echo -e "${BLUE}Testing Phase 1.3 Standardized API Endpoints${NC}\n"

# Start the server in background
echo -e "${YELLOW}Starting server...${NC}"
./agentic-engine-test &
SERVER_PID=$!
sleep 5

# Wait for server to be ready
echo -e "${YELLOW}Waiting for server to be ready...${NC}"
for i in {1..10}; do
    if curl -s "$API_BASE/agents" >/dev/null 2>&1; then
        echo -e "${GREEN}Server is ready!${NC}"
        break
    fi
    if [ $i -eq 10 ]; then
        echo -e "${RED}Server failed to start${NC}"
        kill $SERVER_PID 2>/dev/null || true
        exit 1
    fi
    sleep 1
done

echo -e "\n${BLUE}=== Core CRUD Operations ===${NC}"

# Test GET /agents
test_endpoint "GET" "/agents" "List All Agents" "" "200"

# Test GET /agents/{id} with existing agent
AGENT_ID=$(curl -s "$API_BASE/agents" | jq -r '.data[0].id // "test-id"')
test_endpoint "GET" "/agents/$AGENT_ID" "Get Agent by ID" "" "200"

# Test GET /agents/{id} with non-existent agent
test_endpoint "GET" "/agents/non-existent-id" "Get Non-existent Agent" "" "404"

# Test POST /agents (Create Agent)
CREATE_DATA='{"name":"Test Agent","type":"test","config":{"collection":"test","status":"idle"}}'
test_endpoint "POST" "/agents" "Create Agent" "$CREATE_DATA" "201"

echo -e "\n${BLUE}=== Advanced Operations ===${NC}"

# Test GET /agents/discover
test_endpoint "GET" "/agents/discover" "Discover Agents" "" "200"

# Test GET /agents/search
test_endpoint "GET" "/agents/search?q=Alpha" "Search Agents" "" "200"

# Test POST /agents/register
REGISTER_DATA='{"id":"test-register-id","name":"Registered Agent","type":"test","owner_id":1}'
test_endpoint "POST" "/agents/register" "Register Agent" "$REGISTER_DATA" "201"

echo -e "\n${BLUE}=== Agent Lifecycle Operations ===${NC}"

# Test POST /agents/{id}/activate
test_endpoint "POST" "/agents/$AGENT_ID/activate" "Activate Agent" "" "200"

# Test POST /agents/{id}/deactivate  
test_endpoint "POST" "/agents/$AGENT_ID/deactivate" "Deactivate Agent" "" "200"

echo -e "\n${BLUE}=== Configuration Operations ===${NC}"

# Test GET /agents/{id}/config
test_endpoint "GET" "/agents/$AGENT_ID/config" "Get Agent Config" "" "200"

# Test PUT /agents/{id}/config
CONFIG_DATA='{"collection":"updated","status":"active","capabilities":["test"]}'
test_endpoint "PUT" "/agents/$AGENT_ID/config" "Update Agent Config" "$CONFIG_DATA" "200"

echo -e "\n${BLUE}=== Filtering Operations ===${NC}"

# Test GET /agents/by-type/{type}
test_endpoint "GET" "/agents/by-type/Genesis" "Filter by Type" "" "200"

# Test GET /agents/by-status/{status}
test_endpoint "GET" "/agents/by-status/idle" "Filter by Status" "" "200"

# Test GET /agents/by-build-target/{target}
test_endpoint "GET" "/agents/by-build-target/plugin" "Filter by Build Target" "" "200"

echo -e "\n${BLUE}=== Error Handling Tests ===${NC}"

# Test invalid JSON
test_endpoint "POST" "/agents" "Invalid JSON" '{"invalid":json}' "400"

# Test missing required fields
test_endpoint "POST" "/agents" "Missing Required Fields" '{"name":"Test"}' "400"

# Cleanup
echo -e "\n${YELLOW}Cleaning up...${NC}"
kill $SERVER_PID 2>/dev/null || true
rm -rf "$TEMP_DIR"

# Print summary
echo -e "\n${BLUE}=== Test Summary ===${NC}"
echo -e "Total Tests: $TOTAL_TESTS"
echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
echo -e "${RED}Failed: $FAILED_TESTS${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "\n${GREEN}🎉 All tests passed! API standardization is working correctly.${NC}"
    exit 0
else
    echo -e "\n${RED}❌ Some tests failed. Please review the issues above.${NC}"
    exit 1
fi
