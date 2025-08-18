#!/bin/bash
echo "🚀 Starting KNIRVENGINE Complete System..."

# Start Desktop Host in background
cd "$(dirname "$0")"
./start-desktop-client.sh &
DESKTOP_PID=$!

echo "Desktop Host started (PID: $DESKTOP_PID)"
echo "Mobile Tool available at: ./mobile-controller/index.html"
echo "API available at: http://localhost:8082"
echo "MCP WebSocket at: ws://localhost:8082/api/mcp/ws"
echo ""
echo "Press Ctrl+C to stop the system"

# Wait for interrupt
trap "echo 'Stopping system...'; kill $DESKTOP_PID; exit 0" INT
wait $DESKTOP_PID
