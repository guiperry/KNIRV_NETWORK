#!/bin/bash
set -e

echo "Starting NANDA ANS (Agent Registry)..."

# Get the correct base directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(dirname "$SCRIPT_DIR")"

# Create necessary directories
mkdir -p "$BASE_DIR/logs"

# Check if node_modules exists
if [ ! -d "$BASE_DIR/nanda_ans/node_modules" ]; then
    echo "Installing NANDA ANS dependencies..."
    cd "$BASE_DIR/nanda_ans" && npm install
fi

# Build the Next.js application
echo "Building NANDA ANS application..."
cd "$BASE_DIR/nanda_ans" && npm run build

# Start NANDA ANS on port 9002
echo "Starting NANDA ANS on port 9002..."
cd "$BASE_DIR/nanda_ans" && npm start -- -p 9002 > "$BASE_DIR/logs/nanda-ans.log" 2>&1 &

NANDA_PID=$!
cd "$BASE_DIR"
echo $NANDA_PID > data/nanda-ans.pid

echo "NANDA ANS started with PID $NANDA_PID"
echo "Access at: http://localhost:9002"
echo "Logs: $BASE_DIR/logs/nanda-ans.log"
