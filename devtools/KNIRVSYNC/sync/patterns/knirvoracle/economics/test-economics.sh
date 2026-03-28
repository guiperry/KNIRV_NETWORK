#!/bin/bash

# KNIRV Economics Service Test Script
# This script tests the economics service API endpoints

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
ECONOMICS_URL="http://localhost:8090"
TEST_USER_ID="test_user_123"
TEST_SKILL_ID="test_skill_456"
TEST_LLM_ID="test_llm_789"
TEST_VALIDATOR_ID="test_validator_101"

# Function to print colored output
print_info() {
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

# Function to make API call and check response
test_api_call() {
    local method=$1
    local endpoint=$2
    local data=$3
    local description=$4
    
    print_info "Testing: $description"
    
    if [ "$method" = "GET" ]; then
        response=$(curl -s -w "\n%{http_code}" "$ECONOMICS_URL$endpoint")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
            -H "Content-Type: application/json" \
            -d "$data" \
            "$ECONOMICS_URL$endpoint")
    fi
    
    # Extract HTTP status code (last line)
    http_code=$(echo "$response" | tail -n1)
    # Extract response body (all but last line)
    body=$(echo "$response" | head -n -1)
    
    if [ "$http_code" -eq 200 ]; then
        print_success "$description - HTTP $http_code"
        echo "Response: $body" | jq . 2>/dev/null || echo "Response: $body"
        echo ""
        return 0
    else
        print_error "$description - HTTP $http_code"
        echo "Response: $body"
        echo ""
        return 1
    fi
}

# Function to wait for service to be ready
wait_for_service() {
    local max_attempts=30
    local attempt=1
    
    print_info "Waiting for economics service to be ready..."
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s -f "$ECONOMICS_URL/economics/health" > /dev/null 2>&1; then
            print_success "Economics service is ready!"
            return 0
        fi
        
        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    print_error "Economics service failed to start within $((max_attempts * 2)) seconds"
    return 1
}

# Check if jq is available for JSON formatting
if ! command -v jq &> /dev/null; then
    print_warning "jq not found. JSON responses will not be formatted."
fi

print_info "Starting KNIRV Economics Service API Tests"
print_info "Service URL: $ECONOMICS_URL"
echo ""

# Wait for service to be ready
if ! wait_for_service; then
    print_error "Cannot proceed with tests - service is not ready"
    exit 1
fi

# Test 1: Health Check
test_api_call "GET" "/economics/health" "" "Health Check"

# Test 2: Service Info
test_api_call "GET" "/economics/info" "" "Service Information"

# Test 3: Get Economic Metrics
test_api_call "GET" "/economics/metrics" "" "Get Economic Metrics"

# Test 4: Get Economic Rules
test_api_call "GET" "/economics/rules" "" "Get Economic Rules"

# Test 5: Process Skill Invocation
skill_data='{
    "user_id": "'$TEST_USER_ID'",
    "skill_id": "'$TEST_SKILL_ID'",
    "amount": "100000"
}'
test_api_call "POST" "/economics/skill/invoke" "$skill_data" "Process Skill Invocation"

# Test 6: Process LLM Registration
llm_data='{
    "user_id": "'$TEST_USER_ID'",
    "llm_id": "'$TEST_LLM_ID'",
    "registration_fee": "1000000"
}'
test_api_call "POST" "/economics/llm/register" "$llm_data" "Process LLM Registration"

# Test 7: Process Validation Reward
validation_data='{
    "validator_id": "'$TEST_VALIDATOR_ID'",
    "target_id": "'$TEST_LLM_ID'",
    "validation_result": true
}'
test_api_call "POST" "/economics/validation/reward" "$validation_data" "Process Validation Reward"

# Test 8: Calculate Network Fees
fees_data='{
    "gas_used": 21000,
    "priority": "medium"
}'
test_api_call "POST" "/economics/fees/calculate" "$fees_data" "Calculate Network Fees"

# Test 9: Get Transactions
test_api_call "GET" "/economics/transactions?limit=10" "" "Get Recent Transactions"

# Test 10: Get Burn History
test_api_call "GET" "/economics/burn/history?limit=5" "" "Get Burn History"

# Test 11: Get Total Burned
test_api_call "GET" "/economics/burn/total" "" "Get Total Burned Amount"

# Test 12: Get Service Metrics for KNIRVCHAIN
test_api_call "GET" "/economics/service/knirvchain/metrics" "" "Get KNIRVCHAIN Service Metrics"

# Test 13: Get Integration Status
test_api_call "GET" "/economics/integration/status" "" "Get Integration Status"

# Test 14: Update Economic Rules (this might fail if validation is strict)
new_rules='{
    "skill_invocation_cost": "150000",
    "llm_registration_fee": "1500000",
    "validation_reward": "75000",
    "burn_rates": {
        "skill_invocation": "150000",
        "llm_registration": "750000",
        "validation": "37500"
    },
    "minting_rules": {
        "max_supply": "1000000000000000",
        "inflation_rate": 0.05,
        "validator_rewards": "10000000",
        "developer_rewards": "5000000",
        "community_rewards": "2000000"
    },
    "staking_requirements": {
        "min_validator_stake": "100000000000",
        "min_developer_stake": "10000000000",
        "slashing_penalty": 0.05,
        "unbonding_period": "504h"
    },
    "governance_thresholds": {
        "proposal_deposit": "1000000000",
        "voting_threshold": 0.5,
        "quorum_threshold": 0.33,
        "voting_period": "168h"
    }
}'
test_api_call "PUT" "/economics/rules" "$new_rules" "Update Economic Rules"

# Test 15: Verify Updated Metrics
test_api_call "GET" "/economics/metrics" "" "Get Updated Economic Metrics"

print_info "All API tests completed!"
print_success "Economics service is functioning correctly"

# Optional: Run load test
read -p "Do you want to run a load test? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    print_info "Running load test with 10 concurrent skill invocations..."
    
    for i in {1..10}; do
        (
            skill_data='{
                "user_id": "load_test_user_'$i'",
                "skill_id": "load_test_skill_'$i'",
                "amount": "100000"
            }'
            curl -s -X POST \
                -H "Content-Type: application/json" \
                -d "$skill_data" \
                "$ECONOMICS_URL/economics/skill/invoke" > /dev/null
        ) &
    done
    
    wait
    print_success "Load test completed"
    
    # Check final metrics
    test_api_call "GET" "/economics/metrics" "" "Final Metrics After Load Test"
fi

print_success "All tests completed successfully!"
