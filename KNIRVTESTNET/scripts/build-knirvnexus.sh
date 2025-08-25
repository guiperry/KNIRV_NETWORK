#!/bin/bash
set -e

echo "Building KNIRV-NEXUS unified binary for testnet..."

# Check if KNIRVNEXUS directory exists
if [ ! -d "../KNIRVNEXUS" ]; then
    echo "❌ KNIRVNEXUS directory not found"
    exit 1
fi

cd ../KNIRVNEXUS

# Install frontend dependencies
echo "Installing frontend dependencies..."
if [ -f "package.json" ]; then
    npm install
else
    echo "⚠️  No package.json found, skipping npm install"
fi

# Build frontend
echo "Building frontend..."
if [ -f "package.json" ]; then
    npm run build
else
    echo "⚠️  No package.json found, skipping frontend build"
fi

# Build backend
echo "Building unified backend..."
if [ -d "backend" ]; then
    cd backend
    go mod tidy
    go build -tags testnet -o ../bin/nexus-backend ./main.go
    cd ..
else
    echo "❌ Backend directory not found"
    exit 1
fi

# Build unified binary with embedded frontend and backend
echo "Building unified KNIRV-NEXUS binary..."
go mod tidy
go build -tags testnet -ldflags="-s -w" -o knirv-nexus main.go

# Copy unified binary to testnet bin directory
echo "Copying unified binary to testnet..."
cp knirv-nexus ../KNIRVTESTNET/bin/knirvnexus

cd ../KNIRVTESTNET
echo "✅ Built and copied KNIRV-NEXUS unified binary"

# Create testnet data directories
echo "Setting up testnet data directories..."
mkdir -p data/knirvnexus

# Create testnet-specific configuration
echo "Creating testnet configuration..."
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

# Copy configuration to config directory as well
cp data/knirvnexus/config.yaml config/knirvnexus-testnet-config.yaml

echo "KNIRV-NEXUS testnet build completed successfully!"
