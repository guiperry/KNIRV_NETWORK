#!/bin/bash
set -e

echo "Building KNIRV-GATEWAY for testnet..."

cd ../KNIRVGATEWAY

# Install dependencies
echo "Installing Node.js dependencies..."
npm install

# Build the application with testnet configuration
echo "Building KNIRV-GATEWAY with testnet features..."
npm run build

# Create gateway data directory
mkdir -p ../KNIRVTESTNET/data/knirvgateway

# Copy testnet environment configuration
echo "Setting up testnet configuration..."
cp .env.testnet ../KNIRVTESTNET/data/knirvgateway/.env 2>/dev/null || echo "TESTNET_MODE=true" > ../KNIRVTESTNET/data/knirvgateway/.env

# Copy Netlify functions to testnet directory
mkdir -p ../KNIRVTESTNET/data/knirvgateway/netlify/functions
cp -r netlify/functions/* ../KNIRVTESTNET/data/knirvgateway/netlify/functions/ 2>/dev/null || true

# Copy package.json and install dependencies in testnet directory
cp package.json ../KNIRVTESTNET/data/knirvgateway/ 2>/dev/null || true
cp package-lock.json ../KNIRVTESTNET/data/knirvgateway/ 2>/dev/null || true

# Copy main files
cp index.html ../KNIRVTESTNET/data/knirvgateway/ 2>/dev/null || true
cp -r assets ../KNIRVTESTNET/data/knirvgateway/ 2>/dev/null || true

# Create testnet-specific configuration
cat > ../KNIRVTESTNET/data/knirvgateway/netlify.toml << 'EOF'
[build]
  command = "npm install"
  functions = "netlify/functions"
  publish = "."

[dev]
  command = "npm run dev"
  port = 8888
  targetPort = 8888

[[redirects]]
  from = "/gateway/*"
  to = "/.netlify/functions/gateway-sse"
  status = 200

[[redirects]]
  from = "/auth/*"
  to = "/.netlify/functions/gateway-sse"
  status = 200

[functions]
  node_bundler = "esbuild"

[context.testnet.environment]
  TESTNET_MODE = "true"
  NODE_ENV = "testnet"
  KNIRVROOT_URL = "http://localhost:1317"
  KNIRVCHAIN_URL = "http://localhost:8080"
  KNIRVGRAPH_URL = "http://localhost:8081"
  KNIRVNEXUS_URL = "http://localhost:8082"
  KNIRVROUTER_URL = "http://localhost:5001"
EOF

cd ../KNIRVTESTNET

echo "KNIRV-GATEWAY testnet build completed successfully!"
echo "Gateway will be available at: http://localhost:8888"
echo "Testnet endpoints:"
echo "  - Health: http://localhost:8888/gateway/health"
echo "  - Services: http://localhost:8888/gateway/services"
echo "  - Testnet Status: http://localhost:8888/gateway/testnet/status"
echo "  - Auth Tokens: http://localhost:8888/auth/testnet-tokens"
