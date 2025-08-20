#!/bin/bash

# Enhanced KNIRV-NEXUS Frontend Build Script with Robust Error Handling
# Handles EIO errors, build failures, and provides automatic recovery

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

# Function to clean build environment
clean_build_environment() {
    print_status "Cleaning build environment..."

    # Remove node_modules and package-lock.json
    if [ -d "node_modules" ]; then
        print_status "Removing node_modules..."
        rm -rf node_modules
    fi

    if [ -f "package-lock.json" ]; then
        print_status "Removing package-lock.json..."
        rm -f package-lock.json
    fi

    # Clear npm cache
    print_status "Clearing npm cache..."
    npm cache clean --force 2>/dev/null || true

    print_success "Build environment cleaned"
}

# Function to install dependencies with retry logic
install_dependencies() {
    local max_attempts=3
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        print_status "Installing dependencies (attempt $attempt/$max_attempts)..."

        # Use timeout to prevent hanging
        if timeout 300 npm install --no-audit --no-fund; then
            print_success "Dependencies installed successfully"
            return 0
        else
            print_error "Dependency installation failed (attempt $attempt/$max_attempts)"

            if [ $attempt -lt $max_attempts ]; then
                print_warning "Cleaning environment and retrying..."
                clean_build_environment
                sleep 5
            fi

            attempt=$((attempt + 1))
        fi
    done

    print_error "Failed to install dependencies after $max_attempts attempts"
    return 1
}

# Function to build frontend with retry logic
build_frontend() {
    local max_attempts=2
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        print_status "Building NEXUS frontend (attempt $attempt/$max_attempts)..."

        # Set environment variables to handle terminal issues
        export CI=true
        export FORCE_COLOR=0

        # Use timeout and handle EIO errors
        if timeout 600 npm run build 2>&1; then
            print_success "Frontend build completed successfully"
            return 0
        else
            local exit_code=$?
            print_error "Frontend build failed with exit code $exit_code (attempt $attempt/$max_attempts)"

            # Check for specific error patterns
            if [ $attempt -lt $max_attempts ]; then
                print_warning "Attempting recovery..."

                # Clean and retry for EIO or other terminal-related errors
                clean_build_environment

                if ! install_dependencies; then
                    print_error "Recovery failed - dependency installation failed"
                    return 1
                fi

                sleep 3
            fi

            attempt=$((attempt + 1))
        fi
    done

    print_error "Frontend build failed after $max_attempts attempts"
    return 1
}

# Function to create build log with timestamp
create_build_log() {
    local build_log_path="build.log"
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local git_hash=""

    # Get git hash if available
    if command -v git >/dev/null 2>&1 && [ -d ".git" ]; then
        git_hash=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    else
        git_hash="unknown"
    fi

    # Create build log
    cat > "$build_log_path" << EOF
{
  "buildTimestamp": "$timestamp",
  "gitHash": "$git_hash",
  "buildStatus": "success",
  "nodeVersion": "$(node --version 2>/dev/null || echo 'unknown')",
  "npmVersion": "$(npm --version 2>/dev/null || echo 'unknown')",
  "buildDuration": "$build_duration",
  "buildHost": "$(hostname 2>/dev/null || echo 'unknown')"
}
EOF

    print_success "Build log created: $build_log_path"
}

# Main execution starts here
print_status "Building KNIRV-NEXUS Frontend for testnet..."

# Check for force rebuild flag
FORCE_REBUILD=false
if [ "$1" = "--force" ] || [ "$FORCE_REBUILD_NEXUS" = "true" ]; then
    FORCE_REBUILD=true
    print_status "🔄 Force rebuild requested, skipping optimization check..."
fi

# Check if build can be optimized (unless force rebuild is requested)
if [ "$FORCE_REBUILD" = "false" ]; then
    print_status "Checking if build can be optimized..."

    # Get the absolute path to the script directory
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

    # Run the build skip check directly
    if node -e "
        try {
            const { canSkipBuild } = require('$SCRIPT_DIR/check-nexus-health.js');
            const canSkip = canSkipBuild();
            if (canSkip) {
                console.log('✅ Build optimization: Frontend is up-to-date, skipping rebuild!');
                console.log('📁 Frontend available at: ./data/knirvnexus/portal/');
                console.log('🌐 Will be served on port 8083 (nexus gui_port)');
                console.log('⚡ Build skipped - no changes detected');
                process.exit(0);
            } else {
                console.log('🔄 Build optimization: Changes detected, proceeding with build...');
                process.exit(1);
            }
        } catch (error) {
            console.log('⚠️  Build optimization check failed:', error.message);
            console.log('Proceeding with build...');
            process.exit(1);
        }
    " 2>/dev/null; then
        # This should not be reached if canSkipBuild returns true
        print_success "✅ Build optimization: Frontend is up-to-date, skipping rebuild!"
        print_status "📁 Frontend available at: ./data/knirvnexus/portal/"
        print_status "🌐 Will be served on port 8083 (nexus gui_port)"
        print_status "⚡ Build skipped - no changes detected"
        exit 0
    else
        print_status "🔄 Build optimization: Changes detected, proceeding with build..."
    fi
else
    print_status "🔄 Force rebuild mode - proceeding with full build..."
fi

# Record build start time
build_start=$(date +%s)

# Create necessary directories
mkdir -p data/knirvnexus/portal

# Check if KNIRVNEXUS directory exists
if [ ! -d "KNIRVNEXUS" ]; then
    print_error "KNIRVNEXUS directory not found in current directory."
    print_status "Looking for KNIRVNEXUS in parent directory..."

    if [ ! -d "../KNIRVNEXUS" ]; then
        print_error "KNIRVNEXUS directory not found. Please ensure KNIRVNEXUS is available."
        exit 1
    else
        NEXUS_SOURCE="../KNIRVNEXUS"
    fi
else
    NEXUS_SOURCE="KNIRVNEXUS"
fi

print_success "Found KNIRVNEXUS source at: $NEXUS_SOURCE"

print_status "Copying NEXUS source files to portal directory..."
# Copy source files to portal directory with better error handling
if ! cp -r "$NEXUS_SOURCE"/* data/knirvnexus/portal/ 2>/dev/null; then
    print_warning "Some files may not have been copied (this is usually normal)"
fi

# Navigate to portal directory
cd data/knirvnexus/portal

# Clean environment first
clean_build_environment

# Install dependencies with retry logic
if ! install_dependencies; then
    print_error "Failed to install dependencies"
    exit 1
fi

# Build frontend with retry logic
if ! build_frontend; then
    print_error "Failed to build frontend"
    exit 1
fi

# Calculate build duration
build_end=$(date +%s)
build_duration=$((build_end - build_start))

# Create build log
create_build_log

print_success "✅ NEXUS frontend build completed successfully!"
print_status "📁 Frontend available at: ./data/knirvnexus/portal/"
print_status "🌐 Will be served on port 8083 (nexus gui_port)"
print_status "📦 Dependencies installed in portal directory"
print_status "⏱️  Build completed in ${build_duration} seconds"
