#!/bin/bash

# KNIRV-ORACLE Startup Script with Economics Integration
# This script starts KNIRV-ORACLE with the integrated economics service
# Run from project root: ./scripts/start-with-economics.sh

set -e

echo "🚀 Starting KNIRV-ORACLE with Economics Integration..."

# Get script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
KNIRVORACLE_DIR="$PROJECT_ROOT/KNIRVORACLE"

# Set environment variables for economics integration
export ECONOMICS_LOCAL_MODE=true
export ECONOMICS_SERVICE_URL=http://localhost:8090

# Set KNIRV service URLs (update these to match your deployment)
export KNIRVCHAIN_URL=https://chain.knirv.com
export KNIRVGRAPH_URL=https://graph.knirv.com
export KNIRVNEXUS_URL=https://nexus.knirv.com
export KNIRVORACLE_URL=https://root.knirv.com

# Economics service configuration
export NRN_CONTRACT=nrn_contract_address_placeholder
export XION_RPC=https://rpc.xion-testnet-1.burnt.com:443

echo "📊 Economics service will run in LOCAL mode"
echo "🌐 Service URLs configured:"
echo "  - KNIRVCHAIN: $KNIRVCHAIN_URL"
echo "  - KNIRVGRAPH: $KNIRVGRAPH_URL"
echo "  - KNIRVNEXUS: $KNIRVNEXUS_URL"
echo "  - KNIRVORACLE: $KNIRVORACLE_URL"

# Change to KNIRVORACLE directory
cd "$KNIRVORACLE_DIR"

# Check if go.mod exists
if [ ! -f "go.mod" ]; then
    echo "❌ Error: go.mod not found in $KNIRVORACLE_DIR"
    exit 1
fi

# Check if economics module exists
if [ ! -d "economics" ]; then
    echo "❌ Error: economics directory not found. Please ensure the economics module has been moved to KNIRVORACLE."
    exit 1
fi

# Install dependencies
echo "📦 Installing Go dependencies..."
go mod tidy

# Build the application
echo "🔨 Building KNIRVORACLE..."
go build -o bin/knirvoracle .

# Check if build was successful
if [ ! -f "bin/knirvoracle" ]; then
    echo "❌ Error: Build failed. Please check for compilation errors."
    exit 1
fi

echo "✅ Build successful!"

# Create logs directory if it doesn't exist
mkdir -p logs

# Function to handle cleanup on exit
cleanup() {
    echo ""
    echo "🛑 Shutting down KNIRVORACLE..."
    if [ ! -z "$KNIRVORACLE_PID" ]; then
        kill $KNIRVORACLE_PID 2>/dev/null || true
        wait $KNIRVORACLE_PID 2>/dev/null || true
    fi
    echo "✅ KNIRVORACLE stopped"
    exit 0
}

# Set up signal handlers
trap cleanup SIGINT SIGTERM

# Start KNIRVORACLE
echo "🚀 Starting KNIRVORACLE with economics integration..."
echo "📝 Logs will be written to logs/KNIRVORACLE.log"
echo "📊 Economics service will be available at http://localhost:8090"
echo ""
echo "Press Ctrl+C to stop..."

# Start the application and capture PID
./bin/knirvoracle 2>&1 | tee logs/KNIRVORACLE.log &
KNIRVORACLE_PID=$!

# Wait for the application to start
sleep 3

# Check if the application is still running
if ! kill -0 $KNIRVORACLE_PID 2>/dev/null; then
    echo "❌ Error: KNIRVORACLE failed to start. Check logs/KNIRVORACLE.log for details."
    exit 1
fi

echo "✅ KNIRVORACLE started successfully (PID: $KNIRVORACLE_PID)"

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
echo "🎉 KNIRVORACLE with Economics Integration is running!"
echo ""
echo "📋 Available endpoints:"
echo "  - KNIRVORACLE API: http://localhost:8080"
echo "  - Economics Service: http://localhost:8090"
echo "  - Economics Health: http://localhost:8090/economics/health"
echo "  - Economics Status: http://localhost:8090/economics/status"
echo "  - Economics Metrics: http://localhost:8090/economics/metrics"
echo ""
echo "📖 To test the economics service:"
echo "  curl http://localhost:8090/economics/health"
echo "  curl http://localhost:8090/economics/status"
echo ""
echo "🔍 Monitor logs: tail -f logs/KNIRVORACLE.log"
echo ""

# Wait for the application to finish
wait $KNIRVORACLE_PID
