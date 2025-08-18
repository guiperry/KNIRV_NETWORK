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

# Start KNIRVGRAPH in testnet mode with in-memory storage (no memory limit due to Go 1.23.3 compatibility)
echo "Starting KNIRVGRAPH with testnet features and resource optimizations..."
(
    # Note: Memory will be managed by Go runtime and system limits (Go 1.23.3 compatibility)
    exec ./bin/knirvgraph \
        --testnet \
        --memory \
        --populate \
        --max-nodes 250 \
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
echo "Resource optimizations:"
echo "  - Reduced max nodes: 250 (50% reduction from default 500)"
echo "  - In-memory storage (faster, reduced disk I/O)"
echo "  - No memory limit (Go 1.23.3 runtime managed)"
echo "Log file: ./logs/knirvgraph.log"

# Wait a moment and check if process is still running
sleep 3
if ! kill -0 $(cat ./data/knirvgraph.pid) 2>/dev/null; then
    echo "Error: KNIRVGRAPH failed to start. Check logs:"
    tail -20 ./logs/knirvgraph.log
    exit 1
fi

echo "KNIRVGRAPH testnet is running successfully!"
