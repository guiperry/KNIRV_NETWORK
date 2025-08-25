#!/bin/bash
set -e

echo "Starting KNIRV-NEXUS unified testnet node..."

# Create necessary directories
mkdir -p logs data config

# Check if unified binary exists
if [ ! -f "./bin/knirvnexus" ]; then
    echo "Error: KNIRV-NEXUS unified binary not found. Please run build-knirvnexus.sh first."
    exit 1
fi

# Get the correct base directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(dirname "$SCRIPT_DIR")"

# Copy testnet configuration
echo "Setting up testnet configuration..."
if [ -f "../KNIRVNEXUS/config/nexus-testnet.yaml" ]; then
    cp ../KNIRVNEXUS/config/nexus-testnet.yaml $BASE_DIR/config/nexus-testnet.yaml
else
    echo "⚠️  Testnet config not found, using default configuration"
fi

# Create data directory for NEXUS
mkdir -p $BASE_DIR/data/knirvnexus

# Start unified KNIRV-NEXUS binary with testnet mode
echo "Starting KNIRV-NEXUS unified binary with testnet mode and embedded frontend..."
cd $BASE_DIR && (
    # Note: Memory will be managed by Go runtime and system limits
    exec ./bin/knirvnexus \
        -testnet \
        -port 8084 \
        -config config/nexus-testnet.yaml
) > ./logs/knirvnexus.log 2>&1 &

NEXUS_PID=$!
echo $NEXUS_PID > data/knirvnexus.pid

# Wait a moment for startup
sleep 5

# Check if the process is still running
if ! kill -0 $(cat ./data/knirvnexus.pid) 2>/dev/null; then
    echo "Error: KNIRV-NEXUS failed to start. Check logs:"
    tail -20 ./logs/knirvnexus.log
    exit 1
fi

echo "KNIRV-NEXUS unified service is running successfully!"
echo "🌐 Frontend and API available at: http://localhost:8084"
echo "📋 Logs: ./logs/knirvnexus.log"
echo "🔧 PID file: ./data/knirvnexus.pid"
echo "Testnet features:"
echo "  - Unified binary with embedded frontend"
echo "  - TEE simulation enabled"
echo "  - Mock validation responses"
echo "  - Simplified validation proofs"
echo "  - Testnet optimizations"
