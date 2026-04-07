#!/bin/bash
set -e

# KNIRV Native Orchestrator Build Script
# Builds all components and creates a single self-contained binary

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔨 KNIRV Native Orchestrator Build${NC}"
echo "===================================="
echo ""

# Ensure we're in KNIRVTESTNET directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Step 1: Build all components using existing build-all.sh
echo -e "${BLUE}Step 1: Building all KNIRV components...${NC}"
if [ ! -f "scripts/build-all.sh" ]; then
    echo -e "${RED}❌ build-all.sh not found!${NC}"
    exit 1
fi

./scripts/build-all.sh

echo ""
echo -e "${GREEN}✅ All components built successfully${NC}"
echo ""

# Step 2: Verify all required binaries exist
echo -e "${BLUE}Step 2: Verifying binaries...${NC}"
REQUIRED_BINS=("knirvoracle" "knirvchain" "knirvgraph" "knirvrouter")
OPTIONAL_BINS=("knirvserver")
MISSING_BINS=()

echo "Required binaries (public testnet):"
for bin in "${REQUIRED_BINS[@]}"; do
    if [ ! -f "bin/$bin" ]; then
        echo -e "${RED}❌ Missing binary: bin/$bin${NC}"
        MISSING_BINS+=("$bin")
    else
        BIN_SIZE=$(du -h "bin/$bin" | cut -f1)
        echo -e "${GREEN}✅ Found: bin/$bin${NC} (${BIN_SIZE})"
    fi
done

echo ""
echo "Optional binaries (private/corporate testing):"
for bin in "${OPTIONAL_BINS[@]}"; do
    if [ ! -f "bin/$bin" ]; then
        echo -e "${YELLOW}ℹ️  Optional: bin/$bin${NC} (not found - will skip)"
    else
        BIN_SIZE=$(du -h "bin/$bin" | cut -f1)
        echo -e "${GREEN}✅ Found: bin/$bin${NC} (${BIN_SIZE})"
    fi
done

if [ ${#MISSING_BINS[@]} -ne 0 ]; then
    echo -e "${RED}❌ Build failed: Missing required binaries: ${MISSING_BINS[*]}${NC}"
    exit 1
fi

echo ""

# Step 3: Verify .env file exists
echo -e "${BLUE}Step 3: Checking .env file...${NC}"
if [ ! -f ".env" ]; then
    echo -e "${YELLOW}⚠️  .env file not found. Creating placeholder...${NC}"
    cat > .env << 'EOF'
# KNIRV Testnet Environment Configuration
# This file is embedded in the orchestrator binary

TESTNET_MODE=true
LOCAL_NETWORK_MODE=true
DEPLOYMENT_ENV=testnet
EOF
    echo -e "${GREEN}✅ Created .env placeholder${NC}"
else
    echo -e "${GREEN}✅ .env file exists${NC}"
fi

echo ""

# Step 4: Build the orchestrator binary with all files embedded
echo -e "${BLUE}Step 4: Building orchestrator with embedded files...${NC}"
echo "This may take a moment as we're embedding all binaries..."
echo ""

# Check if Go is available
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go compiler not found!${NC}"
    echo "Please install Go 1.21+ to build the orchestrator"
    exit 1
fi

GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
echo "Go version: $GO_VERSION"

# Build with optimization flags
echo "Building knirvtestnet binary..."
go build -ldflags="-s -w" -o knirvtestnet main.go

if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Build failed!${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}✅ Orchestrator build complete!${NC}"
echo ""

# Step 5: Display build summary
BINARY_SIZE=$(du -h knirvtestnet | cut -f1)
BINARY_PATH=$(readlink -f knirvtestnet)

echo "=========================================="
echo -e "${GREEN}🎉 Build Summary${NC}"
echo "=========================================="
echo ""
echo "Binary Information:"
echo "  📦 File: ./knirvtestnet"
echo "  📏 Size: $BINARY_SIZE"
echo "  📍 Path: $BINARY_PATH"
echo ""
echo "Embedded Components (Public Testnet):"
for bin in "${REQUIRED_BINS[@]}"; do
    BIN_SIZE=$(du -h "bin/$bin" | cut -f1)
    echo "  ✓ $bin ($BIN_SIZE)"
done

# Show optional components if present
OPTIONAL_FOUND=false
for bin in "${OPTIONAL_BINS[@]}"; do
    if [ -f "bin/$bin" ]; then
        if [ "$OPTIONAL_FOUND" = false ]; then
            echo ""
            echo "Optional Components (Private Testing):"
            OPTIONAL_FOUND=true
        fi
        BIN_SIZE=$(du -h "bin/$bin" | cut -f1)
        echo "  ✓ $bin ($BIN_SIZE)"
    fi
done
echo ""
echo "Additional Embedded:"
echo "  ✓ Configuration files (config/)"
echo "  ✓ Data files (data/)"
echo "  ✓ Environment file (.env)"
echo ""
echo "=========================================="
echo -e "${YELLOW}Usage Examples:${NC}"
echo "=========================================="
echo ""
echo "Local Test:"
echo "  ./knirvtestnet ."
echo ""
echo "Remote Deployment:"
echo "  1. Upload: scp knirvtestnet ubuntu@\$IP:/tmp/"
echo "  2. Deploy: ssh ubuntu@\$IP 'sudo mv /tmp/knirvtestnet /usr/local/bin/'"
echo "  3. Start:  ssh ubuntu@\$IP 'sudo knirvtestnet /opt/knirv-testnet'"
echo ""
echo "Or use the simplified deployment:"
echo "  cd .. && make deploy-testnet-services"
echo ""
echo -e "${GREEN}✅ Ready for deployment!${NC}"
echo ""
