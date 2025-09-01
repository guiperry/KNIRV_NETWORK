#!/bin/bash
set -e

# KNIRV Testnet Docker Deployment Script for Render
# This script handles both Render cloud deployment and local Docker testing

echo "🐳 KNIRV TESTNET - DOCKER DEPLOYMENT"
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

# Detect environment
if [ "$RENDER" = "true" ] || [ -n "$RENDER_SERVICE_ID" ]; then
    DEPLOYMENT_ENV="render"
    print_status "Render cloud deployment detected"
    print_status "Service ID: ${RENDER_SERVICE_ID:-'not set'}"
    print_status "External URL: ${RENDER_EXTERNAL_URL:-'not set'}"
else
    DEPLOYMENT_ENV="local"
    print_status "Local development environment detected"
fi

# Set environment variables
export KNIRV_ENV=testnet
export TESTNET_MODE=true
export DEPLOYMENT_ENV=$DEPLOYMENT_ENV

# Handle PORT configuration
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

print_status "Starting KNIRV Testnet..."
print_status "Environment: $KNIRV_ENV"
print_status "Deployment: $DEPLOYMENT_ENV"
print_status "Port: $PORT"

# Create necessary directories
mkdir -p logs data config

if [ "$DEPLOYMENT_ENV" = "render" ]; then
    # Render deployment - services are managed by render.yaml
    # This script only needs to start the main gateway service

    print_status "Render deployment mode - Docker containers managed by Render"
    print_status "Starting testnet gateway service..."

    # Check which service this is based on RENDER_SERVICE_NAME or default behavior
    if [ "$RENDER_SERVICE_NAME" = "knirv-testnet-gateway" ] || [ -z "$RENDER_SERVICE_NAME" ]; then
        # This is the main gateway service
        print_status "Starting testnet gateway (nginx + static files)"

        # Verify testnet-gateway files
        if [ ! -d "data/testnet-gateway" ]; then
            print_error "testnet-gateway directory not found!"
            exit 1
        fi

        # For the gateway service, we need to start nginx or a simple file server
        # Since this is containerized, the Dockerfile handles the nginx setup
        print_status "Gateway service ready - container will handle nginx startup"

        # Keep the container running (this should be handled by the Dockerfile CMD)
        print_success "Testnet gateway service initialized"

        # If we reach here in a container, something is wrong with the Dockerfile
        print_error "This script should not be called directly in a containerized gateway"
        print_error "The Dockerfile should handle nginx startup directly"
        exit 1

    else
        # This might be a backend service - shouldn't happen with our setup
        print_error "Unknown service: ${RENDER_SERVICE_NAME:-'undefined'}"
        print_error "Backend services should be managed by their respective Dockerfiles"
        exit 1
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
