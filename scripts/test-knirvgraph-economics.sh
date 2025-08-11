#!/bin/bash

# KNIRVGRAPH Economics Integration Test Script
# This script tests the economics integration between KNIRVGRAPH and KNIRVROOT
# Run from project root: ./scripts/test-knirvgraph-economics.sh

set -e

echo "🧪 Testing KNIRVGRAPH Economics Integration..."

# Configuration
KNIRVGRAPH_URL=${KNIRVGRAPH_URL:-http://localhost:8081}
KNIRVROOT_URL=${KNIRVROOT_URL:-http://localhost:1317}
TEST_USER_ID="test_user_$(date +%s)"
TEST_SKILL_ID=""
TEST_ERROR_ID=""
TEST_NRV_ID=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Test function wrapper
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    log_info "Testing: $test_name"
    
    if eval "$test_command"; then
        log_success "$test_name passed"
        return 0
    else
        log_error "$test_name failed"
        return 1
    fi
}

# Test KNIRVGRAPH connectivity
test_knirvgraph_health() {
    curl -s --max-time 10 "$KNIRVGRAPH_URL/health" > /dev/null
}

# Test KNIRVROOT connectivity
test_knirvroot_health() {
    curl -s --max-time 10 "$KNIRVROOT_URL/ping" > /dev/null
}

# Test economics metrics endpoint
test_economics_metrics() {
    local response=$(curl -s --max-time 10 "$KNIRVGRAPH_URL/economics/metrics")
    echo "$response" | grep -q "total_nrvs_created"
}

# Create an error node for testing
create_test_error_node() {
    local response=$(curl -s -X POST "$KNIRVGRAPH_URL/nrv/errors" \
        -H "Content-Type: application/json" \
        -d '{
            "error_type": "test_integration_error",
            "description": "Test error for economics integration",
            "context": {
                "test": true,
                "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
            },
            "severity": 2
        }')
    
    TEST_ERROR_ID=$(echo "$response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    [ ! -z "$TEST_ERROR_ID" ]
}

# Create a skill node for testing
create_test_skill_node() {
    local response=$(curl -s -X POST "$KNIRVGRAPH_URL/nrv/skills" \
        -H "Content-Type: application/json" \
        -d '{
            "skill_type": "test_integration_skill",
            "capabilities": ["solve_test_error", "integration_testing"],
            "requirements": {
                "test": true,
                "min_confidence": 0.8
            }
        }')
    
    TEST_SKILL_ID=$(echo "$response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    [ ! -z "$TEST_SKILL_ID" ]
}

# Create an NRV vector for testing
create_test_nrv_vector() {
    local response=$(curl -s -X POST "$KNIRVGRAPH_URL/nrv/vectors" \
        -H "Content-Type: application/json" \
        -d '{
            "target_hash": "test_hash_'$(date +%s)'",
            "coordinates": [0.1, 0.2, 0.3, 0.4, 0.5],
            "metadata": {
                "test": true,
                "error_id": "'$TEST_ERROR_ID'",
                "skill_id": "'$TEST_SKILL_ID'"
            }
        }')
    
    TEST_NRV_ID=$(echo "$response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    [ ! -z "$TEST_NRV_ID" ]
}

# Test skill confirmation for KNIRVCHAIN commitment
test_skill_confirmation() {
    local response=$(curl -s -X POST "$KNIRVGRAPH_URL/economics/skill/confirm" \
        -H "Content-Type: application/json" \
        -d '{
            "skill_id": "'$TEST_SKILL_ID'",
            "nrv_id": "'$TEST_NRV_ID'",
            "creator_id": "'$TEST_USER_ID'"
        }')

    echo "$response" | grep -q '"success":true'
}

# Test reward distribution
test_reward_distribution() {
    local response=$(curl -s -X POST "$KNIRVGRAPH_URL/economics/rewards/distribute" \
        -H "Content-Type: application/json" \
        -d '{
            "recipient_id": "'$TEST_USER_ID'",
            "amount": "500000",
            "reason": "test_reward_for_integration"
        }')
    
    echo "$response" | grep -q '"success":true'
}

# Test solution proof submission
test_solution_proof() {
    local response=$(curl -s -X POST "$KNIRVGRAPH_URL/economics/proof/solution" \
        -H "Content-Type: application/json" \
        -d '{
            "error_node_id": "'$TEST_ERROR_ID'",
            "skill_node_id": "'$TEST_SKILL_ID'",
            "solver_id": "'$TEST_USER_ID'",
            "efficiency_score": 0.95,
            "quality_score": 0.88
        }')
    
    echo "$response" | grep -q '"success":true'
}

# Test metrics after operations
test_updated_metrics() {
    local response=$(curl -s "$KNIRVGRAPH_URL/economics/metrics")
    
    # Check if metrics have been updated
    local skills_invoked=$(echo "$response" | grep -o '"total_skills_invoked":[0-9]*' | cut -d':' -f2)
    local nrn_rewards=$(echo "$response" | grep -o '"total_nrn_rewards":"[^"]*"' | cut -d'"' -f4)
    
    [ "$skills_invoked" -gt 0 ] && [ ! -z "$nrn_rewards" ]
}

# Main test execution
main() {
    echo "🚀 Starting KNIRVGRAPH Economics Integration Tests"
    echo "📊 Test Configuration:"
    echo "  - KNIRVGRAPH URL: $KNIRVGRAPH_URL"
    echo "  - KNIRVROOT URL: $KNIRVROOT_URL"
    echo "  - Test User ID: $TEST_USER_ID"
    echo ""
    
    local failed_tests=0
    local total_tests=0
    
    # Basic connectivity tests
    ((total_tests++))
    if ! run_test "KNIRVGRAPH Health Check" "test_knirvgraph_health"; then
        ((failed_tests++))
        log_error "KNIRVGRAPH is not accessible. Please start KNIRVGRAPH first."
        exit 1
    fi
    
    ((total_tests++))
    if ! run_test "KNIRVROOT Connectivity" "test_knirvroot_health"; then
        ((failed_tests++))
        log_warning "KNIRVROOT is not accessible. Economics integration will be limited."
    fi
    
    # Economics endpoint tests
    ((total_tests++))
    if ! run_test "Economics Metrics Endpoint" "test_economics_metrics"; then
        ((failed_tests++))
    fi
    
    # Create test data
    log_info "Creating test data..."
    
    ((total_tests++))
    if ! run_test "Create Test Error Node" "create_test_error_node"; then
        ((failed_tests++))
        log_error "Failed to create test error node. Cannot continue with economics tests."
        exit 1
    fi
    log_info "Created error node: $TEST_ERROR_ID"
    
    ((total_tests++))
    if ! run_test "Create Test Skill Node" "create_test_skill_node"; then
        ((failed_tests++))
        log_error "Failed to create test skill node. Cannot continue with economics tests."
        exit 1
    fi
    log_info "Created skill node: $TEST_SKILL_ID"
    
    ((total_tests++))
    if ! run_test "Create Test NRV Vector" "create_test_nrv_vector"; then
        ((failed_tests++))
        log_error "Failed to create test NRV vector. Cannot continue with economics tests."
        exit 1
    fi
    log_info "Created NRV vector: $TEST_NRV_ID"
    
    # Economics operation tests
    ((total_tests++))
    if ! run_test "Skill Confirmation (KNIRVCHAIN Commitment)" "test_skill_confirmation"; then
        ((failed_tests++))
    fi
    
    ((total_tests++))
    if ! run_test "Reward Distribution" "test_reward_distribution"; then
        ((failed_tests++))
    fi
    
    ((total_tests++))
    if ! run_test "Solution Proof Submission" "test_solution_proof"; then
        ((failed_tests++))
    fi
    
    # Verify metrics updated
    ((total_tests++))
    if ! run_test "Updated Economics Metrics" "test_updated_metrics"; then
        ((failed_tests++))
    fi
    
    # Test summary
    echo ""
    echo "📊 Test Summary:"
    echo "  - Total Tests: $total_tests"
    echo "  - Passed: $((total_tests - failed_tests))"
    echo "  - Failed: $failed_tests"
    
    if [ $failed_tests -eq 0 ]; then
        log_success "All tests passed! KNIRVGRAPH economics integration is working correctly."
        echo ""
        echo "🎉 Economics Integration Test Results:"
        echo "  ✅ NRV System operational"
        echo "  ✅ Economics metrics tracking"
        echo "  ✅ Skill confirmation for KNIRVCHAIN commitment"
        echo "  ✅ Reward distribution system"
        echo "  ✅ Proof-of-Solution mechanism"
        echo ""
        echo "📈 View current metrics:"
        echo "  curl $KNIRVGRAPH_URL/economics/metrics | jq"
        return 0
    else
        log_error "Some tests failed. Please check the KNIRVGRAPH logs and ensure all services are running."
        return 1
    fi
}

# Run main function
main "$@"
