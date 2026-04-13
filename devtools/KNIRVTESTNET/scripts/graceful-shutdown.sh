#!/bin/bash
# Graceful shutdown script for KNIRV Testnet
# Properly terminates all services started by start-testnet.sh in reverse order
# This script reads the stored PID files and ensures clean shutdown

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# PID files in reverse startup order (services started last are killed first)
PID_FILES=(
    "data/knirvverifier.pid"           # 9. Formal verification (optional, started last)
    "data/health-monitor.pid"          # 8. Health Monitor
    "data/testnet-gateway.pid"         # 7. KNIRV-GATEWAY
    "data/knirvwallet.pid"             # 6. KNIRVWALLET (may not exist)
    "data/knirvrouter.pid"             # 5. KNIRV-ROUTER
    "data/knirvserver.pid"             # 4. KNIRV-SERVER (optional)
    "data/knirvgraph.pid"              # 3. KNIRVGRAPH
    "data/knirvchain.pid"              # 2. KNIRVCHAIN
    "data/knirvoracle.pid"             # 1. KNIRV-ORACLE
    "data/knirvtestnet-server.pid"     # 0. KNIRVTESTNET Server
)

# Track all PIDs for final cleanup
ALL_PIDS=""

print_status() {
    echo -e "${BLUE}[STATUS]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[⚠]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1"
}

# Gracefully kill a process
kill_process() {
    local pid=$1
    local service_name=$2
    local timeout=${3:-10}  # Default 10 seconds

    if ! kill -0 "$pid" 2>/dev/null; then
        print_warning "$service_name (PID: $pid) is not running"
        return 0
    fi

    print_status "Terminating $service_name (PID: $pid)..."

    # Send SIGTERM for graceful shutdown
    kill -TERM "$pid" 2>/dev/null || true

    # Wait for graceful shutdown
    local end_time=$((SECONDS + timeout))
    while [ $SECONDS -lt $end_time ]; do
        if ! kill -0 "$pid" 2>/dev/null; then
            print_success "$service_name terminated gracefully"
            return 0
        fi
        sleep 0.5
    done

    # If still running, force kill
    print_warning "$service_name did not terminate gracefully, force killing..."
    kill -KILL "$pid" 2>/dev/null || true

    # Verify it's dead
    sleep 1
    if kill -0 "$pid" 2>/dev/null; then
        print_error "Failed to kill $service_name (PID: $pid)"
        return 1
    fi

    print_success "$service_name force killed"
    return 0
}

# Shutdown handler for signals
shutdown_handler() {
    print_warning "Received shutdown signal, gracefully terminating all services..."
    perform_shutdown
    exit 0
}

# Main shutdown logic
perform_shutdown() {
    echo ""
    print_status "=== KNIRV Testnet Graceful Shutdown ==="
    echo ""

    local failed_count=0
    local shutdown_count=0

    # Kill services in reverse order
    for pid_file in "${PID_FILES[@]}"; do
        if [ -f "$pid_file" ]; then
            pid=$(cat "$pid_file" 2>/dev/null)
            if [ -n "$pid" ]; then
                service_name=$(basename "$pid_file" .pid)
                
                # Map PID file names to service names
                case "$service_name" in
                    "knirvoracle") service_name="KNIRV-ORACLE" ;;
                    "knirvchain") service_name="KNIRVCHAIN" ;;
                    "knirvgraph") service_name="KNIRVGRAPH" ;;
                    "knirvserver") service_name="KNIRV-SERVER" ;;
                    "knirvrouter") service_name="KNIRV-ROUTER" ;;
                    "testnet-gateway") service_name="KNIRV-GATEWAY" ;;
                    "health-monitor") service_name="Health Monitor" ;;
                    "knirvverifier") service_name="KNIRV-VERIFIER" ;;
                    "knirvwallet") service_name="KNIRVWALLET" ;;
                    "knirvtestnet-server") service_name="KNIRVTESTNET Server" ;;
                esac

                if kill_process "$pid" "$service_name"; then
                    ((shutdown_count++))
                else
                    ((failed_count++))
                    ALL_PIDS+="$pid "
                fi
            fi

            # Clean up the PID file
            rm -f "$pid_file"
        fi
    done

    echo ""
    print_status "Shutdown Summary:"
    print_success "Successfully terminated: $shutdown_count services"
    
    if [ $failed_count -gt 0 ]; then
        print_error "Failed to terminate: $failed_count services"
    fi

    # Additional cleanup: Kill any remaining KNIRV processes
    print_status "Checking for any remaining KNIRV processes..."

    local orphan_pids=$(pgrep -f "knirv|KNIRV" 2>/dev/null || true)
    if [ -n "$orphan_pids" ]; then
        print_warning "Found orphaned KNIRV processes: $orphan_pids"
        for orphan_pid in $orphan_pids; do
            print_warning "Force killing orphaned process (PID: $orphan_pid)..."
            kill -KILL "$orphan_pid" 2>/dev/null || true
        done
    fi

    # Clear socket files
    if [ -d "data" ]; then
        print_status "Cleaning up socket files..."
        find data -name "*.sock" -delete 2>/dev/null || true
    fi

    # Final verification
    sleep 1
    remaining=$(pgrep -f "knirv|KNIRV" 2>/dev/null || true)
    if [ -z "$remaining" ]; then
        print_success "All KNIRV processes terminated successfully!"
        echo ""
    else
        print_error "Warning: Some KNIRV processes still running: $remaining"
        echo ""
    fi
}

# Main entry point
main() {
    # Check if running in KNIRVTESTNET directory
    if [ ! -d "scripts" ] || [ ! -f "scripts/start-testnet.sh" ]; then
        print_error "This script must be run from the KNIRVTESTNET directory"
        exit 1
    fi

    # Set up signal handlers for graceful shutdown
    trap shutdown_handler SIGINT SIGTERM

    # Perform shutdown
    perform_shutdown

    exit 0
}

main "$@"
