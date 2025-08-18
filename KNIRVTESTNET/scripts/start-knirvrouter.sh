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

# Start KNIRV-ROUTER in testnet mode (no memory limit due to Go 1.23.1 compatibility)
echo "Starting KNIRV-ROUTER with testnet features and resource optimizations..."

# Copy optimized environment file to runtime directory
mkdir -p ./data/knirvrouter
cp ./config/knirvrouter-testnet.env ./data/knirvrouter/.env

# Set resource optimization environment variables
export CONNECTIVITY_LOG_LEVEL=warn
export REDUCE_CONNECTIVITY_LOGS=true

# Start KNIRV-ROUTER from the base directory to avoid path issues
(
    # Note: Memory will be managed by Go runtime and system limits
    exec ./bin/knirvrouter \
        -testnet \
        -local-network \
        -mock-nrn \
        -port 8086 \
        -miners_address KNIRVROUTER_Testnet_Miner
) > ./logs/knirvrouter.log 2>&1 &

echo $! > ./data/knirvrouter.pid
echo "KNIRV-ROUTER testnet started with PID $(cat ./data/knirvrouter.pid)"
echo "API endpoint: http://localhost:8086"
echo "Testnet features:"
echo "  - Local network mode enabled"
echo "  - Mock NRN minting enabled"
echo "  - Simplified consensus enabled"
echo "  - XION bridge disabled"
echo "  - Chain ID: knirvrouter-testnet-1"
echo "Resource optimizations:"
echo "  - No memory limit (Go 1.23.1 runtime managed)"
echo "  - Local network mode (reduced external network calls)"
echo "  - Mock NRN minting (simplified token operations)"
echo "  - Simplified consensus (reduced computational overhead)"
echo "  - Increased mining difficulty: 6 (reduced CPU usage)"
echo "  - Reduced connectivity measurements: every 2 minutes (reduced bandwidth)"
echo "  - Warning-level logging only (reduced log verbosity)"
echo "Log file: ./logs/knirvrouter.log"

# Wait a moment and check if process is still running
sleep 3
if ! kill -0 $(cat ./data/knirvrouter.pid) 2>/dev/null; then
    echo "Error: KNIRV-ROUTER failed to start. Check logs:"
    tail -20 ./logs/knirvrouter.log
    exit 1
fi

echo "KNIRV-ROUTER testnet is running successfully!"
