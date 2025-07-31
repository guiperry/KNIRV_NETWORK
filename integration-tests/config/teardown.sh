#!/bin/bash

# KNIRV Integration Test Teardown Script
# This script cleans up the test environment and stops all services

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TEST_DIR="$PROJECT_ROOT/integration-tests"

# Default values
PRESERVE_LOGS=true
PRESERVE_DATA=false
FORCE_KILL=false
VERBOSE=false

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

# Function to check if a process is running
process_running() {
    kill -0 "$1" 2>/dev/null
}

# Function to stop service by PID
stop_service() {
    local service_name=$1
    local pid_file="$TEST_DIR/pids/$service_name.pid"
    
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if process_running "$pid"; then
            print_status "Stopping $service_name (PID: $pid)..."
            
            # Try graceful shutdown first
            kill -TERM "$pid" 2>/dev/null || true
            
            # Wait for graceful shutdown
            local timeout=10
            local elapsed=0
            while [ $elapsed -lt $timeout ] && process_running "$pid"; do
                sleep 1
                elapsed=$((elapsed + 1))
            done
            
            # Force kill if still running
            if process_running "$pid"; then
                if [ "$FORCE_KILL" = true ]; then
                    print_warning "Force killing $service_name (PID: $pid)"
                    kill -KILL "$pid" 2>/dev/null || true
                else
                    print_warning "$service_name (PID: $pid) did not stop gracefully"
                fi
            else
                print_success "$service_name stopped successfully"
            fi
        else
            print_status "$service_name was not running"
        fi
        
        # Remove PID file
        rm -f "$pid_file"
    else
        print_status "No PID file found for $service_name"
    fi
}

# Function to stop services by port
stop_services_by_port() {
    print_status "Stopping services by port..."
    
    local ports=(8080 8081 8082 8083 8084 8085 8086)
    
    for port in "${ports[@]}"; do
        local pids=$(lsof -ti :"$port" 2>/dev/null || true)
        if [ -n "$pids" ]; then
            print_status "Stopping processes on port $port..."
            for pid in $pids; do
                if process_running "$pid"; then
                    print_status "Killing process $pid on port $port"
                    kill -TERM "$pid" 2>/dev/null || true
                    sleep 2
                    if process_running "$pid"; then
                        kill -KILL "$pid" 2>/dev/null || true
                    fi
                fi
            done
        fi
    done
}

# Function to stop all KNIRV services
stop_services() {
    print_status "Stopping KNIRV services..."
    
    # Stop services by PID files
    local services=("knirvchain" "knirvgraph" "knirvnexus" "knirvroot" "knirvrouter")
    
    for service in "${services[@]}"; do
        stop_service "$service"
    done
    
    # Also stop by port as backup
    stop_services_by_port
    
    # Clean up PID directory
    if [ -d "$TEST_DIR/pids" ]; then
        rm -rf "$TEST_DIR/pids"
    fi
    
    print_success "All services stopped"
}

# Function to clean up test data
cleanup_test_data() {
    if [ "$PRESERVE_DATA" = true ]; then
        print_status "Preserving test data"
        return
    fi
    
    print_status "Cleaning up test data..."
    
    # Remove test databases
    if [ -d "$TEST_DIR/data" ]; then
        rm -rf "$TEST_DIR/data"
        print_success "Test data cleaned up"
    fi
    
    # Remove temporary files
    find "$TEST_DIR" -name "*.tmp" -delete 2>/dev/null || true
    find "$TEST_DIR" -name "*.lock" -delete 2>/dev/null || true
    
    # Clean up component-specific data
    rm -rf "$PROJECT_ROOT/KNIRVCHAIN/sledchain.db" 2>/dev/null || true
    rm -rf "$PROJECT_ROOT/KNIRVGRAPH/data" 2>/dev/null || true
    rm -rf "$PROJECT_ROOT/KNIRVNEXUS/data" 2>/dev/null || true
    rm -rf "$PROJECT_ROOT/KNIRVROOT/data" 2>/dev/null || true
    rm -rf "$PROJECT_ROOT/KNIRVROUTER/data" 2>/dev/null || true
}

# Function to clean up logs
cleanup_logs() {
    if [ "$PRESERVE_LOGS" = true ]; then
        print_status "Preserving test logs"
        return
    fi
    
    print_status "Cleaning up test logs..."
    
    if [ -d "$TEST_DIR/logs" ]; then
        rm -rf "$TEST_DIR/logs"
        print_success "Test logs cleaned up"
    fi
}

