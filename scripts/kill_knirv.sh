#!/bin/bash
# KNIRV Network Process Termination Script
#
# This script comprehensively terminates all KNIRV-related processes across the entire network.
# It handles all KNIRV services including KNIRVCHAIN, KNIRVSERVER, KNIRVGRAPH, KNIRVORACLE,
# KNIRVROUTER, KNIRVGATEWAY, Economics Service, and associated frontend processes.
#
# Features:
# - Graceful shutdown with SIGTERM followed by SIGKILL if needed
# - Comprehensive process discovery by name, port, and working directory
# - Cleanup of temporary files, logs, and build artifacts
# - Support for dry-run, verbose, and force modes
# - Emergency kill mode for stubborn processes
# - Network status checking
#
# Usage Examples:
#   ./kill_knirv.sh                    # Normal graceful shutdown
#   ./kill_knirv.sh --force            # Immediate force kill
#   ./kill_knirv.sh --dry-run          # Show what would be killed
#   ./kill_knirv.sh --status           # Check current network status
#   ./kill_knirv.sh --emergency        # Emergency kill mode
#
# Author: KNIRV Development Team
# Version: 2.0 - Enhanced for full network coverage

# Configuration
# All KNIRV service ports
PORTS_TO_CHECK=(
    # Core KNIRV Services
    8000    # KNIRVGATEWAY API Gateway
    8080    # KNIRVCHAIN / KNIRVSERVER Frontend
    8081    # KNIRVSERVER API
    8082    # KNIRVORACLE
    8083    # KNIRVGRAPH
    8090    # Economics Service
    8091    # KNIRVROUTER

    # Legacy/Alternative ports
    5000 5001 6000 6001    # KNIRVORACLE legacy ports
    3000 3001              # Development servers
    4000 4001              # Additional services
    9000 9001              # Monitoring/metrics
)

TEMP_DIRS=("/tmp/go-build*" "/tmp/KNIRV*" "/tmp/knirvserver*" "/tmp/economics*")
# Lock files to clean (excluding package-lock.json to prevent npm corruption)
LOCK_FILES=("gateway.lock" "economics.lock" "*.pid" "knirv*.lock")

