#!/bin/bash
set -e

echo "Building KNIRVCHAIN for testnet..."

# Use existing KNIRVCHAIN binary (with testnet features)
echo "Using existing KNIRVCHAIN binary..."
if [ -f "../KNIRVCHAIN/target/release/knirvchain" ]; then
    cp ../KNIRVCHAIN/target/release/knirvchain bin/knirvchain
    echo "✅ Copied existing KNIRVCHAIN binary"
else
    echo "⚠️  No existing binary found. Building KNIRVCHAIN with testnet features..."
    cd ../KNIRVCHAIN
    cargo build --features testnet --release
    cp target/release/knirvchain ../KNIRVTESTNET/bin/
    cd ../KNIRVTESTNET
    echo "✅ Built and copied KNIRVCHAIN binary"
fi

# Initialize database
echo "Initializing KNIRVCHAIN database..."
mkdir -p data/knirvchain

# Create testnet-specific configuration
echo "Creating testnet configuration..."
cat > data/knirvchain/config.toml << 'EOF'
[testnet]
enabled = true
single_validator = true
simplified_consensus = true
mock_ipfs = true
mock_llm = true
in_memory_db = true

[blockchain]
chain_id = "knirvchain-testnet-1"
block_time = 5
difficulty = 1

[api]
host = "127.0.0.1"
port = 8080

[storage]
database_path = "./data/knirvchain/testnet.db"
in_memory = true

[llm]
mock_mode = true
validation_timeout = 5000

[skills]
mock_validation = true
simplified_testing = true
EOF

# Copy configuration to config directory as well
cp data/knirvchain/config.toml config/knirvchain-testnet-config.toml

echo "KNIRVCHAIN testnet build completed successfully!"
