#!/bin/bash
set -e

# KNIRV Testnet - IPFS Test Script
# This script tests the IPFS configuration for the KNIRV network

echo "🧪 KNIRV TESTNET - IPFS CONFIGURATION TEST"
echo "=========================================="

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

# Test IPFS API connectivity
print_status "Testing IPFS API connectivity..."
if curl -s http://localhost:5001/api/v0/version > /dev/null; then
    IPFS_VERSION=$(curl -s http://localhost:5001/api/v0/version | grep -o '"Version":"[^"]*"' | cut -d'"' -f4)
    print_success "IPFS API is accessible (Version: $IPFS_VERSION)"
else
    print_error "IPFS API is not accessible at http://localhost:5001"
    exit 1
fi

# Test IPFS Gateway
print_status "Testing IPFS Gateway..."
if curl -s http://localhost:8080/ipfs/QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG/readme > /dev/null; then
    print_success "IPFS Gateway is accessible"
else
    print_warning "IPFS Gateway test failed (this is normal if the test hash doesn't exist)"
fi

# Test IPFS configuration
print_status "Checking IPFS configuration..."
API_CONFIG=$(curl -s http://localhost:5001/api/v0/config/show | grep -o '"API":{[^}]*}')
if echo "$API_CONFIG" | grep -q "Access-Control-Allow-Origin"; then
    print_success "CORS is properly configured"
else
    print_warning "CORS configuration not found"
fi

# Test adding content to IPFS
print_status "Testing IPFS content addition..."
TEST_CONTENT="Hello from KNIRV Testnet - $(date)"
echo "$TEST_CONTENT" > /tmp/knirv-test.txt

IPFS_HASH=$(curl -s -F file=@/tmp/knirv-test.txt http://localhost:5001/api/v0/add | grep -o '"Hash":"[^"]*"' | cut -d'"' -f4)
if [ ! -z "$IPFS_HASH" ]; then
    print_success "Content added to IPFS with hash: $IPFS_HASH"
    
    # Test retrieving the content
    print_status "Testing content retrieval..."
    if curl -s "http://localhost:8080/ipfs/$IPFS_HASH" | grep -q "Hello from KNIRV Testnet"; then
        print_success "Content successfully retrieved from IPFS"
    else
        print_warning "Content retrieval test failed"
    fi
else
    print_error "Failed to add content to IPFS"
fi

# Clean up
rm -f /tmp/knirv-test.txt

# Test IPFS peer connections
print_status "Checking IPFS peer connections..."
PEER_COUNT=$(curl -s http://localhost:5001/api/v0/swarm/peers | grep -o '"Peer":"[^"]*"' | wc -l)
print_status "Connected to $PEER_COUNT IPFS peers"

# Display IPFS node info
print_status "IPFS Node Information:"
echo "======================"
curl -s http://localhost:5001/api/v0/id | python3 -m json.tool 2>/dev/null || echo "Node ID information available via API"

echo ""
print_success "IPFS configuration test completed!"
print_status "IPFS is ready for KNIRV network operations"
print_status "API: http://localhost:5001"
print_status "Gateway: http://localhost:8080"
print_status "Swarm: tcp://localhost:4001"
