#!/bin/bash
set -e

echo "🧪 Testing Netlify Functions Integration"
echo "======================================="

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
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

# Detect environment
detect_environment() {
    if [ "$RENDER" = "true" ] || [ -n "$RENDER_SERVICE_ID" ]; then
        echo "staging"
    elif [ "$NETLIFY" = "true" ] || [ -n "$NETLIFY_DEV" ]; then
        echo "production"
    else
        echo "local"
    fi
}

ENVIRONMENT=$(detect_environment)
print_status "Detected environment: $ENVIRONMENT"

# Set base URL based on environment
case $ENVIRONMENT in
    "local")
        BASE_URL="http://localhost:10000"
        GATEWAY_URL="http://localhost:8888"
        ;;
    "staging")
        BASE_URL="http://localhost:10000"
        GATEWAY_URL="http://localhost:8888"
        ;;
    "production")
        BASE_URL="https://testnet.knirv.com"
        GATEWAY_URL="https://testnet.knirv.com"
        ;;
esac

print_status "Testing against: $BASE_URL"
print_status "Gateway URL: $GATEWAY_URL"

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0
TOTAL_TESTS=0

# Function to run a test
run_test() {
    local test_name="$1"
    local url="$2"
    local expected_status="$3"
    local timeout="${4:-10}"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    print_status "Testing: $test_name"
    
    # Make the request with timeout
    response=$(curl -s -w "%{http_code}" -m $timeout "$url" 2>/dev/null || echo "000")
    status_code="${response: -3}"
    
    if [ "$status_code" = "$expected_status" ]; then
        print_success "✓ $test_name - Status: $status_code"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        print_error "✗ $test_name - Expected: $expected_status, Got: $status_code"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Function to test JSON response
test_json_response() {
    local test_name="$1"
    local url="$2"
    local timeout="${3:-10}"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    print_status "Testing JSON: $test_name"
    
    response=$(curl -s -m $timeout "$url" 2>/dev/null || echo "{}")
    
    # Check if response is valid JSON
    if echo "$response" | jq . >/dev/null 2>&1; then
        print_success "✓ $test_name - Valid JSON response"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        print_error "✗ $test_name - Invalid JSON response"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

echo ""
print_status "Running Netlify Functions Tests..."
echo ""

# Test 1: Basic connectivity
run_test "Basic Connectivity" "$BASE_URL" "200"

# Test 2: Health monitor endpoint
run_test "Health Monitor Page" "$BASE_URL/health-monitor" "200"

# Test 3: Health monitor API
test_json_response "Health Monitor API" "$BASE_URL/api/health-monitor/status"

# Test 4: Configuration endpoint
test_json_response "Configuration API" "$BASE_URL/config"

# Test 5: Gateway health (if available)
if [ "$ENVIRONMENT" != "local" ] || pgrep -f "netlify dev" > /dev/null; then
    run_test "Gateway Health" "$GATEWAY_URL/gateway/health" "200"
    test_json_response "Gateway Status" "$GATEWAY_URL/gateway/status"
else
    print_warning "Gateway not running, skipping gateway tests"
fi

# Test 6: Netlify Functions (if in staging/production)
if [ "$ENVIRONMENT" = "staging" ] || [ "$ENVIRONMENT" = "production" ]; then
    print_status "Testing Netlify Functions..."
    
    # Test health monitor function
    test_json_response "Netlify Health Function" "$BASE_URL/.netlify/functions/health-monitor"
    
    # Test gateway SSE function
    run_test "Gateway SSE Function" "$BASE_URL/.netlify/functions/gateway-sse" "200"
    
else
    print_warning "Local environment detected, skipping Netlify function tests"
fi

# Test 7: NEXUS Frontend
if [ "$ENVIRONMENT" = "local" ]; then
    # In local environment, expect redirect to testnet subdomain
    run_test "NEXUS Frontend (redirect)" "$BASE_URL/nexus-portal" "301"
else
    # In staging/production, expect direct access
    run_test "NEXUS Frontend" "$BASE_URL/nexus-portal" "200"
fi

# Test 8: Agent Developer Portal
run_test "Agent Developer Portal" "$BASE_URL/agent-developer-portal/" "200"

# Test 9: GraphChain Explorer
run_test "GraphChain Explorer" "$BASE_URL/graphchain-explorer/" "200"

echo ""
print_status "Test Results Summary"
echo "===================="
print_status "Total Tests: $TOTAL_TESTS"
print_success "Passed: $TESTS_PASSED"
print_error "Failed: $TESTS_FAILED"

if [ $TESTS_FAILED -eq 0 ]; then
    echo ""
    print_success "🎉 All tests passed! Netlify functions integration is working correctly."
    exit 0
else
    echo ""
    print_error "❌ Some tests failed. Please check the configuration and try again."
    exit 1
fi
