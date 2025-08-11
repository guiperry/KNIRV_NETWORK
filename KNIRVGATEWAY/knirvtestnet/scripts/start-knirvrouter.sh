#!/bin/bash
set -e

echo "Starting KNIRV-ROUTER testnet node..."

# Create necessary directories
mkdir -p logs data

# Check if binary exists
if [ ! -f "./bin/knirvrouter" ]; then
    echo "Error: KNIRV-ROUTER binary not found. Please run build-knirvrouter.sh first."
    exit 1
fi

# Copy testnet environment file to working directory
cp data/knirvrouter/test.env ./test.env

# Start KNIRV-ROUTER in testnet mode
echo "Starting KNIRV-ROUTER with testnet features..."
./bin/knirvrouter \
    --testnet \
    --local-network \
    --mock-nrn \
    --port 8086 \
    --miners_address KNIRVROUTER_Testnet_Miner \
    > ./logs/knirvrouter.log 2>&1 &

echo $! > ./data/knirvrouter.pid
echo "KNIRV-ROUTER testnet started with PID $(cat ./data/knirvrouter.pid)"
echo "API endpoint: http://localhost:8086"
echo "Testnet features:"
echo "  - Local network mode enabled"
echo "  - Mock NRN minting enabled"
echo "  - Simplified consensus enabled"
echo "  - XION bridge disabled"
echo "  - Chain ID: knirvrouter-testnet-1"
echo "Log file: ./logs/knirvrouter.log"

# Wait a moment and check if process is still running
sleep 3
if ! kill -0 $(cat ./data/knirvrouter.pid) 2>/dev/null; then
    echo "Error: KNIRV-ROUTER failed to start. Check logs:"
    tail -20 ./logs/knirvrouter.log
    exit 1
fi

echo "KNIRV-ROUTER testnet is running successfully!"
