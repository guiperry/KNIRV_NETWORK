#!/bin/bash
set -e

# Render.com Build Script for KNIRV Testnet
# This script is optimized for Render's build environment

echo "🚀 KNIRV Testnet Build for Render.com"
echo "====================================="

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

# Install toolchains first
print_status "Installing required toolchains..."
bash scripts/install-deps.sh

# Load the environment using our environment loader
print_status "Loading toolchain environment..."
source scripts/load-env.sh

# Export environment variables for all subsequent processes
export PATH="$HOME/.local/bin:$HOME/.cargo/bin:$HOME/.local/go/bin:$PATH"
export GOROOT="$HOME/.local/go"
export GOPATH="$HOME/.local/go-workspace"

# Create a startup script that sets environment for npm start
print_status "Creating startup environment script..."
cat > start-with-env.sh << 'EOF'
#!/bin/bash
# Load environment and start the application
source scripts/load-env.sh 2>/dev/null || true
export PATH="$HOME/.local/bin:$HOME/.cargo/bin:$HOME/.local/go/bin:$PATH"
export GOROOT="$HOME/.local/go"
export GOPATH="$HOME/.local/go-workspace"
exec npm run server:start
EOF

chmod +x start-with-env.sh

# Verify toolchains are available
print_status "Verifying toolchains..."

if ! command -v go &> /dev/null; then
    print_error "Go toolchain not available after installation"
    exit 1
fi

if ! command -v rustc &> /dev/null; then
    print_error "Rust toolchain not available after installation"
    exit 1
fi

print_success "Toolchains verified successfully"

# Install Node.js dependencies
print_status "Installing Node.js dependencies..."
npm install

# Install dependencies for sub-projects
if [ -d "nexus-portal" ]; then
    print_status "Installing NEXUS portal dependencies..."
    cd nexus-portal && npm install && cd ..
fi

if [ -d "nanda_ans" ]; then
    print_status "Installing NANDA ANS dependencies..."
    cd nanda_ans && npm install && cd ..
fi

# Build all components
print_status "Building KNIRV components..."

# Create necessary directories
mkdir -p logs data bin config

# Build components that don't require external dependencies first
print_status "Building KNIRV-ORACLE..."
if [ -f "scripts/build-knirvoracle.sh" ]; then
    bash scripts/build-knirvoracle.sh
else
    print_warning "KNIRV-ORACLE build script not found, skipping"
fi

print_status "Building KNIRVCHAIN..."
if [ -f "scripts/build-knirvchain.sh" ]; then
    bash scripts/build-knirvchain.sh
else
    print_warning "KNIRVCHAIN build script not found, skipping"
fi

print_status "Building KNIRVGRAPH..."
if [ -f "scripts/build-knirvgraph.sh" ]; then
    bash scripts/build-knirvgraph.sh
else
    print_warning "KNIRVGRAPH build script not found, skipping"
fi

print_status "Building KNIRV-NEXUS..."
if [ -f "scripts/build-knirvserver.sh" ]; then
    bash scripts/build-knirvserver.sh
else
    print_warning "KNIRV-NEXUS build script not found, skipping"
fi

print_status "Building KNIRV-ROUTER..."
if [ -f "scripts/build-knirvrouter.sh" ]; then
    bash scripts/build-knirvrouter.sh
else
    print_warning "KNIRV-ROUTER build script not found, skipping"
fi

print_status "Building KNIRV-GATEWAY..."
if [ -f "scripts/build-knirvgateway.sh" ]; then
    bash scripts/build-knirvgateway.sh
else
    print_warning "KNIRV-GATEWAY build script not found, skipping"
fi

print_status "Building NANDA ANS..."
if [ -f "scripts/build-nanda-ans.sh" ]; then
    bash scripts/build-nanda-ans.sh
else
    print_warning "NANDA ANS build script not found, skipping"
fi

# Build frontend components
if [ -d "nexus-portal" ]; then
    print_status "Building NEXUS portal frontend..."
    cd nexus-portal
    npm run build 2>/dev/null || print_warning "NEXUS portal build failed"
    cd ..
fi

if [ -d "nanda_ans" ]; then
    print_status "Building NANDA ANS frontend..."
    cd nanda_ans
    npm run build 2>/dev/null || print_warning "NANDA ANS build failed"
    cd ..
fi

# Load endpoints for production
print_status "Loading production endpoints..."
npm run load-endpoints:testnet 2>/dev/null || print_warning "Endpoint loading failed"

print_success "KNIRV Testnet build completed for Render!"

# Display build summary
echo ""
echo "📋 Build Summary:"
echo "=================="

# Check which binaries were built
for binary in bin/*; do
    if [ -f "$binary" ] && [ -x "$binary" ]; then
        print_success "Built: $(basename "$binary")"
    fi
done

# Check frontend builds
if [ -d "nexus-portal/.next" ] || [ -d "nexus-portal/dist" ]; then
    print_success "NEXUS portal frontend built"
fi

if [ -d "nanda_ans/.next" ] || [ -d "nanda_ans/dist" ]; then
    print_success "NANDA ANS frontend built"
fi

echo ""
print_status "Build artifacts ready for deployment"
print_status "Use 'npm start' to run the testnet"
