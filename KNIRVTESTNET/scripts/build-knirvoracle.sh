#!/bin/bash
set -e

echo "Building KNIRV-ORACLE for testnet..."

# Build KNIRV-ORACLE from source with testnet support
echo "Building KNIRV-ORACLE from source with testnet support..."
cd ../KNIRVORACLE

# Clean any previous builds
rm -f knirvoracle KNIRVORACLE bin/knirvoracle

# Download and tidy dependencies
echo "Downloading dependencies..."
go mod tidy

# Build with all dependencies (build entire package, not just main.go)
echo "Compiling KNIRV-ORACLE..."
go build -o knirvoracle .

# Copy to testnet bin directory
cp knirvoracle ../KNIRVTESTNET/bin/knirvoracle
echo "✅ Built and copied KNIRV-ORACLE binary with testnet support"

cd ../KNIRVTESTNET

# Create simplified testnet data directory
echo "Setting up testnet data directory..."
mkdir -p data/knirvoracle
mkdir -p data/testnet

# Copy testnet configuration
echo "Setting up testnet configuration..."
cp ../KNIRVORACLE/config/testnet_config.json config/knirvoracle-testnet-config.json

# Create genesis file for testnet
echo "Creating testnet genesis file..."
mkdir -p data/knirvoracle/genesis
cat > data/knirvoracle/genesis/genesis.json << 'EOF'
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

echo "KNIRV-ORACLE testnet build completed successfully!"
