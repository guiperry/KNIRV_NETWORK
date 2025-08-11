#!/bin/bash
set -e

echo "=========================================="
echo "KNIRV-TESTNET: Network Startup"
echo "=========================================="

# Already in KNIRVTESTNET directory

# Function to check if service is ready
wait_for_service() {
    local url=$1
    local name=$2
    local max_attempts=30
    local attempt=1
    
    echo "Waiting for $name to be ready..."
    while [ $attempt -le $max_attempts ]; do
        if curl -s -f "$url" > /dev/null 2>&1; then
            echo "✅ $name is ready!"
            return 0
        fi
        echo "⏳ Attempt $attempt/$max_attempts: $name not ready yet..."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    echo "❌ ERROR: $name failed to start within timeout"
    return 1
}

# Skip IPFS for now (testnet can work without it)
echo "⏭️  Skipping IPFS setup for testnet..."

# Start core blockchain layers
echo "🚀 Starting KNIRV-ROOT..."
./scripts/start-knirvroot.sh
wait_for_service "http://localhost:1317/status" "KNIRV-ROOT"

echo "🚀 Starting KNIRVCHAIN..."
./scripts/start-knirvchain.sh
wait_for_service "http://localhost:8080/health" "KNIRVCHAIN"

echo "🚀 Starting KNIRVGRAPH..."
./scripts/start-knirvgraph.sh
wait_for_service "http://localhost:8081/health" "KNIRVGRAPH"

# Start compute layer
echo "🚀 Starting KNIRV-NEXUS nodes..."
./scripts/start-knirvnexus.sh
wait_for_service "http://localhost:8082/status" "KNIRV-NEXUS-1"
wait_for_service "http://localhost:8083/status" "KNIRV-NEXUS-2"

echo "🚀 Starting KNIRV-ROUTER..."
./scripts/start-knirvrouter.sh
wait_for_service "http://localhost:8086/status" "KNIRV-ROUTER"

# Start interface layer
echo "🚀 Starting KNIRV-GATEWAY..."
./scripts/start-knirvgateway.sh
wait_for_service "http://localhost:8087/health" "KNIRV-GATEWAY"

# Populate sample data
echo "📊 Populating sample data..."
sleep 5  # Allow services to fully initialize
./scripts/populate-knirvgraph.sh

echo ""
echo "=========================================="
echo "🎉 KNIRV-TESTNET: Startup Complete!"
echo "=========================================="
echo ""
echo "🌐 Service Endpoints:"
echo "  KNIRV-ROOT:     http://localhost:1317"
echo "  KNIRVCHAIN:     http://localhost:8080"
echo "  KNIRVGRAPH:     http://localhost:8081"
echo "  KNIRV-NEXUS-1:  http://localhost:8082"
echo "  KNIRV-NEXUS-2:  http://localhost:8083"
echo "  KNIRV-ROUTER:   http://localhost:8086"
echo "  KNIRV-GATEWAY:  http://localhost:8087"
echo "  IPFS API:       http://localhost:5001"
echo ""
echo "📋 Management:"
echo "  Logs: ./logs/"
echo "  PIDs: ./data/*.pid"
echo "  Stop: ./scripts/stop-testnet.sh"
echo "  Health: ./scripts/health-check.sh"
echo "  Tests: ./scripts/run-tests.sh"
