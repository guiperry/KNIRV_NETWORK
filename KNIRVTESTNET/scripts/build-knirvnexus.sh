#!/bin/bash
set -e

echo "Building KNIRV-NEXUS unified binary for testnet using new architecture..."

# Check if KNIRVNEXUS directory exists
if [ ! -d "../KNIRVNEXUS" ]; then
    echo "❌ KNIRVNEXUS directory not found"
    exit 1
fi

cd ../KNIRVNEXUS

# Set build variables for testnet
export VERSION="testnet-$(date +%Y%m%d-%H%M%S)"
export BUILD_TIME="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
export GIT_COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")"

echo "🔧 Building with:"
echo "  Version: $VERSION"
echo "  Build Time: $BUILD_TIME"
echo "  Git Commit: $GIT_COMMIT"

# Install frontend dependencies
echo "📦 Installing frontend dependencies..."
if [ -f "package.json" ]; then
    npm install
else
    echo "❌ No package.json found in KNIRVNEXUS"
    exit 1
fi

# Compile socket.io TypeScript
echo "🔌 Compiling Socket.IO TypeScript..."
npm run compile:socket

# Build frontend using Next.js
echo "🎨 Building Next.js frontend..."
npm run build

# Verify frontend build output
if [ ! -d "out" ]; then
    echo "❌ Frontend build failed - 'out' directory not found"
    exit 1
fi
echo "✅ Frontend build completed - found $(find out -type f | wc -l) files"

# Build backend using new architecture
echo "⚙️ Building unified backend..."
if [ -d "backend" ]; then
    cd backend
    echo "  Installing backend dependencies..."
    go mod tidy

    echo "  Building complete backend package..."
    go build -ldflags "-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME -X main.GitCommit=$GIT_COMMIT -w -s" -o bin/nexus-backend .

    # Verify backend binary
    if [ ! -f "bin/nexus-backend" ]; then
        echo "❌ Backend build failed - binary not found"
        exit 1
    fi
    echo "  ✅ Backend binary built successfully"

    # Copy backend binary to root bin directory
    mkdir -p ../bin
    cp bin/nexus-backend ../bin/
    cd ..
else
    echo "❌ Backend directory not found"
    exit 1
fi

# Build unified binary with embedded frontend and backend
echo "🔗 Building unified KNIRV-NEXUS binary with embedded components..."
go mod tidy
go build -ldflags "-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME -X main.GitCommit=$GIT_COMMIT -w -s" -o knirv-nexus main.go

# Verify unified binary
if [ ! -f "knirv-nexus" ]; then
    echo "❌ Unified binary build failed"
    exit 1
fi

echo "✅ Unified binary built successfully ($(du -h knirv-nexus | cut -f1))"

# Copy unified binary to testnet bin directory
echo "📋 Copying unified binary to testnet..."
cp knirv-nexus ../KNIRVTESTNET/bin/knirvnexus

cd ../KNIRVTESTNET

# Verify the binary was copied successfully
if [ ! -f "bin/knirvnexus" ]; then
    echo "❌ Failed to copy unified binary to testnet"
    exit 1
fi

echo "✅ Built and copied KNIRV-NEXUS unified binary"

# Create testnet data directories
echo "📁 Setting up testnet data directories..."
mkdir -p data/knirvnexus
mkdir -p logs

# Copy testnet configuration from KNIRVNEXUS if available
echo "⚙️ Setting up testnet configuration..."
if [ -f "../KNIRVNEXUS/config/nexus-testnet.yaml" ]; then
    echo "  Copying testnet config from KNIRVNEXUS..."
    cp ../KNIRVNEXUS/config/nexus-testnet.yaml config/nexus-testnet.yaml
else
    echo "  Creating default testnet configuration..."
    cat > config/nexus-testnet.yaml << 'EOF'
# KNIRV-NEXUS Testnet Configuration
host: "0.0.0.0"
port: 8084
backend_port: 8080
log_level: "info"
testnet: true

# Testnet-specific settings
testnet_config:
  enabled: true
  tee:
    simulation_mode: true
    mock_validation: true
  validation:
    mock_responses: true
    simplified_proofs: true
    timeout_ms: 5000
  database:
    clean_on_start: true
    in_memory: false
    path: "./data/knirvnexus/testnet.db"
EOF
fi

# Also create the legacy config format for backward compatibility
cat > data/knirvnexus/config.yaml << 'EOF'
testnet:
  enabled: true
  tee:
    simulation_mode: true
    mock_validation: true

nexus:
  gui_port: 8082
  api_port: 8083
  tee_port: 8182

tee:
  simulation_mode: true
  mock_validation: true
  simplified_validation: true

validation:
  mock_responses: true
  simplified_proofs: true
  timeout_ms: 5000

database:
  clean_on_start: true
  in_memory: false
  path: "./data/knirvnexus/testnet.db"
EOF

# Copy configuration to config directory as well for backward compatibility
cp data/knirvnexus/config.yaml config/knirvnexus-testnet-config.yaml

echo ""
echo "🎉 KNIRV-NEXUS testnet build completed successfully!"
echo "📋 Summary:"
echo "  ✅ Frontend built with Next.js and embedded"
echo "  ✅ Backend built as unified package"
echo "  ✅ Unified binary created with embedded components"
echo "  ✅ Binary copied to testnet ($(du -h bin/knirvnexus | cut -f1))"
echo "  ✅ Testnet configuration created"
echo ""
echo "🚀 Ready to start with: ./scripts/start-knirvnexus.sh"
