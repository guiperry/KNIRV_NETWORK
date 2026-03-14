#!/bin/bash

# Script to kill all gorgonite processes
# Usage: ./scripts/kill_gorgonite.sh

echo "Finding and killing gorgonite processes..."

# Find all processes containing 'gorgonite' in their command line
PIDS=$(pgrep -f gorgonite)

if [ -z "$PIDS" ]; then
    echo "No gorgonite processes found."
    exit 0
fi

echo "Found processes: $PIDS"

# Kill the processes
kill -9 $PIDS

if [ $? -eq 0 ]; then
    echo "Successfully killed gorgonite processes."
else
    echo "Failed to kill some processes."
    exit 1
fi