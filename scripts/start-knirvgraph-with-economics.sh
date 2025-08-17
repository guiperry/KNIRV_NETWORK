#!/bin/bash

# KNIRVGRAPH Startup Script with Economics Integration
# This script starts KNIRVGRAPH with the integrated NRN economics and Proof-of-Solution
# Run from project root: ./scripts/start-knirvgraph-with-economics.sh

set -e

echo "🚀 Starting KNIRVGRAPH with Economics Integration..."

# Get script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
KNIRVGRAPH_DIR="$PROJECT_ROOT/KNIRVGRAPH"

# Set environment variables for economics integration
export ECONOMICS_ENABLED=true
export KNIRVORACLE_URL=${KNIRVORACLE_URL:-http://localhost:1317}
export KNIRVGRAPH_PORT=${KNIRVGRAPH_PORT:-8081}
export KNIRVGRAPH_HOME=${KNIRVGRAPH_HOME:-$HOME/.knirvgraph}

# Set KNIRV service URLs for integration
export KNIRVCHAIN_URL=${KNIRVCHAIN_URL:-http://localhost:8080}
export KNIRVNEXUS_URL=${KNIRVNEXUS_URL:-http://localhost:3000}
export KNIRVGATEWAY_URL=${KNIRVGATEWAY_URL:-http://localhost:8000}

echo "📊 Economics integration enabled"
echo "🌐 Service URLs configured:"
echo "  - KNIRVORACLE: $KNIRVORACLE_URL"
echo "  - KNIRVCHAIN: $KNIRVCHAIN_URL"
echo "  - KNIRVNEXUS: $KNIRVNEXUS_URL"
echo "  - KNIRVGATEWAY: $KNIRVGATEWAY_URL"
echo "  - KNIRVGRAPH Port: $KNIRVGRAPH_PORT"

# Change to KNIRVGRAPH directory
cd "$KNIRVGRAPH_DIR"

# Check if go.mod exists
if [ ! -f "go.mod" ]; then
    echo "❌ Error: go.mod not found in $KNIRVGRAPH_DIR"
    exit 1
fi

# Check if economics module exists
if [ ! -d "internal/economics" ]; then
    echo "❌ Error: economics directory not found. Please ensure the economics module has been implemented."
    exit 1
fi

# Install dependencies
echo "📦 Installing Go dependencies..."
go mod tidy

# Build the application
echo "🔨 Building KNIRVGRAPH..."
go build -o bin/knirvgraph ./cmd/node/main.go

# Check if build was successful
if [ ! -f "bin/knirvgraph" ]; then
    echo "❌ Error: Build failed. Please check for compilation errors."
    exit 1
fi

echo "✅ Build successful!"

# Create necessary directories
mkdir -p logs
mkdir -p "$KNIRVGRAPH_HOME"
mkdir -p "$KNIRVGRAPH_HOME/data"

# Function to handle cleanup on exit
cleanup() {
    echo ""
    echo "🛑 Shutting down KNIRVGRAPH..."
    if [ ! -z "$KNIRVGRAPH_PID" ]; then
        kill $KNIRVGRAPH_PID 2>/dev/null || true
        wait $KNIRVGRAPH_PID 2>/dev/null || true
    fi
    echo "✅ KNIRVGRAPH stopped"
    exit 0
}

# Set up signal handlers
trap cleanup SIGINT SIGTERM

# Test KNIRVORACLE connectivity
echo "🔗 Testing KNIRVORACLE connectivity..."
if curl -s --max-time 5 "$KNIRVORACLE_URL/ping" > /dev/null 2>&1; then
    echo "✅ KNIRVORACLE is accessible at $KNIRVORACLE_URL"
else
    echo "⚠️  Warning: KNIRVORACLE not accessible at $KNIRVORACLE_URL"
    echo "   Economics integration will be disabled until KNIRVORACLE is available"
fi

# Start KNIRVGRAPH
echo "🚀 Starting KNIRVGRAPH with economics integration..."
echo "📝 Logs will be written to logs/knirvgraph.log"
echo "📊 Economics endpoints will be available at http://localhost:$KNIRVGRAPH_PORT/economics/"
echo ""
echo "Press Ctrl+C to stop..."

# Start the application and capture PID
./bin/knirvgraph --home "$KNIRVGRAPH_HOME" --port "$KNIRVGRAPH_PORT" 2>&1 | tee logs/knirvgraph.log &
KNIRVGRAPH_PID=$!

# Wait for the application to start
sleep 3

# Check if the application is still running
if ! kill -0 $KNIRVGRAPH_PID 2>/dev/null; then
    echo "❌ Error: KNIRVGRAPH failed to start. Check logs/knirvgraph.log for details."
    exit 1
fi

echo "✅ KNIRVGRAPH started successfully (PID: $KNIRVGRAPH_PID)"

# Test service endpoints
echo "🧪 Testing KNIRVGRAPH endpoints..."
sleep 2

# Test health endpoint
if curl -s http://localhost:$KNIRVGRAPH_PORT/health > /dev/null 2>&1; then
    echo "✅ KNIRVGRAPH health endpoint is responding"
else
    echo "⚠️  KNIRVGRAPH health endpoint not yet available"
fi

# Test economics metrics endpoint
if curl -s http://localhost:$KNIRVGRAPH_PORT/economics/metrics > /dev/null 2>&1; then
    echo "✅ Economics metrics endpoint is responding"
else
    echo "⚠️  Economics metrics endpoint not yet available (normal if KNIRVORACLE is not connected)"
fi

# Test NRV system endpoint
if curl -s http://localhost:$KNIRVGRAPH_PORT/nrv/vectors > /dev/null 2>&1; then
    echo "✅ NRV system endpoint is responding"
else
    echo "⚠️  NRV system endpoint not yet available"
fi

echo ""
echo "🎉 KNIRVGRAPH with Economics Integration is running!"
echo ""
echo "📋 Available endpoints:"
echo "  - KNIRVGRAPH API: http://localhost:$KNIRVGRAPH_PORT"
echo "  - Health Check: http://localhost:$KNIRVGRAPH_PORT/health"
echo "  - Graph Operations: http://localhost:$KNIRVGRAPH_PORT/graph/"
echo "  - NRV System: http://localhost:$KNIRVGRAPH_PORT/nrv/"
echo "  - Economics Metrics: http://localhost:$KNIRVGRAPH_PORT/economics/metrics"
echo "  - Skill Confirmation: http://localhost:$KNIRVGRAPH_PORT/economics/skill/confirm"
echo "  - Reward Distribution: http://localhost:$KNIRVGRAPH_PORT/economics/rewards/distribute"
echo "  - Solution Proofs: http://localhost:$KNIRVGRAPH_PORT/economics/proof/solution"
echo ""
echo "📖 To test the economics integration:"
echo "  # Get economic metrics"
echo "  curl http://localhost:$KNIRVGRAPH_PORT/economics/metrics"
echo ""
echo "  # Create an error node"
echo "  curl -X POST http://localhost:$KNIRVGRAPH_PORT/nrv/errors \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"error_type\":\"test_error\",\"description\":\"Test error\",\"context\":{\"test\":true},\"severity\":1}'"
echo ""
echo "  # Create a skill node"
echo "  curl -X POST http://localhost:$KNIRVGRAPH_PORT/nrv/skills \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"skill_type\":\"test_skill\",\"capabilities\":[\"solve_test\"],\"requirements\":{\"test\":true}}'"
echo ""
echo "  # Confirm a skill for KNIRVCHAIN commitment"
echo "  curl -X POST http://localhost:$KNIRVGRAPH_PORT/economics/skill/confirm \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"skill_id\":\"skill_123\",\"nrv_id\":\"nrv_456\",\"creator_id\":\"creator_789\"}'"
echo ""
echo "  # Submit a solution proof"
echo "  curl -X POST http://localhost:$KNIRVGRAPH_PORT/economics/proof/solution \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"error_node_id\":\"error_123\",\"skill_node_id\":\"skill_456\",\"solver_id\":\"solver_789\",\"efficiency_score\":0.95,\"quality_score\":0.88}'"
echo ""
echo "🔍 Monitor logs: tail -f logs/knirvgraph.log"
echo ""

# Wait for the application to finish
wait $KNIRVGRAPH_PID
