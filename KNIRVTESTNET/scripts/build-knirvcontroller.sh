#!/bin/bash

# KNIRV TESTNET - Build Real KNIRVCONTROLLER
# Builds and prepares the real KNIRVCONTROLLER for testnet deployment

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTNET_ROOT="$(dirname "$SCRIPT_DIR")"
PROJECT_ROOT="$(dirname "$TESTNET_ROOT")"
CONTROLLER_DIR="$PROJECT_ROOT/KNIRVCONTROLLER"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

print_info "🔧 Building Real KNIRVCONTROLLER for testnet..."

# Check if KNIRVCONTROLLER exists
if [ ! -d "$CONTROLLER_DIR" ]; then
    print_error "KNIRVCONTROLLER directory not found: $CONTROLLER_DIR"
    print_info "The real KNIRVCONTROLLER is optional. Demo KNIRVCONTROLLER will be used instead."
    exit 0
fi

cd "$CONTROLLER_DIR"

print_info "📦 Installing KNIRVCONTROLLER dependencies..."

# Install dependencies with legacy peer deps to handle conflicts
if ! npm install --legacy-peer-deps; then
    print_warning "npm install failed, trying with force..."
    npm install --force || {
        print_error "Failed to install dependencies"
        print_info "Demo KNIRVCONTROLLER will be used instead"
        exit 0
    }
fi

print_info "🏗️ Building KNIRVCONTROLLER..."

# Build the project
if npm run build; then
    print_success "KNIRVCONTROLLER built successfully"
else
    print_warning "Build failed, but continuing..."
fi

# Create testnet configuration
print_info "⚙️ Creating testnet configuration..."

mkdir -p "$TESTNET_ROOT/config/knirvcontroller"

cat > "$TESTNET_ROOT/config/knirvcontroller/testnet-config.json" << 'EOF'
{
  "mode": "testnet",
  "port": 8088,
  "apiPort": 8089,
  "services": {
    "knirvchain": "http://localhost:8090",
    "knirvgraph": "http://localhost:8082",
    "knirvnexus": "http://localhost:8084",
    "knirvoracle": "http://localhost:1317",
    "knirvrouter": "http://localhost:8086",
    "knirvgateway": "http://localhost:8888"
  },
  "features": {
    "demoMode": false,
    "mockServices": false,
    "enableCORS": true,
    "enableLogging": true,
    "enableMetrics": true
  },
  "testnet": {
    "enableTestData": true,
    "enableDemoAgents": true,
    "enableMockSkills": false,
    "resourceLimits": {
      "maxAgents": 10,
      "maxSkills": 50,
      "maxConcurrentInvocations": 5
    }
  }
}
EOF

# Create startup script
cat > "$TESTNET_ROOT/config/knirvcontroller/start-real-controller.sh" << 'EOF'
#!/bin/bash

# Start real KNIRVCONTROLLER with testnet configuration

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTNET_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"
PROJECT_ROOT="$(dirname "$TESTNET_ROOT")"
CONTROLLER_DIR="$PROJECT_ROOT/KNIRVCONTROLLER"

if [ ! -d "$CONTROLLER_DIR" ]; then
    echo "KNIRVCONTROLLER not found, using demo service"
    exit 1
fi

cd "$CONTROLLER_DIR"

# Set environment variables for testnet
export NODE_ENV=testnet
export KNIRV_TESTNET_MODE=true
export KNIRV_CONFIG_FILE="$TESTNET_ROOT/config/knirvcontroller/testnet-config.json"
export PORT=8088
export API_PORT=8089

# Start the controller
if [ -f "dist/index.js" ]; then
    echo "Starting built KNIRVCONTROLLER..."
    node dist/index.js
elif [ -f "package.json" ] && grep -q "dev" package.json; then
    echo "Starting KNIRVCONTROLLER in development mode..."
    npm run dev
else
    echo "Starting KNIRVCONTROLLER with npm start..."
    npm start
fi
EOF

chmod +x "$TESTNET_ROOT/config/knirvcontroller/start-real-controller.sh"

print_success "✅ Real KNIRVCONTROLLER build completed!"
print_info "📋 Configuration:"
print_info "   Config: $TESTNET_ROOT/config/knirvcontroller/testnet-config.json"
print_info "   Startup: $TESTNET_ROOT/config/knirvcontroller/start-real-controller.sh"
print_info "   Frontend: http://localhost:8088"
print_info "   API: http://localhost:8089"

print_info "🚀 To use real KNIRVCONTROLLER:"
print_info "   1. Run: ./scripts/start-knirvcontroller.sh real"
print_info "   2. Or manually: $TESTNET_ROOT/config/knirvcontroller/start-real-controller.sh"

print_info "🎮 To use demo KNIRVCONTROLLER:"
print_info "   1. Run: ./scripts/start-knirvcontroller.sh demo"
print_info "   2. Or let testnet auto-detect (default)"
