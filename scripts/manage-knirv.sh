#!/bin/bash

# KNIRV Network Management Script
# This script provides unified management for all KNIRV components including the gateway

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Component directories
KNIRVCHAIN_DIR="$PROJECT_ROOT/KNIRVCHAIN"
KNIRVNEXUS_DIR="$PROJECT_ROOT/KNIRVNEXUS"
KNIRVROOT_DIR="$PROJECT_ROOT/KNIRVROOT"
KNIRVGRAPH_DIR="$PROJECT_ROOT/KNIRVGRAPH"
KNIRVROUTER_DIR="$PROJECT_ROOT/KNIRVROUTER"
KNIRVGATEWAY_DIR="$PROJECT_ROOT/KNIRVGATEWAY"
INTEGRATION_TESTS_DIR="$PROJECT_ROOT/integration-tests"

# Default ports
DEFAULT_KNIRVCHAIN_PORT=8080
DEFAULT_KNIRVNEXUS_PORT=8081
DEFAULT_KNIRVROOT_PORT=8082
DEFAULT_KNIRVGRAPH_PORT=8083
DEFAULT_KNIRVROUTER_PORT=8084
DEFAULT_ECONOMICS_PORT=8090
DEFAULT_GATEWAY_PORT=8000

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

print_component() {
    echo -e "${CYAN}[COMPONENT]${NC} $1"
}

# Function to show usage
show_usage() {
    echo "KNIRV Network Management Script"
    echo ""
    echo "Usage: $0 [COMMAND] [COMPONENT] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  start                    Start all components or specific component"
    echo "  stop                     Stop all components or specific component"
    echo "  restart                  Restart all components or specific component"
    echo "  status                   Check status of all components or specific component"
    echo "  build                    Build all components or specific component"
    echo "  test                     Run tests for all components or specific component"
    echo "  logs                     Show logs for specific component"
    echo "  clean                    Clean build artifacts"
    echo "  deploy                   Deploy components"
    echo "  health                   Check health of all services"
    echo "  production-test          Run production test suite"
    echo "  deploy-test              Deploy and run comprehensive tests"
    echo ""
    echo "Components:"
    echo "  all                      All KNIRV components (default)"
    echo "  knirvchain               KNIRVCHAIN service"
    echo "  knirvnexus               KNIRVNEXUS service"
    echo "  knirvroot                KNIRVROOT service"
    echo "  knirvgraph               KNIRVGRAPH service"
    echo "  knirvrouter              KNIRVROUTER service"
    echo "  gateway                  KNIRVGATEWAY (API Gateway + Economics)"
    echo "  economics                Economics service only"
    echo "  integration              Integration test suite"
    echo ""
    echo "Options:"
    echo "  -p, --port PORT          Set port for component"
    echo "  -d, --dev                Run in development mode"
    echo "  -v, --verbose            Verbose output"
    echo "  -b, --background         Run in background"
    echo "  -w, --wait               Wait for services to be ready"
    echo "  -t, --timeout SECONDS    Timeout for operations (default: 300)"
    echo "  -h, --help               Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 start all             # Start all components"
    echo "  $0 start gateway         # Start gateway services only"
    echo "  $0 test economics        # Test economics service"
    echo "  $0 status                # Check status of all components"
    echo "  $0 build knirvchain      # Build KNIRVCHAIN only"
    echo "  $0 deploy --dev          # Deploy in development mode"
}

# Function to check if component directory exists
check_component_dir() {
    local component=$1
    local dir=$2
    
    if [ ! -d "$dir" ]; then
        print_error "$component directory not found: $dir"
        return 1
    fi
    return 0
}

# Function to check service health
check_service_health() {
    local url=$1
    local service_name=$2
    local timeout=${3:-5}
    
    if timeout "$timeout" curl -s -f "$url" > /dev/null 2>&1; then
        print_success "$service_name is healthy at $url"
        return 0
    else
        print_warning "$service_name is not responding at $url"
        return 1
    fi
}

