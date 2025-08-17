#!/bin/bash
set -e

echo "Starting KNIRV-GATEWAY testnet node..."

# Create necessary directories
mkdir -p logs data

# Check if testnet gateway directory exists
if [ ! -d "./data/knirvgateway" ]; then
    echo "Error: KNIRV-GATEWAY testnet version not found. Please run build-knirvgateway.sh first."
    exit 1
fi

# Check if Node.js is available
if ! command -v node &> /dev/null; then
    echo "Error: Node.js is required but not installed."
    exit 1
fi

# Check if npm is available
if ! command -v npm &> /dev/null; then
    echo "Error: npm is required but not installed."
    exit 1
fi

# Navigate to testnet gateway directory
echo "Starting KNIRV-GATEWAY testnet version..."
cd data/knirvgateway

# Install dependencies if needed
if [ ! -d "node_modules" ]; then
    echo "Installing dependencies..."
    npm install || {
        echo "npm install failed, running netlify-cli fix..."
        if [ -f "scripts/fix-netlify-cli.sh" ]; then
            ./scripts/fix-netlify-cli.sh --auto
            npm install
        else
            echo "Fix script not found, installation failed"
            exit 1
        fi
    }
fi

# Check if netlify-cli is working
if ! npx netlify --version >/dev/null 2>&1; then
    echo "netlify-cli issues detected, running fix..."
    if [ -f "scripts/fix-netlify-cli.sh" ]; then
        ./scripts/fix-netlify-cli.sh --auto
    else
        echo "Fix script not found, netlify-cli may not work properly"
    fi
fi

# Set testnet environment variables with correct service URLs
export TESTNET_MODE=true
export NODE_ENV=testnet
export KNIRVORACLE_URL=http://localhost:1317
export KNIRVCHAIN_URL=http://localhost:8090
export KNIRVGRAPH_URL=http://localhost:8082
export KNIRVNEXUS_DVE_URL=http://localhost:8084
export KNIRVNEXUS_VAL_URL=http://localhost:8085
export KNIRVROUTER_URL=http://localhost:8086

# Start KNIRV-GATEWAY using npm start with specified port
echo "Starting KNIRV-GATEWAY on port 8888..."
npx netlify dev --port 8888 > ../../logs/knirvgateway.log 2>&1 &

GATEWAY_PID=$!
echo $GATEWAY_PID > ../../data/knirvgateway.pid
cd ../..

echo "KNIRV-GATEWAY testnet started with PID $(cat ./data/knirvgateway.pid)"
echo "Gateway endpoint: http://localhost:8888"
echo "Testnet endpoints:"
echo "  - Health: http://localhost:8888/gateway/health"
echo "  - Services: http://localhost:8888/gateway/services"
echo "  - Testnet Status: http://localhost:8888/gateway/testnet/status"
echo "  - Auth Tokens: http://localhost:8888/auth/testnet-tokens"
echo "  - Auth Validate: http://localhost:8888/auth/validate"
echo "Testnet features:"
echo "  - Static service discovery enabled"
echo "  - Simplified authentication enabled"
echo "  - Local service proxying enabled"
echo "  - SSE support enabled"
echo "Log file: ./logs/knirvgateway.log"

# Wait a moment and check if process is still running
sleep 5
if ! kill -0 $(cat ./data/knirvgateway.pid) 2>/dev/null; then
    echo "Error: KNIRV-GATEWAY failed to start. Check logs:"
    tail -20 ./logs/knirvgateway.log
    exit 1
fi

echo "KNIRV-GATEWAY testnet is running successfully!"
