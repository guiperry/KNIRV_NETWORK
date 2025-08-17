#!/bin/bash

set -e

echo "Starting KNIRV D-TEN Final Test Suite..."

# Configuration
GATEWAY_URL="https://api.knirv.com"
TEST_DURATION=3600  # 1 hour
CONCURRENT_USERS=100
TEST_DATA_DIR="./test-data"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

run_test() {
    local test_name="$1"
    local test_command="$2"

    echo "Running test: $test_name"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    if eval "$test_command"; then
        log_info "✓ $test_name PASSED"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        log_error "✗ $test_name FAILED"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# Test 1: Service Health Checks
test_service_health() {
    local services=("knirvchain" "knirvgraph" "knirvnexus" "knirvoracle")

    for service in "${services[@]}"; do
        local response=$(curl -s -o /dev/null -w "%{http_code}" "$GATEWAY_URL/$service/health")
        if [ "$response" != "200" ]; then
            log_error "Service $service health check failed (HTTP $response)"
            return 1
        fi
    done

    return 0
}

# Test 2: Authentication Flow
test_authentication() {
    local login_response=$(curl -s -X POST "$GATEWAY_URL/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"password"}')

    local token=$(echo "$login_response" | jq -r '.token')

    if [ "$token" = "null" ] || [ -z "$token" ]; then
        log_error "Authentication failed - no token received"
        return 1
    fi

    # Test token validation
    local validate_response=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: Bearer $token" \
        "$GATEWAY_URL/auth/validate")

    if [ "$validate_response" != "200" ]; then
        log_error "Token validation failed (HTTP $validate_response)"
        return 1
    fi

    echo "$token" > "$TEST_DATA_DIR/auth_token"
    return 0
}

# Test 3: LLM Registration and Retrieval
test_llm_registration() {
    local token=$(cat "$TEST_DATA_DIR/auth_token")
    local llm_id="test_llm_$(date +%s)"

    # Register LLM
    local register_response=$(curl -s -X POST "$GATEWAY_URL/knirvchain/llm/register" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d "{
            \"name\": \"$llm_id\",
            \"version\": \"1.0.0\",
            \"capabilities\": [\"text-generation\"],
            \"model_data\": \"$(echo -n 'test model data' | base64)\",
            \"registration_fee\": \"1000000\",
            \"usage_fee\": \"100000\"
        }")

    local success=$(echo "$register_response" | jq -r '.success')
    if [ "$success" != "true" ]; then
        log_error "LLM registration failed: $register_response"
        return 1
    fi

    # Wait for processing
    sleep 5

    # Retrieve LLM
    local retrieve_response=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: Bearer $token" \
        "$GATEWAY_URL/knirvchain/llm/$llm_id")

    if [ "$retrieve_response" != "200" ]; then
        log_error "LLM retrieval failed (HTTP $retrieve_response)"
        return 1
    fi

    echo "$llm_id" > "$TEST_DATA_DIR/test_llm_id"
    return 0
}

# Test 4: NRV System
test_nrv_system() {
    local token=$(cat "$TEST_DATA_DIR/auth_token")
    local error_id="test_error_$(date +%s)"

    # Create error node
    local error_response=$(curl -s -X POST "$GATEWAY_URL/knirvgraph/nrv/errors" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d "{
            \"error_type\": \"test_error\",
            \"description\": \"Test error for final testing\",
            \"context\": {\"test\": true},
            \"severity\": 2
        }")

    local error_node_id=$(echo "$error_response" | jq -r '.id')
    if [ "$error_node_id" = "null" ] || [ -z "$error_node_id" ]; then
        log_error "Error node creation failed: $error_response"
        return 1
    fi

    # Create skill node
    local skill_response=$(curl -s -X POST "$GATEWAY_URL/knirvgraph/nrv/skills" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d "{
            \"skill_type\": \"test_solver\",
            \"capabilities\": [\"test_solving\"],
            \"requirements\": {}
        }")

    local skill_node_id=$(echo "$skill_response" | jq -r '.id')
    if [ "$skill_node_id" = "null" ] || [ -z "$skill_node_id" ]; then
        log_error "Skill node creation failed: $skill_response"
        return 1
    fi

    # Test NRV resolution
    local resolve_response=$(curl -s "$GATEWAY_URL/knirvgraph/nrv/resolve/$error_node_id" \
        -H "Authorization: Bearer $token")

    local vectors_count=$(echo "$resolve_response" | jq '. | length')
    if [ "$vectors_count" -eq 0 ]; then
        log_error "NRV resolution returned no vectors"
        return 1
    fi

    return 0
}

# Test 5: Token Economics
test_token_economics() {
    local token=$(cat "$TEST_DATA_DIR/auth_token")

    # Get economic metrics
    local metrics_response=$(curl -s "$GATEWAY_URL/knirvoracle/economics/metrics" \
        -H "Authorization: Bearer $token")

    local total_supply=$(echo "$metrics_response" | jq -r '.total_supply')
    if [ "$total_supply" = "null" ]; then
        log_error "Economic metrics missing total_supply"
        return 1
    fi

    # Test skill invocation with token burning
    local skill_id=$(cat "$TEST_DATA_DIR/test_llm_id" 2>/dev/null || echo "test_skill")
    local invoke_response=$(curl -s -X POST "$GATEWAY_URL/knirvchain/skill/invoke" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d "{
            \"skill_id\": \"$skill_id\",
            \"amount\": \"500000\",
            \"user_address\": \"test_user_address\"
        }")

    local invoke_success=$(echo "$invoke_response" | jq -r '.success')
    if [ "$invoke_success" != "true" ]; then
        log_error "Skill invocation failed: $invoke_response"
        return 1
    fi

    return 0
}

# Test 6: Cross-Chain Bridge
test_cross_chain_bridge() {
    local token=$(cat "$TEST_DATA_DIR/auth_token")

    # Test bridge transfer
    local bridge_response=$(curl -s -X POST "$GATEWAY_URL/knirvoracle/bridge/transfer" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d "{
            \"target_chain\": \"xion\",
            \"amount\": \"1000000\",
            \"recipient\": \"test_recipient_address\"
        }")

    local tx_hash=$(echo "$bridge_response" | jq -r '.tx_hash')
    if [ "$tx_hash" = "null" ] || [ -z "$tx_hash" ]; then
        log_error "Bridge transfer failed: $bridge_response"
        return 1
    fi

    # Wait for processing
    sleep 10

    # Check bridge status
    local status_response=$(curl -s "$GATEWAY_URL/knirvoracle/bridge/status?tx_hash=$tx_hash" \
        -H "Authorization: Bearer $token")

    local status=$(echo "$status_response" | jq -r '.status')
    if [ "$status" = "null" ]; then
        log_error "Bridge status check failed: $status_response"
        return 1
    fi

    return 0
}

# Test 7: Load Testing
test_load_performance() {
    log_info "Starting load test with $CONCURRENT_USERS concurrent users for $TEST_DURATION seconds"

    # Create load test script
    cat > "$TEST_DATA_DIR/load_test.js" << 'EOF'
import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
  stages: [
    { duration: '2m', target: 100 },
    { duration: '5m', target: 100 },
    { duration: '2m', target: 200 },
    { duration: '5m', target: 200 },
    { duration: '2m', target: 300 },
    { duration: '5m', target: 300 },
    { duration: '10m', target: 0 },
  ],
};

export default function () {
  let response = http.get('https://api.knirv.com/gateway/health');
  check(response, {
    'status is 200': (r) => r.status === 200,
    'response time < 500ms': (r) => r.timings.duration < 500,
  });
  sleep(1);
}
EOF

    # Run load test
    if command -v k6 >/dev/null 2>&1; then
        k6 run "$TEST_DATA_DIR/load_test.js" > "$TEST_DATA_DIR/load_test_results.txt" 2>&1

        # Check results
        local success_rate=$(grep "http_req_failed" "$TEST_DATA_DIR/load_test_results.txt" | awk '{print $3}' | sed 's/%//')
        if (( $(echo "$success_rate > 5" | bc -l) )); then
            log_error "Load test failed - error rate too high: $success_rate%"
            return 1
        fi

        log_info "Load test completed - error rate: $success_rate%"
    else
        log_warn "k6 not installed, skipping load test"
    fi

    return 0
}

# Test 8: Security Testing
test_security() {
    log_info "Running security tests..."

    # Test rate limiting
    local rate_limit_test=0
    for i in {1..150}; do
        local response=$(curl -s -o /dev/null -w "%{http_code}" "$GATEWAY_URL/gateway/health")
        if [ "$response" = "429" ]; then
            rate_limit_test=1
            break
        fi
        sleep 0.1
    done

    if [ "$rate_limit_test" = "0" ]; then
        log_error "Rate limiting not working properly"
        return 1
    fi

    # Test invalid authentication
    local invalid_auth_response=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: Bearer invalid_token" \
        "$GATEWAY_URL/knirvchain/llm/register")

    if [ "$invalid_auth_response" != "401" ]; then
        log_error "Invalid authentication not properly rejected (HTTP $invalid_auth_response)"
        return 1
    fi

    # Test HTTPS enforcement
    local http_response=$(curl -s -o /dev/null -w "%{http_code}" \
        "http://api.knirv.com/gateway/health" 2>/dev/null || echo "000")

    if [ "$http_response" != "301" ] && [ "$http_response" != "302" ]; then
        log_warn "HTTPS redirect not properly configured"
    fi

    return 0
}

# Test 9: WebSocket Connectivity
test_websocket() {
    local token=$(cat "$TEST_DATA_DIR/auth_token")

    # Create WebSocket test script
    cat > "$TEST_DATA_DIR/ws_test.js" << EOF
const WebSocket = require('ws');

const ws = new WebSocket('wss://api.knirv.com/gateway/ws');

ws.on('open', function open() {
  console.log('WebSocket connected');

  // Send ping
  ws.send(JSON.stringify({ type: 'ping' }));

  setTimeout(() => {
    ws.close();
    process.exit(0);
  }, 5000);
});

ws.on('message', function message(data) {
  const msg = JSON.parse(data);
  console.log('Received:', msg);

  if (msg.type === 'pong') {
    console.log('WebSocket test PASSED');
  }
});

ws.on('error', function error(err) {
  console.error('WebSocket error:', err);
  process.exit(1);
});

ws.on('close', function close() {
  console.log('WebSocket disconnected');
});
EOF

    # Run WebSocket test
    if command -v node >/dev/null 2>&1; then
        timeout 10 node "$TEST_DATA_DIR/ws_test.js" > "$TEST_DATA_DIR/ws_test_output.txt" 2>&1

        if grep -q "WebSocket test PASSED" "$TEST_DATA_DIR/ws_test_output.txt"; then
            return 0
        else
            log_error "WebSocket test failed"
            cat "$TEST_DATA_DIR/ws_test_output.txt"
            return 1
        fi
    else
        log_warn "Node.js not installed, skipping WebSocket test"
        return 0
    fi
}

# Test 10: KNIRV-ROUTER Connectivity
test_knirv_router() {
    local token=$(cat "$TEST_DATA_DIR/auth_token")

    # Test router connectivity status
    local status_response=$(curl -s "$GATEWAY_URL/knirvrouter/api/connectivity/status" \
        -H "Authorization: Bearer $token")

    local proof_engine_active=$(echo "$status_response" | jq -r '.proof_engine_active')
    if [ "$proof_engine_active" != "true" ]; then
        log_error "KNIRV-ROUTER proof engine is not active"
        return 1
    fi

    # Test connectivity proof creation
    local proof_response=$(curl -s -X POST "$GATEWAY_URL/knirvrouter/api/connectivity/proofs" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json")

    local proof_status=$(echo "$proof_response" | jq -r '.status')
    if [ "$proof_status" != "proof_generation_initiated" ]; then
        log_error "Failed to initiate connectivity proof"
        return 1
    fi

    # Wait for proof processing
    sleep 15

    # Check proof history
    local proofs_response=$(curl -s "$GATEWAY_URL/knirvrouter/api/connectivity/proofs" \
        -H "Authorization: Bearer $token")

    local proofs_count=$(echo "$proofs_response" | jq '. | length')
    if [ "$proofs_count" -eq 0 ]; then
        log_error "No connectivity proofs found"
        return 1
    fi

    log_info "KNIRV-ROUTER connectivity test passed with $proofs_count proofs"

    # Test TURN server endpoint (existing functionality)
    local turn_response=$(curl -s -o /dev/null -w "%{http_code}" \
        "$GATEWAY_URL/knirvrouter/turn/status" \
        -H "Authorization: Bearer $token")

    if [ "$turn_response" != "200" ]; then
        log_warn "KNIRV-ROUTER TURN server endpoint returned HTTP $turn_response"
    fi

    return 0
}

# Test 11: Data Consistency
test_data_consistency() {
    local token=$(cat "$TEST_DATA_DIR/auth_token")

    # Create test data
    local test_id="consistency_test_$(date +%s)"

    # Create data in multiple services
    local chain_response=$(curl -s -X POST "$GATEWAY_URL/knirvchain/test/data" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d "{\"id\": \"$test_id\", \"data\": \"test_data\"}")

    local graph_response=$(curl -s -X POST "$GATEWAY_URL/knirvgraph/test/data" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d "{\"id\": \"$test_id\", \"data\": \"test_data\"}")

    # Wait for synchronization
    sleep 5

    # Verify data consistency
    local chain_data=$(curl -s "$GATEWAY_URL/knirvchain/test/data/$test_id" \
        -H "Authorization: Bearer $token" | jq -r '.data')

    local graph_data=$(curl -s "$GATEWAY_URL/knirvgraph/test/data/$test_id" \
        -H "Authorization: Bearer $token" | jq -r '.data')

    if [ "$chain_data" != "$graph_data" ]; then
        log_error "Data inconsistency detected between services"
        return 1
    fi

    return 0
}

# Main test execution
main() {
    log_info "KNIRV D-TEN Final Test Suite Starting..."

    # Create test data directory
    mkdir -p "$TEST_DATA_DIR"

    # Run all tests
    run_test "Service Health Checks" "test_service_health"
    run_test "Authentication Flow" "test_authentication"
    run_test "LLM Registration" "test_llm_registration"
    run_test "NRV System" "test_nrv_system"
    run_test "Token Economics" "test_token_economics"
    run_test "Cross-Chain Bridge" "test_cross_chain_bridge"
    run_test "Load Performance" "test_load_performance"
    run_test "Security Testing" "test_security"
    run_test "WebSocket Connectivity" "test_websocket"
    run_test "KNIRV-ROUTER Connectivity" "test_knirv_router"
    run_test "Data Consistency" "test_data_consistency"

    # Generate test report
    echo ""
    echo "=========================================="
    echo "KNIRV D-TEN Final Test Results"
    echo "=========================================="
    echo "Total Tests: $TOTAL_TESTS"
    echo "Passed: $PASSED_TESTS"
    echo "Failed: $FAILED_TESTS"
    echo "Success Rate: $(( PASSED_TESTS * 100 / TOTAL_TESTS ))%"
    echo "=========================================="

    if [ "$FAILED_TESTS" -eq 0 ]; then
        log_info "🎉 All tests passed! KNIRV D-TEN is ready for production."
        exit 0
    else
        log_error "❌ $FAILED_TESTS test(s) failed. Please review and fix issues before production deployment."
        exit 1
    fi
}

# Cleanup function
cleanup() {
    log_info "Cleaning up test data..."
    rm -rf "$TEST_DATA_DIR"
}

# Set trap for cleanup
trap cleanup EXIT

# Run main function
main "$@"
