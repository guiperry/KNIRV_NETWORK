#!/bin/bash
set -e

echo "Starting KNIRV Testnet Health Monitor..."

# Get the correct base directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(dirname "$SCRIPT_DIR")"

# Create necessary directories
mkdir -p "$BASE_DIR/logs"

# Start Health Monitor on port 10001
echo "Starting Health Monitor on port 10001..."
cd "$BASE_DIR" && node server/health-monitor.js > logs/health-monitor.log 2>&1 &

HEALTH_PID=$!
echo $HEALTH_PID > data/health-monitor.pid

echo "Health Monitor started with PID $HEALTH_PID"
echo "Access at: http://localhost:10001/health-monitor"
echo "API at: http://localhost:10001/health-monitor/status"
echo "Logs: $BASE_DIR/logs/health-monitor.log"
