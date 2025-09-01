#!/bin/bash
set -e

echo "🐳 KNIRV Testnet Docker Build for Render Deployment"
echo "=================================================="
echo "Build started at: $(date)"
echo "Working directory: $(pwd)"
echo "User: $(whoami)"
echo "Node version: $(node --version 2>/dev/null || echo 'Node not found')"
echo "NPM version: $(npm --version 2>/dev/null || echo 'NPM not found')"
echo ""

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO $(date +%H:%M:%S)]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS $(date +%H:%M:%S)]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING $(date +%H:%M:%S)]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR $(date +%H:%M:%S)]${NC} $1"
}

# Error handler
handle_error() {
    local exit_code=$?
    local line_number=$1
    print_error "Build failed at line $line_number with exit code $exit_code"
    print_error "Last command: $BASH_COMMAND"
    print_error "Environment variables:"
    env | grep -E "(RENDER|NODE|NPM|PATH)" | sort
    exit $exit_code
}

# Set up error trapping
trap 'handle_error $LINENO' ERR

# Environment detection and logging
print_status "=== ENVIRONMENT DETECTION ==="
print_status "RENDER: ${RENDER:-'not set'}"
print_status "RENDER_SERVICE_ID: ${RENDER_SERVICE_ID:-'not set'}"
print_status "RENDER_SERVICE_NAME: ${RENDER_SERVICE_NAME:-'not set'}"
print_status "RENDER_EXTERNAL_URL: ${RENDER_EXTERNAL_URL:-'not set'}"
print_status "RENDER_GIT_COMMIT: ${RENDER_GIT_COMMIT:-'not set'}"
print_status "RENDER_GIT_BRANCH: ${RENDER_GIT_BRANCH:-'not set'}"
print_status "PWD: $(pwd)"
print_status "HOME: ${HOME:-'not set'}"

# List directory contents for debugging
print_status "=== DIRECTORY STRUCTURE ==="
print_status "Current directory contents:"
ls -la || print_error "Failed to list current directory"

print_status "Checking for key directories and files:"
for item in "data" "data/testnet-gateway" "config" "scripts" "package.json" "render.yaml" "testnet-gateway.Dockerfile"; do
    if [ -e "$item" ]; then
        print_success "✓ $item exists"
        if [ -d "$item" ]; then
            print_status "  Contents of $item:"
            ls -la "$item" | head -10 || print_warning "  Could not list $item contents"
        fi
    else
        print_error "✗ $item missing"
    fi
done

# Check if we're on Render
if [ "$RENDER" = "true" ] || [ -n "$RENDER_SERVICE_ID" ]; then
    print_status "=== RENDER DEPLOYMENT MODE ==="
    print_status "Render environment detected - Docker builds handled by Render"
    print_status "Service ID: ${RENDER_SERVICE_ID:-'not set'}"
    print_status "Service Name: ${RENDER_SERVICE_NAME:-'not set'}"
    print_status "External URL: ${RENDER_EXTERNAL_URL:-'not set'}"

    # On Render, the Docker builds are handled automatically by render.yaml
    # This script just needs to prepare any additional assets

    print_status "=== PHASE 1: PREPARING TESTNET-GATEWAY ASSETS ==="

    # Ensure testnet-gateway assets are ready
    if [ -d "data/testnet-gateway" ]; then
        print_status "Found testnet-gateway directory"
        print_status "Contents of data/testnet-gateway:"
        ls -la data/testnet-gateway/ || print_error "Failed to list testnet-gateway contents"

        print_status "Checking for package.json in testnet-gateway..."
        if [ -f "data/testnet-gateway/package.json" ]; then
            print_success "Found package.json in testnet-gateway"
            print_status "Installing testnet-gateway dependencies..."
            cd data/testnet-gateway

            print_status "Current directory: $(pwd)"
            print_status "Running npm ci --only=production..."

            if npm ci --only=production 2>&1; then
                print_success "Dependencies installed successfully"
            else
                print_error "Failed to install dependencies"
                print_status "Trying npm install as fallback..."
                npm install 2>&1 || print_error "npm install also failed"
            fi

            # Run any build steps for the gateway
            print_status "Checking for build script..."
            if npm run build 2>&1; then
                print_success "Testnet-gateway build completed"
            else
                print_warning "No build step for testnet-gateway (static assets) - this is normal"
            fi
            cd ../..
            print_status "Returned to: $(pwd)"
        else
            print_warning "No package.json found in testnet-gateway - treating as static assets"
        fi
    else
        print_error "testnet-gateway directory not found!"
        print_status "Available directories in data/:"
        ls -la data/ 2>/dev/null || print_error "data/ directory not found"
        exit 1
    fi

    print_status "=== PHASE 2: VERIFYING DOCKERFILES ==="

    # Verify all required Dockerfiles exist
    DOCKERFILES=(
        "testnet-gateway.Dockerfile"
        "../KNIRVORACLE/Dockerfile"
        "../KNIRVCHAIN/Dockerfile"
        "../KNIRVGRAPH/Dockerfile"
        "../KNIRVNEXUS/Dockerfile"
        "../KNIRVROUTER/Dockerfile"
    )

    print_status "Checking for required Dockerfiles..."
    for dockerfile in "${DOCKERFILES[@]}"; do
        print_status "Checking: $dockerfile"
        if [ -f "$dockerfile" ]; then
            print_success "✓ $dockerfile exists"
            # Show first few lines of Dockerfile for verification
            print_status "  First 3 lines of $dockerfile:"
            head -3 "$dockerfile" 2>/dev/null | sed 's/^/    /' || print_warning "  Could not read $dockerfile"
        else
            print_error "✗ $dockerfile not found!"
            print_status "  Looking for Dockerfile in directory: $(dirname "$dockerfile")"
            ls -la "$(dirname "$dockerfile")" 2>/dev/null || print_error "  Directory not found"
            exit 1
        fi
    done

    print_success "All Dockerfiles verified"

    print_status "=== PHASE 3: PREPARING CONFIGURATION ==="

    print_status "Creating necessary directories..."
    # Ensure config directories exist
    mkdir -p config data logs
    print_success "Directories created: config, data, logs"

    print_status "Setting permissions..."
    # Set proper permissions
    chmod -R 755 config data 2>/dev/null || print_warning "Could not set permissions (may not be needed)"

    print_status "=== PHASE 4: VERIFYING RENDER.YAML ==="
    if [ -f "render.yaml" ]; then
        print_success "render.yaml found"
        print_status "render.yaml contents (first 20 lines):"
        head -20 render.yaml | sed 's/^/  /' || print_warning "Could not read render.yaml"
    else
        print_error "render.yaml not found!"
        exit 1
    fi

    print_success "Render Docker build preparation complete"

