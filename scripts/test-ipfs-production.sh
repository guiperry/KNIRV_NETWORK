#!/bin/bash
set -e

# KNIRV Production Network - IPFS Integration Test Script
# This script tests IPFS functionality in the production KNIRV network

echo "🧪 KNIRV PRODUCTION NETWORK - IPFS INTEGRATION TEST"
echo "=================================================="

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# Configuration
IPFS_API_URL="http://localhost:5001"
IPFS_GATEWAY_URL="http://localhost:8080"
TEST_TIMEOUT=30
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Test tracking
run_test() {
    local test_name="$1"
    local test_function="$2"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    print_status "Running test: $test_name"
    
    if $test_function; then
        PASSED_TESTS=$((PASSED_TESTS + 1))
        print_success "✅ $test_name PASSED"
    else
        FAILED_TESTS=$((FAILED_TESTS + 1))
        print_error "❌ $test_name FAILED"
    fi
    echo ""
}

# Test 1: IPFS API Connectivity
test_ipfs_connectivity() {
    local response=$(curl -s -w "%{http_code}" -o /tmp/ipfs_version.json "$IPFS_API_URL/api/v0/version" --max-time $TEST_TIMEOUT)
    local http_code="${response: -3}"
    
    if [ "$http_code" = "200" ]; then
        local version=$(cat /tmp/ipfs_version.json | grep -o '"Version":"[^"]*"' | cut -d'"' -f4)
        print_status "IPFS Version: $version"
        return 0
    else
        print_error "IPFS API not accessible (HTTP $http_code)"
        return 1
    fi
}

# Test 2: IPFS Node Configuration
test_ipfs_configuration() {
    local response=$(curl -s "$IPFS_API_URL/api/v0/id" --max-time $TEST_TIMEOUT)
    
    if echo "$response" | grep -q "knirv-production"; then
        local node_id=$(echo "$response" | grep -o '"ID":"[^"]*"' | cut -d'"' -f4)
        print_status "IPFS Node ID: ${node_id:0:20}..."
        print_status "KNIRV production configuration detected"
        return 0
    else
        print_error "IPFS not configured for KNIRV production network"
        return 1
    fi
}

# Test 3: Content Storage and Retrieval
test_content_operations() {
    local test_content="KNIRV Production Network Test - $(date +%s)"
    local temp_file="/tmp/knirv_test_content.txt"
    
    # Create test content
    echo "$test_content" > "$temp_file"
    
    # Add content to IPFS
    local add_response=$(curl -s -F "file=@$temp_file" "$IPFS_API_URL/api/v0/add" --max-time $TEST_TIMEOUT)
    local content_hash=$(echo "$add_response" | grep -o '"Hash":"[^"]*"' | cut -d'"' -f4)
    
    if [ -z "$content_hash" ]; then
        print_error "Failed to add content to IPFS"
        rm -f "$temp_file"
        return 1
    fi
    
    print_status "Content added with hash: $content_hash"
    
    # Retrieve content via API
    local retrieved_content=$(curl -s "$IPFS_API_URL/api/v0/cat?arg=$content_hash" --max-time $TEST_TIMEOUT)
    
    if [ "$retrieved_content" = "$test_content" ]; then
        print_status "Content retrieved successfully via API"
    else
        print_error "Content retrieval via API failed"
        rm -f "$temp_file"
        return 1
    fi
    
    # Test gateway access
    local gateway_content=$(curl -s "$IPFS_GATEWAY_URL/ipfs/$content_hash" --max-time $TEST_TIMEOUT)
    
    if [ "$gateway_content" = "$test_content" ]; then
        print_status "Content retrieved successfully via gateway"
    else
        print_error "Content retrieval via gateway failed"
        rm -f "$temp_file"
        return 1
    fi
    
    # Clean up
    rm -f "$temp_file"
    return 0
}

# Test 4: IPFS Pinning
test_ipfs_pinning() {
    # Use a known hash for testing (empty directory)
    local test_hash="QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn"
    
    # Pin the content
    local pin_response=$(curl -s -X POST "$IPFS_API_URL/api/v0/pin/add?arg=$test_hash" --max-time $TEST_TIMEOUT)
    
    if echo "$pin_response" | grep -q "Pins"; then
        print_status "Content pinned successfully"
    else
        print_error "Failed to pin content"
        return 1
    fi
    
    # Verify pin exists
    local pin_list=$(curl -s "$IPFS_API_URL/api/v0/pin/ls" --max-time $TEST_TIMEOUT)
    
    if echo "$pin_list" | grep -q "$test_hash"; then
        print_status "Pin verified in pin list"
        return 0
    else
        print_error "Pin not found in pin list"
        return 1
    fi
}

