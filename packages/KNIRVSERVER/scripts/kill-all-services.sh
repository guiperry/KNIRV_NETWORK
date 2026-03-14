#!/bin/bash
# Script to kill all KNIRVSERVER services
# Usage: ./kill-all-services.sh [--force]

set -e

FORCE=false
if [[ "$1" == "--force" ]]; then
    FORCE=true
fi

echo "=== Killing all KNIRVSERVER services ==="

# Function to kill process by name pattern
kill_by_pattern() {
    local pattern="$1"
    local signal="${2:-SIGTERM}"
    
    echo "Looking for processes matching: $pattern"
    
    # Get PIDs
    local pids=$(pgrep -f "$pattern" 2>/dev/null || true)
    
    if [[ -z "$pids" ]]; then
        echo "  No processes found matching '$pattern'"
        return 0
    fi
    
    echo "  Found PIDs: $pids"
    
    # Send signal
    for pid in $pids; do
        if kill -0 "$pid" 2>/dev/null; then
            echo "  Sending $signal to PID $pid"
            if ! kill -"$signal" "$pid" 2>/dev/null; then
                echo "  Warning: Failed to send $signal to PID $pid"
            fi
        fi
    done
    
    # Wait for processes to terminate
    local timeout=10
    local start_time=$(date +%s)
    
    for pid in $pids; do
        while kill -0 "$pid" 2>/dev/null; do
            local current_time=$(date +%s)
            local elapsed=$((current_time - start_time))
            
            if [[ $elapsed -ge $timeout ]]; then
                echo "  Timeout waiting for PID $pid to terminate"
                if [[ "$FORCE" == "true" ]]; then
                    echo "  Force killing PID $pid with SIGKILL"
                    kill -9 "$pid" 2>/dev/null || true
                fi
                break
            fi
            sleep 0.5
        done
    done
    
    # Final check
    local remaining_pids=$(pgrep -f "$pattern" 2>/dev/null || true)
    if [[ -n "$remaining_pids" ]]; then
        echo "  Warning: Some processes still running: $remaining_pids"
        if [[ "$FORCE" == "true" ]]; then
            echo "  Force killing remaining processes"
            kill -9 $remaining_pids 2>/dev/null || true
        fi
        return 1
    else
        echo "  All processes terminated"
        return 0
    fi
}

# Kill KNIRV-SERVER unified binary
echo ""
echo "1. Killing KNIRV-SERVER unified binary..."
kill_by_pattern "./dist/knirv-server"
kill_by_pattern "knirv-server"

# Kill backend server
echo ""
echo "2. Killing backend server..."
kill_by_pattern "backend_server"
kill_by_pattern "bin/backend_server"

# Kill container deployer
echo ""
echo "3. Killing container deployer..."
kill_by_pattern "container_deployer"

# Kill OS builder
echo ""
echo "4. Killing OS builder..."
kill_by_pattern "os_builder"

# Kill any Go processes related to KNIRV
echo ""
echo "5. Killing other KNIRV-related Go processes..."
kill_by_pattern "go run.*knirv"
kill_by_pattern "go build.*knirv"

# Kill Node.js services (Playwright test servers, etc.)
echo ""
echo "6. Killing KNIRV-related Node.js processes..."
kill_by_pattern "node.*knirv"
kill_by_pattern "node.*KNIRV"

# Kill any processes listening on KNIRV ports
echo ""
echo "7. Checking for processes on KNIRV ports..."
PORTS="8080 8081 8082 8090 4001 9080 9090"
for port in $PORTS; do
    if command -v lsof >/dev/null 2>&1; then
        pid=$(lsof -ti:$port 2>/dev/null || true)
        if [[ -n "$pid" ]]; then
            echo "  Found process $pid on port $port"
            kill -TERM $pid 2>/dev/null || true
            if [[ "$FORCE" == "true" ]]; then
                sleep 1
                kill -9 $pid 2>/dev/null || true
            fi
        fi
    elif command -v ss >/dev/null 2>&1; then
        # Alternative using ss and awk
        pid=$(ss -ltnp | grep ":$port " | awk '{print $6}' | cut -d= -f2 | cut -d, -f1 | head -1)
        if [[ -n "$pid" ]]; then
            echo "  Found process $pid on port $port"
            kill -TERM $pid 2>/dev/null || true
            if [[ "$FORCE" == "true" ]]; then
                sleep 1
                kill -9 $pid 2>/dev/null || true
            fi
        fi
    fi
done

# Clean up any temporary files
echo ""
echo "8. Cleaning up temporary files..."
find /tmp -name "*knirv*" -type f -mtime +1 -delete 2>/dev/null || true
find /tmp -name "*KNIRV*" -type f -mtime +1 -delete 2>/dev/null || true

echo ""
echo "=== Service cleanup complete ==="
echo ""
echo "Remaining KNIRV processes:"
ps aux | grep -i knirv | grep -v grep | grep -v "$0" || echo "  None"

# Check if Docker containers are running
if command -v docker >/dev/null 2>&1; then
    echo ""
    echo "=== Docker containers ==="
    docker ps --filter "name=knirv" --format "table {{.Names}}\t{{.Status}}" || true
fi