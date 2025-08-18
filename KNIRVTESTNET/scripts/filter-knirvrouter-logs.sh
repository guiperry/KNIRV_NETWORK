#!/bin/bash

# Log filter for KNIRV-ROUTER to reduce verbose connectivity logs
# This script filters out excessive "Measured connectivity to peer" messages
# while preserving important log information

# Read from stdin and filter logs
while IFS= read -r line; do
    # Skip "Measured connectivity to peer" messages (reduce by 90%)
    if [[ "$line" == *"Measured connectivity to peer"* ]]; then
        # Only show every 10th connectivity measurement (90% reduction)
        if (( RANDOM % 10 == 0 )); then
            echo "$line"
        fi
    # Skip "No transactions to mine, waiting..." messages (reduce by 80%)
    elif [[ "$line" == *"No transactions to mine, waiting"* ]]; then
        # Only show every 5th mining wait message (80% reduction)
        if (( RANDOM % 5 == 0 )); then
            echo "$line"
        fi
    # Show all other log messages
    else
        echo "$line"
    fi
done
