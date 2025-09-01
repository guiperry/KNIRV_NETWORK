#!/bin/bash
# Unified Render Start Script for KNIRV Testnet
# This script detects the service type and runs the appropriate command

set -e

echo "🚀 KNIRV Testnet Unified Start Script"
echo "===================================="
echo "Started at: $(date)"
echo "Service: ${RENDER_SERVICE_NAME:-'unknown'}"
echo "Working directory: $(pwd)"
echo ""

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

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

# Detect environment
print_info "=== ENVIRONMENT DETECTION ==="
print_info "RENDER: ${RENDER:-'not set'}"
print_info "RENDER_SERVICE_ID: ${RENDER_SERVICE_ID:-'not set'}"
print_info "RENDER_SERVICE_NAME: ${RENDER_SERVICE_NAME:-'not set'}"
print_info "PORT: ${PORT:-'not set'}"

# Check if we're in a Docker container
if [ -f /.dockerenv ]; then
    print_success "✓ Running inside Docker container"
    CONTAINER_MODE=true
else
    print_warning "Not running in Docker container"
    CONTAINER_MODE=false
fi

# Service detection and startup
print_info "=== SERVICE STARTUP ==="

case "${RENDER_SERVICE_NAME}" in
    "knirv-testnet-gateway")
        print_info "🌐 Starting KNIRV Testnet Gateway (nginx)"
        if [ "$CONTAINER_MODE" = "true" ]; then
            # Verify nginx is available
            if command -v nginx >/dev/null 2>&1; then
                print_success "✓ nginx found"
                print_info "Testing nginx configuration..."
                nginx -t
                print_success "✓ nginx configuration valid"
                print_info "Starting nginx..."
                exec nginx -g "daemon off;"
            else
                print_error "✗ nginx not found in container"
                exit 1
            fi
        else
            print_error "Gateway service should run in Docker container"
            exit 1
        fi
        ;;
        
    "knirv-oracle")
        print_info "🔮 Starting KNIRV Oracle"
        if [ "$CONTAINER_MODE" = "true" ]; then
            if [ -f "/usr/local/bin/knirv-oracle" ]; then
                print_success "✓ knirv-oracle binary found"
                print_info "Starting oracle service..."
                exec knirv-oracle --testnet --disable-p2p --port 1317
            else
                print_error "✗ knirv-oracle binary not found"
                ls -la /usr/local/bin/ || echo "Cannot list /usr/local/bin/"
                exit 1
            fi
        else
            print_error "Oracle service should run in Docker container"
            exit 1
        fi
        ;;
        
    "knirv-chain")
        print_info "⛓️ Starting KNIRV Chain"
        if [ "$CONTAINER_MODE" = "true" ]; then
            if [ -f "./knirvchain" ]; then
                print_success "✓ knirvchain binary found"
                print_info "Starting chain service..."
                exec ./knirvchain
            else
                print_error "✗ knirvchain binary not found"
                ls -la . || echo "Cannot list current directory"
                exit 1
            fi
        else
            print_error "Chain service should run in Docker container"
            exit 1
        fi
        ;;
        
    "knirv-graph")
        print_info "🕸️ Starting KNIRV Graph"
        if [ "$CONTAINER_MODE" = "true" ]; then
            if [ -f "./knirvgraph" ]; then
                print_success "✓ knirvgraph binary found"
                print_info "Starting graph service..."
                exec ./knirvgraph
            else
                print_error "✗ knirvgraph binary not found"
                ls -la . || echo "Cannot list current directory"
                exit 1
            fi
        else
            print_error "Graph service should run in Docker container"
            exit 1
        fi
        ;;
        
    "knirv-nexus")
        print_info "🏢 Starting KNIRV Nexus"
        if [ "$CONTAINER_MODE" = "true" ]; then
            if [ -f "./bin/knirv-nexus" ]; then
                print_success "✓ knirv-nexus binary found"
                print_info "Starting nexus service..."
                exec ./bin/knirv-nexus --config /opt/knirv-nexus/config/nexus.yaml
            else
                print_error "✗ knirv-nexus binary not found"
                ls -la ./bin/ || echo "Cannot list ./bin/"
                exit 1
            fi
        else
            print_error "Nexus service should run in Docker container"
            exit 1
        fi
        ;;
        
    "knirv-router")
        print_info "🌐 Starting KNIRV Router"
        if [ "$CONTAINER_MODE" = "true" ]; then
            if [ -f "./knirvrouter" ]; then
                print_success "✓ knirvrouter binary found"
                print_info "Starting router service..."
                exec ./knirvrouter
            else
                print_error "✗ knirvrouter binary not found"
                ls -la . || echo "Cannot list current directory"
                exit 1
            fi
        else
            print_error "Router service should run in Docker container"
            exit 1
        fi
        ;;
        
    *)
        print_error "Unknown service: ${RENDER_SERVICE_NAME:-'undefined'}"
        print_info "Available services:"
        print_info "  - knirv-testnet-gateway"
        print_info "  - knirv-oracle"
        print_info "  - knirv-chain"
        print_info "  - knirv-graph"
        print_info "  - knirv-nexus"
        print_info "  - knirv-router"
        print_info ""
        print_info "Environment variables:"
        env | grep -E "RENDER" | sort
        exit 1
        ;;
esac

# This should never be reached due to exec calls above
print_error "Service startup failed - exec command did not replace process"
exit 1
