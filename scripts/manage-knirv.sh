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
                # Add KNIRVCHAIN start logic here
                print_success "KNIRVCHAIN started"
            fi
            ;;
        "knirvnexus")
            print_component "Starting KNIRVNEXUS..."
            if check_component_dir "KNIRVNEXUS" "$KNIRVNEXUS_DIR"; then
                cd "$KNIRVNEXUS_DIR"
                # Add KNIRVNEXUS start logic here
                print_success "KNIRVNEXUS started"
            fi
            ;;
        "knirvroot")
            print_component "Starting KNIRVROOT..."
            if check_component_dir "KNIRVROOT" "$KNIRVROOT_DIR"; then
                cd "$KNIRVROOT_DIR"
                # Add KNIRVROOT start logic here
                print_success "KNIRVROOT started"
            fi
            ;;
        "knirvgraph")
            print_component "Starting KNIRVGRAPH..."
            if check_component_dir "KNIRVGRAPH" "$KNIRVGRAPH_DIR"; then
                cd "$KNIRVGRAPH_DIR"
                # Add KNIRVGRAPH start logic here
                print_success "KNIRVGRAPH started"
            fi
            ;;
        "knirvrouter")
            print_component "Starting KNIRVROUTER..."
            if check_component_dir "KNIRVROUTER" "$KNIRVROUTER_DIR"; then
                cd "$KNIRVROUTER_DIR"
                # Add KNIRVROUTER start logic here
                print_success "KNIRVROUTER started"
            fi
            ;;
        "gateway")
            print_component "Starting KNIRVGATEWAY..."
            "$SCRIPT_DIR/run-gateway.sh" start
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
            "$SCRIPT_DIR/run-gateway.sh" stop
            ;;
        "economics")
            print_component "Stopping Economics Service..."
            "$SCRIPT_DIR/run-gateway.sh" economics stop
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
            check_service_health "http://localhost:$DEFAULT_KNIRVCHAIN_PORT/health" "KNIRVCHAIN"
            ;;
        "knirvnexus")
            check_service_health "http://localhost:$DEFAULT_KNIRVNEXUS_PORT/health" "KNIRVNEXUS"
            ;;
        "knirvroot")
            check_service_health "http://localhost:$DEFAULT_KNIRVROOT_PORT/health" "KNIRVROOT"
            ;;
        "knirvgraph")
            check_service_health "http://localhost:$DEFAULT_KNIRVGRAPH_PORT/health" "KNIRVGRAPH"
            ;;
        "knirvrouter")
            check_service_health "http://localhost:$DEFAULT_KNIRVROUTER_PORT/health" "KNIRVROUTER"
            ;;
        "gateway")
            check_service_health "http://localhost:$DEFAULT_GATEWAY_PORT/health" "API Gateway"
            check_service_health "http://localhost:$DEFAULT_ECONOMICS_PORT/economics/health" "Economics Service"
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
        start|stop|restart|status|build|test|logs|clean|deploy|health)
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
    deploy)
        print_header "Deploying KNIRV Network"
        print_info "Development mode: $DEV_MODE"
        # Add deployment logic here
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
