#!/bin/bash
set -e

# KNIRV Testnet Unified Stop Script
# This script stops all KNIRV testnet components gracefully

echo "🛑 KNIRV TESTNET SHUTDOWN"
echo "========================"
echo "Stopping KNIRV Decentralized Trusted Execution Network testnet..."
echo ""

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

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

# Function to stop a service
stop_service() {
    local name=$1
    local pid_file=$2
    
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        
        if kill -0 "$pid" 2>/dev/null; then
            print_status "Stopping $name (PID: $pid)..."
            
            # Try graceful shutdown first
            if kill -TERM "$pid" 2>/dev/null; then
                # Wait up to 10 seconds for graceful shutdown
                local count=0
                while [ $count -lt 10 ] && kill -0 "$pid" 2>/dev/null; do
                    sleep 1
                    count=$((count + 1))
                done
                
                # If still running, force kill
                if kill -0 "$pid" 2>/dev/null; then
                    print_warning "$name didn't stop gracefully, forcing shutdown..."
                    kill -KILL "$pid" 2>/dev/null || true
                fi
            fi
            
            # Verify process is stopped
            if ! kill -0 "$pid" 2>/dev/null; then
                print_success "$name stopped successfully"
                rm -f "$pid_file"
            else
                print_error "Failed to stop $name"
            fi
        else
            print_warning "$name PID file exists but process is not running"
            rm -f "$pid_file"
        fi
    else
        print_status "$name is not running (no PID file found)"
    fi
}

# Stop services in reverse order (opposite of startup)
print_status "Stopping services in reverse order..."

# 6. Stop KNIRV-GATEWAY
stop_service "KNIRV-GATEWAY" "data/knirvgateway.pid"

# 5. Stop KNIRV-ROUTER
stop_service "KNIRV-ROUTER" "data/knirvrouter.pid"

# 4. Stop KNIRV-NEXUS
stop_service "KNIRV-NEXUS" "data/knirvnexus.pid"

# 3. Stop KNIRVGRAPH
stop_service "KNIRVGRAPH" "data/knirvgraph.pid"

# 2. Stop KNIRVCHAIN
stop_service "KNIRVCHAIN" "data/knirvchain.pid"

# 1. Stop KNIRV-ROOT
stop_service "KNIRV-ROOT" "data/knirvroot.pid"

# Clean up any remaining processes
print_status "Cleaning up any remaining processes..."

# Kill any processes listening on our ports
ports=(1317 8090 8082 8084 8086 8888)
for port in "${ports[@]}"; do
    pids=$(lsof -ti:$port 2>/dev/null || true)
    if [ -n "$pids" ]; then
        print_warning "Found processes still using port $port, terminating..."
        echo "$pids" | xargs kill -TERM 2>/dev/null || true
        sleep 2
        echo "$pids" | xargs kill -KILL 2>/dev/null || true
    fi
done

# Clean up lock files and temporary data
print_status "Cleaning up temporary files..."

# Remove any lock files
find data -name "*.lock" -delete 2>/dev/null || true

# Remove any temporary database files if they exist
find data -name "*.tmp" -delete 2>/dev/null || true

# Optional: Clean up log files (commented out by default)
# print_status "Cleaning up log files..."
# rm -f logs/*.log 2>/dev/null || true

print_success "All services stopped successfully!"

echo ""
echo "🏁 KNIRV TESTNET SHUTDOWN COMPLETE"
echo "=================================="
echo ""
echo "All testnet services have been stopped."
echo "To restart the testnet, run: ./start-testnet.sh"
echo ""

# Display final status
echo "Final Status:"
if [ -f "data/knirvroot.pid" ] || [ -f "data/knirvchain.pid" ] || [ -f "data/knirvgraph.pid" ] || [ -f "data/knirvnexus.pid" ] || [ -f "data/knirvrouter.pid" ] || [ -f "data/knirvgateway.pid" ]; then
    print_warning "Some PID files still exist - manual cleanup may be required"
else
    print_success "Clean shutdown - no PID files remaining"
fi

# Check if any ports are still in use
active_ports=()
for port in "${ports[@]}"; do
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        active_ports+=($port)
    fi
done

if [ ${#active_ports[@]} -gt 0 ]; then
    print_warning "Some ports are still in use: ${active_ports[*]}"
    print_status "You may need to manually kill processes using these ports"
else
    print_success "All testnet ports are now available"
fi

echo ""
print_success "KNIRV Testnet shutdown completed!"
