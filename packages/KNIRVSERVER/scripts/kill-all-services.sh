#!/bin/bash
# Script to kill all KNIRVSERVER services
# Usage: ./kill-all-services.sh [--force]
#
# NOTE: This script is AGGRESSIVE - it will use SIGKILL as fallback and searches
# in common installation directories including ~/.local/share/

set -e

FORCE=false
if [[ "$1" == "--force" ]]; then
    FORCE=true
fi

echo "=== Killing all KNIRVSERVER services (AGGRESSIVE MODE) ==="

# Function to kill process by name pattern - AGGRESSIVE version
kill_by_pattern() {
    local pattern="$1"
    local signal="${2:-SIGTERM}"
    local aggressive="${3:-false}"
    
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
    
    # Wait for processes to terminate - shorter timeout, more aggressive
    local timeout=3
    local start_time=$(date +%s)
    
    for pid in $pids; do
        while kill -0 "$pid" 2>/dev/null; do
            local current_time=$(date +%s)
            local elapsed=$((current_time - start_time))
            
            if [[ $elapsed -ge $timeout ]]; then
                echo "  Timeout waiting for PID $pid to terminate"
                # ALWAYS force kill if FORCE mode or aggressive pattern
                if [[ "$FORCE" == "true" ]] || [[ "$aggressive" == "true" ]]; then
                    echo "  Force killing PID $pid with SIGKILL"
                    kill -9 "$pid" 2>/dev/null || true
                fi
                break
            fi
            sleep 0.5
        done
    done
    
    # Final check - ALWAYS force kill remaining if aggressive
    local remaining_pids=$(pgrep -f "$pattern" 2>/dev/null || true)
    if [[ -n "$remaining_pids" ]]; then
        echo "  Warning: Some processes still running: $remaining_pids"
        if [[ "$FORCE" == "true" ]] || [[ "$aggressive" == "true" ]]; then
            echo "  Force killing remaining processes with SIGKILL"
            kill -9 $remaining_pids 2>/dev/null || true
        fi
        return 1
    else
        echo "  All processes terminated"
        return 0
    fi
}

# Kill processes by binary path (handles ~/.local/share/knirvserver/bin/*)
kill_by_binary_path() {
    local bin_dir="$1"
    
    if [[ ! -d "$bin_dir" ]]; then
        return 0
    fi
    
    echo "Searching in binary directory: $bin_dir"
    
    for binary in "$bin_dir"/*; do
        if [[ -x "$binary" ]]; then
            local bin_name=$(basename "$binary")
            echo "  Checking for: $bin_name"
            local pids=$(pgrep -f "$bin_name" 2>/dev/null || true)
            if [[ -n "$pids" ]]; then
                echo "    Found PIDs: $pids - force killing"
                kill -9 $pids 2>/dev/null || true
            fi
        fi
    done
}

# Kill KNIRV-SERVER unified binary (aggressive)
echo ""
echo "1. Killing KNIRV-SERVER unified binary..."
kill_by_pattern "./dist/knirv-server" "SIGTERM" "true"
kill_by_pattern "knirv-server" "SIGTERM" "true"

# Kill binaries in ~/.local/share/knirvserver/bin/
echo ""
echo "1.5. Killing binaries in ~/.local/share/knirvserver/bin/..."
kill_by_binary_path "$HOME/.local/share/knirvserver/bin"
kill_by_binary_path "/home/gperry/.local/share/knirvserver/bin"

# Kill backend server
echo ""
echo "2. Killing backend server..."
kill_by_pattern "backend_server" "SIGTERM" "true"
kill_by_pattern "bin/backend_server" "SIGTERM" "true"

# Kill container deployer
echo ""
echo "3. Killing container deployer..."
kill_by_pattern "container_deployer" "SIGTERM" "true"

# Kill OS builder
echo ""
echo "4. Killing OS builder..."
kill_by_pattern "os_builder" "SIGTERM" "true"

# Kill any Go processes related to KNIRV (aggressive)
echo ""
echo "5. Killing other KNIRV-related Go processes..."
kill_by_pattern "go run.*knirv" "SIGTERM" "true"
kill_by_pattern "go build.*knirv" "SIGTERM" "true"

# Kill Node.js services (Playwright test servers, etc.)
echo ""
echo "6. Killing KNIRV-related Node.js processes..."
kill_by_pattern "node.*knirv" "SIGTERM" "true"
kill_by_pattern "node.*KNIRV" "SIGTERM" "true"

# Kill any processes listening on KNIRV ports (aggressive)
echo ""
echo "7. Checking for processes on KNIRV ports..."
PORTS="8080 8081 8082 8084 8086 8090 4001 9080 9090 9002 7090"
for port in $PORTS; do
    if command -v lsof >/dev/null 2>&1; then
        pid=$(lsof -ti:$port 2>/dev/null || true)
        if [[ -n "$pid" ]]; then
            echo "  Found process $pid on port $port - force killing"
            kill -9 $pid 2>/dev/null || true
        fi
    elif command -v ss >/dev/null 2>&1; then
        # Alternative using ss and awk
        pid=$(ss -ltnp | grep ":$port " | awk '{print $6}' | cut -d= -f2 | cut -d, -f1 | head -1)
        if [[ -n "$pid" ]]; then
            echo "  Found process $pid on port $port - force killing"
            kill -9 $pid 2>/dev/null || true
        fi
    fi
done

# Kill any remaining knirvgraph processes specifically
echo ""
echo "8. Aggressively killing any remaining knirvgraph processes..."
pkill -9 -f "knirvgraph" 2>/dev/null || true
pkill -9 -f "KNIRVGRAPH" 2>/dev/null || true

# Clean up any temporary files
echo ""
echo "9. Cleaning up temporary files..."
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