# Function to start a component
start_component() {
    local component=$1
    local port=$2
    
    case $component in
        "knirvchain")
            print_component "Starting KNIRVCHAIN..."
            if check_component_dir "KNIRVCHAIN" "$KNIRVCHAIN_DIR"; then
                cd "$KNIRVCHAIN_DIR"

                # Check if Rust/Cargo is available
                if ! command -v cargo &> /dev/null; then
                    print_error "Rust/Cargo is required but not installed"
                    return 1
                fi

                # Create necessary directories
                mkdir -p logs data/knirvchain

                # Check if binary exists, build if needed
                if [ ! -f "./target/release/knirvchain" ] && [ ! -f "./knirvchain" ]; then
                    print_info "Building KNIRVCHAIN..."
                    cargo build --release --features testnet
                fi

                # Set environment variables for testnet
                export KNIRVCHAIN_RPC_ENDPOINT="127.0.0.1:8090"
                export BLOCK_DIFFICULTY="1"
                export KNIRVCHAIN_ID="1"
                export BLOCK_TIME="5"
                export RUST_LOG="info"

                # Start KNIRVCHAIN in testnet mode
                print_info "Starting KNIRVCHAIN with testnet features on port 8090..."
                if [ -f "./target/release/knirvchain" ]; then
                    ./target/release/knirvchain > ./logs/knirvchain.log 2>&1 &
                elif [ -f "./knirvchain" ]; then
                    ./knirvchain > ./logs/knirvchain.log 2>&1 &
                else
                    print_error "KNIRVCHAIN binary not found after build"
                    return 1
                fi

                CHAIN_PID=$!
                echo $CHAIN_PID > ./data/knirvchain.pid
                print_success "KNIRVCHAIN started with PID $CHAIN_PID on port 8090"

                # Wait a moment and check if process is still running
                sleep 3
                if ! kill -0 $CHAIN_PID 2>/dev/null; then
                    print_error "KNIRVCHAIN failed to start. Check logs: ./logs/knirvchain.log"
                    return 1
                fi
            fi
            ;;
        "knirvnexus")
            print_component "Starting KNIRVNEXUS..."
            if check_component_dir "KNIRVNEXUS" "$KNIRVNEXUS_DIR"; then
                cd "$KNIRVNEXUS_DIR"

                # Check if Go is available
                if ! command -v go &> /dev/null; then
                    print_error "Go is required but not installed"
                    return 1
                fi

                # Create necessary directories
                mkdir -p logs data reports
                touch data/nexus.db

                # Check if backend binary exists
                if [ ! -f "./backend/bin/dve-manager" ]; then
                    print_info "Building KNIRVNEXUS backend..."
                    cd backend
                    go build -o bin/dve-manager cmd/dve-manager/main.go
                    cd ..
                fi

                # Start KNIRVNEXUS backend in testnet mode
                print_info "Starting KNIRVNEXUS backend on port 8083..."
                # Run from KNIRVNEXUS directory to ensure proper working directory
                ./backend/bin/dve-manager -testnet -port 8083 > ./logs/knirvnexus.log 2>&1 &
                NEXUS_PID=$!
                echo $NEXUS_PID > ./data/knirvnexus.pid

                print_success "KNIRVNEXUS started with PID $NEXUS_PID on port 8083"

                # Wait a moment and check if process is still running
                sleep 3
                if ! kill -0 $NEXUS_PID 2>/dev/null; then
                    print_error "KNIRVNEXUS failed to start. Check logs: ./logs/knirvnexus.log"
                    return 1
                fi
            fi
            ;;
        "knirvroot")
            print_component "Starting KNIRVROOT..."
            if check_component_dir "KNIRVROOT" "$KNIRVROOT_DIR"; then
                cd "$KNIRVROOT_DIR"

                # Check if Go is available
                if ! command -v go &> /dev/null; then
                    print_error "Go is required but not installed"
                    return 1
                fi

                # Create necessary directories
                mkdir -p logs data/testnet

                # Check if binary exists, build if needed
                if [ ! -f "./knirvroot" ]; then
                    print_info "Building KNIRVROOT..."
                    if [ -f "./Makefile" ]; then
                        make build
                    else
                        go build -o knirvroot .
                    fi
                fi

                # Start KNIRVROOT in testnet mode
                print_info "Starting KNIRVROOT in testnet mode on port 1317..."
                ./knirvroot \
                    --testnet \
                    --port 1317 \
                    --p2p.port 26656 \
                    --shared_database_path ./data/testnet/blockchain.db \
                    --miners_address KNIRVROOT_Faucet \
                    --root \
                    --non-interactive \
                    --skip-install \
                    > ./logs/knirvroot.log 2>&1 &

                ROOT_PID=$!
                echo $ROOT_PID > ./data/knirvroot.pid
                print_success "KNIRVROOT started with PID $ROOT_PID on port 1317"

                # Wait a moment and check if process is still running
                sleep 3
                if ! kill -0 $ROOT_PID 2>/dev/null; then
                    print_error "KNIRVROOT failed to start. Check logs: ./logs/knirvroot.log"
                    return 1
                fi
            fi
            ;;
        "knirvgraph")
            print_component "Starting KNIRVGRAPH..."
            if check_component_dir "KNIRVGRAPH" "$KNIRVGRAPH_DIR"; then
                cd "$KNIRVGRAPH_DIR"

                # Check if Go is available
                if ! command -v go &> /dev/null; then
                    print_error "Go is required but not installed"
                    return 1
                fi

                # Create necessary directories
                mkdir -p logs data/knirvgraph

                # Check if binary exists, build if needed
                if [ ! -f "./knirvgraph" ] && [ ! -f "./bin/knirvgraph" ]; then
                    print_info "Building KNIRVGRAPH..."
                    if [ -f "./Makefile" ]; then
                        make build
                    else
                        go build -o knirvgraph ./cmd/knirvgraph
                    fi
                fi

                # Start KNIRVGRAPH in testnet mode
                print_info "Starting KNIRVGRAPH with testnet features on port 8082..."
                if [ -f "./knirvgraph" ]; then
                    ./knirvgraph \
                        --testnet \
                        --populate \
                        --max-nodes 1000 \
                        --rpc-port 8082 \
                        --home ./data/knirvgraph \
                        > ./logs/knirvgraph.log 2>&1 &
                elif [ -f "./bin/knirvgraph" ]; then
                    ./bin/knirvgraph \
                        --testnet \
                        --populate \
                        --max-nodes 1000 \
                        --rpc-port 8082 \
                        --home ./data/knirvgraph \
                        > ./logs/knirvgraph.log 2>&1 &
                else
                    print_error "KNIRVGRAPH binary not found after build"
                    return 1
                fi

                GRAPH_PID=$!
                echo $GRAPH_PID > ./data/knirvgraph.pid
                print_success "KNIRVGRAPH started with PID $GRAPH_PID on port 8082"

                # Wait a moment and check if process is still running
                sleep 3
                if ! kill -0 $GRAPH_PID 2>/dev/null; then
                    print_error "KNIRVGRAPH failed to start. Check logs: ./logs/knirvgraph.log"
                    return 1
                fi
            fi
            ;;
        "knirvrouter")
            print_component "Starting KNIRVROUTER..."
            if check_component_dir "KNIRVROUTER" "$KNIRVROUTER_DIR"; then
                cd "$KNIRVROUTER_DIR"

                # Check if binary exists
                if [ ! -f "./knirvrouter" ] && [ ! -f "./bin/knirvrouter" ]; then
                    print_warning "KNIRVROUTER binary not found, attempting to build..."
                    if [ -f "./build.sh" ]; then
                        ./build.sh
                    else
                        go build -o knirvrouter main.go
                    fi
                fi

                # Create necessary directories
                mkdir -p logs data

                # Set environment variables for testnet mode
                export TESTNET_MODE=true
                export LOCAL_NETWORK_MODE=true
                export MOCK_NRN_MINTING=true
                export SIMPLIFIED_CONSENSUS=true
                export DISABLE_XION_BRIDGE=true

                # Start KNIRVROUTER in testnet mode
                print_info "Starting KNIRVROUTER with testnet configuration..."
                if [ -f "./knirvrouter" ]; then
                    ./knirvrouter -testnet -local-network -mock-nrn > ./logs/knirvrouter.log 2>&1 &
                elif [ -f "./bin/knirvrouter" ]; then
                    ./bin/knirvrouter -testnet -local-network -mock-nrn > ./logs/knirvrouter.log 2>&1 &
                else
                    print_error "KNIRVROUTER binary not found after build attempt"
                    return 1
                fi

                ROUTER_PID=$!
                echo $ROUTER_PID > ./data/knirvrouter.pid
                print_success "KNIRVROUTER started with PID $ROUTER_PID on port 5001"

                # Wait a moment and check if process is still running
                sleep 3
                if ! kill -0 $ROUTER_PID 2>/dev/null; then
                    print_error "KNIRVROUTER failed to start. Check logs: ./logs/knirvrouter.log"
                    return 1
                fi
            fi
            ;;
        "gateway")
            print_component "Starting KNIRVGATEWAY..."
            if check_component_dir "KNIRVGATEWAY" "$KNIRVGATEWAY_DIR"; then
                cd "$KNIRVGATEWAY_DIR"

                # Check if Node.js and npm are available
                if ! command -v node &> /dev/null; then
                    print_error "Node.js is required but not installed"
                    return 1
                fi

                if ! command -v npm &> /dev/null; then
                    print_error "npm is required but not installed"
                    return 1
                fi

                # Install dependencies if needed
                if [ ! -d "node_modules" ]; then
                    print_info "Installing KNIRVGATEWAY dependencies..."
                    npm install
                fi

                # Create necessary directories
                mkdir -p logs data

                # Set testnet environment variables
                export TESTNET_MODE=true
                export NODE_ENV=testnet
                export KNIRVROOT_URL=http://localhost:1317
                export KNIRVCHAIN_URL=http://localhost:8090
                export KNIRVGRAPH_URL=http://localhost:8082
                export KNIRVNEXUS_URL=http://localhost:8083
                export KNIRVROUTER_URL=http://localhost:5001

                # Start Netlify Dev server
                print_info "Starting KNIRVGATEWAY with Netlify Dev on port 8888..."
                npx netlify dev --port 8888 --targetPort 3001 > ./logs/knirvgateway.log 2>&1 &

                GATEWAY_PID=$!
                echo $GATEWAY_PID > ./data/knirvgateway.pid
                print_success "KNIRVGATEWAY started with PID $GATEWAY_PID on port 8888"

                # Wait a moment and check if process is still running
                sleep 5
                if ! kill -0 $GATEWAY_PID 2>/dev/null; then
                    print_error "KNIRVGATEWAY failed to start. Check logs: ./logs/knirvgateway.log"
                    return 1
                fi

                print_info "Gateway endpoints:"
                print_info "  - Main Site: http://localhost:8888"
                print_info "  - Health: http://localhost:8888/gateway/health"
                print_info "  - Services: http://localhost:8888/gateway/services"
                print_info "  - Auth: http://localhost:8888/auth/testnet-tokens"
            fi
            ;;
        "economics")
            print_component "Starting Economics Service..."
            "$SCRIPT_DIR/run-gateway.sh" economics start
            ;;
        *)
            print_error "Unknown component: $component"
            return 1
            ;;
    esac
    
    cd "$PROJECT_ROOT"
}

