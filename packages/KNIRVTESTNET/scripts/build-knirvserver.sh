#!/bin/bash
set -e

# KNIRV-SERVER Build Script for Testnet
#
# This script supports two modes:
# 1. BUILD FROM SOURCE: Requires Go, Node.js, make (local development)
# 2. USE PREBUILT BINARY: Uses existing binary (Render deployment)
#
# Environment variables:
#   USE_PREBUILT=true    - Force use of pre-built binary
#   FORCE_REBUILD=true   - Force rebuild even if binary is up to date
#   RENDER=true          - Automatically detected on Render
#   RENDER_SERVICE_ID    - Automatically set on Render
#
# Usage:
#   ./build-knirvserver.sh                    # Auto-detect mode (smart rebuild)
#   ./build-knirvserver.sh --force            # Force rebuild
#   ./build-knirvserver.sh --prebuilt         # Force use prebuilt
#   USE_PREBUILT=true ./build-knirvserver.sh  # Force prebuilt mode (env var)

echo "🚀 Building KNIRV-SERVER unified binary for testnet using new Makefile architecture..."

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --force|--force-rebuild)
            export FORCE_REBUILD=true
            shift
            ;;
        --prebuilt|--use-prebuilt)
            export USE_PREBUILT=true
            shift
            ;;
        --status|--info)
            echo ""
            echo "KNIRV-SERVER Build Status"
            echo ""
            if [ -f "../packages/KNIRVSERVER/build.log" ]; then
                echo "Build Log Information:"
                grep -E '"(buildStatus|buildTimestamp|gitHash|buildDuration)"' "../packages/KNIRVSERVER/build.log" 2>/dev/null | sed 's/^/  /' || echo "  No build information found"
            else
                echo "  No build.log found"
            fi
            echo ""
            if [ -f "../packages/KNIRVSERVER/dist/knirv-nexus" ]; then
                binary_size=$(du -h "../packages/KNIRVSERVER/dist/knirv-nexus" | cut -f1)
                binary_date=$(stat -c %y "../packages/KNIRVSERVER/dist/knirv-nexus" 2>/dev/null | cut -d' ' -f1,2)
                echo "Binary Information:"
                echo "  File: ../packages/KNIRVSERVER/dist/knirv-nexus"
                echo "  Size: $binary_size"
                echo "  Date: $binary_date"
            else
                echo "Binary: Not found"
            fi
            echo ""
            exit 0
            ;;
        --help|-h)
            echo ""
            echo "KNIRV-SERVER Build Script"
            echo ""
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --force, --force-rebuild    Force rebuild even if binary is up to date"
            echo "  --prebuilt, --use-prebuilt  Force use of existing pre-built binary"
            echo "  --status, --info            Show current build status and exit"
            echo "  --help, -h                  Show this help message"
            echo ""
            echo "Environment Variables:"
            echo "  FORCE_REBUILD=true          Force rebuild"
            echo "  USE_PREBUILT=true           Force use of pre-built binary"
            echo "  RENDER=true                 Render deployment mode"
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

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Exit on any error
set -e

# Set script directory variables
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTNET_ROOT="$(dirname "$SCRIPT_DIR")"
PROJECT_ROOT="$(dirname "$TESTNET_ROOT")"

# Check if KNIRVSERVER directory exists
if [ ! -d "../packages/KNIRVSERVER" ]; then
    print_error "KNIRVSERVER directory not found"
    exit 1
fi

print_status "Changing to KNIRVSERVER directory..."
cd ../packages/KNIRVSERVER

# Check if we're on Render or should use pre-built binary
# Check if we need to rebuild by examining build.log and binary status
check_build_needed() {
    local build_log_file="build.log"
    local binary_file="dist/knirv-nexus"
    local testnet_binary_file="../../KNIRVTESTNET/bin/knirvserver"

    # If no build log exists, we need to build
    if [ ! -f "$build_log_file" ]; then
        print_status "No build.log found - build required"
        return 0
    fi

    # If no binary exists, we need to build
    if [ ! -f "$binary_file" ]; then
        print_status "No unified binary found - build required"
        return 0
    fi

    # If testnet binary doesn't exist, we need to copy at least
    if [ ! -f "$testnet_binary_file" ]; then
        print_status "Testnet binary missing - copy required"
        return 0
    fi

    # Check if source files are newer than the binary
    local binary_time=$(stat -c %Y "$binary_file" 2>/dev/null || echo 0)
    local source_dirs=("src" "backend" "main.go" "package.json" "Makefile")

    for source_dir in "${source_dirs[@]}"; do
        if [ -e "$source_dir" ]; then
            local source_time=$(find "$source_dir" -type f -newer "$binary_file" 2>/dev/null | head -1)
            if [ -n "$source_time" ]; then
                print_status "Source files newer than binary - rebuild required"
                return 0
            fi
        fi
    done

    # Check git status for changes
    local current_git_hash=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    local build_git_hash=$(grep '"gitHash"' "$build_log_file" 2>/dev/null | sed 's/.*"gitHash": *"\([^"]*\)".*/\1/' || echo "unknown")

    if [ "$current_git_hash" != "$build_git_hash" ] && [ "$current_git_hash" != "unknown" ]; then
        print_status "Git hash changed ($build_git_hash → $current_git_hash) - rebuild required"
        return 0
    fi

    # Check build status in log
    local build_status=$(grep '"buildStatus"' "$build_log_file" 2>/dev/null | sed 's/.*"buildStatus": *"\([^"]*\)".*/\1/' || echo "unknown")
    if [ "$build_status" != "success" ]; then
        print_status "Previous build was not successful - rebuild required"
        return 0
    fi

    # Check if binary is significantly old (more than 1 day)
    local current_time=$(date +%s)
    local binary_age=$((current_time - binary_time))
    local one_day=$((24 * 60 * 60))

    if [ $binary_age -gt $one_day ]; then
        print_status "Binary is more than 1 day old - rebuild recommended"
        return 0
    fi

    # All checks passed - no rebuild needed
    local build_time=$(grep '"buildTimestamp"' "$build_log_file" 2>/dev/null | sed 's/.*"buildTimestamp": *"\([^"]*\)".*/\1/' || echo "unknown")
    print_success "Existing build is up to date (built: $build_time)"
    return 1
}

