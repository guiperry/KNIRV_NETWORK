#!/bin/bash

# KNIRV Testnet Integration Test Script
# Tests service discovery and communication between all testnet components

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
TIMEOUT=10
VERBOSE=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -t|--timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        -h|--help)
            echo "KNIRV Testnet Integration Test"
            echo ""
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  -v, --verbose         Show detailed test information"
            echo "  -t, --timeout SEC     HTTP timeout in seconds (default: 10)"
            echo "  -h, --help           Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                   Run integration tests"
            echo "  $0 --verbose         Detailed test output"
            echo "  $0 -t 15            Use 15-second timeout"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[⚠]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1"
}

print_header() {
    echo -e "${PURPLE}$1${NC}"
}

print_verbose() {
    if [ "$VERBOSE" = true ]; then
        echo -e "${CYAN}[VERBOSE]${NC} $1"
    fi
}

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Function to increment test counters
test_passed() {
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    PASSED_TESTS=$((PASSED_TESTS + 1))
    print_success "$1"
}

test_failed() {
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    FAILED_TESTS=$((FAILED_TESTS + 1))
    print_error "$1"
}

# Function to make HTTP request and check response
test_http_endpoint() {
    local name=$1
    local url=$2
    local expected_status=${3:-200}
    local expected_content=$4
    
    print_verbose "Testing $name: $url"
    
    local response=$(curl -s -w "%{http_code}" --max-time "$TIMEOUT" "$url" 2>/dev/null)
    local http_code="${response: -3}"
    local body="${response%???}"
    
    if [ "$http_code" = "$expected_status" ]; then
        if [ -n "$expected_content" ]; then
            if echo "$body" | grep -q "$expected_content"; then
                test_passed "$name endpoint responds correctly"
                return 0
            else
                test_failed "$name endpoint missing expected content: $expected_content"
                print_verbose "Response body: $body"
                return 1
            fi
        else
            test_passed "$name endpoint responds with status $http_code"
            return 0
        fi
    else
        test_failed "$name endpoint returned status $http_code (expected $expected_status)"
        print_verbose "Response body: $body"
        return 1
    fi
}

# Function to test service discovery through gateway
test_service_discovery() {
    local service_name=$1
    
    print_verbose "Testing service discovery for $service_name through gateway"
    
    local response=$(curl -s --max-time "$TIMEOUT" "http://localhost:8888/gateway/services" 2>/dev/null)
    
    if echo "$response" | grep -q "$service_name"; then
        test_passed "Gateway discovers $service_name"
        return 0
    else
        test_failed "Gateway cannot discover $service_name"
        print_verbose "Gateway services response: $response"
        return 1
    fi
}

# Function to test authentication
test_authentication() {
    print_verbose "Testing authentication system"
    
    # Test getting testnet tokens
    local tokens_response=$(curl -s --max-time "$TIMEOUT" "http://localhost:8888/auth/testnet-tokens" 2>/dev/null)
    
    if echo "$tokens_response" | grep -q "testnet-token-123"; then
        test_passed "Testnet authentication tokens available"
        
        # Test token validation
        local validation_response=$(curl -s --max-time "$TIMEOUT" \
            -H "Authorization: Bearer testnet-token-123" \
            "http://localhost:8888/auth/validate" 2>/dev/null)
        
        if echo "$validation_response" | grep -q '"valid":true'; then
            test_passed "Token validation works correctly"
            return 0
        else
            test_failed "Token validation failed"
            print_verbose "Validation response: $validation_response"
            return 1
        fi
    else
        test_failed "Testnet authentication tokens not available"
        print_verbose "Tokens response: $tokens_response"
        return 1
    fi
}

# Function to test cross-service communication
test_cross_service_communication() {
    print_verbose "Testing cross-service communication"
    
    # Test KNIRVCHAIN mock LLM validation
    local llm_payload='{"model_id":"test-model"}'
    local llm_response=$(curl -s --max-time "$TIMEOUT" \
        -H "Content-Type: application/json" \
        -d "$llm_payload" \
        "http://localhost:8080/testnet/llm/validate" 2>/dev/null)
    
    if echo "$llm_response" | grep -q '"success":true'; then
        test_passed "KNIRVCHAIN mock LLM validation works"
    else
        test_failed "KNIRVCHAIN mock LLM validation failed"
        print_verbose "LLM response: $llm_response"
    fi
    
    # Test KNIRVCHAIN mock skill validation
    local skill_payload='{"skill_id":"test-skill","skill_code":"console.log(\"test\")"}'
    local skill_response=$(curl -s --max-time "$TIMEOUT" \
        -H "Content-Type: application/json" \
        -d "$skill_payload" \
        "http://localhost:8080/testnet/skill/validate" 2>/dev/null)
    
    if echo "$skill_response" | grep -q '"success":true'; then
        test_passed "KNIRVCHAIN mock skill validation works"
    else
        test_failed "KNIRVCHAIN mock skill validation failed"
        print_verbose "Skill response: $skill_response"
    fi
    
    # Test KNIRV-NEXUS TEE simulation
    local tee_payload='{"skill_code":"test","test_cases":[{"input":"test","expected":"test","name":"test"}]}'
    local tee_response=$(curl -s --max-time "$TIMEOUT" \
        -H "Content-Type: application/json" \
        -d "$tee_payload" \
        "http://localhost:8182/testnet/validate/skill" 2>/dev/null)
    
    if echo "$tee_response" | grep -q '"valid":true'; then
        test_passed "KNIRV-NEXUS TEE simulation works"
    else
        test_failed "KNIRV-NEXUS TEE simulation failed"
        print_verbose "TEE response: $tee_response"
    fi
}

