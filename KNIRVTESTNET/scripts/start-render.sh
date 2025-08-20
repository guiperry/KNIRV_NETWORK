#!/bin/bash
set -e

# Render-specific startup script
# This script starts only the web server components for Render deployment

echo "🚀 KNIRV TESTNET - RENDER DEPLOYMENT"
echo "===================================="

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

# Check if we're running on Render
if [ "$RENDER" != "true" ] && [ -z "$RENDER_SERVICE_ID" ]; then
    print_warning "This script is designed for Render deployment"
    print_warning "For local development, use: npm run dev"
    print_warning "For full testnet, use: npm run testnet:start"
fi

# Set environment variables for Render
export NODE_ENV=staging-testnet
export TESTNET_MODE=true

# Render.com sets PORT automatically - don't override it
if [ -z "$PORT" ]; then
    print_warning "PORT environment variable not set by Render"
    export PORT=10000
    print_warning "Defaulting to PORT=10000"
else
    print_success "Render provided PORT=$PORT"
fi

print_status "Starting KNIRV Testnet Web Server for Render..."
print_status "Environment: $NODE_ENV"
print_status "Port: $PORT"
print_status "Render Service ID: ${RENDER_SERVICE_ID:-'not set'}"
print_status "Render External URL: ${RENDER_EXTERNAL_URL:-'not set'}"

# Create necessary directories
mkdir -p logs data bin config

# Run smart initialization for Render deployment
print_status "Running smart initialization for Render deployment..."
if [ -f "scripts/smart-start.js" ]; then
    # Set environment for staging-testnet
    export NODE_ENV=staging-testnet

    # Run smart initialization (includes axios fix, health check, endpoint loading)
    if node scripts/smart-start.js --init-only; then
        print_success "Smart initialization completed for Render deployment"
    else
        print_error "Smart initialization failed"
        exit 1
    fi
else
    print_warning "smart-start.js not found, using basic initialization"

    # Load endpoints configuration
    print_status "Loading staging endpoint configuration..."
    node scripts/load-endpoints.js staging-testnet
fi

# Initialize testnet services for Render deployment
print_status "Initializing testnet services for Render deployment..."

# Check if we're on Render and need to start mock services
if [ "$RENDER" = "true" ] || [ -n "$RENDER_SERVICE_ID" ]; then
    print_status "Running on Render - starting mock services for testnet simulation..."

    # Start mock services in background for Render deployment
    print_status "Starting mock KNIRV services..."

    # Start mock services using Python scripts (lightweight for Render)
    if [ -f "scripts/mock-knirvoracle.py" ]; then
        python3 scripts/mock-knirvoracle.py &
        echo $! > data/mock-knirvoracle.pid
        print_success "Mock KNIRV-ORACLE started"
    fi

    if [ -f "scripts/mock-knirvchain.py" ]; then
        python3 scripts/mock-knirvchain.py &
        echo $! > data/mock-knirvchain.pid
        print_success "Mock KNIRVCHAIN started"
    fi

    if [ -f "scripts/mock-knirvgraph.py" ]; then
        python3 scripts/mock-knirvgraph.py &
        echo $! > data/mock-knirvgraph.pid
        print_success "Mock KNIRVGRAPH started"
    fi

    if [ -f "scripts/mock-knirvnexus.py" ]; then
        python3 scripts/mock-knirvnexus.py &
        echo $! > data/mock-knirvnexus.pid
        print_success "Mock KNIRV-NEXUS started"
    fi

    if [ -f "scripts/mock-knirvrouter.py" ]; then
        python3 scripts/mock-knirvrouter.py &
        echo $! > data/mock-knirvrouter.pid
        print_success "Mock KNIRV-ROUTER started"
    fi

    if [ -f "scripts/mock-knirvgateway.py" ]; then
        python3 scripts/mock-knirvgateway.py &
        echo $! > data/mock-knirvgateway.pid
        print_success "Mock KNIRV-GATEWAY started"
    fi

    print_success "Mock testnet services initialized for Render deployment"

    # Give mock services time to start
    print_status "Waiting for mock services to initialize..."
    sleep 5
else
    print_status "Local development mode - skipping mock service initialization"
fi

# Start the main web server
print_status "Starting KNIRVTESTNET Express server..."
exec node server/app.js
