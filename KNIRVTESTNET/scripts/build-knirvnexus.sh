#!/bin/bash
set -e

echo "🚀 Building KNIRV-NEXUS unified binary for testnet using new Makefile architecture..."

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

# Check if KNIRVNEXUS directory exists
if [ ! -d "../KNIRVNEXUS" ]; then
    print_error "KNIRVNEXUS directory not found"
    exit 1
fi

print_status "Changing to KNIRVNEXUS directory..."
cd ../KNIRVNEXUS

# Check for required build tools
print_status "Checking build prerequisites..."
if ! command -v make >/dev/null 2>&1; then
    print_error "make is required but not installed"
    exit 1
fi

if ! command -v go >/dev/null 2>&1; then
    print_error "Go is required but not installed"
    exit 1
fi

if ! command -v node >/dev/null 2>&1; then
    print_error "Node.js is required but not installed"
    exit 1
fi

print_success "All build prerequisites found"

# Set build variables for testnet
export VERSION="testnet-$(date +%Y%m%d-%H%M%S)"
export BUILD_TIME="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
export GIT_COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")"

print_status "Build configuration:"
echo "  Version: $VERSION"
echo "  Build Time: $BUILD_TIME"
echo "  Git Commit: $GIT_COMMIT"
echo ""

# Clean previous builds
print_status "Cleaning previous builds..."
make clean || print_warning "Clean failed (may be first build)"

# Use the new Makefile-based build system
print_status "Building unified binary using Makefile..."
print_status "This will: install deps → build frontend → build backend → create unified binary"

# Run the unified build process
if make binary; then
    print_success "Unified binary build completed successfully"
else
    print_error "Unified binary build failed"
    exit 1
fi

# Verify the unified binary was created
if [ -f "dist/knirv-nexus" ]; then
    print_success "Unified binary created: dist/knirv-nexus ($(du -h dist/knirv-nexus | cut -f1))"
else
    print_error "Unified binary not found at dist/knirv-nexus"
    exit 1
fi

# Copy unified binary to testnet bin directory
print_status "Copying unified binary to testnet..."
mkdir -p ../KNIRVTESTNET/bin
cp dist/knirv-nexus ../KNIRVTESTNET/bin/knirvnexus

cd ../KNIRVTESTNET

# Verify the binary was copied successfully
if [ ! -f "bin/knirvnexus" ]; then
    print_error "Failed to copy unified binary to testnet"
    exit 1
fi

print_success "Built and copied KNIRV-NEXUS unified binary"

# Create testnet data directories
print_status "Setting up testnet data directories..."
mkdir -p data/knirvnexus
mkdir -p logs
mkdir -p config

# Copy testnet configuration from KNIRVNEXUS if available
print_status "Setting up testnet configuration..."
if [ -f "../KNIRVNEXUS/config/nexus-testnet.yaml" ]; then
    print_status "Copying testnet config from KNIRVNEXUS..."
    cp ../KNIRVNEXUS/config/nexus-testnet.yaml config/nexus-testnet.yaml
    print_success "Testnet configuration copied"
else
    print_warning "No testnet config found, creating default configuration..."
    cat > config/nexus-testnet.yaml << 'EOF'
# KNIRV-NEXUS Testnet Configuration
# Updated for new unified architecture (no Socket.io)

server:
  host: "0.0.0.0"
  port: 8084
  environment: "testnet"
  log_level: "info"

# Frontend configuration (embedded in binary)
frontend:
  embedded: true
  static_path: "/static"
  api_prefix: "/api/v1"

# Backend configuration (unified service)
backend:
  api_port: 8084
  health_check_path: "/api/v1/health"
  cors_enabled: true
  cors_origins: ["*"]

# Testnet-specific settings
testnet:
  enabled: true
  simulation_mode: true
  mock_validation: true
  simplified_proofs: true
  timeout_ms: 5000

# Database configuration
database:
  type: "sqlite"
  path: "./data/knirvnexus/testnet.db"
  clean_on_start: true
  auto_migrate: true

# DVE (Distributed Virtual Environment) settings
dve:
  enabled: true
  max_environments: 10
  default_timeout: 300
  cleanup_interval: 3600

# TEE (Trusted Execution Environment) settings
tee:
  simulation_mode: true
  mock_validation: true
  simplified_validation: true

# Logging configuration
logging:
  level: "info"
  format: "json"
  output: "./logs/knirvnexus.log"
  max_size: 100
  max_backups: 5
  max_age: 30
EOF
    print_success "Default testnet configuration created"
fi

# Create a simplified config for backward compatibility
print_status "Creating backward compatibility configuration..."
cat > data/knirvnexus/config.yaml << 'EOF'
# Legacy configuration format for backward compatibility
testnet:
  enabled: true
  simulation_mode: true

nexus:
  port: 8084
  api_port: 8084
  log_level: "info"

database:
  clean_on_start: true
  in_memory: false
  path: "./data/knirvnexus/testnet.db"

dve:
  enabled: true
  max_environments: 10

tee:
  simulation_mode: true
  mock_validation: true
EOF

# Copy configuration to config directory as well for backward compatibility
cp data/knirvnexus/config.yaml config/knirvnexus-testnet-config.yaml

print_status "Setting executable permissions..."
chmod +x bin/knirvnexus

echo ""
print_success "🎉 KNIRV-NEXUS testnet build completed successfully!"
echo ""
print_status "📋 Build Summary:"
print_success "  ✅ Dependencies installed via Makefile"
print_success "  ✅ Frontend built with Next.js (no Socket.io)"
print_success "  ✅ Backend built as unified Go service"
print_success "  ✅ Unified binary created with embedded components"
print_success "  ✅ Binary copied to testnet ($(du -h bin/knirvnexus | cut -f1))"
print_success "  ✅ Testnet configuration created"
print_success "  ✅ Backward compatibility configs created"
echo ""
print_status "🚀 Ready to start with:"
print_status "  ./bin/knirvnexus --config config/nexus-testnet.yaml"
print_status "  OR"
print_status "  ./scripts/start-knirvnexus.sh (if available)"
echo ""
print_status "🔍 Binary info:"
print_status "  Location: $(pwd)/bin/knirvnexus"
print_status "  Size: $(du -h bin/knirvnexus | cut -f1)"
print_status "  Permissions: $(ls -la bin/knirvnexus | cut -d' ' -f1)"
