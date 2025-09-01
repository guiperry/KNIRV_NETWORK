#!/bin/bash
set -e

echo "🐳 KNIRV Testnet Docker Build for Render Deployment"
echo "=================================================="

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
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

# Check if we're on Render
if [ "$RENDER" = "true" ] || [ -n "$RENDER_SERVICE_ID" ]; then
    print_status "Render environment detected - Docker builds handled by Render"
    print_status "Service ID: ${RENDER_SERVICE_ID:-'not set'}"
    print_status "External URL: ${RENDER_EXTERNAL_URL:-'not set'}"

    # On Render, the Docker builds are handled automatically by render.yaml
    # This script just needs to prepare any additional assets

    print_status "Phase 1: Preparing testnet-gateway assets..."

    # Ensure testnet-gateway assets are ready
    if [ -d "data/testnet-gateway" ]; then
        print_status "Installing testnet-gateway dependencies..."
        cd data/testnet-gateway
        npm ci --only=production --silent

        # Run any build steps for the gateway
        if npm run build > /dev/null 2>&1; then
            print_success "Testnet-gateway build completed"
        else
            print_warning "No build step for testnet-gateway (static assets)"
        fi
        cd ../..
    else
        print_error "testnet-gateway directory not found!"
        exit 1
    fi

    print_status "Phase 2: Verifying Dockerfiles..."

    # Verify all required Dockerfiles exist
    DOCKERFILES=(
        "testnet-gateway.Dockerfile"
        "../KNIRVORACLE/Dockerfile"
        "../KNIRVCHAIN/Dockerfile"
        "../KNIRVGRAPH/Dockerfile"
        "../KNIRVNEXUS/Dockerfile"
        "../KNIRVROUTER/Dockerfile"
    )

    for dockerfile in "${DOCKERFILES[@]}"; do
        if [ -f "$dockerfile" ]; then
            print_success "✓ $dockerfile"
        else
            print_error "✗ $dockerfile not found!"
            exit 1
        fi
    done

    print_success "All Dockerfiles verified"

    print_status "Phase 3: Preparing configuration..."

    # Ensure config directories exist
    mkdir -p config data logs

    # Set proper permissions
    chmod -R 755 config data

    print_success "Render Docker build preparation complete"

else
    # Local development - build Docker images
    print_status "Local development environment detected"
    print_status "Building Docker images for local testing..."

    # Check if Docker is available
    if ! command -v docker &> /dev/null; then
        print_error "Docker not found! Please install Docker for local development."
        exit 1
    fi

    print_status "Building testnet-gateway image..."
    docker build -f testnet-gateway.Dockerfile -t knirv-testnet-gateway:latest .

    print_status "Building service images..."
    docker build -f ../KNIRVORACLE/Dockerfile -t knirv-oracle:latest ../KNIRVORACLE/
    docker build -f ../KNIRVCHAIN/Dockerfile -t knirv-chain:latest ../KNIRVCHAIN/
    docker build -f ../KNIRVGRAPH/Dockerfile -t knirv-graph:latest ../KNIRVGRAPH/
    docker build -f ../KNIRVNEXUS/Dockerfile -t knirv-nexus:latest ../KNIRVNEXUS/
    docker build -f ../KNIRVROUTER/Dockerfile -t knirv-router:latest ../KNIRVROUTER/

    print_success "All Docker images built successfully"
fi

# Display build summary
echo ""
echo "📋 Build Summary:"
echo "=================="
if [ "$RENDER" = "true" ] || [ -n "$RENDER_SERVICE_ID" ]; then
    print_success "Render Docker deployment prepared"
    print_success "Testnet Gateway: Ready for nginx serving"
    print_success "Backend Services: Dockerfiles verified"
    print_success "Configuration: Prepared"
else
    print_success "Local Docker images built"
    print_success "Ready for local testing with docker-compose"
fi
echo ""
print_status "Build complete - ready for deployment"