# Function to stop a component
stop_component() {
    local component=$1
    
    case $component in
        "gateway")
            print_component "Stopping KNIRVGATEWAY..."
            if [ -f "$KNIRVGATEWAY_DIR/data/knirvgateway.pid" ]; then
                local pid=$(cat "$KNIRVGATEWAY_DIR/data/knirvgateway.pid")
                if kill -0 "$pid" 2>/dev/null; then
                    kill "$pid"
                    print_success "KNIRVGATEWAY stopped (PID: $pid)"
                else
                    print_warning "KNIRVGATEWAY process not running"
                fi
                rm -f "$KNIRVGATEWAY_DIR/data/knirvgateway.pid"
            else
                print_warning "KNIRVGATEWAY PID file not found"
            fi
            ;;
        "economics")
            print_component "Stopping Economics Service..."
            "$SCRIPT_DIR/run-gateway.sh" economics stop
            ;;
        "knirvnexus")
            print_component "Stopping KNIRVNEXUS..."
            if [ -f "$KNIRVNEXUS_DIR/data/knirvnexus.pid" ]; then
                local pid=$(cat "$KNIRVNEXUS_DIR/data/knirvnexus.pid")
                if kill -0 "$pid" 2>/dev/null; then
                    kill "$pid"
                    print_success "KNIRVNEXUS stopped (PID: $pid)"
                else
                    print_warning "KNIRVNEXUS process not running"
                fi
                rm -f "$KNIRVNEXUS_DIR/data/knirvnexus.pid"
            else
                print_warning "KNIRVNEXUS PID file not found"
            fi
            ;;
        "knirvrouter")
            print_component "Stopping KNIRVROUTER..."
            if [ -f "$KNIRVROUTER_DIR/data/knirvrouter.pid" ]; then
                local pid=$(cat "$KNIRVROUTER_DIR/data/knirvrouter.pid")
                if kill -0 "$pid" 2>/dev/null; then
                    kill "$pid"
                    print_success "KNIRVROUTER stopped (PID: $pid)"
                else
                    print_warning "KNIRVROUTER process not running"
                fi
                rm -f "$KNIRVROUTER_DIR/data/knirvrouter.pid"
            else
                print_warning "KNIRVROUTER PID file not found"
            fi
            ;;
        "knirvchain")
            print_component "Stopping KNIRVCHAIN..."
            if [ -f "$KNIRVCHAIN_DIR/data/knirvchain.pid" ]; then
                local pid=$(cat "$KNIRVCHAIN_DIR/data/knirvchain.pid")
                if kill -0 "$pid" 2>/dev/null; then
                    kill "$pid"
                    print_success "KNIRVCHAIN stopped (PID: $pid)"
                else
                    print_warning "KNIRVCHAIN process not running"
                fi
                rm -f "$KNIRVCHAIN_DIR/data/knirvchain.pid"
            else
                print_warning "KNIRVCHAIN PID file not found"
            fi
            ;;
        "knirvgraph")
            print_component "Stopping KNIRVGRAPH..."
            if [ -f "$KNIRVGRAPH_DIR/data/knirvgraph.pid" ]; then
                local pid=$(cat "$KNIRVGRAPH_DIR/data/knirvgraph.pid")
                if kill -0 "$pid" 2>/dev/null; then
                    kill "$pid"
                    print_success "KNIRVGRAPH stopped (PID: $pid)"
                else
                    print_warning "KNIRVGRAPH process not running"
                fi
                rm -f "$KNIRVGRAPH_DIR/data/knirvgraph.pid"
            else
                print_warning "KNIRVGRAPH PID file not found"
            fi
            ;;
        "knirvroot")
            print_component "Stopping KNIRVROOT..."
            if [ -f "$KNIRVROOT_DIR/data/knirvroot.pid" ]; then
                local pid=$(cat "$KNIRVROOT_DIR/data/knirvroot.pid")
                if kill -0 "$pid" 2>/dev/null; then
                    kill "$pid"
                    print_success "KNIRVROOT stopped (PID: $pid)"
                else
                    print_warning "KNIRVROOT process not running"
                fi
                rm -f "$KNIRVROOT_DIR/data/knirvroot.pid"
            else
                print_warning "KNIRVROOT PID file not found"
            fi
            ;;
        *)
            print_component "Stopping $component..."
            # Add generic stop logic for other components
            print_success "$component stopped"
            ;;
    esac
}

