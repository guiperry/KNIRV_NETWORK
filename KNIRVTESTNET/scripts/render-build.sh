#!/bin/bash
set -e

echo "🚀 KNIRV Testnet Pre-built Artifact Deployment"
echo "============================================="

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_status "Build step on Render is now a preparation step."
print_status "All binaries are pre-built and checked into the repository."

# --- Phase 1: Verify Artifacts ---
print_status "Phase 1: Verifying pre-built artifacts..."

if [ ! -f "bin/knirv-orchestrator" ]; then
    echo -e "${RED}[ERROR]${NC} Core artifact 'bin/knirv-orchestrator' not found!"
    echo "Please run 'scripts/build-local-release.sh' locally and commit the 'bin' directory."
    exit 1
fi
print_success "All required artifacts are present."

# --- Phase 2: Prepare Runtime Environment ---
print_status "Phase 2: Preparing runtime environment..."

print_status "Making all binaries executable..."
chmod +x ./bin/*
print_success "Binaries are now executable."
 
WASMTIME_VERSION="v22.0.0"
# WASM runtime not needed for native orchestrator deployment
print_status "Skipping WASM runtime download - using native orchestrator only"

print_success "Render build step complete. Ready for startup."

# Display build summary
echo ""
echo "📋 Build Summary:"
echo "=================="
print_success "Native Orchestrator: bin/knirv-orchestrator"
print_success "KNIRV Services:      bin/knirv*"
echo ""
print_status "Build artifacts ready for deployment"
print_status "Use './scripts/start-render.sh' to run the native orchestrator"