# KNIRV service patterns to search for
KNIRV_PATTERNS=(
    "KNIRVORACLE"
    "KNIRVCHAIN"
    "KNIRVSERVER"
    "KNIRVGRAPH"
    "KNIRVROUTER"
    "economics"
    "knirvserver"
    "knirvserver"
    "knirvchain"
    "knirvgraph"
    "knirvoracle"
    "knirvrouter"
    "api-gateway"
)

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Find all KNIRV processes across the network
find_processes() {
    local all_pids=""

    # Find by KNIRV service patterns (redirect output to stderr for logging)
    >&2 echo -e "${YELLOW}Searching for all KNIRV network processes...${NC}"
    >&2 echo -e "${YELLOW}  Searching by process names and patterns...${NC}"
    for pattern in "${KNIRV_PATTERNS[@]}"; do
        local pattern_pids=$(pgrep -f "$pattern" 2>/dev/null)
        if [ -n "$pattern_pids" ]; then
            >&2 echo -e "${YELLOW}    Found processes matching '$pattern': $pattern_pids${NC}"
            all_pids+="$pattern_pids "
        fi
    done

    # Find by Go processes in KNIRV directories
    >&2 echo -e "${YELLOW}  Searching by working directory...${NC}"
    local knirv_dirs=$(find /home -maxdepth 3 -type d -name "*KNIRV*" 2>/dev/null | head -10)
    for dir in $knirv_dirs; do
        local dir_pids=$(pgrep -a -f "go run" | grep "$dir" | awk '{print $1}' 2>/dev/null)
        if [ -n "$dir_pids" ]; then
            >&2 echo -e "${YELLOW}    Found Go processes in $dir: $dir_pids${NC}"
            all_pids+="$dir_pids "
        fi
    done

    # Find by specific ports
    >&2 echo -e "${YELLOW}  Searching by network ports...${NC}"
    for port in "${PORTS_TO_CHECK[@]}"; do
        local port_pids=$(lsof -i :$port -t 2>/dev/null)
        if [ -n "$port_pids" ]; then
            >&2 echo -e "${YELLOW}    Found processes on port $port: $port_pids${NC}"
            all_pids+="$port_pids "
        fi
    done

    # Find Node.js/Vite processes (for frontend services)
    >&2 echo -e "${YELLOW}  Searching for Node.js/Vite processes...${NC}"
    local node_pids=$(pgrep -f "vite|node.*knirv|npm.*dev" 2>/dev/null)
    if [ -n "$node_pids" ]; then
        >&2 echo -e "${YELLOW}    Found Node.js/Vite processes: $node_pids${NC}"
        all_pids+="$node_pids "
    fi

    # Find child processes of all found PIDs
    local parent_pids=$(echo "$all_pids" | tr ' ' '\n' | sort -u | grep -v '^$')
    local children=""
    if [ -n "$parent_pids" ]; then
        >&2 echo -e "${YELLOW}  Searching for child processes...${NC}"
        for pid in $parent_pids; do
            local child_pids=$(pgrep -P "$pid" 2>/dev/null)
            if [ -n "$child_pids" ]; then
                children+="$child_pids "
            fi
        done
    fi

    # Combine and dedupe all found PIDs - only output the final result
    local final_pids=$(echo "$all_pids $children" | tr ' ' '\n' | sort -u | grep -v '^$')

    # Filter out system processes that should not be killed (e.g., dnsmasq)
    local filtered_pids=""
    for pid in $final_pids; do
        # Get the command line for this PID
        local cmd=$(ps -p $pid -o args= 2>/dev/null)
        if [[ -n "$cmd" && "$cmd" != *"dnsmasq"* ]]; then
            filtered_pids+="$pid "
        else
            >&2 echo -e "${YELLOW}    Excluding system process $pid: $cmd${NC}"
        fi
    done
    final_pids=$(echo "$filtered_pids" | tr ' ' '\n' | sort -u | grep -v '^$')

    echo "$final_pids"
}

# Kill processes
kill_processes() {
    local pids=$1
    local signal=$2
    local signal_name=$3 # e.g., "TERM", "KILL"

    if [ -z "$pids" ]; then
        echo -e "${GREEN}No processes to send $signal_name signal to.${NC}"
        return
    fi

    for pid in $pids; do
        # Check if the process still exists before trying to kill
        if ps -p $pid > /dev/null; then
            echo -e "${YELLOW}Sending $signal_name signal (-$signal) to PID $pid...${NC}"
            kill -$signal $pid 2>/dev/null
            if [ $? -ne 0 ]; then
                 # Check again if it died right after the signal attempt
                 if ps -p $pid > /dev/null; then
                    echo -e "${RED}Failed to send $signal_name signal to PID $pid (Process might require sudo or already exited)${NC}"
                 else
                    echo -e "${GREEN}PID $pid terminated after $signal_name signal.${NC}"
                 fi
            fi
        else
             echo -e "${YELLOW}PID $pid already terminated.${NC}"
        fi
    done
}

