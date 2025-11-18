#!/bin/bash

# Comprehensive KNIRVCHAIN process shutdown script

# Stage 1: Identify all KNIRVCHAIN-related processes
# - By process name/args
# - By working directory
# - By network ports
PIDS=$(
    # Processes with KNIRVCHAIN in command line
    pgrep -f "KNIRVCHAIN"
    
    # Processes running from project directory
    pgrep -a -f "go" | grep "$(pwd)" | awk '{print $1}'
    
    # Processes using blockchain ports (adjust ports as needed)
    lsof -i :3000-3010 -t
) | sort -u

if [ -z "$PIDS" ]; then
    echo "No KNIRVCHAIN processes found"
    exit 0
fi

echo "Found KNIRVCHAIN processes with PIDs:"
ps -fp $PIDS

# Stage 2: Graceful shutdown
echo "Initiating graceful shutdown..."
kill -TERM $PIDS 2>/dev/null

# Stage 3: Wait for shutdown with timeout
TIMEOUT=30
END_TIME=$((SECONDS + TIMEOUT))
REMAINING=$PIDS

while [ $SECONDS -lt $END_TIME ]; do
    REMAINING=$(echo "$REMAINING" | xargs -n1 ps -p 2>/dev/null | awk '{print $1}')
    if [ -z "$REMAINING" ]; then
        echo "All processes shut down gracefully"
        exit 0
    fi
    echo "Waiting for PIDs: $REMAINING"
    sleep 2
done

# Stage 4: Forceful termination
echo "Timeout reached, forcing termination..."
kill -9 $PIDS 2>/dev/null
sleep 1

# Final verification
REMAINING=$(echo "$PIDS" | xargs -n1 ps -p 2>/dev/null | awk '{print $1}')
if [ -n "$REMAINING" ]; then
    echo "Warning: Could not terminate PIDs: $REMAINING"
    exit 1
else
    echo "All KNIRVCHAIN processes terminated"
    exit 0
fi