#!/bin/bash

# KNIRV Gateway Management Script
# This script manages the KNIRVGATEWAY services including economics and API gateway

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
GATEWAY_DIR="$PROJECT_ROOT/KNIRVGATEWAY"
ECONOMICS_DIR="$GATEWAY_DIR/economics"
API_GATEWAY_DIR="$GATEWAY_DIR/api-gateway"

# Default configuration
DEFAULT_ECONOMICS_PORT="8090"
DEFAULT_GATEWAY_PORT="8000"

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

print_header() {
    echo -e "${PURPLE}[HEADER]${NC} $1"
}

# Function to show usage
show_usage() {
    echo "KNIRV Gateway Management Script"
    echo ""
    echo "Usage: $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  start                    Start all gateway services"
    echo "  stop                     Stop all gateway services"
    echo "  restart                  Restart all gateway services"
    echo "  status                   Check status of all services"
    echo "  test                     Run comprehensive tests"
    echo "  economics                Manage economics service only"
    echo "  api-gateway              Manage API gateway only"
    echo "  build                    Build all services"
    echo "  clean                    Clean build artifacts"
    echo "  logs                     Show service logs"
    echo "  verify                   Verify installation and configuration"
    echo ""
    echo "Economics Service Options:"
    echo "  economics start          Start economics service"
    echo "  economics stop           Stop economics service"
    echo "  economics test           Test economics service"
    echo "  economics verify         Verify economics implementation"
    echo ""
    echo "API Gateway Options:"
    echo "  api-gateway start        Start API gateway"
    echo "  api-gateway stop         Stop API gateway"
    echo "  api-gateway test         Test API gateway"
    echo ""
    echo "Global Options:"
    echo "  -p, --port PORT          Set port for services"
    echo "  -e, --economics-port     Set economics service port (default: $DEFAULT_ECONOMICS_PORT)"
    echo "  -g, --gateway-port       Set API gateway port (default: $DEFAULT_GATEWAY_PORT)"
    echo "  -d, --dev                Run in development mode"
    echo "  -v, --verbose            Verbose output"
    echo "  -h, --help               Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  ECONOMICS_PORT           Economics service port"
    echo "  GATEWAY_PORT             API gateway port"
    echo "  NRN_CONTRACT             NRN token contract address"
    echo "  XION_RPC                 XION RPC endpoint"
    echo "  KNIRVCHAIN_URL           KNIRVCHAIN service URL"
    echo "  KNIRVNEXUS_URL           KNIRVSERVER service URL"
    echo "  KNIRVORACLE_URL            KNIRVORACLE service URL"
    echo "  KNIRVGRAPH_URL           KNIRVGRAPH service URL"
}

# Function to check if directory exists
check_directory() {
    local dir=$1
    local name=$2
    
    if [ ! -d "$dir" ]; then
        print_error "$name directory not found: $dir"
        return 1
    fi
    return 0
}

# Function to check service status
check_service_status() {
    local url=$1
    local service_name=$2
    
    if curl -s -f "$url" > /dev/null 2>&1; then
        print_success "$service_name is running at $url"
        return 0
    else
        print_warning "$service_name is not responding at $url"
        return 1
    fi
}

# Function to start economics service
start_economics() {
    print_header "Starting Economics Service"
    
    if ! check_directory "$ECONOMICS_DIR" "Economics"; then
        return 1
    fi
    
    cd "$ECONOMICS_DIR"
    
    # Set environment variables
    export ECONOMICS_PORT=${ECONOMICS_PORT:-$DEFAULT_ECONOMICS_PORT}
    export NRN_CONTRACT=${NRN_CONTRACT:-"nrn_contract_placeholder"}
    export XION_RPC=${XION_RPC:-"https://rpc.xion-testnet-1.burnt.com:443"}
    export KNIRVCHAIN_URL=${KNIRVCHAIN_URL:-"http://localhost:8080"}
    export KNIRVNEXUS_URL=${KNIRVNEXUS_URL:-"http://localhost:8081"}
    export KNIRVORACLE_URL=${KNIRVORACLE_URL:-"http://localhost:8082"}
    export KNIRVGRAPH_URL=${KNIRVGRAPH_URL:-"http://localhost:8083"}
    
    print_info "Starting economics service on port $ECONOMICS_PORT"
    
    if [ -x "./start-economics.sh" ]; then
        ./start-economics.sh --no-deps-check &
        ECONOMICS_PID=$!
        echo $ECONOMICS_PID > /tmp/economics.pid
        print_success "Economics service started with PID $ECONOMICS_PID"
    else
        print_error "Economics startup script not found or not executable"
        return 1
    fi
    
    cd "$PROJECT_ROOT"
}

# Function to stop economics service
stop_economics() {
    print_header "Stopping Economics Service"
    
    if [ -f "/tmp/economics.pid" ]; then
        local pid=$(cat /tmp/economics.pid)
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid"
            rm -f /tmp/economics.pid
            print_success "Economics service stopped"
        else
            print_warning "Economics service was not running"
            rm -f /tmp/economics.pid
        fi
    else
        print_warning "Economics service PID file not found"
    fi
}

# Function to test economics service
test_economics() {
    print_header "Testing Economics Service"
    
    if ! check_directory "$ECONOMICS_DIR" "Economics"; then
        return 1
    fi
    
    cd "$ECONOMICS_DIR"
    
    if [ -x "./test-economics.sh" ]; then
        ./test-economics.sh
    else
        print_error "Economics test script not found or not executable"
        return 1
    fi
    
    cd "$PROJECT_ROOT"
}

