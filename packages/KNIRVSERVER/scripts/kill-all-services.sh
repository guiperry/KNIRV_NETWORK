#!/bin/bash
# Script to kill all KNIRVSERVER services
# Usage: ./kill-all-services.sh [--force]
#
# NOTE: This script is AGGRESSIVE - it will use SIGKILL as fallback and searches
# in common installation directories including ~/.local/share/
#
# IMPORTANT: Many KNIRV services run as root. If you see "Operation not
# permitted" errors below, re-run with: sudo ./kill-all-services.sh

set -e

FORCE=false
if [[ "$1" == "--force" ]]; then
    FORCE=true
fi

echo "=== Killing all KNIRVSERVER services (AGGRESSIVE MODE) ==="

# Continue past individual kill failures so every service gets a chance.
set +e

# Check whether sudo is usable without interaction
SUDO_AVAILABLE=false
if command -v sudo &>/dev/null; then
    if sudo -n true 2>/dev/null; then
        SUDO_AVAILABLE=true
    fi
fi

# If not root and sudo requires a password, warn early
if [[ "$(id -u)" -ne 0 ]] && ! $SUDO_AVAILABLE; then
    echo ""
    echo "WARNING: You are not running as root, and sudo requires a password."
    echo "         Some KNIRV services run as root and cannot be killed without"
    echo "         elevated privileges. Consider re-running with:"
    echo "           sudo $0 $@"
    echo ""
fi

# ── Privilege-aware kill helper ───────────────────────────────────────────
# Tries plain kill first. Falls back to sudo kill when the process is owned
# by another user.  Always logs what happened so failures are visible.
sudo_kill() {
    local pid="$1"
    local sig="$2"

    if ! kill -0 "$pid" 2>/dev/null; then
        return 0  # already dead
    fi

    if kill "-$sig" "$pid" 2>/dev/null; then
        return 0
    fi

    # Plain kill failed — try with sudo (non-interactive, so it fails
    # immediately if a password is required rather than hanging).
    if $SUDO_AVAILABLE; then
        echo "  (escalating with sudo -n for PID $pid)"
        sudo -n kill "-$sig" "$pid" && return 0
        echo "  WARNING: sudo kill failed for PID $pid"
    fi

    echo "  WARNING: Could not kill PID $pid (owned by $(ps -o user= -p $pid 2>/dev/null || echo '?'))."
    echo "           Re-run as root: sudo $0"
    return 1
}

# ── Kill by pgrep pattern ─────────────────────────────────────────────────
kill_by_pattern() {
    local pattern="$1"
    local signal="${2:-SIGTERM}"
    local aggressive="${3:-false}"

    echo "Looking for processes matching: $pattern"

    local pids=$(pgrep -f "$pattern" 2>/dev/null || true)

    if [[ -z "$pids" ]]; then
        echo "  No processes found matching '$pattern'"
        return 0
    fi

    echo "  Found PIDs: $pids"

    # Send signal
    for pid in $pids; do
        echo "  Sending $signal to PID $pid"
        sudo_kill "$pid" "$signal" || true
    done

    # Wait for graceful termination (up to 3 s)
    local timeout=3
    local start_time=$(date +%s)

    for pid in $pids; do
        while kill -0 "$pid" 2>/dev/null; do
            local current_time=$(date +%s)
            local elapsed=$((current_time - start_time))

            if [[ $elapsed -ge $timeout ]]; then
                echo "  Timeout waiting for PID $pid to terminate"
                if [[ "$FORCE" == "true" ]] || [[ "$aggressive" == "true" ]]; then
                    echo "  Force killing PID $pid with SIGKILL"
                    sudo_kill "$pid" 9 || true
                fi
                break
            fi
            sleep 0.5
        done
    done

    # Final check — SIGKILL any survivors
    local remaining_pids=$(pgrep -f "$pattern" 2>/dev/null || true)
    if [[ -n "$remaining_pids" ]]; then
        echo "  Warning: Some processes still running: $remaining_pids"
        if [[ "$FORCE" == "true" ]] || [[ "$aggressive" == "true" ]]; then
            echo "  Force killing remaining processes with SIGKILL"
            for pid in $remaining_pids; do
                sudo_kill "$pid" 9 || true
            done
        fi
        return 1
    else
        echo "  All processes terminated"
        return 0
    fi
}

# ── Kill by binary path — iterates over executables in a directory ────────
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
                for pid in $pids; do
                    sudo_kill "$pid" 9 || true
                done
            fi
        fi
    done
}

# Kill KNIRV-SERVER unified binary (aggressive)
echo ""
echo "1. Killing KNIRV-SERVER unified binary..."
kill_by_pattern "./dist/knirv-server" "SIGTERM" "true"
kill_by_pattern "knirv-server" "SIGTERM" "true"

# Kill binaries in KNIRVSERVER runtime bin directories
echo ""
echo "1.5. Killing binaries in KNIRVSERVER runtime bin directories..."
kill_by_binary_path "/var/lib/knirvserver/bin"
kill_by_binary_path "$HOME/.local/share/knirvserver/bin"

# Kill subprocess services individually (aggressive, with graceful SIGTERM first)
# These are managed subprocesses launched by the backend — knirvoracle binds a
# Unix socket, knirvgateway proxies API calls, knirvhasher handles content hashing.
echo ""
echo "1.6. Killing KNIRV subprocess services (knirvoracle, knirvgateway, knirvhasher)..."
kill_by_pattern "knirvoracle" "SIGTERM" "true" || true
kill_by_pattern "knirvgateway" "SIGTERM" "true" || true
kill_by_pattern "knirvhasher" "SIGTERM" "true" || true

# Kill backend server
echo ""
echo "2. Killing backend server..."
kill_by_pattern "backend_server" "SIGTERM" "true"
kill_by_pattern "bin/backend_server" "SIGTERM" "true"

# Kill installer / GUI monitor
echo ""
echo "2.5. Killing installer and detached GUI monitor..."
kill_by_pattern "knirvserver" "SIGTERM" "true"

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
            sudo_kill "$pid" 9 || true
        fi
    elif command -v ss >/dev/null 2>&1; then
        # Alternative using ss and awk
        pid=$(ss -ltnp | grep ":$port " | awk '{print $6}' | cut -d= -f2 | cut -d, -f1 | head -1)
        if [[ -n "$pid" ]]; then
            echo "  Found process $pid on port $port - force killing"
            sudo_kill "$pid" 9 || true
        fi
    fi
done

# Kill any remaining knirvgraph processes specifically
echo ""
echo "8. Aggressively killing any remaining knirvgraph processes..."
kill_by_pattern "knirvgraph" "SIGTERM" "true" || true
kill_by_pattern "KNIRVGRAPH" "SIGTERM" "true" || true

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
