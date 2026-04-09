#!/bin/bash

# Cross-compilation script for KNIRVSHELL
# This script builds the CLI for multiple platforms

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Default values
VERSION=${VERSION:-"dev"}
BINARY_NAME=${BINARY_NAME:-"knirv"}
OUTPUT_DIR=${OUTPUT_DIR:-"./dist"}
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Platforms to build for
PLATFORMS=(
    "windows/amd64"
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64"
    "linux/arm64"
)

# Print header
echo -e "${YELLOW}=======================================${NC}"
echo -e "${YELLOW}Cross-compiling KNIRVSHELL v${VERSION}${NC}"
echo -e "${YELLOW}=======================================${NC}"
echo -e "Build Time: ${BUILD_TIME}"
echo -e "Git Commit: ${GIT_COMMIT}"
echo -e "Output Directory: ${OUTPUT_DIR}/${VERSION}"
echo -e "Platforms: ${PLATFORMS[*]}"
echo -e "${YELLOW}=======================================${NC}\n"

# Create output directory
mkdir -p "${OUTPUT_DIR}/${VERSION}"

# Build for each platform
for PLATFORM in "${PLATFORMS[@]}"; do
    # Split platform into OS and ARCH
    IFS='/' read -r OS ARCH <<< "${PLATFORM}"
    
    # Set output directory for this platform
    PLATFORM_DIR="${OUTPUT_DIR}/${VERSION}/${OS}_${ARCH}"
    mkdir -p "${PLATFORM_DIR}"
    
    # Set binary name with extension for Windows
    BINARY="${BINARY_NAME}"
    if [ "${OS}" = "windows" ]; then
        BINARY="${BINARY}.exe"
    fi
    
    # Set build flags
    LDFLAGS="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"
    
    echo -e "${YELLOW}Building for ${OS}/${ARCH}...${NC}"
    
    # Build the binary
    if [ "${OS}" = "darwin" ]; then
        # macOS cross-compilation (disable CGO to avoid linking host libraries)
        echo -e "  Building for darwin with CGO disabled (pure Go binary)"
        GOOS=${OS} GOARCH=${ARCH} CGO_ENABLED=0 go build -ldflags="${LDFLAGS}" -o "${PLATFORM_DIR}/${BINARY}" .
    else
        # Default build configuration for other platforms
        GOOS=${OS} GOARCH=${ARCH} CGO_ENABLED=0 go build -ldflags="${LDFLAGS}" -o "${PLATFORM_DIR}/${BINARY}" .
    fi
    
    echo -e "${GREEN}✓ Built ${BINARY} for ${OS}/${ARCH}${NC}"
    echo -e "  Output: ${PLATFORM_DIR}/${BINARY}\n"
done

echo -e "${GREEN}Cross-compilation complete!${NC}"
echo -e "Binaries are available in ${OUTPUT_DIR}/${VERSION}"