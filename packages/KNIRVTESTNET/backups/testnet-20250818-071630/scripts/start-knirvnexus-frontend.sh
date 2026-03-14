#!/bin/bash
set -e

echo "Starting KNIRV-NEXUS Frontend..."

# Create necessary directories
mkdir -p logs data

# Check if NEXUS frontend build exists
if [ ! -d "./data/knirvnexus/portal" ]; then
    echo "Error: NEXUS frontend not found. Please run build-nexus-frontend.sh first."
    exit 1
fi

# Check if Node.js is available
if ! command -v node &> /dev/null; then
    echo "Error: Node.js is required but not installed."
    exit 1
fi

# Navigate to NEXUS frontend directory
cd data/knirvnexus/portal

# Check if package.json exists
if [ ! -f "package.json" ]; then
    echo "Error: NEXUS frontend package.json not found."
    exit 1
fi

# Install dependencies if needed
if [ ! -d "node_modules" ]; then
    echo "Installing NEXUS frontend dependencies..."
    npm install
fi

# Set environment variables for testnet
export NODE_ENV=production
export PORT=8083
export TESTNET_MODE=true

# Start NEXUS frontend using custom server
echo "Starting NEXUS frontend on port 8083..."
node server.js > ../../../logs/knirvnexus-frontend.log 2>&1 &

NEXUS_FRONTEND_PID=$!
echo $NEXUS_FRONTEND_PID > ../../../data/knirvnexus-frontend.pid
cd ../../..

# Wait a moment for startup
sleep 3

# Check if the process is still running
if ! kill -0 $(cat ./data/knirvnexus-frontend.pid) 2>/dev/null; then
    echo "Error: NEXUS frontend failed to start. Check logs:"
    tail -20 ./logs/knirvnexus-frontend.log
    exit 1
fi

echo "NEXUS frontend is running successfully on port 8083!"
echo "Frontend available at: http://localhost:8083"