# Function to verify economics implementation
verify_economics() {
    print_header "Verifying Economics Implementation"
    
    if ! check_directory "$ECONOMICS_DIR" "Economics"; then
        return 1
    fi
    
    cd "$ECONOMICS_DIR"
    
    if [ -x "./verify-month11.sh" ]; then
        ./verify-month11.sh
    else
        print_error "Economics verification script not found or not executable"
        return 1
    fi
    
    cd "$PROJECT_ROOT"
}

# Function to start API gateway
start_api_gateway() {
    print_header "Starting API Gateway"
    
    if ! check_directory "$API_GATEWAY_DIR" "API Gateway"; then
        return 1
    fi
    
    cd "$API_GATEWAY_DIR"
    
    export GATEWAY_PORT=${GATEWAY_PORT:-$DEFAULT_GATEWAY_PORT}
    
    print_info "Starting API gateway on port $GATEWAY_PORT"
    
    # Check if there's a startup script
    if [ -f "start-gateway.sh" ] && [ -x "start-gateway.sh" ]; then
        ./start-gateway.sh &
        GATEWAY_PID=$!
        echo $GATEWAY_PID > /tmp/gateway.pid
        print_success "API gateway started with PID $GATEWAY_PID"
    elif [ -f "main.go" ]; then
        go run main.go -port "$GATEWAY_PORT" &
        GATEWAY_PID=$!
        echo $GATEWAY_PID > /tmp/gateway.pid
        print_success "API gateway started with PID $GATEWAY_PID"
    else
        print_error "API gateway startup method not found"
        return 1
    fi
    
    cd "$PROJECT_ROOT"
}

# Function to stop API gateway
stop_api_gateway() {
    print_header "Stopping API Gateway"
    
    if [ -f "/tmp/gateway.pid" ]; then
        local pid=$(cat /tmp/gateway.pid)
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid"
            rm -f /tmp/gateway.pid
            print_success "API gateway stopped"
        else
            print_warning "API gateway was not running"
            rm -f /tmp/gateway.pid
        fi
    else
        print_warning "API gateway PID file not found"
    fi
}

# Function to build all services
build_services() {
    print_header "Building Gateway Services"
    
    # Build economics service
    if check_directory "$ECONOMICS_DIR" "Economics"; then
        cd "$ECONOMICS_DIR"
        print_info "Building economics service..."
        if go build -o bin/economics-service cmd/main.go; then
            print_success "Economics service built successfully"
        else
            print_error "Failed to build economics service"
            return 1
        fi
        cd "$PROJECT_ROOT"
    fi
    
    # Build API gateway
    if check_directory "$API_GATEWAY_DIR" "API Gateway"; then
        cd "$API_GATEWAY_DIR"
        print_info "Building API gateway..."
        if [ -f "main.go" ]; then
            if go build -o bin/gateway main.go; then
                print_success "API gateway built successfully"
            else
                print_error "Failed to build API gateway"
                return 1
            fi
        else
            print_warning "API gateway main.go not found, skipping build"
        fi
        cd "$PROJECT_ROOT"
    fi
}

# Function to check status of all services
check_status() {
    print_header "Checking Gateway Services Status"
    
    local economics_port=${ECONOMICS_PORT:-$DEFAULT_ECONOMICS_PORT}
    local gateway_port=${GATEWAY_PORT:-$DEFAULT_GATEWAY_PORT}
    
    check_service_status "http://localhost:$economics_port/economics/health" "Economics Service"
    check_service_status "http://localhost:$gateway_port/health" "API Gateway"
}

# Parse command line arguments
COMMAND=""
ECONOMICS_PORT=${ECONOMICS_PORT:-$DEFAULT_ECONOMICS_PORT}
GATEWAY_PORT=${GATEWAY_PORT:-$DEFAULT_GATEWAY_PORT}
DEV_MODE=false
VERBOSE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        start|stop|restart|status|test|build|clean|logs|verify)
            COMMAND="$1"
            shift
            ;;
        economics|api-gateway)
            COMMAND="$1"
            if [[ $# -gt 1 ]] && [[ $2 =~ ^(start|stop|test|verify)$ ]]; then
                COMMAND="$1_$2"
                shift
            fi
            shift
            ;;
        -e|--economics-port)
            ECONOMICS_PORT="$2"
            shift 2
            ;;
        -g|--gateway-port)
            GATEWAY_PORT="$2"
            shift 2
            ;;
        -d|--dev)
            DEV_MODE=true
            shift
            ;;
        -v|--verbose)
            VERBOSE=true
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

# Main command execution
case $COMMAND in
    start)
        print_header "Starting All Gateway Services"
        start_economics
        sleep 5  # Give economics service time to start
        start_api_gateway
        sleep 3
        check_status
        ;;
    stop)
        print_header "Stopping All Gateway Services"
        stop_api_gateway
        stop_economics
        ;;
    restart)
        print_header "Restarting All Gateway Services"
        stop_api_gateway
        stop_economics
        sleep 2
        start_economics
        sleep 5
        start_api_gateway
        sleep 3
        check_status
        ;;
    status)
        check_status
        ;;
    test)
        print_header "Running Comprehensive Gateway Tests"
        test_economics
        # Add API gateway tests here when available
        ;;
    economics_start)
        start_economics
        ;;
    economics_stop)
        stop_economics
        ;;
    economics_test)
        test_economics
        ;;
    economics_verify)
        verify_economics
        ;;
    api-gateway_start)
        start_api_gateway
        ;;
    api-gateway_stop)
        stop_api_gateway
        ;;
    build)
        build_services
        ;;
    verify)
        verify_economics
        ;;
    *)
        if [ -z "$COMMAND" ]; then
            print_error "No command specified"
        else
            print_error "Unknown command: $COMMAND"
        fi
        show_usage
        exit 1
        ;;
esac
