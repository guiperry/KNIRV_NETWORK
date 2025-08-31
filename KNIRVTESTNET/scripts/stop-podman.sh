#!/bin/bash
set -e

# KNIRV Testnet - Podman Stop Script
# This script stops the KNIRV testnet Podman containers

echo "🛑 KNIRV TESTNET - STOPPING PODMAN SERVICES"
echo "==========================================="

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
    print_error "Podman is not installed."
    exit 1
fi

# Check if podman-compose is available
if ! command -v podman-compose &> /dev/null; then
    print_warning "podman-compose not found. Falling back to manual container stopping..."
    
    # Stop containers manually
    print_status "Stopping KNIRV containers manually..."
    podman stop $(podman ps -q --filter "name=knirv") 2>/dev/null || print_warning "No running KNIRV containers found"
    
    # Remove containers
    print_status "Removing KNIRV containers..."
    podman rm $(podman ps -aq --filter "name=knirv") 2>/dev/null || print_warning "No KNIRV containers to remove"
    
else
    # Use podman-compose to stop services
    print_status "Stopping KNIRV testnet services with podman-compose..."
    
    if [ -f "docker-compose.yml" ]; then
        podman-compose down
        print_success "Services stopped successfully"
    else
        print_error "docker-compose.yml not found in current directory"
        exit 1
    fi
fi

# Clean up any remaining resources
print_status "Cleaning up remaining resources..."

# Remove any dangling volumes
print_status "Removing unused volumes..."
podman volume prune -f 2>/dev/null || print_warning "No volumes to clean"

# Remove any dangling images (optional)
if [ "$1" = "--clean-images" ]; then
    print_status "Removing unused images..."
    podman image prune -f 2>/dev/null || print_warning "No images to clean"
fi

print_success "KNIRV testnet stopped successfully!"

# Show remaining containers (should be empty)
echo ""
echo "📋 Remaining KNIRV Containers:"
echo "=============================="
podman ps -a --filter "name=knirv" --format "table {{.Names}}\t{{.Status}}" || echo "No KNIRV containers found"

echo ""
print_status "All KNIRV testnet services have been stopped"
print_status "To restart: ./scripts/start-podman.sh"
print_status "To clean images: ./scripts/stop-podman.sh --clean-images"
