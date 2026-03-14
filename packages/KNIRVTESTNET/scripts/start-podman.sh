#!/bin/bash
set -e

# KNIRV Testnet - Podman Deployment Script
# This script starts the KNIRV testnet using Podman instead of Docker

echo "🚀 KNIRV TESTNET - PODMAN DEPLOYMENT"
echo "===================================="

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

# Check if Podman is installed
if ! command -v podman &> /dev/null; then
    print_error "Podman is not installed. Please install Podman first."
    echo "Installation instructions:"
    echo "  Ubuntu/Debian: sudo apt-get install podman"
    echo "  RHEL/CentOS: sudo dnf install podman"
    echo "  macOS: brew install podman"
    exit 1
fi

# Check if podman-compose is available
if ! command -v podman-compose &> /dev/null; then
    print_warning "podman-compose not found. Installing via pip..."
    pip3 install podman-compose || {
        print_error "Failed to install podman-compose. Please install manually:"
        echo "  pip3 install podman-compose"
        exit 1
    }
fi

print_success "Podman and podman-compose are available"

# Create necessary directories
print_status "Creating necessary directories..."
mkdir -p data/ipfs data/knirvoracle data/knirvchain data/knirvgraph data/knirvserver data/knirvrouter logs

# Initialize IPFS directory structure if needed
print_status "Setting up IPFS for KNIRV network..."
if [ ! -f "data/ipfs/config" ]; then
    print_status "IPFS will be initialized on first container start"
    print_status "IPFS will be configured for KNIRV testnet with proper CORS settings"
fi

# Set proper permissions for rootless Podman
print_status "Setting up permissions for rootless Podman..."
chmod -R 755 data config

# Ensure IPFS data directory has correct permissions
chmod -R 755 data/ipfs

# Start services using podman-compose
print_status "Starting KNIRV testnet services with Podman..."
print_status "This will start all services in rootless containers for better security"

# Use podman-compose to start services
podman-compose up -d

print_success "KNIRV testnet started with Podman!"
print_status "Services are running in rootless containers"

# Display running containers
echo ""
echo "📋 Running Containers:"
echo "======================"
podman ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

echo ""
echo "🔗 Service URLs:"
echo "================"
echo "IPFS API:                http://localhost:5001"
echo "IPFS Gateway:            http://localhost:8080"
echo "IPFS Swarm:              tcp://localhost:4001"
echo "KNIRV Oracle:            http://localhost:1317"
echo "KNIRV Chain:             http://localhost:8090"
echo "KNIRV Graph:             http://localhost:8082"
echo "KNIRV Nexus:             http://localhost:8084"
echo "KNIRV Router:            http://localhost:8086"
echo "KNIRV Testnet Gateway:   http://localhost:10000"
echo ""
echo "🌐 Web Applications (via Testnet Gateway):"
echo "==========================================="
echo "Main Portal:             http://localhost:10000"
echo "GraphChain Explorer:     http://localhost:10000/graphchain-explorer"
echo "Nexus Portal:            http://localhost:10000/nexus-portal"
echo "Developer Portal:  http://localhost:10000/developer-portal"

echo ""
print_status "To stop services: podman-compose down"
print_status "To view logs: podman-compose logs -f [service_name]"
print_status "To check status: podman ps"
