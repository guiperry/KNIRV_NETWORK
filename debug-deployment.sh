#!/bin/bash

# Debug deployment script to identify where the failure occurs

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TESTNET_IP="18.189.74.120"
SSH_KEY="~/.ssh/AEGONG.pem"

print_step() {
    echo -e "${BLUE}[DEBUG STEP]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Test 1: Basic SSH connection
print_step "Testing basic SSH connection..."
if ssh -i "${SSH_KEY/#\~/$HOME}" -o ConnectTimeout=10 -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" "echo 'SSH test successful'" > /dev/null 2>&1; then
    print_success "SSH connection works"
else
    print_error "SSH connection failed"
    exit 1
fi

# Test 2: Check if directory exists
print_step "Checking if /opt/knirv-testnet exists..."
if ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" "test -d /opt/knirv-testnet"; then
    print_success "/opt/knirv-testnet exists"
else
    print_error "/opt/knirv-testnet does not exist"
    print_step "Creating directory..."
    ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" \
        "sudo mkdir -p /opt/knirv-testnet && sudo chown ubuntu:ubuntu /opt/knirv-testnet"
    print_success "Directory created"
fi

# Test 3: Check container runtime availability
print_step "Checking container runtime availability..."
ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" << 'EOF'
echo "=== Container Runtime Check ==="
if command -v docker &> /dev/null; then
    echo "Docker is available"
    if docker info &> /dev/null; then
        echo "Docker daemon is running"
    else
        echo "Docker daemon is not running"
    fi
else
    echo "Docker is not available"
fi

if command -v podman &> /dev/null; then
    echo "Podman is available"
else
    echo "Podman is not available"
fi
echo "=== End Container Runtime Check ==="
EOF

# Test 4: Check if we can create a simple test file
print_step "Testing file creation in /opt/knirv-testnet..."
ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" \
    "echo 'test' > /opt/knirv-testnet/test.txt && cat /opt/knirv-testnet/test.txt && rm /opt/knirv-testnet/test.txt"
print_success "File creation test passed"

# Test 5: Test Docker permissions setup
print_step "Testing Docker permissions setup..."
ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" << 'EOF'
echo "=== Docker Permissions Setup ==="
# Add ubuntu user to docker group if not already added
if ! groups ubuntu | grep -q docker; then
    sudo usermod -aG docker ubuntu
    echo "Added ubuntu user to docker group"
else
    echo "Ubuntu user already in docker group"
fi

# Ensure Docker daemon is running
sudo systemctl enable docker
sudo systemctl start docker

echo "Docker status:"
sudo systemctl status docker --no-pager -l

echo "=== End Docker Permissions Setup ==="
EOF

print_success "All debug tests completed successfully"
