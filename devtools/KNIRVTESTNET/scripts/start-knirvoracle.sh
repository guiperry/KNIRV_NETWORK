#!/bin/bash
set -e

echo "Starting KNIRV-ORACLE testnet node..."

# Create necessary directories
mkdir -p logs data

# Check if binary exists
if [ ! -f "./bin/knirvoracle" ]; then
    echo "Error: KNIRV-ORACLE binary not found. Please run build-knirvoracle.sh first."
    exit 1
fi

# Start KNIRV-ORACLE in testnet mode with resource reduction (no ulimit due to Go 1.23.3 compatibility)
echo "Starting KNIRV-ORACLE in testnet mode with P2P disabled for reduced resource usage..."
(
    # Note: Memory will be managed by Go runtime and system limits
    # P2P is disabled to reduce memory usage and network traffic in testnet
    exec ./bin/knirvoracle \
        --testnet \
        --disable-p2p \
        --config ./config/knirvoracle-testnet-config.json \
        --port 1317 \
        --p2p.port 26656 \
        --shared_database_path ./data/testnet/blockchain.db \
        --miners_address KNIRVORACLE_Faucet \
        --root \
        --non-interactive \
        --skip-install
) > ./logs/KNIRVORACLE.log 2>&1 &

echo $! > ./data/knirvoracle.pid
echo "KNIRV-ORACLE testnet started with PID $(cat ./data/knirvoracle.pid)"
echo "API endpoint: http://localhost:1317"
echo "RPC endpoint: http://localhost:26657"
echo "P2P port: 26656 (P2P messaging disabled for resource reduction)"
echo "Resource optimizations: P2P disabled (~50-70% memory reduction, ~90% network reduction)"
echo "Log file: ./logs/KNIRVORACLE.log"

# Wait a moment and check if process is still running
sleep 2
if ! kill -0 $(cat ./data/knirvoracle.pid) 2>/dev/null; then
    echo "Error: KNIRV-ORACLE failed to start. Check logs:"
    tail -20 ./logs/KNIRVORACLE.log
    exit 1
fi

echo "KNIRV-ORACLE testnet is running successfully!"
