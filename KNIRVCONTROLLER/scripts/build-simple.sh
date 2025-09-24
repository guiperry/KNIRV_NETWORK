#!/bin/bash

# Simplified build script for KNIRV Controller
# This script focuses on building the frontend without complex WASM dependencies

set -e

echo "🚀 Building KNIRV Controller (simplified version)..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

echo -e "${BLUE}📁 Working directory: $ROOT_DIR${NC}"

# Check prerequisites
echo -e "\n${BLUE}🔍 Checking prerequisites...${NC}"

# Check if Node.js is installed
if ! command -v node &> /dev/null; then
    echo -e "${RED}❌ Node.js is not installed${NC}"
    exit 1
fi

# Check if npm is installed
if ! command -v npm &> /dev/null; then
    echo -e "${RED}❌ npm is not installed${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Prerequisites check passed${NC}"

# Build the frontend only (skip complex WASM builds)
echo -e "\n${BLUE}🔨 Building frontend application...${NC}"

cd "$ROOT_DIR"

# TypeScript compilation check
echo -e "${YELLOW}📝 TypeScript compilation...${NC}"
npx tsc --noEmit

# Vite build
echo -e "${YELLOW}📦 Building with Vite...${NC}"
npx vite build

# Create basic WASM placeholder if needed
echo -e "${YELLOW}📝 Creating WASM placeholder...${NC}"
mkdir -p dist/wasm
mkdir -p public/build

# Create placeholder WASM files
cat > dist/wasm/placeholder-info.json << EOF
{
  "name": "knirv-controller-placeholder",
  "version": "1.0.0",
  "description": "Placeholder for WASM modules - full build requires additional dependencies",
  "status": "placeholder",
  "buildDate": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF

cp dist/wasm/placeholder-info.json public/build/

echo -e "\n${GREEN}🎉 Simplified build completed successfully!${NC}"
echo -e "${BLUE}📊 Build summary:${NC}"
echo "  - Frontend application: Built"
echo "  - TypeScript compilation: Passed"
echo "  - WASM modules: Placeholder (full build requires additional setup)"
echo "  - Output directory: dist/"

echo -e "\n${GREEN}✨ KNIRV Controller is ready for deployment!${NC}"