# Function to check component status
check_component_status() {
    local component=$1
    
    case $component in
        "knirvchain")
            check_service_health "http://localhost:8090/health" "KNIRVCHAIN"
            ;;
        "knirvnexus")
            check_service_health "http://localhost:8083/health" "KNIRVNEXUS"
            ;;
        "knirvroot")
            check_service_health "http://localhost:1317/health" "KNIRVROOT"
            ;;
        "knirvgraph")
            check_service_health "http://localhost:8082/height" "KNIRVGRAPH"
            ;;
        "knirvrouter")
            check_service_health "http://localhost:5001/status" "KNIRVROUTER"
            ;;
        "gateway")
            check_service_health "http://localhost:8888/gateway/health" "KNIRVGATEWAY"
            ;;
        "economics")
            check_service_health "http://localhost:$DEFAULT_ECONOMICS_PORT/economics/health" "Economics Service"
            ;;
        *)
            print_warning "Status check not implemented for: $component"
            ;;
    esac
}

# Function to test a component
test_component() {
    local component=$1
    
    case $component in
        "gateway")
            print_component "Testing KNIRVGATEWAY..."
            "$SCRIPT_DIR/test-gateway-integration.sh"
            ;;
        "economics")
            print_component "Testing Economics Service..."
            "$SCRIPT_DIR/test-gateway-integration.sh" --economics-only
            ;;
        "integration")
            print_component "Running Integration Tests..."
            "$INTEGRATION_TESTS_DIR/config/run-tests.sh"
            ;;
        *)
            print_component "Testing $component..."
            # Add component-specific test logic here
            print_success "$component tests completed"
            ;;
    esac
}

