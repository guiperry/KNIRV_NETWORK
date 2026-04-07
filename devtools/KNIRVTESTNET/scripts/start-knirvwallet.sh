#!/bin/bash

# KNIRV TESTNET - Unified KNIRVWALLET Starter
# Automatically detects and starts either real or demo KNIRVWALLET

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTNET_ROOT="$(dirname "$SCRIPT_DIR")"
PROJECT_ROOT="$(dirname "$TESTNET_ROOT")"

# Configuration
REAL_CONTROLLER_PORT=8088
DEMO_CONTROLLER_PORT=8089
CONTROLLER_PID_FILE="$TESTNET_ROOT/data/knirvwallet.pid"
CONTROLLER_LOG_FILE="$TESTNET_ROOT/logs/knirvwallet.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m'

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

print_choice() {
    echo -e "${PURPLE}[CHOICE]${NC} $1"
}

# Check if real KNIRVWALLET is available
check_real_controller_available() {
    local controller_dir="$PROJECT_ROOT/packages/KNIRVWALLET"
    
    if [ ! -d "$controller_dir" ]; then
        return 1
    fi
    
    if [ ! -f "$controller_dir/package.json" ]; then
        return 1
    fi
    
    # Check if dependencies are installed
    if [ ! -d "$controller_dir/node_modules" ]; then
        return 1
    fi
    
    return 0
}

# Start real KNIRVWALLET
start_real_controller() {
    print_info "🚀 Starting Real KNIRVWALLET..."
    
    local controller_dir="$PROJECT_ROOT/packages/KNIRVWALLET"
    local config_script="$TESTNET_ROOT/config/knirvwallet/start-real-controller.sh"
    
    if [ ! -f "$config_script" ]; then
        print_error "Real KNIRVWALLET not built. Run: ./scripts/build-knirvwallet.sh"
        return 1
    fi
    
    # Start real controller in background
    cd "$controller_dir"
    
    # Set environment variables
    export NODE_ENV=testnet
    export KNIRV_TESTNET_MODE=true
    export PORT="$REAL_CONTROLLER_PORT"
    export API_PORT="$DEMO_CONTROLLER_PORT"
    
    # Start the service
    nohup bash "$config_script" > "$CONTROLLER_LOG_FILE" 2>&1 &
    local pid=$!
    echo "$pid" > "$CONTROLLER_PID_FILE"
    
    # Wait for service to start
    sleep 5
    
    # Verify service is running
    if curl -s --max-time 10 "http://localhost:$REAL_CONTROLLER_PORT" > /dev/null 2>&1; then
        print_success "Real KNIRVWALLET started successfully!"
        print_info "Frontend: http://localhost:$REAL_CONTROLLER_PORT"
        print_info "API: http://localhost:$DEMO_CONTROLLER_PORT"
        print_info "PID: $pid"
        print_info "Log: $CONTROLLER_LOG_FILE"
        return 0
    else
        print_warning "Real KNIRVWALLET failed to start, falling back to demo..."
        kill "$pid" 2>/dev/null || true
        rm -f "$CONTROLLER_PID_FILE"
        return 1
    fi
}

# Start demo KNIRVWALLET
start_demo_controller() {
    print_info "🎮 Starting Demo KNIRVWALLET..."
    
    # Use the demo controller script
    if "$SCRIPT_DIR/start-demo-knirvwallet.sh" start; then
        print_success "Demo KNIRVWALLET started successfully!"
        return 0
    else
        print_error "Failed to start Demo KNIRVWALLET"
        return 1
    fi
}

# Auto-detect and start appropriate controller
auto_start_controller() {
    print_info "🔍 Auto-detecting KNIRVWALLET configuration..."
    
    # Check if any controller is already running
    if curl -s --max-time 2 "http://localhost:$REAL_CONTROLLER_PORT" > /dev/null 2>&1; then
        print_success "Real KNIRVWALLET already running on port $REAL_CONTROLLER_PORT"
        return 0
    fi
    
    if curl -s --max-time 2 "http://localhost:$DEMO_CONTROLLER_PORT" > /dev/null 2>&1; then
        print_success "Demo KNIRVWALLET already running on port $DEMO_CONTROLLER_PORT"
        return 0
    fi
    
    # Try real controller first if available
    if check_real_controller_available; then
        print_choice "Real KNIRVWALLET available - attempting to start..."
        if start_real_controller; then
            return 0
        fi
    fi
    
    # Fall back to demo controller
    print_choice "Using Demo KNIRVWALLET for testnet development"
    start_demo_controller
}