if [ "$RENDER" = "true" ] || [ -n "$RENDER_SERVICE_ID" ] || [ "$USE_PREBUILT" = "true" ]; then
    print_status "Render/prebuilt mode detected - will use existing binary"
    USE_PREBUILT_BINARY=true
else
    print_status "Local development mode - checking if build is needed"

    # Check if we need to rebuild (unless forced)
    if [ "$FORCE_REBUILD" = "true" ]; then
        print_status "Force rebuild requested - will build from source"
        USE_PREBUILT_BINARY=false
    elif check_build_needed; then
        print_status "Build required - will build from source"
        USE_PREBUILT_BINARY=false

        # Check for required build tools only when building
        print_status "Checking build prerequisites..."
        if ! command -v make >/dev/null 2>&1; then
            print_error "make is required but not installed"
            exit 1
        fi

        if ! command -v go >/dev/null 2>&1; then
            print_error "Go is required but not installed"
            exit 1
        fi

        if ! command -v node >/dev/null 2>&1; then
            print_error "Node.js is required but not installed"
            exit 1
        fi

        if ! command -v npm >/dev/null 2>&1; then
            print_error "npm is required but not installed"
            exit 1
        fi

        print_success "All build prerequisites found"
    else
        print_status "Using existing build - no rebuild needed"
        USE_PREBUILT_BINARY=true
    fi
fi

if [ "$USE_PREBUILT_BINARY" = "true" ]; then
    # Use pre-built binary mode (for Render deployment)
    print_status "=== PREBUILT BINARY MODE ==="
    print_status "Looking for existing knirv-nexus binary..."

    # Check for pre-built binary in various locations
    BINARY_LOCATIONS=(
        "dist/knirv-nexus"
        "knirv-nexus"
        "bin/knirv-nexus"
        "../packages/KNIRVSERVER/dist/knirv-nexus"
        "../packages/KNIRVSERVER/knirv-nexus"
    )

    FOUND_BINARY=""
    for location in "${BINARY_LOCATIONS[@]}"; do
        if [ -f "$location" ]; then
            FOUND_BINARY="$location"
            print_success "Found pre-built binary at: $location"
            break
        fi
    done

    if [ -z "$FOUND_BINARY" ]; then
        print_error "No pre-built knirv-nexus binary found!"
        print_status "Searched locations:"
        for location in "${BINARY_LOCATIONS[@]}"; do
            print_status "  - $location"
        done
        print_status ""
        print_status "Please build the binary locally first:"
        print_status "  cd ../packages/KNIRVSERVER && make binary"
        exit 1
    fi

    # Create dist directory if it doesn't exist
    mkdir -p dist

    # Copy the binary to the expected location if needed
    if [ "$FOUND_BINARY" != "dist/knirv-nexus" ]; then
        print_status "Copying binary to dist/knirv-nexus..."
        cp "$FOUND_BINARY" dist/knirv-nexus
    fi

    print_success "Using pre-built binary ($(du -h dist/knirv-nexus | cut -f1))"

else
    # Build from source mode (for local development)
    print_status "=== BUILD FROM SOURCE MODE ==="

    # Set build variables for testnet
    export VERSION="testnet-$(date +%Y%m%d-%H%M%S)"
    export BUILD_TIME="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
    export GIT_COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")"

    print_status "Build configuration:"
    echo "  Version: $VERSION"
    echo "  Build Time: $BUILD_TIME"
    echo "  Git Commit: $GIT_COMMIT"
    echo ""

    # Clean previous builds
    print_status "Cleaning previous builds..."
    make clean || print_warning "Clean failed (may be first build)"

    # Use the new Makefile-based build system
    print_status "Building unified binary using Makefile..."
    print_status "This will: install deps → build frontend → build backend → create unified binary"

    # Run the unified build process
    if make binary; then
        print_success "Unified binary build completed successfully"
    else
        print_error "Unified binary build failed"
        exit 1
    fi

    # Verify the unified binary was created
    if [ -f "dist/knirv-nexus" ]; then
        print_success "Unified binary created: dist/knirv-nexus ($(du -h dist/knirv-nexus | cut -f1))"
    else
        print_error "Unified binary not found at dist/knirv-nexus"
        exit 1
    fi
