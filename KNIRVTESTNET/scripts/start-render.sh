#!/bin/bash
set -e

# KNIRV Wizened Environment - Render Deployment Script
# This script starts the wizened WASM module with instant startup

echo "🚀 KNIRV WIZENED TESTNET - RENDER DEPLOYMENT"
echo "============================================="
echo "Using WizenedEnvironmentGuide approach with pre-initialized WASM module"

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

# Check if we're running on Render
if [ "$RENDER" != "true" ] && [ -z "$RENDER_SERVICE_ID" ]; then
    print_warning "This script is designed for Render deployment"
    print_warning "For local development, use: npm run dev"
    print_warning "For full testnet, use: npm run testnet:start"
fi

# Set environment variables for wizened deployment
export KNIRV_ENV=wizened-testnet
export TESTNET_MODE=true

# Render.com sets PORT automatically - don't override it
if [ -z "$PORT" ]; then
    print_warning "PORT environment variable not set by Render"
    export PORT=10000
    print_warning "Defaulting to PORT=10000"
else
    print_success "Render provided PORT=$PORT"
fi

print_status "Starting KNIRV Wizened Testnet for Render..."
print_status "Environment: $KNIRV_ENV"
print_status "Port: $PORT"
print_status "Render Service ID: ${RENDER_SERVICE_ID:-'not set'}"
print_status "Render External URL: ${RENDER_EXTERNAL_URL:-'not set'}"

# Verify wizened artifacts
print_status "Verifying wizened artifacts..."
if [ ! -f "bin/knirv-server.wasm" ]; then
    print_error "knirv-server.wasm not found!"
    print_error "Please run 'scripts/build-local-release.sh' locally and commit the bin/ directory."
    exit 1
fi

if [ ! -f "bin/wasmtime" ]; then
    print_error "wasmtime runtime not found!"
    print_error "Please run 'scripts/build-local-release.sh' locally and commit the bin/ directory."
    exit 1
fi

print_success "All wizened artifacts verified."

# Create necessary directories
mkdir -p logs data

# The wizened approach doesn't need mock services - everything runs inside WASM
print_status "Wizened deployment detected - skipping mock service initialization"
print_status "All services will be orchestrated by the wizened WASM module"

# Start the native KNIRV orchestrator
print_status "Starting KNIRV Native Orchestrator..."
print_status "Native orchestrator manages KNIRV services on host"
print_status "VFS toolchain available separately for development tools"

# Execute the native orchestrator
print_status "Executing: ./bin/knirv-orchestrator"
exec ./bin/knirv-orchestrator
