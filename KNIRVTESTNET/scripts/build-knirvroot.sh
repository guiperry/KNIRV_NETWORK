#!/bin/bash
set -e

echo "Building KNIRV-ROOT for testnet..."

# Build KNIRV-ROOT from source with testnet support
echo "Building KNIRV-ROOT from source with testnet support..."
cd ../KNIRVROOT

# Clean any previous builds
rm -f knirvroot KNIRVROOT bin/knirvroot

# Build with all dependencies (build entire package, not just main.go)
echo "Compiling KNIRV-ROOT..."
go build -o knirvroot .

# Copy to testnet bin directory
cp knirvroot ../KNIRVTESTNET/bin/knirvroot
echo "✅ Built and copied KNIRV-ROOT binary with testnet support"

cd ../KNIRVTESTNET

# Create simplified testnet data directory
echo "Setting up testnet data directory..."
mkdir -p data/knirvroot
mkdir -p data/testnet

# Copy testnet configuration
echo "Setting up testnet configuration..."
cp ../KNIRVROOT/config/testnet_config.json config/knirvroot-testnet-config.json

# Create genesis file for testnet
echo "Creating testnet genesis file..."
mkdir -p data/knirvroot/genesis
cat > data/knirvroot/genesis/genesis.json << 'EOF'
{
  "chain_id": "knirv-testnet-1",
  "validators": [
    {
      "address": "validator1",
      "power": 100
    },
    {
      "address": "validator2",
      "power": 100
    },
    {
      "address": "validator3",
      "power": 100
    }
  ],
  "initial_nrn": 1000000000000,
  "timestamp": "2025-08-06T00:00:00Z"
}
EOF

echo "KNIRV-ROOT testnet build completed successfully!"
