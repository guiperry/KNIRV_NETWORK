#!/bin/bash
# Test script to verify build process works before Render deployment

set -e

echo "🧪 KNIRV Testnet Build Test"
echo "=========================="
echo "This script tests the build process locally to catch issues before Render deployment"
echo ""

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_status() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

print_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Test 1: Check required files exist
print_status "Test 1: Checking required files..."
REQUIRED_FILES=(
    "package.json"
    "config/render.yml"
    "testnet-gateway.Dockerfile"
    "scripts/render-build.sh"
    "scripts/start-render.sh"
    "data/testnet-gateway/index.html"
    "data/testnet-gateway/package.json"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [ -f "$file" ]; then
        print_success "✓ $file exists"
    else
        print_error "✗ $file missing"
        exit 1
    fi
done

# Test 2: Check Dockerfiles exist
print_status "Test 2: Checking Dockerfiles..."
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
        print_success "✓ $dockerfile exists"
    else
        print_error "✗ $dockerfile missing"
        exit 1
    fi
done

# Test 3: Validate config/render.yml syntax
print_status "Test 3: Validating config/render.yml..."
if command -v python3 &> /dev/null; then
    if python3 -c "import yaml; yaml.safe_load(open('config/render.yml'))" 2>/dev/null; then
        print_success "✓ config/render.yml is valid YAML"
    else
        print_error "✗ config/render.yml has syntax errors"
        exit 1
    fi
else
    print_warning "Python3 not available - skipping YAML validation"
fi

# Test 4: Check testnet-gateway package.json
print_status "Test 4: Checking testnet-gateway package.json..."
cd data/testnet-gateway
if npm list --depth=0 &> /dev/null; then
    print_success "✓ testnet-gateway dependencies are valid"
else
    print_warning "Dependencies may need installation"
fi
cd ../..

# Test 5: Test build script (dry run)
print_status "Test 5: Testing build script..."
export RENDER=false  # Force local mode for testing
if bash scripts/render-build.sh &> /tmp/build-test.log; then
    print_success "✓ Build script executed successfully"
else
    print_error "✗ Build script failed"
    echo "Build log:"
    cat /tmp/build-test.log
    exit 1
fi

# Test 6: Check Docker availability (for local testing)
print_status "Test 6: Checking Docker availability..."
if command -v docker &> /dev/null; then
    if docker info &> /dev/null; then
        print_success "✓ Docker is available and running"
    else
        print_warning "Docker is installed but not running"
    fi
else
    print_warning "Docker not available (OK for Render deployment)"
fi

echo ""
print_success "🎉 All tests passed! Build should work on Render."
echo ""
print_status "Next steps:"
print_status "1. Commit and push changes to trigger Render deployment"
print_status "2. Monitor Render build logs for detailed output"
print_status "3. Check service health after deployment"