# Test 5: IPFS Swarm Connectivity
test_swarm_connectivity() {
    local peers_response=$(curl -s "$IPFS_API_URL/api/v0/swarm/peers" --max-time $TEST_TIMEOUT)
    
    if [ $? -eq 0 ]; then
        local peer_count=$(echo "$peers_response" | grep -o '"Peer":"[^"]*"' | wc -l)
        print_status "Swarm API accessible"
        print_status "Connected to $peer_count peers"
        return 0
    else
        print_error "Failed to access swarm API"
        return 1
    fi
}

# Test 6: IPFS Repository Status
test_repo_status() {
    local repo_response=$(curl -s "$IPFS_API_URL/api/v0/repo/stat" --max-time $TEST_TIMEOUT)
    
    if echo "$repo_response" | grep -q "RepoSize"; then
        local repo_size=$(echo "$repo_response" | grep -o '"RepoSize":[0-9]*' | cut -d':' -f2)
        local storage_max=$(echo "$repo_response" | grep -o '"StorageMax":[0-9]*' | cut -d':' -f2)
        
        print_status "Repository size: $repo_size bytes"
        print_status "Storage max: $storage_max bytes"
        
        # Check if repo is not full (less than 90% capacity)
        if [ "$repo_size" -lt $((storage_max * 90 / 100)) ]; then
            print_status "Repository has sufficient space"
            return 0
        else
            print_warning "Repository is approaching capacity"
            return 0  # Still pass the test, just warn
        fi
    else
        print_error "Failed to get repository status"
        return 1
    fi
}

# Test 7: KNIRV Network Integration
test_knirv_integration() {
    print_status "Testing KNIRV network integration..."
    
    # Test if other KNIRV services can access IPFS
    local services=("8083" "8080" "8081" "8082")  # Oracle, Chain, Graph, Nexus
    local integration_success=true
    
    for port in "${services[@]}"; do
        if curl -s "http://localhost:$port/health" --max-time 5 >/dev/null 2>&1; then
            print_status "Service on port $port is accessible"
        else
            print_warning "Service on port $port not accessible (may not be running)"
        fi
    done
    
    # Test IPFS configuration for KNIRV
    local config_response=$(curl -s "$IPFS_API_URL/api/v0/config/show" --max-time $TEST_TIMEOUT)
    
    if echo "$config_response" | grep -q "Access-Control-Allow-Origin"; then
        print_status "CORS configuration detected for web applications"
    else
        print_warning "CORS configuration not found"
        integration_success=false
    fi
    
    if echo "$config_response" | grep -q "knirv-production"; then
        print_status "KNIRV production agent version detected"
    else
        print_warning "KNIRV production agent version not detected"
        integration_success=false
    fi
    
    return $($integration_success && echo 0 || echo 1)
}

# Main test execution
main() {
    print_status "Starting IPFS integration tests for KNIRV production network..."
    print_status "IPFS API: $IPFS_API_URL"
    print_status "IPFS Gateway: $IPFS_GATEWAY_URL"
    print_status "Test timeout: ${TEST_TIMEOUT}s"
    echo ""
    
    # Wait for IPFS to be ready
    print_status "Waiting for IPFS to be ready..."
    local ready=false
    for i in {1..30}; do
        if curl -s "$IPFS_API_URL/api/v0/version" >/dev/null 2>&1; then
            ready=true
            break
        fi
        sleep 1
    done
    
    if [ "$ready" = false ]; then
        print_error "IPFS not ready after 30 seconds"
        exit 1
    fi
    
    print_success "IPFS is ready, starting tests..."
    echo ""
    
    # Run all tests
    run_test "IPFS API Connectivity" test_ipfs_connectivity
    run_test "IPFS Node Configuration" test_ipfs_configuration
    run_test "Content Storage and Retrieval" test_content_operations
    run_test "IPFS Pinning" test_ipfs_pinning
    run_test "Swarm Connectivity" test_swarm_connectivity
    run_test "Repository Status" test_repo_status
    run_test "KNIRV Network Integration" test_knirv_integration
    
    # Test summary
    echo "=========================================="
    echo "🧪 IPFS INTEGRATION TEST SUMMARY"
    echo "=========================================="
    echo "Total Tests: $TOTAL_TESTS"
    echo "Passed: $PASSED_TESTS"
    echo "Failed: $FAILED_TESTS"
    echo ""
    
    if [ $FAILED_TESTS -eq 0 ]; then
        print_success "🎉 All IPFS integration tests PASSED!"
        print_status "IPFS is properly configured for KNIRV production network"
        exit 0
    else
        print_error "❌ $FAILED_TESTS test(s) FAILED"
        print_status "Please check IPFS configuration and network connectivity"
        exit 1
    fi
}

# Run main function
main "$@"
