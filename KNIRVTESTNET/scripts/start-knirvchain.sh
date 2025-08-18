#!/bin/bash
set -e

echo "Starting KNIRVCHAIN testnet node..."

# Create necessary directories
mkdir -p logs data

# Check if binary exists
if [ ! -f "./bin/knirvchain" ]; then
    echo "Error: KNIRVCHAIN binary not found. Please run build-knirvchain.sh first."
    exit 1
fi

# Set environment variables for testnet
export KNIRVCHAIN_RPC_ENDPOINT="127.0.0.1:8090"
export BLOCK_DIFFICULTY="1"
export KNIRVCHAIN_ID="1"
export BLOCK_TIME="5"
export RUST_LOG="info"

# Start KNIRVCHAIN in testnet mode with memory limit (80MB)
echo "Starting KNIRVCHAIN with testnet features and 80MB memory limit..."
cd data/knirvchain
(
    # Set memory limit for this process (80MB = 81920KB)
    ulimit -v 81920
    exec ../../bin/knirvchain
) > ../../logs/knirvchain.log 2>&1 &
cd ../..

echo $! > ./data/knirvchain.pid
echo "KNIRVCHAIN testnet started with PID $(cat ./data/knirvchain.pid)"
echo "API endpoint: http://localhost:8090"
echo "Testnet endpoints:"
echo "  - Health check: http://localhost:8090/health"
echo "  - Testnet status: http://localhost:8090/testnet/status"
echo "  - Mock LLM validate: http://localhost:8090/testnet/llm/validate"
echo "  - Mock skill validate: http://localhost:8090/testnet/skill/validate"
echo "Log file: ./logs/knirvchain.log"

# Wait a moment and check if process is still running
sleep 3
if ! kill -0 $(cat ./data/knirvchain.pid) 2>/dev/null; then
    echo "Error: KNIRVCHAIN failed to start. Check logs:"
    tail -20 ./logs/knirvchain.log
    exit 1
fi

echo "KNIRVCHAIN testnet is running successfully!"
