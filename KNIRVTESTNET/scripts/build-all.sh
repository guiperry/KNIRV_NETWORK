#!/bin/bash
set -e

# Parse command line arguments
FORCE_REBUILD=""
while [[ $# -gt 0 ]]; do
    case $1 in
        --force|--force-rebuild)
            FORCE_REBUILD="--force"
            shift
            ;;
        --help|-h)
            echo "Build All KNIRV Components"
            echo ""
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --force, --force-rebuild    Force rebuild all components"
            echo "  --help, -h                  Show this help message"
            echo ""
            echo "This script builds all KNIRV components with smart rebuild detection."
            echo "Components will only rebuild if source changes are detected."
            echo ""
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

echo "=========================================="
echo "KNIRV-TESTNET: Complete Build Process"
echo "=========================================="
if [ -n "$FORCE_REBUILD" ]; then
    echo "🔥 Force rebuild mode enabled"
fi

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

echo "4/7 Building KNIRV-NEXUS (optional - for private/corporate testing)..."
if [ -f "scripts/build-knirvnexus.sh" ]; then
    echo "  ℹ️  KNIRV-NEXUS is optional and only used for private testing"
    read -p "  Build KNIRV-NEXUS? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        ./scripts/build-knirvnexus.sh $FORCE_REBUILD
    else
        echo "  ⏭️  Skipping KNIRV-NEXUS build (public testnet mode)"
    fi
else
    echo "  ⚠️  build-knirvnexus.sh not found, skipping"
fi

echo "5/6 Building KNIRV-ROUTER with simplified connectivity..."
./scripts/build-knirvrouter.sh

echo "6/7 Building KNIRVCONTROLLER (optional)..."
./scripts/build-knirvcontroller.sh

echo "7/7 Building KNIRV-GATEWAY..."
./scripts/build-knirvgateway.sh

echo "8/8 Building KNIRVTESTNET orchestrator..."
go build -o knirvtestnet main.go

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
