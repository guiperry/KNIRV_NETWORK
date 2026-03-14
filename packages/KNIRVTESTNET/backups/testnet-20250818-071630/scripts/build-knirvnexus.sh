#!/bin/bash
set -e

echo "Building KNIRV-NEXUS for testnet..."

# Build KNIRV-NEXUS backend services
echo "Building KNIRV-NEXUS backend services..."
if [ -d "../KNIRVNEXUS/backend" ]; then
    cd ../KNIRVNEXUS/backend

    # Build DVE Manager
    echo "Building DVE Manager..."
    go mod tidy
    go build -tags testnet -o dve-manager ./cmd/dve-manager/main.go

    # Build Validation Core
    echo "Building Validation Core..."
    go build -tags testnet -o validation-core ./cmd/validation-core/main.go

    # Copy binaries to testnet bin directory
    cp dve-manager ../../KNIRVTESTNET/bin/knirvnexus-dve-manager
    cp validation-core ../../KNIRVTESTNET/bin/knirvnexus-validation-core

    cd ../../KNIRVTESTNET
    echo "✅ Built and copied KNIRV-NEXUS backend services"
else
    echo "❌ KNIRV-NEXUS backend directory not found"
    exit 1
fi

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