fi

# Copy unified binary to testnet bin directory
print_status "Copying unified binary to testnet..."
mkdir -p ../../KNIRVTESTNET/bin
cp dist/knirv-nexus ../../KNIRVTESTNET/bin/knirvserver

cd ../../KNIRVTESTNET

# Verify the binary was copied successfully
if [ ! -f "bin/knirvserver" ]; then
    print_error "Failed to copy unified binary to testnet"
    exit 1
fi

print_success "Built and copied KNIRV-SERVER unified binary"

# Create testnet data directories
print_status "Setting up testnet data directories..."
mkdir -p data/knirvserver
mkdir -p logs
mkdir -p config

# Copy testnet configuration from KNIRVSERVER if available
print_status "Setting up testnet configuration..."
if [ -f "../packages/KNIRVSERVER/config/nexus-testnet.yaml" ]; then
    print_status "Copying testnet config from KNIRVSERVER..."
    cp ../packages/KNIRVSERVER/config/nexus-testnet.yaml config/nexus-testnet.yaml
    print_success "Testnet configuration copied"
else
    print_warning "No testnet config found, creating default configuration..."
    cat > config/nexus-testnet.yaml << 'EOF'
# KNIRV-SERVER Testnet Configuration
# Updated for new unified architecture (no Socket.io)

server:
  host: "0.0.0.0"
  port: 8084
  environment: "testnet"
  log_level: "info"

# Frontend configuration (embedded in binary)
frontend:
  embedded: true
  static_path: "/static"
  api_prefix: "/api/v1"

# Backend configuration (unified service)
backend:
  api_port: 8084
  health_check_path: "/api/v1/health"
  cors_enabled: true
  cors_origins: ["*"]

# Testnet-specific settings
testnet:
  enabled: true
  simulation_mode: true
  mock_validation: true
  simplified_proofs: true
  timeout_ms: 5000

# Database configuration
database:
  type: "sqlite"
  path: "./data/knirvserver/testnet.db"
  clean_on_start: true
  auto_migrate: true

# DVE (Distributed Virtual Environment) settings
dve:
  enabled: true
  max_environments: 10
  default_timeout: 300
  cleanup_interval: 3600

# TEE (Trusted Execution Environment) settings
tee:
  simulation_mode: true
  mock_validation: true
  simplified_validation: true

# Logging configuration
logging:
  level: "info"
  format: "json"
  output: "./logs/knirvserver.log"
  max_size: 100
  max_backups: 5
  max_age: 30
EOF
    print_success "Default testnet configuration created"
fi

# Create a simplified config for backward compatibility
print_status "Creating backward compatibility configuration..."
cat > data/knirvserver/config.yaml << 'EOF'
# Legacy configuration format for backward compatibility
testnet:
  enabled: true
  simulation_mode: true

nexus:
  port: 8084
  api_port: 8084
  log_level: "info"

database:
  clean_on_start: true
  in_memory: false
  path: "./data/knirvserver/testnet.db"

dve:
  enabled: true
  max_environments: 10

tee:
  simulation_mode: true
  mock_validation: true
EOF

# Copy configuration to config directory as well for backward compatibility
cp data/knirvserver/config.yaml config/knirvserver-testnet-config.yaml

print_status "Setting executable permissions..."
chmod +x bin/knirvserver

echo ""
print_success "🎉 KNIRV-SERVER testnet build completed successfully!"
echo ""
print_status "📋 Build Summary:"
if [ "$USE_PREBUILT_BINARY" = "true" ]; then
    print_success "  ✅ Used pre-built binary (no compilation needed)"
    print_success "  ✅ Skipped Go/Node.js build dependencies"
else
    print_success "  ✅ Dependencies installed via Makefile"
    print_success "  ✅ Frontend built with Next.js (no Socket.io)"
    print_success "  ✅ Backend built as unified Go service"
    print_success "  ✅ Unified binary created with embedded components"
fi
print_success "  ✅ Binary copied to testnet ($(du -h bin/knirvserver | cut -f1))"
print_success "  ✅ Testnet configuration created"
print_success "  ✅ Backward compatibility configs created"
echo ""
print_status "🚀 Ready to start with:"
print_status "  ./bin/knirvserver --config config/nexus-testnet.yaml"
print_status "  OR"
print_status "  ./scripts/start-knirvserver.sh (if available)"
echo ""
print_status "🔍 Binary info:"
print_status "  Location: $(pwd)/bin/knirvserver"
print_status "  Size: $(du -h bin/knirvserver | cut -f1)"
print_status "  Permissions: $(ls -la bin/knirvserver | cut -d' ' -f1)"
