#!/bin/bash

echo "=== Tiny-LLM Shutdown Test ==="

# Cleanup any existing running processes
pkill -9 -f llama-server 2>/dev/null || true
pkill -9 -f tinyllm 2>/dev/null || true
sleep 2

# Check initial server state
if curl -s http://localhost:8000/health > /dev/null; then
    echo "❌ Server already running before test"
    exit 1
fi
echo "✅ Server is not running initially"

# Start the application
echo -e "/quit" | timeout 20 ./tinyllm &
APP_PID=$!
echo "Starting application with PID: $APP_PID"

# Give server time to start and quit
echo "Waiting for application to complete..."
wait $APP_PID

# Check if server is still running
if curl -s http://localhost:8000/health > /dev/null; then
    echo "❌ Server failed to shut down"
    # Force kill the server
    pkill -9 -f llama-server 2>/dev/null || true
    pkill -9 -f tinyllm 2>/dev/null || true
    sleep 1
    if curl -s http://localhost:8000/health > /dev/null; then
        echo "❌ Failed to force kill server"
        exit 1
    else
        echo "⚠️ Server force killed"
    fi
else
    echo "✅ Server shut down successfully"
fi