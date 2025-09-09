#!/bin/bash
set -e

# KNIRV Testnet Docker Deployment Script for Render
# This script handles both Render cloud deployment and local Docker testing

echo "🐳 KNIRV TESTNET - DOCKER DEPLOYMENT"
echo "===================================="
echo "Startup initiated at: $(date)"
echo "Working directory: $(pwd)"
echo "User: $(whoami)"
echo "Process ID: $$"
echo ""

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO $(date +%H:%M:%S)]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS $(date +%H:%M:%S)]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING $(date +%H:%M:%S)]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR $(date +%H:%M:%S)]${NC} $1"
}

# Error handler
handle_error() {
    local exit_code=$?
    local line_number=$1
    print_error "Startup failed at line $line_number with exit code $exit_code"
    print_error "Last command: $BASH_COMMAND"
    print_error "Environment variables:"
    env | grep -E "(RENDER|NODE|NPM|PORT|KNIRV)" | sort
    exit $exit_code
}

# Set up error trapping
trap 'handle_error $LINENO' ERR

# CRITICAL CHECK: If we're in a Docker container, this script should NOT be running
if [ -f /.dockerenv ]; then
    print_error "🚨 CRITICAL ERROR: npm start should NOT run in Docker containers!"
    print_error ""
    print_error "This indicates a configuration problem:"
    print_error "1. Render is running npm start inside a Docker container"
    print_error "2. For Docker services, use the Docker command directly"
    print_error "3. npm start should only run for non-Docker services"
    print_error ""
    print_error "SOLUTION FOR TESTNET GATEWAY:"
    print_error "1. Go to Render dashboard → Service Settings"
    print_error "2. Change Start Command from 'npm start' to:"
    print_error "   nginx -g \"daemon off;\""
    print_error "3. This matches the Dockerfile CMD instruction"
    print_error ""
    print_error "SOLUTION FOR BACKEND SERVICES:"
    print_error "1. Use the command from each service's Dockerfile CMD"
    print_error "2. Example: './knirv-oracle' or './knirv-chain'"
    print_error ""
    print_error "Service: ${RENDER_SERVICE_NAME:-'unknown'}"
    print_error "Container ID: $(hostname)"
    print_error "Working directory: $(pwd)"
    exit 1
fi

# Environment detection and comprehensive logging
print_status "=== ENVIRONMENT DETECTION ==="
print_status "RENDER: ${RENDER:-'not set'}"
print_status "RENDER_SERVICE_ID: ${RENDER_SERVICE_ID:-'not set'}"
print_status "RENDER_SERVICE_NAME: ${RENDER_SERVICE_NAME:-'not set'}"
print_status "RENDER_EXTERNAL_URL: ${RENDER_EXTERNAL_URL:-'not set'}"
print_status "RENDER_GIT_COMMIT: ${RENDER_GIT_COMMIT:-'not set'}"
print_status "RENDER_GIT_BRANCH: ${RENDER_GIT_BRANCH:-'not set'}"
print_status "NODE_ENV: ${NODE_ENV:-'not set'}"
print_status "PWD: $(pwd)"
print_status "HOME: ${HOME:-'not set'}"

# Detect environment
if [ "$RENDER" = "true" ] || [ -n "$RENDER_SERVICE_ID" ]; then
    DEPLOYMENT_ENV="render"
    print_status "🌐 Render cloud deployment detected"
    print_status "Service ID: ${RENDER_SERVICE_ID:-'not set'}"
    print_status "Service Name: ${RENDER_SERVICE_NAME:-'not set'}"
    print_status "External URL: ${RENDER_EXTERNAL_URL:-'not set'}"
    print_status "Git Commit: ${RENDER_GIT_COMMIT:-'not set'}"
    print_status "Git Branch: ${RENDER_GIT_BRANCH:-'not set'}"
else
    DEPLOYMENT_ENV="local"
    print_status "💻 Local development environment detected"
fi

# Set environment variables
export KNIRV_ENV=testnet
export TESTNET_MODE=true
export DEPLOYMENT_ENV=$DEPLOYMENT_ENV

print_status "=== ENVIRONMENT CONFIGURATION ==="
print_status "KNIRV_ENV: $KNIRV_ENV"
print_status "TESTNET_MODE: $TESTNET_MODE"
print_status "DEPLOYMENT_ENV: $DEPLOYMENT_ENV"

# Handle PORT configuration
print_status "=== PORT CONFIGURATION ==="
if [ -z "$PORT" ]; then
    if [ "$DEPLOYMENT_ENV" = "render" ]; then
        print_warning "PORT environment variable not set by Render"
        export PORT=80
        print_warning "Defaulting to PORT=80 for Render"
    else
        export PORT=10000
        print_status "Using PORT=10000 for local development"
    fi
else
    print_success "Using provided PORT=$PORT"
fi

print_status "=== STARTUP INITIALIZATION ==="
print_status "Starting KNIRV Testnet..."
print_status "Environment: $KNIRV_ENV"
print_status "Deployment: $DEPLOYMENT_ENV"
print_status "Port: $PORT"

# Run comprehensive diagnostics first
print_status "=== RUNNING DEPLOYMENT DIAGNOSTICS ==="
if [ -f "scripts/diagnose-deployment.sh" ]; then
    print_status "Running deployment diagnostics..."
    bash scripts/diagnose-deployment.sh
    DIAG_EXIT_CODE=$?
    if [ $DIAG_EXIT_CODE -ne 0 ]; then
        print_error "Diagnostics detected configuration issues"
        print_error "Check the diagnostic output above for recommendations"
        exit $DIAG_EXIT_CODE
    fi
