#!/bin/bash
set -e

echo "Starting KNIRVGRAPH testnet node..."

# Create necessary directories
mkdir -p logs data

# Check if binary exists
if [ ! -f "./bin/knirvgraph" ]; then
    echo "Error: KNIRVGRAPH binary not found. Please run build-knirvgraph.sh first."
    exit 1
fi

# Start KNIRVGRAPH in testnet mode with in-memory storage and memory limit (60MB)
echo "Starting KNIRVGRAPH with testnet features and 60MB memory limit..."
(
    # Set memory limit for this process (60MB = 61440KB)
    ulimit -v 61440
    exec ./bin/knirvgraph \
        --testnet \
        --memory \
        --populate \
        --max-nodes 500 \
        --rpc-port 8082 \
        --home ./data/knirvgraph
) > ./logs/knirvgraph.log 2>&1 &

echo $! > ./data/knirvgraph.pid
echo "KNIRVGRAPH testnet started with PID $(cat ./data/knirvgraph.pid)"
echo "API endpoint: http://localhost:8082"
echo "Testnet features:"
echo "  - In-memory storage enabled"
echo "  - Pre-populated test data"
echo "  - Real DHT implementation"
echo "  - Full graph operations"
echo "Log file: ./logs/knirvgraph.log"

# Wait a moment and check if process is still running
sleep 3
if ! kill -0 $(cat ./data/knirvgraph.pid) 2>/dev/null; then
    echo "Error: KNIRVGRAPH failed to start. Check logs:"
    tail -20 ./logs/knirvgraph.log
    exit 1
fi

echo "KNIRVGRAPH testnet is running successfully!"