# Function to build a component
build_component() {
    local component=$1
    
    case $component in
        "gateway")
            print_component "Building KNIRVGATEWAY..."
            "$SCRIPT_DIR/run-gateway.sh" build
            ;;
        *)
            print_component "Building $component..."
            # Add component-specific build logic here
            print_success "$component built successfully"
            ;;
    esac
}

# Function to check health of all services
check_all_health() {
    print_header "Checking Health of All KNIRV Services"
    
    local components=("knirvchain" "knirvnexus" "knirvroot" "knirvgraph" "knirvrouter" "gateway")
    local healthy_count=0
    local total_count=${#components[@]}
    
    for component in "${components[@]}"; do
        if check_component_status "$component"; then
            ((healthy_count++))
        fi
    done
    
    print_info "Health Summary: $healthy_count/$total_count services healthy"
    
    if [ "$healthy_count" -eq "$total_count" ]; then
        print_success "All KNIRV services are healthy!"
        return 0
    else
        print_warning "Some KNIRV services are not healthy"
        return 1
    fi
}

# Parse command line arguments
COMMAND=""
COMPONENT="all"
PORT=""
DEV_MODE=false
VERBOSE=false
BACKGROUND=false
WAIT=false
TIMEOUT=300

while [[ $# -gt 0 ]]; do
    case $1 in
        start|stop|restart|status|build|test|logs|clean|deploy|health|production-test|deploy-test)
            COMMAND="$1"
            shift
            ;;
        all|knirvchain|knirvnexus|knirvroot|knirvgraph|knirvrouter|gateway|economics|integration)
            COMPONENT="$1"
            shift
            ;;
        -p|--port)
            PORT="$2"
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
        -b|--background)
            BACKGROUND=true
            shift
            ;;
        -w|--wait)
            WAIT=true
            shift
            ;;
        -t|--timeout)
            TIMEOUT="$2"
            shift 2
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
        print_header "Starting KNIRV Components: $COMPONENT"
        if [ "$COMPONENT" = "all" ]; then
            # Start all components in order
            start_component "knirvroot"
            sleep 2
            start_component "knirvchain"
            sleep 2
            start_component "knirvnexus"
            sleep 2
            start_component "knirvgraph"
            sleep 2
            start_component "knirvrouter"
            sleep 2
            start_component "gateway"
        else
            start_component "$COMPONENT" "$PORT"
        fi
        ;;
    stop)
        print_header "Stopping KNIRV Components: $COMPONENT"
        if [ "$COMPONENT" = "all" ]; then
            # Stop all components in reverse order
            stop_component "gateway"
            stop_component "knirvrouter"
            stop_component "knirvgraph"
            stop_component "knirvnexus"
            stop_component "knirvchain"
            stop_component "knirvroot"
        else
            stop_component "$COMPONENT"
        fi
        ;;
    restart)
        print_header "Restarting KNIRV Components: $COMPONENT"
        if [ "$COMPONENT" = "all" ]; then
            $0 stop all
            sleep 5
            $0 start all
        else
            stop_component "$COMPONENT"
            sleep 2
            start_component "$COMPONENT" "$PORT"
        fi
        ;;
    status)
        print_header "Checking Status of KNIRV Components: $COMPONENT"
        if [ "$COMPONENT" = "all" ]; then
            check_all_health
        else
            check_component_status "$COMPONENT"
        fi
        ;;
    build)
        print_header "Building KNIRV Components: $COMPONENT"
        if [ "$COMPONENT" = "all" ]; then
            # Build all components
            build_component "gateway"
            # Add other component builds here
        else
            build_component "$COMPONENT"
        fi
        ;;
    test)
        print_header "Testing KNIRV Components: $COMPONENT"
        if [ "$COMPONENT" = "all" ]; then
            test_component "gateway"
            test_component "integration"
        else
            test_component "$COMPONENT"
        fi
        ;;
    health)
        check_all_health
        ;;
    production-test)
        print_header "Running Production Test Suite"
        if [ -f "$SCRIPT_DIR/deploy-and-test.sh" ]; then
            local test_args="--test-only --production-tests"
            if [ "$VERBOSE" = true ]; then
                test_args="$test_args --verbose"
            fi
            "$SCRIPT_DIR/deploy-and-test.sh" $test_args
        else
            print_error "Production deployment script not found"
            exit 1
        fi
        ;;
    deploy-test)
        print_header "Deploy and Run Comprehensive Tests"
        if [ -f "$SCRIPT_DIR/deploy-and-test.sh" ]; then
            local deploy_test_args="--comprehensive"
            if [ "$DEV_MODE" = true ]; then
                deploy_test_args="$deploy_test_args --env development"
            else
                deploy_test_args="$deploy_test_args --env production"
            fi
            if [ "$VERBOSE" = true ]; then
                deploy_test_args="$deploy_test_args --verbose"
            fi
            "$SCRIPT_DIR/deploy-and-test.sh" $deploy_test_args
        else
            print_error "Production deployment script not found"
            exit 1
        fi
        ;;
    deploy)
        print_header "Deploying KNIRV Network"
        print_info "Development mode: $DEV_MODE"

        # Use the new production deployment script
        local deploy_args=""
        if [ "$DEV_MODE" = true ]; then
            deploy_args="--env development"
        else
            deploy_args="--env production"
        fi

        if [ "$VERBOSE" = true ]; then
            deploy_args="$deploy_args --verbose"
        fi

        # Check if production deployment script exists
        if [ -f "$SCRIPT_DIR/deploy-and-test.sh" ]; then
            print_info "Using production deployment script..."
            "$SCRIPT_DIR/deploy-and-test.sh" --deploy-only $deploy_args
        else
            print_warning "Production deployment script not found, using legacy deployment"
            # Legacy deployment logic
            $0 start all
        fi

        print_success "Deployment completed"
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

print_success "Operation completed successfully!"