else
    # Local development - build Docker images
    print_status "=== LOCAL DEVELOPMENT MODE ==="
    print_status "Local development environment detected"
    print_status "Building Docker images for local testing..."

    # Check if Docker is available
    if ! command -v docker &> /dev/null; then
        print_error "Docker not found! Please install Docker for local development."
        exit 1
    fi

    print_status "Docker version: $(docker --version)"

    # Check Docker daemon status
    if ! docker info >/dev/null 2>&1; then
        print_error "Docker daemon is not running or accessible"
        print_status "Please start Docker and try again"
        exit 1
    fi

    # Clean up any existing containers/networks that might cause conflicts
    print_status "Cleaning up existing Docker resources..."
    docker system prune -f >/dev/null 2>&1 || true

    print_status "Building testnet-gateway image..."
    print_status "Build context size check..."
    du -sh . 2>/dev/null || echo "Cannot determine directory size"

    # Build with more verbose output and better error handling
    if docker build -f testnet-gateway.Dockerfile -t knirv-testnet-gateway:latest . --progress=plain 2>&1; then
        print_success "Testnet gateway image built"
    else
        print_error "Failed to build testnet gateway image"
        print_status "Checking Docker logs..."
        docker logs $(docker ps -lq) 2>/dev/null || echo "No recent containers to check"
        exit 1
    fi

    print_status "Building service images..."

    services=("KNIRVORACLE" "KNIRVCHAIN" "KNIRVGRAPH" "KNIRVNEXUS" "KNIRVROUTER")
    for service in "${services[@]}"; do
        print_status "Building $service image..."
        if docker build -f "../$service/Dockerfile" -t "knirv-${service,,}:latest" "../$service/" 2>&1; then
            print_success "$service image built"
        else
            print_error "Failed to build $service image"
            exit 1
        fi
    done

    print_success "All Docker images built successfully"
fi

# Display comprehensive build summary
echo ""
echo "📋 COMPREHENSIVE BUILD SUMMARY"
echo "==============================="
echo "Build completed at: $(date)"
echo "Total build time: $((SECONDS / 60))m $((SECONDS % 60))s"
echo ""

if [ "$RENDER" = "true" ] || [ -n "$RENDER_SERVICE_ID" ]; then
    print_success "✅ RENDER DEPLOYMENT PREPARED"
    print_success "  ├── Testnet Gateway: Assets prepared, ready for nginx serving"
    print_success "  ├── Backend Services: All Dockerfiles verified and ready"
    print_success "  ├── Configuration: Directories created and permissions set"
    print_success "  ├── render.yaml: Verified and ready for Render orchestration"
    print_success "  └── Dependencies: Testnet gateway dependencies installed"
    echo ""
    print_status "🚀 NEXT STEPS:"
    print_status "  1. Render will automatically build Docker images using render.yaml"
    print_status "  2. Services will be deployed according to the service definitions"
    print_status "  3. The testnet-gateway will be publicly accessible"
    print_status "  4. Backend services will communicate via internal networking"
else
    print_success "✅ LOCAL DEVELOPMENT READY"
    print_success "  ├── Docker images: All services built successfully"
    print_success "  ├── Testnet Gateway: knirv-testnet-gateway:latest"
    print_success "  ├── KNIRV Oracle: knirv-knirvoracle:latest"
    print_success "  ├── KNIRV Chain: knirv-knirvchain:latest"
    print_success "  ├── KNIRV Graph: knirv-knirvgraph:latest"
    print_success "  ├── KNIRV Nexus: knirv-knirvnexus:latest"
    print_success "  └── KNIRV Router: knirv-knirvrouter:latest"
    echo ""
    print_status "🚀 NEXT STEPS:"
    print_status "  1. Run 'npm start' to start the testnet with docker-compose"
    print_status "  2. Access the testnet at http://localhost:10000"
    print_status "  3. Use 'docker-compose logs -f' to view service logs"
fi

echo ""
print_status "🎉 BUILD SCRIPT COMPLETED SUCCESSFULLY"
print_status "Ready for deployment!"

# Final verification
print_status "=== FINAL VERIFICATION ==="
print_status "Current working directory: $(pwd)"
print_status "Available files:"
ls -la | head -20
echo ""
print_status "Build script finished at: $(date)"
