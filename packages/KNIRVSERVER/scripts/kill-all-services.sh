#!/bin/bash
# Script to kill all KNIRVSERVER services
# Usage: ./kill-all-services.sh [--force]
#
# Services handled (what the launcher and backend_server start):
#   launcher        knirv-server
#   launcher-owned  backend_server, ulorad (KNIRVULORA), knirvagent, knirvllama and its
#                   llama-server child, text-embedder, ipfs, xiond (Xion), the
#                   Transaction Chain (node main.js) and Validation Chain, KNIRVMONITOR
#                   (in-process)
#   backend-owned   knirvmodels (HASHER, HEART, HEAD), knirvgateway, knirvgraph,
#                   knirvchain, knirvoracle, KNIRVARENA (in-process)
#   data pipeline   data-connector, data-mapper, data-encoder, data-seeder,
#                   data-trainer, head-trainer; ASIC driver hasher-host / hasher-server
#   other           container_deployer, os_builder, a stale pre-rename knirvhasher
#
# Ctrl+C on a running knirv-server already stops all of these: the launcher stops each
# service in order and then ends any descendant still alive. Use this script for a
# server that was killed uncleanly or started elsewhere.
#
# The process checks in this script are deliberately conservative.  A matching
# port or a matching word in an arbitrary command line is not enough to kill a
# process: it must also be a KNIRV executable/command.
#
# IMPORTANT: Many KNIRV services run as root. If you see "Operation not
# permitted" errors below, re-run with: sudo ./kill-all-services.sh

set -e

FORCE=false
if [[ "$1" == "--force" ]]; then
    FORCE=true
fi

echo "=== Stopping all KNIRVSERVER services ==="

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

# Return success only for a process which is safe for this script to manage.
#
# Do not remove these checks in favour of a bare `pgrep -f ... | xargs kill`.
# This script is commonly started from a terminal, so its own path (and the
# path of the shell that started it) can contain KNIRVSERVER.  More
# importantly, ports such as 8080 and 4001 are not owned exclusively by KNIRV.
is_knirv_process() {
    local pid="$1"
    local cmd comm ancestor

    [[ "$pid" =~ ^[0-9]+$ ]] || return 1
    [[ "$pid" -gt 1 && "$pid" != "$$" ]] || return 1

    # Never kill this script or one of the shells which launched it.
    ancestor="$PPID"
    while [[ "$ancestor" =~ ^[0-9]+$ && "$ancestor" -gt 1 ]]; do
        [[ "$pid" == "$ancestor" ]] && return 1
        ancestor=$(ps -o ppid= -p "$ancestor" 2>/dev/null | tr -d ' ')
    done

    comm=$(ps -o comm= -p "$pid" 2>/dev/null | tr -d ' ')
    cmd=$(ps -o args= -p "$pid" 2>/dev/null)
    [[ -n "$cmd" ]] || return 1
    [[ "$comm" != "pgrep" && "$comm" != "pkill" && "$comm" != "grep" ]] || return 1
    [[ "$cmd" != *"kill-all-services.sh"* ]] || return 1

    # Source-tree/runtime commands carry a KNIRV path/name.  The explicit
    # executable names cover installed services whose argv contains only the
    # binary name.
    if printf '%s\n' "$cmd" | grep -Eqi 'knirv|/KNIRV'; then
        return 0
    fi
    case "$comm" in
        backend_server|container_deployer|os_builder|hasher-host|hasher-server) return 0 ;;
        ulorad|knirvagent|knirvllama|knirvmodels|knirvmodels-inf|knirvgateway|knirvgraph|knirvchain|knirvoracle) return 0 ;;
        data-connector|data-mapper|data-encoder|data-seeder|data-trainer|head-trainer) return 0 ;;
    esac

    # Services whose argv says nothing about KNIRV (ipfs daemon, text-embedder,
    # `node main.js` of the Transaction Chain, validationchain): accept them only when
    # their environment, working directory or executable is KNIRV's own.
    local env_knirv exe cwd
    env_knirv=$(tr '\0' '\n' < "/proc/$pid/environ" 2>/dev/null | grep -E '^(IPFS_PATH|TEXT_EMBEDDER_BINARY_PATH|XION_HOME)=.*knirv' || true)
    [[ -n "$env_knirv" ]] && return 0
    exe=$(readlink "/proc/$pid/exe" 2>/dev/null)
    cwd=$(readlink "/proc/$pid/cwd" 2>/dev/null)
    case "$comm" in
        ipfs|xiond|text-embedder|embedder|validationchain|node)
            if printf '%s\n%s\n' "$exe" "$cwd" | grep -Eqi 'knirv'; then
                return 0
            fi
            ;;
    esac
    return 1
}

