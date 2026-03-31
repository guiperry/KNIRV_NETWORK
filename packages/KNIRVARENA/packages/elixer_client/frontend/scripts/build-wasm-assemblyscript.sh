#!/bin/bash

# KNIRV Controller AssemblyScript WASM Build Script
# Builds WASM modules using AssemblyScript for TypeScript-to-WASM compilation

set -e

echo "🚀 Building KNIRV Controller WASM modules with AssemblyScript..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
BUILD_DIR="$ROOT_DIR/build"
DIST_DIR="$ROOT_DIR/dist"

echo -e "${BLUE}📁 Working directories:${NC}"
echo "  Root: $ROOT_DIR"
echo "  Build: $BUILD_DIR"
echo "  Dist: $DIST_DIR"

# Check prerequisites
echo -e "\n${BLUE}🔍 Checking prerequisites...${NC}"

# Check if Node.js is installed
if ! command -v node &> /dev/null; then
    echo -e "${RED}❌ Node.js is not installed${NC}"
    exit 1
fi

# Check if AssemblyScript compiler is available
if ! command -v npx asc &> /dev/null; then
    echo -e "${RED}❌ AssemblyScript compiler not found${NC}"
    echo "Install with: npm install assemblyscript"
    exit 1
fi

echo -e "${GREEN}✅ Prerequisites check passed${NC}"

# Create build directory if it doesn't exist
mkdir -p "$BUILD_DIR"
mkdir -p "$DIST_DIR/wasm"

# Build AssemblyScript WASM modules
echo -e "\n${BLUE}🔨 Building AssemblyScript WASM modules...${NC}"

cd "$ROOT_DIR"

# Build release version
echo -e "${YELLOW}📦 Building release WASM...${NC}"
npx asc assembly/index.ts \
    --target release \
    --outFile build/knirv-controller.wasm \
    --textFile build/knirv-controller.wat \
    --bindings esm \
    --exportRuntime \
    --optimize \
    --converge \
    --noAssert

# Build debug version
echo -e "${YELLOW}📦 Building debug WASM...${NC}"
npx asc assembly/index.ts \
    --target debug \
    --outFile build/knirv-controller-debug.wasm \
    --textFile build/knirv-controller-debug.wat \
    --bindings esm \
    --exportRuntime \
    --sourceMap

# Copy built WASM files to dist and public
echo -e "\n${BLUE}📁 Creating WASM distribution...${NC}"

mkdir -p "$DIST_DIR/wasm"
mkdir -p "$ROOT_DIR/public/build"

if [ -f "build/knirv-controller.wasm" ]; then
    cp build/knirv-controller.wasm "$DIST_DIR/wasm/"
    cp build/knirv-controller.wasm "$ROOT_DIR/public/build/"
    echo -e "${GREEN}✅ Release WASM module copied to dist and public${NC}"
fi

if [ -f "build/knirv-controller-debug.wasm" ]; then
    cp build/knirv-controller-debug.wasm "$DIST_DIR/wasm/"
    cp build/knirv-controller-debug.wasm "$ROOT_DIR/public/build/"
    echo -e "${GREEN}✅ Debug WASM module copied to dist and public${NC}"
fi

# Create WASM module info
cat > "$DIST_DIR/wasm/assemblyscript-info.json" << EOF
{
  "name": "knirv-controller-assemblyscript",
  "version": "1.0.0",
  "description": "KNIRV Controller AssemblyScript WASM modules",
  "compiler": "AssemblyScript",
  "targets": ["release", "debug"],
  "features": [
    "agent-core",
    "model-inference", 
    "typescript-compilation",
    "memory-management"
  ],
  "buildDate": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "assemblyScriptVersion": "$(npx asc --version)",
  "nodeVersion": "$(node --version)"
}
EOF

echo -e "\n${GREEN}🎉 AssemblyScript WASM build completed successfully!${NC}"
echo -e "${BLUE}📊 Build summary:${NC}"
echo "  - AssemblyScript Agent-Core WASM: Ready"
echo "  - TypeScript-to-WASM compilation: Enabled"
echo "  - Memory management: Optimized"
echo "  - Output directory: $DIST_DIR/wasm"

# Display file sizes
echo -e "\n${BLUE}📏 WASM file sizes:${NC}"
if [ -f "$DIST_DIR/wasm/knirv-controller.wasm" ]; then
    ls -lh "$DIST_DIR/wasm/knirv-controller.wasm" | awk '{print "  Release WASM: " $5}'
fi
if [ -f "$DIST_DIR/wasm/knirv-controller-debug.wasm" ]; then
    ls -lh "$DIST_DIR/wasm/knirv-controller-debug.wasm" | awk '{print "  Debug WASM: " $5}'
fi

echo -e "\n${GREEN}✨ KNIRV Controller AssemblyScript WASM modules are ready!${NC}"
