#!/bin/bash
set -e

echo "Starting KNIRV-ROOT testnet node..."

# Create necessary directories
mkdir -p logs data

# Check if binary exists
if [ ! -f "./bin/knirvroot" ]; then
    echo "Error: KNIRV-ROOT binary not found. Please run build-knirvroot.sh first."
    exit 1
fi

# Start KNIRV-ROOT in testnet mode
echo "Starting KNIRV-ROOT in testnet mode..."
./bin/knirvroot \
    --testnet \
    --config ./config/knirvroot-testnet-config.json \
    --port 1317 \
    --p2p.port 26656 \
    --shared_database_path ./data/testnet/blockchain.db \
    --miners_address KNIRVROOT_Faucet \
    --root \
    --non-interactive \
    --skip-install \
    > ./logs/knirvroot.log 2>&1 &

echo $! > ./data/knirvroot.pid
echo "KNIRV-ROOT testnet started with PID $(cat ./data/knirvroot.pid)"
echo "API endpoint: http://localhost:1317"
echo "RPC endpoint: http://localhost:26657"
echo "P2P port: 26656"
echo "Log file: ./logs/knirvroot.log"

# Wait a moment and check if process is still running
sleep 2
if ! kill -0 $(cat ./data/knirvroot.pid) 2>/dev/null; then
    echo "Error: KNIRV-ROOT failed to start. Check logs:"
    tail -20 ./logs/knirvroot.log
    exit 1
fi

echo "KNIRV-ROOT testnet is running successfully!"
