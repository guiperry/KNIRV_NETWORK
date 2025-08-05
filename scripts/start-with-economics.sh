#!/bin/bash

# KNIRV-ROOT Startup Script with Economics Integration
# This script starts KNIRV-ROOT with the integrated economics service
# Run from project root: ./scripts/start-with-economics.sh

set -e

echo "🚀 Starting KNIRV-ROOT with Economics Integration..."

# Get script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
KNIRVROOT_DIR="$PROJECT_ROOT/KNIRVROOT"

# Set environment variables for economics integration
export ECONOMICS_LOCAL_MODE=true
export ECONOMICS_SERVICE_URL=http://localhost:8090

# Set KNIRV service URLs (update these to match your deployment)
export KNIRVCHAIN_URL=https://chain.knirv.com
export KNIRVGRAPH_URL=https://graph.knirv.com
export KNIRVNEXUS_URL=https://nexus.knirv.com
export KNIRVROOT_URL=https://root.knirv.com

# Economics service configuration
export NRN_CONTRACT=nrn_contract_address_placeholder
export XION_RPC=https://rpc.xion-testnet-1.burnt.com:443

echo "📊 Economics service will run in LOCAL mode"
echo "🌐 Service URLs configured:"
echo "  - KNIRVCHAIN: $KNIRVCHAIN_URL"
echo "  - KNIRVGRAPH: $KNIRVGRAPH_URL"
echo "  - KNIRVNEXUS: $KNIRVNEXUS_URL"
echo "  - KNIRVROOT: $KNIRVROOT_URL"

# Change to KNIRVROOT directory
cd "$KNIRVROOT_DIR"

# Check if go.mod exists
if [ ! -f "go.mod" ]; then
    echo "❌ Error: go.mod not found in $KNIRVROOT_DIR"
    exit 1
fi

# Check if economics module exists
if [ ! -d "economics" ]; then
    echo "❌ Error: economics directory not found. Please ensure the economics module has been moved to KNIRVROOT."
    exit 1
fi

# Install dependencies
echo "📦 Installing Go dependencies..."
go mod tidy

# Build the application
echo "🔨 Building KNIRVROOT..."
go build -o bin/knirvroot .

# Check if build was successful
if [ ! -f "bin/knirvroot" ]; then
    echo "❌ Error: Build failed. Please check for compilation errors."
    exit 1
fi

echo "✅ Build successful!"

# Create logs directory if it doesn't exist
mkdir -p logs

# Function to handle cleanup on exit
cleanup() {
    echo ""
    echo "🛑 Shutting down KNIRVROOT..."
    if [ ! -z "$KNIRVROOT_PID" ]; then
        kill $KNIRVROOT_PID 2>/dev/null || true
        wait $KNIRVROOT_PID 2>/dev/null || true
    fi
    echo "✅ KNIRVROOT stopped"
    exit 0
}

# Set up signal handlers
trap cleanup SIGINT SIGTERM

# Start KNIRVROOT
echo "🚀 Starting KNIRVROOT with economics integration..."
echo "📝 Logs will be written to logs/knirvroot.log"
echo "📊 Economics service will be available at http://localhost:8090"
echo ""
echo "Press Ctrl+C to stop..."

# Start the application and capture PID
./bin/knirvroot 2>&1 | tee logs/knirvroot.log &
KNIRVROOT_PID=$!

# Wait for the application to start
sleep 3

# Check if the application is still running
if ! kill -0 $KNIRVROOT_PID 2>/dev/null; then
    echo "❌ Error: KNIRVROOT failed to start. Check logs/knirvroot.log for details."
    exit 1
fi

echo "✅ KNIRVROOT started successfully (PID: $KNIRVROOT_PID)"

# Test economics service
echo "🧪 Testing economics service..."
sleep 2

# Test economics health endpoint
if curl -s http://localhost:8090/economics/health > /dev/null 2>&1; then
    echo "✅ Economics service is healthy"
else
    echo "⚠️  Economics service health check failed (this is normal if it takes time to start)"
fi

# Test economics status endpoint
if curl -s http://localhost:8090/economics/status > /dev/null 2>&1; then
    echo "✅ Economics service status endpoint is responding"
else
    echo "⚠️  Economics service status endpoint not yet available"
fi

echo ""
echo "🎉 KNIRVROOT with Economics Integration is running!"
echo ""
echo "📋 Available endpoints:"
echo "  - KNIRVROOT API: http://localhost:8080"
echo "  - Economics Service: http://localhost:8090"
echo "  - Economics Health: http://localhost:8090/economics/health"
echo "  - Economics Status: http://localhost:8090/economics/status"
echo "  - Economics Metrics: http://localhost:8090/economics/metrics"
echo ""
echo "📖 To test the economics service:"
echo "  curl http://localhost:8090/economics/health"
echo "  curl http://localhost:8090/economics/status"
echo ""
echo "🔍 Monitor logs: tail -f logs/knirvroot.log"
echo ""

# Wait for the application to finish
wait $KNIRVROOT_PID
