#!/bin/bash
set -e

echo "Building KNIRV-ROUTER for testnet..."

# Use existing KNIRV-ROUTER headless binary
echo "Using existing KNIRV-ROUTER headless binary..."
if [ -f "../KNIRVROUTER/bin/knirvrouter-headless" ]; then
    cp ../KNIRVROUTER/bin/knirvrouter-headless bin/knirvrouter
    echo "✅ Copied existing KNIRV-ROUTER headless binary"
else
    echo "⚠️  No existing headless binary found. Building KNIRV-ROUTER with headless mode..."
    cd ../KNIRVROUTER
    go mod tidy
    go build -tags headless -o knirvrouter-headless ./main_headless.go
    cp knirvrouter-headless ../KNIRVTESTNET/bin/knirvrouter
    cd ../KNIRVTESTNET
    echo "✅ Built and copied KNIRV-ROUTER headless binary"
fi

# Create testnet data directories
echo "Setting up testnet data directories..."
mkdir -p data/knirvrouter

# Create testnet-specific configuration
echo "Creating testnet configuration..."
cat > data/knirvrouter/test.env << 'EOF'
# Testnet Configuration for KNIRV-ROUTER
TESTNET_MODE=true
LOCAL_NETWORK_MODE=true
MOCK_NRN_MINTING=true
SIMPLIFIED_CONSENSUS=true
DISABLE_XION_BRIDGE=true

# Basic Configuration
PORT=5001
MINERS_ADDRESS=KNIRVROUTER_Testnet_Miner
BLOCKCHAIN_NAME=KNIRVROUTER_TESTNET
CURRENCY_NAME=NRN
HEX_PREFIX=0x
ADDRESS_PREFIX=KNIRVROUTER-

# Testnet-specific settings
TESTNET_CHAIN_ID=knirvrouter-testnet-1
TESTNET_VALIDATORS=3
TESTNET_INITIAL_NRN=1000000000000

# Mining Configuration
MINING_DIFFICULTY=1
MINING_REWARD=100
DECIMAL=8

# Consensus Configuration
CONSENSUS_PAUSE_TIME=30

# Status Messages
SUCCESS=true
FAILED=false
PENDING=pending
TXN_VERIFICATION_SUCCESS=verified
TXN_VERIFICATION_FAILURE=failed
BLOCKCHAIN_STATUS=active

# Peer Configuration (empty for local testnet)
PEER_ADDRESSES=

# Installation
INSTALL_COMPLETE=true
EOF

# Copy configuration to config directory as well
cp data/knirvrouter/test.env config/knirvrouter-testnet.env

echo "KNIRV-ROUTER testnet build completed successfully!"
