#!/bin/bash

# KNIRV Economics Service Startup Script
# This script starts the economics service with proper configuration

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default configuration
DEFAULT_PORT="8090"
DEFAULT_NRN_CONTRACT="nrn_contract_placeholder"
DEFAULT_XION_RPC="https://rpc.xion-testnet-1.burnt.com:443"
DEFAULT_KNIRVCHAIN_URL="http://localhost:8080"
DEFAULT_KNIRVNEXUS_URL="http://localhost:8081"
DEFAULT_KNIRVORACLE_URL="http://localhost:8082"
DEFAULT_KNIRVGRAPH_URL="http://localhost:8083"

# Function to print colored output
print_info() {
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

# Function to check if a service is running
check_service() {
    local url=$1
    local service_name=$2
    
    if curl -s -f "$url/health" > /dev/null 2>&1; then
        print_success "$service_name is running at $url"
        return 0
    else
        print_warning "$service_name is not responding at $url"
        return 1
    fi
}

# Function to wait for service to be ready
wait_for_service() {
    local url=$1
    local service_name=$2
    local max_attempts=30
    local attempt=1
    
    print_info "Waiting for $service_name to be ready..."
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s -f "$url/health" > /dev/null 2>&1; then
            print_success "$service_name is ready!"
            return 0
        fi
        
        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    print_error "$service_name failed to start within $((max_attempts * 2)) seconds"
    return 1
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -p, --port PORT              Port to run the service on (default: $DEFAULT_PORT)"
    echo "  -c, --config FILE            Configuration file path"
    echo "  -d, --dev                    Run in development mode"
    echo "  -b, --build                  Build the service before starting"
    echo "  -h, --help                   Show this help message"
    echo "  --check-deps                 Check if all dependencies are running"
    echo "  --no-deps-check             Skip dependency checks"
    echo ""
    echo "Environment Variables:"
    echo "  ECONOMICS_PORT               Service port"
    echo "  NRN_CONTRACT                 NRN token contract address"
    echo "  XION_RPC                     XION RPC endpoint"
    echo "  KNIRVCHAIN_URL              KNIRVCHAIN service URL"
    echo "  KNIRVNEXUS_URL              KNIRVNEXUS service URL"
    echo "  KNIRVORACLE_URL               KNIRVORACLE service URL"
    echo "  KNIRVGRAPH_URL              KNIRVGRAPH service URL"
}

# Parse command line arguments
PORT=${ECONOMICS_PORT:-$DEFAULT_PORT}
CONFIG_FILE=""
DEV_MODE=false
BUILD_FIRST=false
CHECK_DEPS=true
SKIP_DEPS_CHECK=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -p|--port)
            PORT="$2"
            shift 2
            ;;
        -c|--config)
            CONFIG_FILE="$2"
            shift 2
            ;;
        -d|--dev)
            DEV_MODE=true
            shift
            ;;
        -b|--build)
            BUILD_FIRST=true
            shift
            ;;
        --check-deps)
            CHECK_DEPS=true
            shift
            ;;
        --no-deps-check)
            SKIP_DEPS_CHECK=true
            shift
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Set environment variables with defaults
export ECONOMICS_PORT=${ECONOMICS_PORT:-$PORT}
export NRN_CONTRACT=${NRN_CONTRACT:-$DEFAULT_NRN_CONTRACT}
export XION_RPC=${XION_RPC:-$DEFAULT_XION_RPC}
export KNIRVCHAIN_URL=${KNIRVCHAIN_URL:-$DEFAULT_KNIRVCHAIN_URL}
export KNIRVNEXUS_URL=${KNIRVNEXUS_URL:-$DEFAULT_KNIRVNEXUS_URL}
export KNIRVORACLE_URL=${KNIRVORACLE_URL:-$DEFAULT_KNIRVORACLE_URL}
export KNIRVGRAPH_URL=${KNIRVGRAPH_URL:-$DEFAULT_KNIRVGRAPH_URL}

print_info "Starting KNIRV Economics Service..."
print_info "Configuration:"
print_info "  Port: $ECONOMICS_PORT"
print_info "  NRN Contract: $NRN_CONTRACT"
print_info "  XION RPC: $XION_RPC"
print_info "  Dev Mode: $DEV_MODE"

# Check if we're in the right directory
if [ ! -f "cmd/main.go" ]; then
    print_error "Please run this script from the economics directory"
    exit 1
fi

# Build the service if requested
if [ "$BUILD_FIRST" = true ]; then
    print_info "Building economics service..."
    go build -o bin/economics-service cmd/main.go
    if [ $? -eq 0 ]; then
        print_success "Build completed successfully"
    else
        print_error "Build failed"
        exit 1
    fi
fi

# Check dependencies if not skipped
if [ "$SKIP_DEPS_CHECK" = false ]; then
    print_info "Checking KNIRV component dependencies..."
    
    DEPS_OK=true
    
    if ! check_service "$KNIRVCHAIN_URL" "KNIRVCHAIN"; then
        DEPS_OK=false
    fi
    
    if ! check_service "$KNIRVNEXUS_URL" "KNIRVNEXUS"; then
        DEPS_OK=false
    fi
    
    if ! check_service "$KNIRVORACLE_URL" "KNIRVORACLE"; then
        DEPS_OK=false
    fi
    
    if ! check_service "$KNIRVGRAPH_URL" "KNIRVGRAPH"; then
        DEPS_OK=false
    fi
    
    if [ "$DEPS_OK" = false ] && [ "$DEV_MODE" = false ]; then
        print_error "Some dependencies are not running. Use --no-deps-check to skip this check or -d for dev mode."
        exit 1
    elif [ "$DEPS_OK" = false ]; then
        print_warning "Some dependencies are not running, but continuing in dev mode..."
    fi
fi

# Start the service
print_info "Starting economics service on port $ECONOMICS_PORT..."

if [ -f "bin/economics-service" ]; then
    # Use pre-built binary
    if [ -n "$CONFIG_FILE" ]; then
        ./bin/economics-service -port "$ECONOMICS_PORT" -config "$CONFIG_FILE"
    else
        ./bin/economics-service -port "$ECONOMICS_PORT"
    fi
else
    # Run with go run
    if [ -n "$CONFIG_FILE" ]; then
        go run cmd/main.go -port "$ECONOMICS_PORT" -config "$CONFIG_FILE"
    else
        go run cmd/main.go -port "$ECONOMICS_PORT"
    fi
fi
