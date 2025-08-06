#!/bin/bash
set -e

echo "Starting KNIRV-NEXUS testnet node..."

# Create necessary directories
mkdir -p logs data

# Check if binary exists
if [ ! -f "./bin/knirvnexus" ]; then
    echo "Error: KNIRV-NEXUS binary not found. Please run build-knirvnexus.sh first."
    exit 1
fi

# Set environment variables for testnet
export JWT_SECRET="testnet-jwt-secret-key-for-development-only"
export RUST_LOG="info"

# Copy port configuration to working directory
cp ./data/knirvnexus/ports.config ./ports.config

# Start KNIRV-NEXUS in testnet mode
echo "Starting KNIRV-NEXUS with TEE simulation..."
./bin/knirvnexus \
    -gui-port 8083 \
    -clean-db \
    > ./logs/knirvnexus.log 2>&1 &

echo $! > ./data/knirvnexus.pid
echo "KNIRV-NEXUS testnet started with PID $(cat ./data/knirvnexus.pid)"
echo "GUI endpoint: http://localhost:8083"
echo "API endpoint: http://localhost:8084"
echo "TEE Simulator: http://localhost:8183"
echo "Testnet features:"
echo "  - TEE simulation enabled"
echo "  - Mock validation responses"
echo "  - Simplified validation proofs"
echo "  - Clean database on start"
echo "Log file: ./logs/knirvnexus.log"

# Wait a moment and check if process is still running
sleep 3
if ! kill -0 $(cat ./data/knirvnexus.pid) 2>/dev/null; then
    echo "Error: KNIRV-NEXUS failed to start. Check logs:"
    tail -20 ./logs/knirvnexus.log
    exit 1
fi

echo "KNIRV-NEXUS testnet is running successfully!"