# Cleanup temp files and lock files for all KNIRV services
# NOTE: This function is designed to be safe and avoid corrupting npm installations
# It specifically excludes package-lock.json files and node_modules directories
cleanup() {
    echo -e "${YELLOW}Cleaning up temporary files and lock files for all KNIRV services...${NC}"

    # Clean Go build cache in /tmp
    echo -e "${YELLOW}  Cleaning Go build cache...${NC}"
    find /tmp -maxdepth 1 -name 'go-build*' -exec rm -rf {} + 2>/dev/null

    # Clean KNIRV-specific temp files
    echo -e "${YELLOW}  Cleaning KNIRV temp files...${NC}"
    for pattern in "${TEMP_DIRS[@]}"; do
        find /tmp -maxdepth 1 -name "${pattern#/tmp/}" -exec rm -rf {} + 2>/dev/null
    done

    # Clean lock and PID files in current directory and subdirectories
    # NOTE: Explicitly avoid package-lock.json files to prevent npm corruption
    echo -e "${YELLOW}  Cleaning lock and PID files (excluding package-lock.json)...${NC}"
    find . -name '*.lock' -not -name 'package-lock.json' -delete 2>/dev/null
    find . -name '*.pid' -delete 2>/dev/null
    find . -name 'gateway.pid' -delete 2>/dev/null
    find . -name 'economics.pid' -delete 2>/dev/null

    # Clean database directories used by various KNIRV services
    echo -e "${YELLOW}  Cleaning database directories...${NC}"
    rm -rf database database_reflection 2>/dev/null
    rm -rf knirvserver.db knirvchain.db knirvgraph.db 2>/dev/null
    rm -rf knirvserver.db 2>/dev/null  # Legacy name

    # Clean log files
    echo -e "${YELLOW}  Cleaning log files...${NC}"
    find . -name '*.log' -size +100M -delete 2>/dev/null  # Only large log files
    rm -rf logs/*.log 2>/dev/null

    # NOTE: Node.js/npm cleanup removed to prevent corruption of package installations
    # The script no longer touches node_modules, dist directories, or npm cache files

    # Clean Docker containers and volumes if any
    echo -e "${YELLOW}  Cleaning Docker resources...${NC}"
    docker ps -a --filter "name=knirv" --format "{{.ID}}" | xargs -r docker rm -f 2>/dev/null
    docker volume ls --filter "name=knirv" --format "{{.Name}}" | xargs -r docker volume rm 2>/dev/null

    echo -e "${GREEN}Cleanup completed.${NC}"
}

# Display help information
show_help() {
    echo "KNIRV Network Process Termination Script"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -h, --help     Show this help message"
    echo "  -f, --force    Skip graceful shutdown, force kill immediately"
    echo "  -v, --verbose  Show detailed process information"
    echo "  -n, --dry-run  Show what would be killed without actually killing"
    echo "  --no-cleanup   Skip cleanup of temp files and logs"
    echo ""
    echo "This script will terminate all KNIRV network processes including:"
    echo "  - KNIRVCHAIN, KNIRVSERVER, KNIRVGRAPH, KNIRVORACLE, KNIRVROUTER"
    echo "  - KNIRVGATEWAY and Economics Service"
    echo "  - Associated Node.js/Vite frontend processes"
    echo "  - All child processes and related services"
    echo ""
}

# Main execution
main() {
    local force_kill=false
    local verbose=false
    local dry_run=false
    local skip_cleanup=false

    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -f|--force)
                force_kill=true
                shift
                ;;
            -v|--verbose)
                verbose=true
                shift
                ;;
            -n|--dry-run)
                dry_run=true
                shift
                ;;
            --no-cleanup)
                skip_cleanup=true
                shift
                ;;
            *)
                echo -e "${RED}Unknown option: $1${NC}"
                show_help
                exit 1
                ;;
        esac
    done

    echo -e "${YELLOW}=== KNIRV Network Process Termination ===${NC}"

    # Find all processes
    pids=$(find_processes)

    if [ -z "$pids" ]; then
        echo -e "${GREEN}No KNIRV network processes found.${NC}"
    else
        echo -e "${YELLOW}Found KNIRV network processes with PIDs: $pids${NC}"

        if [ "$verbose" = true ] || [ "$dry_run" = true ]; then
            echo -e "${YELLOW}Process details:${NC}"
            ps -fp $pids 2>/dev/null || echo -e "${YELLOW}Some processes may have already exited.${NC}"
        fi

        if [ "$dry_run" = true ]; then
            echo -e "${YELLOW}DRY RUN: Would terminate the above processes${NC}"
            exit 0
        fi

        if [ "$force_kill" = false ]; then
            # Graceful shutdown first
            echo -e "${YELLOW}Attempting graceful shutdown...${NC}"
            kill_processes "$pids" "TERM" "TERM" # Signal 15

            # Wait for shutdown
            local timeout=15 # Increased timeout for network services
            local end_time=$((SECONDS + timeout))
            local remaining_pids=$pids

            echo -e "${YELLOW}Waiting up to $timeout seconds for graceful shutdown...${NC}"
            while [ $SECONDS -lt $end_time ]; do
                # Check which of the original PIDs are still running
                current_running=""
                for pid in $remaining_pids; do
                    if ps -p $pid > /dev/null 2>&1; then
                        current_running+="$pid "
                    fi
                done
                remaining_pids=$(echo "$current_running" | sed 's/ *$//') # Trim trailing space

                if [ -z "$remaining_pids" ]; then
                    echo -e "${GREEN}All processes shut down gracefully.${NC}"
                    break
                fi
                if [ "$verbose" = true ]; then
                    echo -e "${YELLOW}Still waiting for PIDs: $remaining_pids${NC}"
                fi
                sleep 2 # Check every 2 seconds
            done
        else
            remaining_pids=$pids
            echo -e "${YELLOW}Force kill mode enabled, skipping graceful shutdown.${NC}"
        fi

        # Force kill any remaining processes
        if [ -n "$remaining_pids" ]; then
            echo -e "${RED}Force killing remaining processes: $remaining_pids${NC}"
            kill_processes "$remaining_pids" "KILL" "KILL" # Signal 9
            sleep 2 # Give OS time to process KILL signals
        fi
    fi

    # Cleanup files unless skipped
    if [ "$skip_cleanup" = false ]; then
        cleanup
    else
        echo -e "${YELLOW}Skipping cleanup as requested.${NC}"
    fi

    # Final verification
    sleep 2 # Longer pause before final check
    remaining_after_kill=$(find_processes)
    if [ -z "$remaining_after_kill" ]; then
        echo -e "${GREEN}✅ All KNIRV network processes terminated successfully.${NC}"
        exit 0
    else
        echo -e "${RED}⚠️  Warning: Could not terminate all processes.${NC}"
        echo -e "${RED}Remaining PIDs: $remaining_after_kill${NC}"
        echo -e "${RED}You may need to kill these manually with 'sudo kill -9 $remaining_after_kill'${NC}"
        exit 1
    fi
}

# Add additional utility functions
check_network_status() {
    echo -e "${YELLOW}Checking KNIRV network status...${NC}"
    local services_found=false

    for port in "${PORTS_TO_CHECK[@]}"; do
        if lsof -i :$port >/dev/null 2>&1; then
            local pid=$(lsof -i :$port -t)
            local process=$(ps -p $pid -o comm= 2>/dev/null)
            echo -e "${YELLOW}  Port $port: $process (PID: $pid)${NC}"
            services_found=true
        fi
    done

    if [ "$services_found" = false ]; then
        echo -e "${GREEN}  No KNIRV services detected on monitored ports.${NC}"
    fi
}

# Emergency kill function for when normal methods fail
emergency_kill() {
    echo -e "${RED}=== EMERGENCY KILL MODE ===${NC}"
    echo -e "${RED}This will forcefully terminate ALL processes matching KNIRV patterns${NC}"
    read -p "Are you sure? (yes/no): " confirm

    if [ "$confirm" = "yes" ]; then
        for pattern in "${KNIRV_PATTERNS[@]}"; do
            pkill -9 -f "$pattern" 2>/dev/null
        done

        for port in "${PORTS_TO_CHECK[@]}"; do
            local pids=$(lsof -i :$port -t 2>/dev/null)
            if [ -n "$pids" ]; then
                kill -9 $pids 2>/dev/null
            fi
        done

        echo -e "${RED}Emergency kill completed.${NC}"
        cleanup
    else
        echo -e "${YELLOW}Emergency kill cancelled.${NC}"
    fi
}

# Check if running as root and warn
if [ "$EUID" -eq 0 ]; then
    echo -e "${RED}⚠️  Warning: Running as root. This will affect system-wide processes.${NC}"
    sleep 2
fi

# Handle special emergency mode
if [ "$1" = "--emergency" ]; then
    emergency_kill
    exit 0
fi

# Show network status if requested
if [ "$1" = "--status" ]; then
    check_network_status
    exit 0
fi

# Ensure script runs with bash and handle arguments
main "$@"

echo -e "${GREEN}KNIRV network termination script completed.${NC}"

# End of script