else
    print_warning "Diagnostic script not found - continuing without diagnostics"
fi

# Create necessary directories
print_status "Creating necessary directories..."
mkdir -p logs data config
print_success "Directories created: logs, data, config"

# List current directory structure for debugging
print_status "=== DIRECTORY STRUCTURE ==="
print_status "Current directory contents:"
ls -la | head -15 || print_error "Failed to list directory contents"

if [ "$DEPLOYMENT_ENV" = "render" ]; then
    # Render deployment - services are managed by config/render.yml
    print_status "=== RENDER DEPLOYMENT MODE ==="
    print_status "Docker containers managed by Render"
    print_status "Service Name: ${RENDER_SERVICE_NAME:-'not set'}"
    print_status "Service Type: ${RENDER_SERVICE_TYPE:-'not set'}"

    # Log all environment variables for debugging
    print_status "=== ENVIRONMENT VARIABLES ==="
    env | grep -E "(RENDER|PORT|KNIRV)" | sort | while read line; do
        print_status "  $line"
    done

    # Check which service this is
    print_status "=== SERVICE IDENTIFICATION ==="
    if [ "$RENDER_SERVICE_NAME" = "knirv-testnet-gateway" ] || [ -z "$RENDER_SERVICE_NAME" ]; then
        print_status "🌐 TESTNET GATEWAY SERVICE DETECTED"

        # Verify we're in the right directory and files exist
        print_status "=== GATEWAY VERIFICATION ==="
        print_status "Current directory: $(pwd)"
        print_status "Directory contents:"
        ls -la | head -10

        # Check for testnet-gateway files
        if [ -d "data/testnet-gateway" ]; then
            print_success "✓ testnet-gateway directory found"
            print_status "Contents of data/testnet-gateway:"
            ls -la data/testnet-gateway/ | head -10
        else
            print_error "✗ testnet-gateway directory not found!"
            print_status "Available directories:"
            find . -maxdepth 2 -type d -name "*gateway*" 2>/dev/null || echo "No gateway directories found"
            exit 1
        fi

        # Check if we're running in a container
        print_status "=== CONTAINER DETECTION ==="
        if [ -f /.dockerenv ]; then
            print_status "🐳 Running inside Docker container"
            print_status "Container should handle nginx startup via Dockerfile CMD"
            print_status "This start script should NOT be called in containerized mode"
            print_error "ERROR: npm start should not be called in Docker container"
            print_error "The Dockerfile CMD should start nginx directly"
            print_error "Check config/render.yml configuration - remove startCommand for Docker services"
            exit 1
        else
            print_status "🖥️  Running on host (not containerized)"
            print_status "This suggests config/render.yml is not using Docker properly"
            print_error "ERROR: Service should be containerized but appears to be running on host"
            print_error "Check config/render.yml - ensure runtime: docker is set"
            exit 1
        fi

    else
        # Backend service
        print_status "🔧 BACKEND SERVICE DETECTED: ${RENDER_SERVICE_NAME}"
        print_status "Backend services should be managed by their Dockerfiles"

        # Check if we're in a container
        if [ -f /.dockerenv ]; then
            print_status "🐳 Running inside Docker container"
            print_error "ERROR: Backend service npm start should not be called in container"
            print_error "The service Dockerfile should handle startup directly"
            exit 1
        else
            print_error "ERROR: Backend service should be containerized"
            print_error "Check config/render.yml configuration for ${RENDER_SERVICE_NAME}"
            exit 1
        fi
    fi

else
    # Local development - use docker-compose
    print_status "Local Docker development mode"

    # Check if Docker is available
    if ! command -v docker &> /dev/null; then
        print_error "Docker not found! Please install Docker for local development."
        exit 1
    fi

    # Check if docker-compose is available
    if command -v docker-compose &> /dev/null; then
        COMPOSE_CMD="docker-compose"
    elif docker compose version &> /dev/null; then
        COMPOSE_CMD="docker compose"
    else
        print_error "Docker Compose not found! Please install Docker Compose."
        exit 1
    fi

    print_status "Using Docker Compose command: $COMPOSE_CMD"

    # Stop any existing containers
    print_status "Stopping any existing containers..."
    $COMPOSE_CMD down --remove-orphans || true

    # Start the testnet with docker-compose
    print_status "Starting KNIRV Testnet with Docker Compose..."

    # Build and start services
    $COMPOSE_CMD up --build -d

    print_success "KNIRV Testnet started successfully!"
    print_status "Services available at:"
    print_status "  - Testnet Gateway: http://localhost:10000"
    print_status "  - KNIRV Oracle: http://localhost:1317"
    print_status "  - KNIRV Chain: http://localhost:8090"
    print_status "  - KNIRV Graph: http://localhost:8082"
    print_status "  - KNIRV Nexus: http://localhost:8084"
    print_status "  - KNIRV Router: http://localhost:8086"
    print_status "  - IPFS Gateway: http://localhost:8080"
    print_status "  - IPFS API: http://localhost:5001"

    print_status "To view logs: $COMPOSE_CMD logs -f"
    print_status "To stop: $COMPOSE_CMD down"

    # Follow logs
    print_status "Following container logs (Ctrl+C to exit)..."
    $COMPOSE_CMD logs -f
fi
