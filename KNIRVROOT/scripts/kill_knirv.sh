#!/bin/bash
# KNIRVROOT Process Termination Script

# Configuration
# Include HTTP API ports (5000, 5001) and Libp2p ports (6000, 6001)
PORTS_TO_CHECK=(5000 5001 6000 6001) # Add other ports if needed
TEMP_DIRS=("/tmp/go-build*" "/tmp/KNIRVROOT*")
LOCK_FILES=("*.lock" "*.pid")

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Find all KNIRVROOT processes
find_processes() {
    echo -e "${YELLOW}Searching for KNIRVROOT processes...${NC}"

    # Find by name/command line arguments
    local pids_name=$(pgrep -f "KNIRVROOT|go run \.|KNIRVROOT_GO_Root") # Added more specific patterns

    # Find by working directory (assuming script is run from project root)
    local pids_dir=$(pgrep -a -f "go run \." | grep "$(pwd)" | awk '{print $1}')

    # Find by specific ports
    local pids_port=""
    for port in "${PORTS_TO_CHECK[@]}"; do
        pids_port+=$(lsof -i :$port -t 2>/dev/null)
        pids_port+=" " # Add space separator
    done

    # Find child processes of the initially found PIDs
    local parent_pids=$(echo "$pids_name $pids_dir $pids_port" | tr ' ' '\n' | sort -u | grep -v '^$')
    local children=""
    if [ -n "$parent_pids" ]; then
        # Create a regex pattern for pgrep -P
        local parent_pattern=$(echo "$parent_pids" | paste -sd,)
        children=$(pgrep -P "$parent_pattern")
    fi

    # Combine and dedupe all found PIDs
    local all_pids=$(echo "$pids_name $pids_dir $pids_port $children" | tr ' ' '\n' | sort -u | grep -v '^$')

    echo "$all_pids"
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

# Cleanup temp files and lock files
cleanup() {
    echo -e "${YELLOW}Cleaning up temporary files and lock files...${NC}"
    # Clean Go build cache in /tmp
    find /tmp -maxdepth 1 -name 'go-build*' -exec rm -rf {} + 2>/dev/null
    # Clean specific KNIRVROOT temp files if any standard pattern exists
    find /tmp -maxdepth 1 -name 'KNIRVROOT*' -exec rm -rf {} + 2>/dev/null
    # Clean lock files in the current directory (adjust path if needed)
    find . -maxdepth 1 -name '*.lock' -delete 2>/dev/null
    find . -maxdepth 1 -name '*.pid' -delete 2>/dev/null
    # Clean database directories used by runNetwork.sh
    echo -e "${YELLOW}Removing database directories (database/, database_reflection/)...${NC}"
    rm -rf database database_reflection
}

# Main execution
main() {
    # Find all processes
    pids=$(find_processes)

    if [ -z "$pids" ]; then
        echo -e "${GREEN}No KNIRVROOT processes found.${NC}"
    else
        echo -e "${YELLOW}Found KNIRVROOT-related processes with PIDs:${NC}"
        # Use ps with PID list for better formatting and reliability
        ps -fp $pids || echo -e "${YELLOW}Some processes may have already exited.${NC}"

        # Graceful shutdown first
        kill_processes "$pids" "TERM" "TERM" # Signal 15

        # Wait for shutdown
        local timeout=10 # Reduced timeout for faster feedback
        local end_time=$((SECONDS + timeout))
        local remaining_pids=$pids

        echo -e "${YELLOW}Waiting up to $timeout seconds for graceful shutdown...${NC}"
        while [ $SECONDS -lt $end_time ]; do
            # Check which of the original PIDs are still running
            current_running=""
            for pid in $remaining_pids; do
                if ps -p $pid > /dev/null; then
                    current_running+="$pid "
                fi
            done
            remaining_pids=$(echo "$current_running" | sed 's/ *$//') # Trim trailing space

            if [ -z "$remaining_pids" ]; then
                echo -e "${GREEN}All processes shut down gracefully.${NC}"
                break
            fi
            echo -e "${YELLOW}Waiting for PIDs: $remaining_pids${NC}"
            sleep 1 # Check more frequently
        done

        # Force kill any remaining processes
        if [ -n "$remaining_pids" ]; then
            echo -e "${RED}Timeout reached or processes still running. Force killing remaining PIDs: $remaining_pids${NC}"
            kill_processes "$remaining_pids" "KILL" "KILL" # Signal 9
            sleep 1 # Give OS time to process KILL signals
        fi
    fi

    # Cleanup files regardless of whether processes were found
    cleanup

    # Final verification
    # Rerun find_processes to be absolutely sure
    sleep 1 # Short pause before final check
    remaining_after_kill=$(find_processes)
    if [ -z "$remaining_after_kill" ]; then
        echo -e "${GREEN}All KNIRVROOT processes terminated and cleanup done.${NC}"
        exit 0
    else
        echo -e "${RED}Warning: Could not terminate all processes. Remaining PIDs: $remaining_after_kill${NC}"
        echo -e "${RED}You may need to kill these manually (e.g., 'kill -9 $remaining_after_kill' or using sudo).${NC}"
        exit 1
    fi
}

# Ensure script runs with bash and handle arguments if any in the future
main "$@"

# End of script