#!/bin/bash

# KNIRV Network API Gateway Startup Script
# This script starts the API Gateway and manages its lifecycle

set -e

# Configuration
GATEWAY_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATEWAY_BINARY="$GATEWAY_DIR/gateway"
CONFIG_FILE="$GATEWAY_DIR/config.yaml"
PID_FILE="$GATEWAY_DIR/gateway.pid"
LOG_FILE="$GATEWAY_DIR/gateway.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[GATEWAY]${NC} $1"
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

# Function to check if gateway is running
is_running() {
    if [ -f "$PID_FILE" ]; then
        local pid=$(cat "$PID_FILE")
        if ps -p "$pid" > /dev/null 2>&1; then
            return 0
        else
            rm -f "$PID_FILE"
            return 1
        fi
    fi
    return 1
}

# Function to start the gateway
start_gateway() {
    print_status "Starting KNIRV Network API Gateway..."
    
    if is_running; then
        print_warning "Gateway is already running (PID: $(cat "$PID_FILE"))"
        return 0
    fi
    
    # Check if binary exists
    if [ ! -f "$GATEWAY_BINARY" ]; then
        print_status "Gateway binary not found. Building..."
        cd "$GATEWAY_DIR/.."
        go build -o api-gateway/gateway api-gateway/gateway.go
        if [ $? -ne 0 ]; then
            print_error "Failed to build gateway binary"
            exit 1
        fi
        print_success "Gateway binary built successfully"
    fi
    
    # Start the gateway in background
    nohup "$GATEWAY_BINARY" > "$LOG_FILE" 2>&1 &
    local pid=$!
    echo "$pid" > "$PID_FILE"
    
    # Wait a moment and check if it's still running
    sleep 2
    if is_running; then
        print_success "Gateway started successfully (PID: $pid)"
        print_status "Gateway is running on http://localhost:8000"
        print_status "Health check: http://localhost:8000/gateway/health"
        print_status "Metrics: http://localhost:8000/gateway/metrics"
        print_status "WebSocket: ws://localhost:8000/gateway/ws"
        print_status "Log file: $LOG_FILE"
    else
        print_error "Gateway failed to start. Check log file: $LOG_FILE"
        exit 1
    fi
}

# Function to stop the gateway
stop_gateway() {
    print_status "Stopping KNIRV Network API Gateway..."
    
    if ! is_running; then
        print_warning "Gateway is not running"
        return 0
    fi
    
    local pid=$(cat "$PID_FILE")
    kill "$pid"
    
    # Wait for graceful shutdown
    local count=0
    while is_running && [ $count -lt 10 ]; do
        sleep 1
        count=$((count + 1))
    done
    
    if is_running; then
        print_warning "Gateway didn't stop gracefully, forcing shutdown..."
        kill -9 "$pid"
        sleep 1
    fi
    
    rm -f "$PID_FILE"
    print_success "Gateway stopped successfully"
}

# Function to restart the gateway
restart_gateway() {
    stop_gateway
    sleep 1
    start_gateway
}

# Function to show gateway status
status_gateway() {
    if is_running; then
        local pid=$(cat "$PID_FILE")
        print_success "Gateway is running (PID: $pid)"
        
        # Try to get health status
        if command -v curl > /dev/null 2>&1; then
            print_status "Checking health status..."
            curl -s http://localhost:8000/gateway/health | jq . 2>/dev/null || echo "Health check failed or jq not available"
        fi
    else
        print_warning "Gateway is not running"
    fi
}

# Function to show logs
logs_gateway() {
    if [ -f "$LOG_FILE" ]; then
        tail -f "$LOG_FILE"
    else
        print_error "Log file not found: $LOG_FILE"
    fi
}

# Function to show help
show_help() {
    echo "KNIRV Network API Gateway Management Script"
    echo ""
    echo "Usage: $0 {start|stop|restart|status|logs|help}"
    echo ""
    echo "Commands:"
    echo "  start    - Start the API Gateway"
    echo "  stop     - Stop the API Gateway"
    echo "  restart  - Restart the API Gateway"
    echo "  status   - Show gateway status"
    echo "  logs     - Show gateway logs (tail -f)"
    echo "  help     - Show this help message"
    echo ""
    echo "Files:"
    echo "  Binary:  $GATEWAY_BINARY"
    echo "  Config:  $CONFIG_FILE"
    echo "  PID:     $PID_FILE"
    echo "  Logs:    $LOG_FILE"
}

# Main script logic
case "${1:-}" in
    start)
        start_gateway
        ;;
    stop)
        stop_gateway
        ;;
    restart)
        restart_gateway
        ;;
    status)
        status_gateway
        ;;
    logs)
        logs_gateway
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        print_error "Invalid command: ${1:-}"
        echo ""
        show_help
        exit 1
        ;;
esac
