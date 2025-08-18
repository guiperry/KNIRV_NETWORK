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

# Install Node.js dependencies with axios fix
print_status "Installing Node.js dependencies..."
npm install || {
    print_warning "npm install failed, checking for axios corruption..."
    if [ -f "scripts/fix-axios-corruption.sh" ]; then
        print_status "Running axios corruption fix..."
        chmod +x scripts/fix-axios-corruption.sh
        ./scripts/fix-axios-corruption.sh || {
            print_warning "Axios fix script failed, trying manual fix..."
            npm install axios@1.6.8 --save-exact
        }
    else
        print_warning "Axios fix script not found, trying manual fix..."
        npm install axios@1.6.8 --save-exact
    fi

    # Retry npm install after axios fix
    npm install || {
        print_error "npm install failed even after axios fix"
        exit 1
    }
}

# Install dependencies for sub-projects
# NEXUS frontend is now built via build-nexus-frontend.sh script
print_status "NEXUS frontend will be built via build-nexus-frontend.sh"



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
if [ -f "scripts/build-knirvnexus.sh" ]; then
    bash scripts/build-knirvnexus.sh
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



# Build frontend components
print_status "Building NEXUS frontend..."
if ./scripts/build-nexus-frontend.sh; then
    print_success "NEXUS frontend built successfully"
else
    print_warning "NEXUS frontend build failed"
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
if [ -d "data/knirvnexus/portal/.next" ] || [ -d "data/knirvnexus/portal/dist" ]; then
    print_success "NEXUS frontend built"
fi



echo ""
print_status "Build artifacts ready for deployment"
print_status "Use 'npm start' to run the testnet"
