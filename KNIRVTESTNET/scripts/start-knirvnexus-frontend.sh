#!/bin/bash

# Enhanced KNIRV-NEXUS Frontend Startup Script with Dependency Verification
# Handles missing dependencies and automatic rebuild when needed

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

print_status "Starting KNIRV-NEXUS Frontend..."

# Create necessary directories
mkdir -p logs data

# Check if NEXUS frontend build exists
if [ ! -d "./data/knirvnexus/portal" ]; then
    print_error "NEXUS frontend not found. Running build script..."
    if ! FORCE_REBUILD_NEXUS=true bash scripts/build-nexus-frontend.sh --force; then
        print_error "Failed to build NEXUS frontend"
        exit 1
    fi
fi

# Check if Node.js is available
if ! command -v node &> /dev/null; then
    print_error "Node.js is required but not installed."
    exit 1
fi

# Navigate to NEXUS frontend directory
cd data/knirvnexus/portal

# Check if package.json exists
if [ ! -f "package.json" ]; then
    print_error "NEXUS frontend package.json not found. Rebuilding..."
    cd ../../..
    if ! FORCE_REBUILD_NEXUS=true bash scripts/build-nexus-frontend.sh --force; then
        print_error "Failed to rebuild NEXUS frontend"
        exit 1
    fi
    cd data/knirvnexus/portal
fi

# Function to verify critical dependencies
verify_dependencies() {
    local missing_deps=()

    # Check for critical dependencies
    if [ ! -d "node_modules/socket.io" ]; then
        missing_deps+=("socket.io")
    fi

    if [ ! -d "node_modules/next" ]; then
        missing_deps+=("next")
    fi

    if [ ! -d "node_modules/react" ]; then
        missing_deps+=("react")
    fi

    if [ ! -f "node_modules/.bin/next" ]; then
        missing_deps+=("next-cli")
    fi

    if [ ${#missing_deps[@]} -gt 0 ]; then
        print_warning "Missing critical dependencies: ${missing_deps[*]}"
        return 1
    fi

    return 0
}

# Function to install dependencies with verification
install_dependencies() {
    local max_attempts=2
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        print_status "Installing/verifying NEXUS frontend dependencies (attempt $attempt/$max_attempts)..."

        # Clean install if this is a retry
        if [ $attempt -gt 1 ]; then
            print_status "Cleaning environment for retry..."
            rm -rf node_modules package-lock.json
            npm cache clean --force 2>/dev/null || true
        fi

        # Install dependencies
        if npm install --no-audit --no-fund; then
            print_success "Dependencies installed successfully"

            # Verify critical dependencies
            if verify_dependencies; then
                print_success "All critical dependencies verified"
                return 0
            else
                print_warning "Dependency verification failed"
            fi
        else
            print_error "npm install failed (attempt $attempt/$max_attempts)"
        fi

        attempt=$((attempt + 1))
    done

    print_error "Failed to install dependencies after $max_attempts attempts"
    return 1
}

# Check and install dependencies
if [ ! -d "node_modules" ]; then
    print_status "Node modules not found, installing dependencies..."
    if ! install_dependencies; then
        print_error "Dependency installation failed"
        exit 1
    fi
else
    print_status "Verifying existing dependencies..."
    if ! verify_dependencies; then
        print_warning "Dependencies incomplete or corrupted, reinstalling..."
        if ! install_dependencies; then
            print_error "Dependency installation failed"
            exit 1
        fi
    else
        print_success "Dependencies verified successfully"
    fi
fi

# Function to start the frontend server
start_frontend_server() {
    local max_attempts=2
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        print_status "Starting NEXUS frontend server (attempt $attempt/$max_attempts)..."

        # Set environment variables for testnet
        export NODE_ENV=production
        export PORT=8083
        export TESTNET_MODE=true

        # Start NEXUS frontend using custom server
        print_status "Starting NEXUS frontend on port 8083..."
        node server.js > ../../../logs/knirvnexus-frontend.log 2>&1 &

        NEXUS_FRONTEND_PID=$!
        echo $NEXUS_FRONTEND_PID > ../../../data/knirvnexus-frontend.pid

        # Wait a moment for startup
        sleep 5

        # Check if the process is still running
        if kill -0 $NEXUS_FRONTEND_PID 2>/dev/null; then
            print_success "NEXUS frontend started successfully!"
            print_success "Frontend available at: http://localhost:8083"
            return 0
        else
            print_error "NEXUS frontend failed to start (attempt $attempt/$max_attempts)"
            print_status "Error logs:"
            tail -10 ../../../logs/knirvnexus-frontend.log

            if [ $attempt -lt $max_attempts ]; then
                print_warning "Attempting to rebuild and retry..."
                cd ../../..

                # Force rebuild with --force flag
                if FORCE_REBUILD_NEXUS=true bash scripts/build-nexus-frontend.sh --force; then
                    print_success "Rebuild completed, retrying startup..."
                    cd data/knirvnexus/portal
                else
                    print_error "Rebuild failed"
                    return 1
                fi
            fi
        fi

        attempt=$((attempt + 1))
    done

    print_error "Failed to start NEXUS frontend after $max_attempts attempts"
    return 1
}

# Start the frontend server with retry logic
if ! start_frontend_server; then
    print_error "Failed to start KNIRV-NEXUS Frontend"
    exit 1
fi

# Return to original directory
cd ../../..

print_success "KNIRV-NEXUS Frontend is running successfully!"
print_status "🌐 Frontend URL: http://localhost:8083"
print_status "📁 Portal directory: ./data/knirvnexus/portal/"
print_status "📋 Logs: ./logs/knirvnexus-frontend.log"
print_status "🔧 PID file: ./data/knirvnexus-frontend.pid"
