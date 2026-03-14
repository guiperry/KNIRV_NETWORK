#!/bin/bash
set -e

echo "Starting KNIRV-GATEWAY testnet node..."

# Create necessary directories
mkdir -p logs data

# Check if testnet gateway directory exists
if [ ! -d "./data/testnet-gateway" ]; then
    echo "Error: Testnet Gateway not found. Please run build-knirvgateway.sh first."
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
echo "Starting Testnet Gateway..."
cd data/testnet-gateway

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
export KNIRVSERVER_URL=http://localhost:8084
export KNIRVSERVER_API_URL=http://localhost:8084/api
export KNIRVROUTER_URL=http://localhost:8086

# Start KNIRV-GATEWAY using npm start with specified port and memory limit (512MB for testnet)
echo "Starting KNIRV-GATEWAY on port 8888 with 512MB memory limit..."
NODE_OPTIONS="--max-old-space-size=512" npx netlify dev --port 8888 > ../../logs/knirvgateway.log 2>&1 &

GATEWAY_PID=$!
echo $GATEWAY_PID > ../../data/testnet-gateway.pid
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
echo "Resource optimizations:"
echo "  - Memory limit: 512MB (optimized for testnet)"
echo "  - Testnet mode with simplified operations"
echo "Log file: ./logs/knirvgateway.log"

# Wait a moment and check if process is still running
sleep 5
if ! kill -0 $(cat ./data/testnet-gateway.pid) 2>/dev/null; then
    echo "Error: Testnet Gateway failed to start. Check logs:"
    tail -20 ./logs/knirvgateway.log
    exit 1
fi

echo "Testnet Gateway is running successfully!"
