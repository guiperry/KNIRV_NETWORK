#!/bin/bash

# KNIRV Network API Gateway Test Script
# This script tests the API Gateway functionality

set -e

# Configuration
GATEWAY_URL="http://localhost:8000"
TEST_RESULTS=()

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_test() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

print_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    TEST_RESULTS+=("PASS: $1")
}

print_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    TEST_RESULTS+=("FAIL: $1")
}

print_skip() {
    echo -e "${YELLOW}[SKIP]${NC} $1"
    TEST_RESULTS+=("SKIP: $1")
}

# Function to test HTTP endpoint
test_endpoint() {
    local method="$1"
    local endpoint="$2"
    local expected_status="$3"
    local description="$4"
    local data="$5"
    local headers="$6"
    
    print_test "Testing $method $endpoint - $description"
    
    local curl_cmd="curl -s -w '%{http_code}' -o /tmp/gateway_test_response"
    
    if [ -n "$headers" ]; then
        curl_cmd="$curl_cmd $headers"
    fi
    
    if [ -n "$data" ]; then
        curl_cmd="$curl_cmd -d '$data' -H 'Content-Type: application/json'"
    fi
    
    curl_cmd="$curl_cmd -X $method $GATEWAY_URL$endpoint"
    
    local status_code
    status_code=$(eval $curl_cmd)
    
    if [ "$status_code" = "$expected_status" ]; then
        print_pass "$description (Status: $status_code)"
        if [ -f /tmp/gateway_test_response ]; then
            local response=$(cat /tmp/gateway_test_response)
            if [ ${#response} -gt 0 ] && [ ${#response} -lt 200 ]; then
                echo "    Response: $response"
            fi
        fi
    else
        print_fail "$description (Expected: $expected_status, Got: $status_code)"
        if [ -f /tmp/gateway_test_response ]; then
            echo "    Response: $(cat /tmp/gateway_test_response)"
        fi
    fi
    
    rm -f /tmp/gateway_test_response
}

# Function to check if gateway is running
check_gateway_running() {
    print_test "Checking if API Gateway is running..."
    
    if curl -s "$GATEWAY_URL/gateway/health" > /dev/null 2>&1; then
        print_pass "API Gateway is running"
        return 0
    else
        print_fail "API Gateway is not running or not accessible"
        echo "Please start the gateway with: ./start-gateway.sh start"
        exit 1
    fi
}

# Function to test authentication
test_authentication() {
    print_test "Testing authentication system..."
    
    # Test login with valid credentials
    local login_response
    login_response=$(curl -s -X POST "$GATEWAY_URL/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"username": "admin", "password": "password"}')
    
    if echo "$login_response" | grep -q "token"; then
        print_pass "Login with valid credentials"
        
        # Extract token
        local token
        token=$(echo "$login_response" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
        
        if [ -n "$token" ]; then
            # Test token validation
            test_endpoint "GET" "/auth/validate" "200" "Token validation" "" "-H 'Authorization: Bearer $token'"
            
            # Test logout
            test_endpoint "POST" "/auth/logout" "200" "Logout" "" "-H 'Authorization: Bearer $token'"
        else
            print_fail "Could not extract token from login response"
        fi
    else
        print_fail "Login with valid credentials"
    fi
    
    # Test login with invalid credentials
    test_endpoint "POST" "/auth/login" "401" "Login with invalid credentials" '{"username": "admin", "password": "wrong"}'
}

# Function to test service routing
test_service_routing() {
    print_test "Testing service routing..."
    
    # Test gateway endpoints
    test_endpoint "GET" "/gateway/health" "200" "Gateway health endpoint"
    test_endpoint "GET" "/gateway/metrics" "200" "Gateway metrics endpoint"
    test_endpoint "GET" "/gateway/services" "200" "Gateway services list"
    
    # Test service proxying (these will likely fail if services aren't running, but we test the routing)
    print_test "Testing service proxy routing (services may not be running)..."
    
    # KNIRVCHAIN routes
    local status_code
    status_code=$(curl -s -w '%{http_code}' -o /dev/null "$GATEWAY_URL/knirvchain/blocks" || echo "000")
    if [ "$status_code" = "200" ]; then
        print_pass "KNIRVCHAIN routing (service is running)"
    elif [ "$status_code" = "503" ]; then
        print_skip "KNIRVCHAIN routing (service unavailable - expected if not running)"
    elif [ "$status_code" = "404" ]; then
        print_pass "KNIRVCHAIN routing (gateway routing works, endpoint not found)"
    else
        print_fail "KNIRVCHAIN routing (unexpected status: $status_code)"
    fi
    
    # KNIRVGRAPH routes
    status_code=$(curl -s -w '%{http_code}' -o /dev/null "$GATEWAY_URL/knirvgraph/height" || echo "000")
    if [ "$status_code" = "200" ]; then
        print_pass "KNIRVGRAPH routing (service is running)"
    elif [ "$status_code" = "503" ]; then
        print_skip "KNIRVGRAPH routing (service unavailable - expected if not running)"
    elif [ "$status_code" = "404" ]; then
        print_pass "KNIRVGRAPH routing (gateway routing works, endpoint not found)"
    else
        print_fail "KNIRVGRAPH routing (unexpected status: $status_code)"
    fi
}

# Function to test WebSocket
test_websocket() {
    print_test "Testing WebSocket functionality..."
    
    # Check if websocat is available for WebSocket testing
    if command -v websocat > /dev/null 2>&1; then
        print_test "Testing WebSocket connection..."
        
        # Test WebSocket ping/pong
        local ws_response
        ws_response=$(echo '{"type":"ping"}' | timeout 5 websocat "ws://localhost:8000/gateway/ws" 2>/dev/null || echo "")
        
        if echo "$ws_response" | grep -q "pong"; then
            print_pass "WebSocket ping/pong"
        else
            print_fail "WebSocket ping/pong"
        fi
    else
        print_skip "WebSocket testing (websocat not available)"
        echo "    Install websocat to test WebSocket functionality: cargo install websocat"
    fi
}

# Function to test rate limiting
test_rate_limiting() {
    print_test "Testing rate limiting..."
    
    # Make multiple rapid requests to test rate limiting
    local success_count=0
    local rate_limited_count=0
    
    for i in {1..10}; do
        local status_code
        status_code=$(curl -s -w '%{http_code}' -o /dev/null "$GATEWAY_URL/gateway/health")
        
        if [ "$status_code" = "200" ]; then
            success_count=$((success_count + 1))
        elif [ "$status_code" = "429" ]; then
            rate_limited_count=$((rate_limited_count + 1))
        fi
    done
    
    if [ $success_count -gt 0 ]; then
        print_pass "Rate limiting allows normal requests ($success_count successful)"
    else
        print_fail "Rate limiting blocks all requests"
    fi
    
    # Note: Rate limiting might not trigger with just 10 requests depending on configuration
    if [ $rate_limited_count -gt 0 ]; then
        print_pass "Rate limiting active ($rate_limited_count rate-limited)"
    else
        print_skip "Rate limiting not triggered (may need more requests or different client)"
    fi
}

# Function to print test summary
print_summary() {
    echo ""
    echo "=================================="
    echo "API Gateway Test Summary"
    echo "=================================="
    
    local pass_count=0
    local fail_count=0
    local skip_count=0
    
    for result in "${TEST_RESULTS[@]}"; do
        echo "$result"
        if [[ $result == PASS:* ]]; then
            pass_count=$((pass_count + 1))
        elif [[ $result == FAIL:* ]]; then
            fail_count=$((fail_count + 1))
        elif [[ $result == SKIP:* ]]; then
            skip_count=$((skip_count + 1))
        fi
    done
    
    echo ""
    echo "Results: $pass_count passed, $fail_count failed, $skip_count skipped"
    
    if [ $fail_count -eq 0 ]; then
        print_pass "All tests passed!"
        exit 0
    else
        print_fail "Some tests failed"
        exit 1
    fi
}

# Main test execution
main() {
    echo "KNIRV Network API Gateway Test Suite"
    echo "====================================="
    echo ""
    
    check_gateway_running
    echo ""
    
    test_authentication
    echo ""
    
    test_service_routing
    echo ""
    
    test_websocket
    echo ""
    
    test_rate_limiting
    echo ""
    
    print_summary
}

# Run tests
main "$@"
