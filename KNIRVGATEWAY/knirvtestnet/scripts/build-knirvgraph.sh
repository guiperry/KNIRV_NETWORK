#!/bin/bash
set -e

echo "Building KNIRVGRAPH for testnet..."

# Build KNIRVGRAPH with testnet support
echo "Building KNIRVGRAPH with testnet features..."
cd ../KNIRVGRAPH

# Update dependencies
go mod tidy

# Build with testnet tags
go build -tags testnet -o knirvgraph ./cmd/node/main.go

# Copy binary to testnet bin directory
cp knirvgraph ../KNIRVTESTNET/bin/
cd ../KNIRVTESTNET

# Initialize database
echo "Initializing KNIRVGRAPH database..."
mkdir -p data/knirvgraph

# Create testnet-specific configuration
echo "Creating testnet configuration..."
cat > data/knirvgraph/config.yaml << 'EOF'
testnet:
  enabled: true
  in_memory: true
  pre_populate: true
  max_nodes: 1000
  chain_id: "knirvgraph-testnet-1"
  port: 8081
  local_mode: true

network:
  chain_id: "knirvgraph-testnet-1"
  port: 8081
  local_mode: true

storage:
  backend: "memory"
  in_memory: true
  max_nodes: 1000

test_data:
  pre_populate: true
  error_nodes: 10
  skill_nodes: 5

dht:
  simulation_mode: true
  mock_connectivity: true
  simplified_proofs: true
EOF

# Copy configuration to config directory as well
cp data/knirvgraph/config.yaml config/knirvgraph-testnet-config.yaml

echo "KNIRVGRAPH testnet build completed successfully!"
