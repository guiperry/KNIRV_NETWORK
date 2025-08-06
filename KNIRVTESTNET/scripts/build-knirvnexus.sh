#!/bin/bash
set -e

echo "Building KNIRV-NEXUS for testnet..."

# Use existing KNIRV-NEXUS binary
echo "Using existing KNIRV-NEXUS binary..."
if [ -f "../KNIRVNEXUS/bin/knirvnexus" ]; then
    cp ../KNIRVNEXUS/bin/knirvnexus bin/knirvnexus
    echo "✅ Copied existing KNIRV-NEXUS binary"
else
    echo "⚠️  No existing binary found. Building KNIRV-NEXUS with testnet features..."
    cd ../KNIRVNEXUS
    go mod tidy
    go build -tags testnet -o knirvnexus ./main.go
    cp knirvnexus ../KNIRVTESTNET/bin/
    cd ../KNIRVTESTNET
    echo "✅ Built and copied KNIRV-NEXUS binary"
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
