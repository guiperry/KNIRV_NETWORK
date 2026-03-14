#!/bin/bash
set -e

echo "=========================================="
echo "KNIRV-TESTNET: Complete Build Process"
echo "=========================================="

# Create necessary directories
mkdir -p bin data logs

echo "Building KNIRV-TESTNET with real components..."

# Create data directories
mkdir -p data/{knirvoracle,knirvchain,knirvgraph,knirvnexus,knirvrouter,knirvgateway,ipfs}

# Build components with testnet features
echo "1/6 Building KNIRV-ORACLE with testnet mode..."
./scripts/build-knirvoracle.sh

echo "2/6 Building KNIRVCHAIN with testnet features..."
./scripts/build-knirvchain.sh

echo "3/6 Building KNIRVGRAPH with testnet mode..."
./scripts/build-knirvgraph.sh

echo "4/7 Building KNIRV-NEXUS with TEE simulation..."
./scripts/build-knirvnexus.sh

echo "5/7 Building KNIRV-NEXUS Frontend..."
./scripts/build-nexus-frontend.sh

echo "6/7 Building KNIRV-ROUTER with simplified connectivity..."
./scripts/build-knirvrouter.sh

echo "7/7 Building KNIRV-GATEWAY..."
./scripts/build-knirvgateway.sh

echo ""
echo "=========================================="
echo "Build Summary:"
echo "=========================================="
echo "✅ All components built successfully!"
echo "📁 Binaries available in: ./bin/"
echo "⚙️  Configuration files in: ./config/"
echo "💾 Data directories in: ./data/"
echo ""
echo "Next steps:"
echo "1. Start the testnet: ./scripts/start-testnet.sh"
echo "2. Check health: ./scripts/health-check.sh"
echo "3. Run tests: ./scripts/run-tests.sh"