# ── Kill by validated pgrep pattern ───────────────────────────────────────
kill_by_pattern() {
    local pattern="$1"
    local signal="${2:-SIGTERM}"
    local aggressive="${3:-false}"
    local timeout="${4:-3}"   # seconds to wait after the signal before SIGKILL

    echo "Looking for processes matching: $pattern"

    local pids=""
    local candidate
    for candidate in $(pgrep -f -- "$pattern" 2>/dev/null || true); do
        if is_knirv_process "$candidate"; then
            pids+="$candidate "
        fi
    done

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

    # Wait for graceful termination (up to $timeout s)
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
    local remaining_pids=""
    for candidate in $(pgrep -f -- "$pattern" 2>/dev/null || true); do
        if is_knirv_process "$candidate"; then
            remaining_pids+="$candidate "
        fi
    done
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
            # Match the executable name, not any occurrence in a command
            # line.  The latter used to kill unrelated desktop applications
            # with common names such as node.
            local pids=$(pgrep -x -- "$bin_name" 2>/dev/null || true)
            if [[ -n "$pids" ]]; then
                for pid in $pids; do
                    if is_knirv_process "$pid"; then
                        echo "    Found PID: $pid - force killing"
                        sudo_kill "$pid" 9 || true
                    fi
                done
            fi
        fi
    done
}

# Kill KNIRV-SERVER unified binary.  SIGTERM is the same as Ctrl+C: the launcher then
# stops backend_server, ulorad, knirvagent, knirvllama/llama-server, text-embedder, IPFS,
# Xion and both chains in order, and sweeps any child left over.  Give it time to do
# that (it allows the backend up to 30 s) before anything is killed outright.
GRACE=35
[[ "$FORCE" == "true" ]] && GRACE=3
echo ""
echo "1. Stopping KNIRV-SERVER unified binary (graceful, up to ${GRACE}s)..."
kill_by_pattern "./dist/knirv-server" "SIGTERM" "true" "$GRACE"
kill_by_pattern "knirv-server" "SIGTERM" "true" "$GRACE"

# Kill binaries in KNIRVSERVER runtime bin directories
echo ""
echo "1.5. Killing binaries in KNIRVSERVER runtime bin directories..."
kill_by_binary_path "/var/lib/knirvserver/bin"
kill_by_binary_path "$HOME/.local/share/knirvserver/bin"

# Anything the launcher or backend started that is still alive.  Services are listed
# individually so each is reported by name.  The backend goes first (it stops the
# daemons it owns), then the daemons, then the launcher's other children.
echo ""
echo "1.6. Killing KNIRV subprocess services..."
for svc in backend_server; do
    kill_by_pattern "$svc" "SIGTERM" "true" 10 || true
done
# Backend-owned daemons: models (HASHER, HEART, HEAD), gateway, graph, chain, oracle.
for svc in knirvmodels knirvgateway knirvgraph knirvchain knirvoracle; do
    kill_by_pattern "$svc" "SIGTERM" "true" || true
done
# Models data pipeline and ASIC driver.
for svc in data-connector data-mapper data-encoder data-seeder data-trainer head-trainer hasher-host hasher-server; do
    kill_by_pattern "$svc" "SIGTERM" "true" || true
done
# Launcher-owned: ULoRA daemon, agent runtime, llama wrapper and its llama-server child,
# text embedder, IPFS, Xion, Transaction Chain (node main.js), Validation Chain.
# knirvmodels-infer is the model harness (Colibri primary, llama.cpp fallback); its
# Colibri engine runs from <app data>/colibri/src and is stopped with `coli stop`.
for svc in knirvmodels-infer colibri/src ulorad knirvagent knirvllama llama-server text-embedder "ipfs daemon" "xiond start" "node main.js" validationchain; do
    kill_by_pattern "$svc" "SIGTERM" "true" || true
done
# Pre-rename daemon left running by an in-place upgrade.
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

# Kill KNIRV processes listening on KNIRV ports.  A port alone is never a
# sufficient reason to terminate a process; these ports may be used by other
# local applications.
echo ""
echo "7. Checking for processes on KNIRV ports..."
# 8089 text-embedder, 5001/8081 IPFS API/gateway, 26657/9091/1317 Xion
PORTS="8080 8081 8082 8084 8086 8089 8090 4001 5001 9080 9090 9002 7090 26657 9091 1317"
for port in $PORTS; do
    if command -v lsof >/dev/null 2>&1; then
        for pid in $(lsof -ti:"$port" 2>/dev/null || true); do
            if is_knirv_process "$pid"; then
                echo "  Found KNIRV process $pid on port $port - force killing"
                sudo_kill "$pid" 9 || true
            else
                echo "  Ignoring non-KNIRV process on port $port (PID $pid)"
            fi
        done
    elif command -v ss >/dev/null 2>&1; then
        # Alternative using ss and awk
        for pid in $(ss -ltnp 2>/dev/null | awk -v port="$port" '$0 ~ ":" port " " { while (match($0, /pid=[0-9]+/)) { print substr($0, RSTART + 4, RLENGTH - 4); $0=substr($0, RSTART + RLENGTH) } }' | sort -u); do
            if is_knirv_process "$pid"; then
                echo "  Found KNIRV process $pid on port $port - force killing"
                sudo_kill "$pid" 9 || true
            else
                echo "  Ignoring non-KNIRV process on port $port (PID $pid)"
            fi
        done
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