# Function to test gateway proxy functionality
test_gateway_proxy() {
    print_verbose "Testing gateway proxy functionality"
    
    # Test proxying to KNIRV-ROOT through gateway
    local proxy_response=$(curl -s --max-time "$TIMEOUT" \
        "http://localhost:8888/knirvroot/health" 2>/dev/null)
    
    if echo "$proxy_response" | grep -q "healthy\|testnet"; then
        test_passed "Gateway proxy to KNIRV-ROOT works"
    else
        test_failed "Gateway proxy to KNIRV-ROOT failed"
        print_verbose "Proxy response: $proxy_response"
    fi
}

# Main integration test function
perform_integration_tests() {
    if [ "$1" != "--silent" ]; then
        print_header "🔗 KNIRV TESTNET INTEGRATION TESTS"
        print_header "=================================="
        echo ""
    fi
    
    # 1. Test individual service health endpoints
    print_status "Testing individual service health endpoints..."
    test_http_endpoint "KNIRV-ROOT" "http://localhost:1317/health" 200 "healthy"
    test_http_endpoint "KNIRVCHAIN" "http://localhost:8080/health" 200 "healthy"
    test_http_endpoint "KNIRVGRAPH" "http://localhost:8081/health" 200 "healthy"
    test_http_endpoint "KNIRV-NEXUS" "http://localhost:8082/health" 200 "healthy"
    test_http_endpoint "KNIRV-ROUTER" "http://localhost:5001/health" 200 "healthy"
    test_http_endpoint "KNIRV-GATEWAY" "http://localhost:8888/gateway/health" 200 "healthy"
    echo ""
    
    # 2. Test testnet-specific endpoints
    print_status "Testing testnet-specific endpoints..."
    test_http_endpoint "KNIRVCHAIN Testnet Status" "http://localhost:8080/testnet/status" 200 "testnet"
    test_http_endpoint "KNIRVGRAPH Testnet Status" "http://localhost:8081/testnet/status" 200 "testnet"
    test_http_endpoint "KNIRV-NEXUS Testnet Status" "http://localhost:8182/testnet/status" 200 "testnet"
    test_http_endpoint "Gateway Testnet Status" "http://localhost:8888/gateway/testnet/status" 200 "testnet"
    echo ""
    
    # 3. Test service discovery
    print_status "Testing service discovery..."
    test_service_discovery "knirvroot"
    test_service_discovery "knirvchain"
    test_service_discovery "knirvgraph"
    test_service_discovery "knirvnexus"
    test_service_discovery "knirvrouter"
    echo ""
    
    # 4. Test authentication
    print_status "Testing authentication system..."
    test_authentication
    echo ""
    
    # 5. Test cross-service communication
    print_status "Testing cross-service communication..."
    test_cross_service_communication
    echo ""
    
    # 6. Test gateway proxy functionality
    print_status "Testing gateway proxy functionality..."
    test_gateway_proxy
    echo ""
    
    # Summary
    if [ "$1" != "--silent" ]; then
        print_header "📊 INTEGRATION TEST SUMMARY"
        print_header "==========================="
        echo "Total tests: $TOTAL_TESTS"
        echo "Passed: $PASSED_TESTS"
        echo "Failed: $FAILED_TESTS"
        echo ""
        
        if [ $FAILED_TESTS -eq 0 ]; then
            print_success "All integration tests passed! 🎉"
            print_status "The KNIRV testnet is fully functional and ready for use."
            echo ""
            echo "🚀 Next Steps:"
            echo "  • Use the gateway at: http://localhost:8888"
            echo "  • Check service status: ./health-check.sh --watch"
            echo "  • View logs in the ./logs/ directory"
            echo "  • Stop the testnet: ./stop-testnet.sh"
        else
            print_error "Integration tests failed with $FAILED_TESTS errors."
            print_status "Please check the service logs and fix the issues."
            echo ""
            echo "🔧 Troubleshooting:"
            echo "  • Check service health: ./health-check.sh"
            echo "  • Validate configuration: ./validate-config.sh"
            echo "  • View service logs: tail -f logs/*.log"
            echo "  • Restart services: ./stop-testnet.sh && ./start-testnet.sh"
        fi
    fi
    
    # Generate JSON report when run from main test suite
    if [ "$1" = "--silent" ]; then
        echo '{
            "testSuite": "KNIRVTestnet",
            "timestamp": "'$(date -u +"%Y-%m-%dT%H:%M:%SZ")'",
            "totalTests": '$TOTAL_TESTS',
            "passedTests": '$PASSED_TESTS',
            "failedTests": '$FAILED_TESTS',
            "success": '$([ $FAILED_TESTS -eq 0 ] && echo "true" || echo "false")'
        }' > "$(dirname "$0")/test-report.json"
    fi
    
    # Return appropriate exit code
    if [ $FAILED_TESTS -eq 0 ]; then
        exit 0
    else
        exit 1
    fi
}

# Check if services are running before testing
print_status "Checking if testnet services are running..."
if ! curl -s -f --max-time 5 "http://localhost:8888/gateway/health" >/dev/null 2>&1; then
    print_error "KNIRV Gateway is not responding. Please start the testnet first:"
    print_status "./start-testnet.sh"
    exit 1
fi

# Main execution
perform_integration_tests