# Function to clean up Docker containers (if any)
cleanup_docker() {
    print_status "Cleaning up Docker containers..."
    
    # Stop and remove KNIRV-related containers
    local containers=$(docker ps -a --filter "name=knirv" --format "{{.Names}}" 2>/dev/null || true)
    
    if [ -n "$containers" ]; then
        for container in $containers; do
            print_status "Stopping and removing container: $container"
            docker stop "$container" 2>/dev/null || true
            docker rm "$container" 2>/dev/null || true
        done
        print_success "Docker containers cleaned up"
    else
        print_status "No KNIRV Docker containers found"
    fi
}

# Function to reset environment variables
reset_environment() {
    print_status "Resetting environment variables..."
    
    unset KNIRV_TEST_MODE
    unset KNIRV_TEST_CONFIG
    unset KNIRV_TEST_DATA_DIR
    unset KNIRV_TEST_LOGS_DIR
    
    print_success "Environment variables reset"
}

# Function to generate cleanup report
generate_cleanup_report() {
    local report_file="$TEST_DIR/reports/cleanup-report-$(date +%Y%m%d-%H%M%S).txt"
    
    mkdir -p "$TEST_DIR/reports"
    
    {
        echo "KNIRV Integration Test Cleanup Report"
        echo "Generated: $(date)"
        echo "========================================"
        echo ""
        echo "Services stopped:"
        echo "- KNIRVCHAIN (port 8080)"
        echo "- KNIRVGRAPH (port 8081)"
        echo "- KNIRVNEXUS (port 8082)"
        echo "- KNIRVROOT (port 8086)"
        echo "- KNIRVROUTER (port 8085)"
        echo ""
        echo "Cleanup actions:"
        echo "- Logs preserved: $PRESERVE_LOGS"
        echo "- Data preserved: $PRESERVE_DATA"
        echo "- Force kill used: $FORCE_KILL"
        echo ""
        echo "Remaining processes on test ports:"
        for port in 8080 8081 8082 8083 8084 8085 8086; do
            local pids=$(lsof -ti :"$port" 2>/dev/null || true)
            if [ -n "$pids" ]; then
                echo "- Port $port: $pids"
            else
                echo "- Port $port: clean"
            fi
        done
        echo ""
        echo "Cleanup completed at: $(date)"
    } > "$report_file"
    
    print_success "Cleanup report generated: $report_file"
}

# Function to verify cleanup
verify_cleanup() {
    print_status "Verifying cleanup..."
    
    local issues=0
    
    # Check for remaining processes on test ports
    local ports=(8080 8081 8082 8083 8084 8085 8086)
    for port in "${ports[@]}"; do
        local pids=$(lsof -ti :"$port" 2>/dev/null || true)
        if [ -n "$pids" ]; then
            print_warning "Processes still running on port $port: $pids"
            issues=$((issues + 1))
        fi
    done
    
    # Check for remaining PID files
    if [ -d "$TEST_DIR/pids" ] && [ "$(ls -A "$TEST_DIR/pids" 2>/dev/null)" ]; then
        print_warning "PID files still exist in $TEST_DIR/pids"
        issues=$((issues + 1))
    fi
    
    if [ $issues -eq 0 ]; then
        print_success "Cleanup verification passed"
    else
        print_warning "Cleanup verification found $issues issues"
        if [ "$FORCE_KILL" = false ]; then
            print_status "Run with --force-kill to forcefully terminate remaining processes"
        fi
    fi
}

# Function to display usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --no-preserve-logs   Remove test logs during cleanup"
    echo "  --preserve-data      Keep test data after cleanup"
    echo "  --force-kill         Force kill processes that don't stop gracefully"
    echo "  --verbose            Enable verbose output"
    echo "  --help               Show this help message"
    echo ""
    echo "This script will:"
    echo "  1. Stop all KNIRV services"
    echo "  2. Clean up test data (unless --preserve-data is used)"
    echo "  3. Clean up logs (unless --no-preserve-logs is used)"
    echo "  4. Reset environment variables"
    echo "  5. Generate cleanup report"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --no-preserve-logs)
            PRESERVE_LOGS=false
            shift
            ;;
        --preserve-data)
            PRESERVE_DATA=true
            shift
            ;;
        --force-kill)
            FORCE_KILL=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            set -x
            shift
            ;;
        --help)
            usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Main execution
main() {
    print_status "Starting KNIRV Integration Test Teardown"
    print_status "Test directory: $TEST_DIR"
    
    stop_services
    cleanup_docker
    cleanup_test_data
    cleanup_logs
    reset_environment
    generate_cleanup_report
    verify_cleanup
    
    print_success "Integration test teardown completed!"
    
    if [ "$PRESERVE_LOGS" = true ]; then
        print_status "Test logs are preserved in: $TEST_DIR/logs"
    fi
    
    if [ "$PRESERVE_DATA" = true ]; then
        print_status "Test data is preserved in: $TEST_DIR/data"
    fi
}

# Run main function
main "$@"