# Stop any running controller
stop_controller() {
    print_info "🛑 Stopping KNIRVWALLET services..."
    
    # Stop real controller
    if [ -f "$CONTROLLER_PID_FILE" ]; then
        local pid=$(cat "$CONTROLLER_PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid"
            print_success "Real KNIRVWALLET stopped (PID: $pid)"
        fi
        rm -f "$CONTROLLER_PID_FILE"
    fi
    
    # Stop demo controller
    "$SCRIPT_DIR/start-demo-knirvwallet.sh" stop
    
    # Kill any remaining processes on controller ports
    for port in "$REAL_CONTROLLER_PORT" "$DEMO_CONTROLLER_PORT"; do
        local pids=$(lsof -ti:$port 2>/dev/null || true)
        if [ -n "$pids" ]; then
            echo "$pids" | xargs kill -9 2>/dev/null || true
            print_info "Killed processes on port $port"
        fi
    done
    
    print_success "All KNIRVWALLET services stopped"
}

# Show controller status
show_status() {
    print_info "📊 KNIRVWALLET Status:"
    
    # Check real controller
    if curl -s --max-time 2 "http://localhost:$REAL_CONTROLLER_PORT/health" > /dev/null 2>&1; then
        print_success "Real KNIRVWALLET: Running on port $REAL_CONTROLLER_PORT"
        curl -s "http://localhost:$REAL_CONTROLLER_PORT/health" | jq . 2>/dev/null || echo "  Health check response received"
    else
        print_warning "Real KNIRVWALLET: Not running"
    fi
    
    # Check demo controller
    if curl -s --max-time 2 "http://localhost:$DEMO_CONTROLLER_PORT/health" > /dev/null 2>&1; then
        print_success "Demo KNIRVWALLET: Running on port $DEMO_CONTROLLER_PORT"
        curl -s "http://localhost:$DEMO_CONTROLLER_PORT/health" | jq . 2>/dev/null || echo "  Health check response received"
    else
        print_warning "Demo KNIRVWALLET: Not running"
    fi
    
    # Show availability
    if check_real_controller_available; then
        print_info "Real KNIRVWALLET: Available for deployment"
    else
        print_warning "Real KNIRVWALLET: Not available (not built or missing dependencies)"
    fi
}

# Main execution
case "${1:-auto}" in
    "auto")
        auto_start_controller
        ;;
    "real")
        if check_real_controller_available; then
            start_real_controller
        else
            print_error "Real KNIRVWALLET not available. Run: ./scripts/build-knirvwallet.sh"
            exit 1
        fi
        ;;
    "demo")
        start_demo_controller
        ;;
    "stop")
        stop_controller
        ;;
    "status")
        show_status
        ;;
    "help"|"--help"|"-h")
        echo "KNIRV TESTNET - KNIRVWALLET Management"
        echo ""
        echo "Usage: $0 [COMMAND]"
        echo ""
        echo "Commands:"
        echo "  auto    Auto-detect and start appropriate controller (default)"
        echo "  real    Start real KNIRVWALLET (requires build)"
        echo "  demo    Start demo KNIRVWALLET"
        echo "  stop    Stop all KNIRVWALLET services"
        echo "  status  Show controller status"
        echo "  help    Show this help message"
        echo ""
        echo "Examples:"
        echo "  $0                    # Auto-start (real if available, demo otherwise)"
        echo "  $0 real              # Force start real KNIRVWALLET"
        echo "  $0 demo              # Force start demo KNIRVWALLET"
        echo "  $0 stop              # Stop all controllers"
        echo ""
        echo "Ports:"
        echo "  Real Controller:     http://localhost:$REAL_CONTROLLER_PORT"
        echo "  Demo Controller:     http://localhost:$DEMO_CONTROLLER_PORT"
        echo ""
        echo "Setup:"
        echo "  1. Build real controller: ./scripts/build-knirvwallet.sh"
        echo "  2. Start testnet: ./scripts/start-testnet.sh"
        echo "  3. Controller will auto-start with testnet"
        ;;
    *)
        print_error "Unknown command: $1"
        echo "Use '$0 help' for usage information"
        exit 1
        ;;
esac
