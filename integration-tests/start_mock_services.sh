#!/bin/bash

# Start Mock Services for Integration Testing
# This script starts mock versions of all KNIRV services for testing

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PID_DIR="$SCRIPT_DIR/mock_pids"

# Service configuration
declare -A SERVICES=(
    ["KNIRVCHAIN"]="8080"
    ["KNIRVGRAPH"]="8081"
    ["KNIRVNEXUS"]="8082"
    ["KNIRVWALLET"]="8083"
    ["KNIRVSHELL"]="8084"
    ["KNIRVROUTER"]="8085"
    ["KNIRVROOT"]="8086"
)

# Function to print colored output
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

# Function to check if port is in use
port_in_use() {
    lsof -i :"$1" >/dev/null 2>&1
}

# Function to start a mock service
start_mock_service() {
    local service_name=$1
    local port=$2
    
    print_status "Starting mock $service_name on port $port..."
    
    # Check if port is already in use
    if port_in_use "$port"; then
        print_warning "Port $port is already in use, killing existing process..."
        lsof -ti :"$port" | xargs kill -9 2>/dev/null || true
        sleep 1
    fi
    
    # Start the mock service
    cd "$SCRIPT_DIR/mock"
    go run mock_server.go "$port" "$service_name" > "../mock_logs/${service_name}.log" 2>&1 &
    local pid=$!
    
    # Save PID
    echo "$pid" > "$PID_DIR/${service_name}.pid"
    
    # Wait a moment and check if service started
    sleep 2
    if kill -0 "$pid" 2>/dev/null; then
        print_success "Mock $service_name started successfully (PID: $pid)"
        
        # Test health endpoint
        local health_url="http://localhost:$port/health"
        if curl -s "$health_url" >/dev/null 2>&1; then
            print_success "Mock $service_name health check passed"
        else
            print_warning "Mock $service_name health check failed"
        fi
    else
        print_error "Failed to start mock $service_name"
        return 1
    fi
}

# Function to stop all mock services
stop_mock_services() {
    print_status "Stopping all mock services..."
    
    if [ -d "$PID_DIR" ]; then
        for pid_file in "$PID_DIR"/*.pid; do
            if [ -f "$pid_file" ]; then
                local service_name=$(basename "$pid_file" .pid)
                local pid=$(cat "$pid_file")
                
                if kill -0 "$pid" 2>/dev/null; then
                    print_status "Stopping mock $service_name (PID: $pid)..."
                    kill "$pid" 2>/dev/null || true
                    sleep 1
                    
                    # Force kill if still running
                    if kill -0 "$pid" 2>/dev/null; then
                        kill -9 "$pid" 2>/dev/null || true
                    fi
                fi
                
                rm -f "$pid_file"
            fi
        done
        
        rmdir "$PID_DIR" 2>/dev/null || true
    fi
    
    print_success "All mock services stopped"
}

# Function to check service status
check_services_status() {
    print_status "Checking mock services status..."
    
    local all_healthy=true
    
    for service_name in "${!SERVICES[@]}"; do
        local port=${SERVICES[$service_name]}
        local health_url="http://localhost:$port/health"
        
        if curl -s "$health_url" >/dev/null 2>&1; then
            print_success "Mock $service_name (port $port): HEALTHY"
        else
            print_error "Mock $service_name (port $port): UNHEALTHY"
            all_healthy=false
        fi
    done
    
    if [ "$all_healthy" = true ]; then
        print_success "All mock services are healthy"
        return 0
    else
        print_error "Some mock services are unhealthy"
        return 1
    fi
}

# Function to display usage
usage() {
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  start     Start all mock services (default)"
    echo "  stop      Stop all mock services"
    echo "  restart   Restart all mock services"
    echo "  status    Check status of all mock services"
    echo "  help      Show this help message"
    echo ""
    echo "Mock Services:"
    for service_name in "${!SERVICES[@]}"; do
        echo "  • $service_name: http://localhost:${SERVICES[$service_name]}"
    done
}

# Main execution
main() {
    local command=${1:-start}
    
    case $command in
        start)
            print_status "Starting KNIRV mock services for integration testing..."
            
            # Create necessary directories
            mkdir -p "$PID_DIR"
            mkdir -p "$SCRIPT_DIR/mock_logs"
            
            # Start all services
            for service_name in "${!SERVICES[@]}"; do
                start_mock_service "$service_name" "${SERVICES[$service_name]}"
            done
            
            # Final status check
            sleep 3
            check_services_status
            
            print_success "All mock services started successfully!"
            print_status "You can now run integration tests with: go test -v"
            ;;
        stop)
            stop_mock_services
            ;;
        restart)
            stop_mock_services
            sleep 2
            main start
            ;;
        status)
            check_services_status
            ;;
        help)
            usage
            ;;
        *)
            print_error "Unknown command: $command"
            usage
            exit 1
            ;;
    esac
}

# Handle script interruption
cleanup() {
    print_status "Received interrupt signal, stopping mock services..."
    stop_mock_services
    exit 0
}

trap cleanup INT TERM

# Run main function
main "$@"
