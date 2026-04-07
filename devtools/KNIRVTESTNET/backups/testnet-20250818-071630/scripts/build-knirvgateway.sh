#!/bin/bash
set -e

echo "Building KNIRV-GATEWAY for testnet..."

# Ensure testnet gateway directory exists
mkdir -p data/testnet-gateway

# Check if we need to copy from production KNIRVGATEWAY first
if [ ! -f "data/testnet-gateway/package.json" ]; then
    echo "Setting up initial KNIRV-GATEWAY testnet configuration..."

    # Copy essential files from production KNIRVGATEWAY
    if [ -d "../KNIRVGATEWAY" ]; then
        cp ../KNIRVGATEWAY/package.json data/testnet-gateway/ 2>/dev/null || true
        cp ../KNIRVGATEWAY/package-lock.json data/testnet-gateway/ 2>/dev/null || true
        cp -r ../KNIRVGATEWAY/assets data/testnet-gateway/ 2>/dev/null || true
        cp -r ../KNIRVGATEWAY/netlify data/testnet-gateway/ 2>/dev/null || true
        cp -r ../KNIRVGATEWAY/graphchain-explorer data/testnet-gateway/ 2>/dev/null || true
        cp -r ../KNIRVGATEWAY/developer-portal data/testnet-gateway/ 2>/dev/null || true
        # Note: index.html is already copied from KNIRVTESTNET/index.html
    fi
fi

cd data/testnet-gateway

# Install dependencies and run fix script if needed
echo "Installing Node.js dependencies..."
npm install || {
    echo "npm install failed, running netlify-cli fix..."
    if [ -f "scripts/fix-netlify-cli.sh" ]; then
        ./scripts/fix-netlify-cli.sh --auto
    else
        echo "Fix script not found, continuing..."
    fi
    npm install
}

# Build the application with testnet configuration
echo "Building KNIRV-GATEWAY with testnet features..."
npm run build || {
    echo "Build failed, running netlify-cli fix..."
    if [ -f "scripts/fix-netlify-cli.sh" ]; then
        ./scripts/fix-netlify-cli.sh --auto
        npm run build
    else
        echo "Fix script not found, build failed"
        exit 1
    fi
}

# Return to testnet directory
cd ../../

echo "KNIRV-GATEWAY testnet build completed successfully!"
echo "Gateway will be available at: http://localhost:8888"
echo "Testnet endpoints:"
echo "  - Health: http://localhost:8888/gateway/health"
echo "  - Services: http://localhost:8888/gateway/services"
echo "  - Testnet Status: http://localhost:8888/gateway/testnet/status"
echo "  - Auth Tokens: http://localhost:8888/auth/testnet-tokens"
