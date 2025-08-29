#!/bin/bash
echo "🚀 Starting KNIRVENGINE Complete System..."

# Start Desktop Host in background
cd "$(dirname "$0")"
./start-desktop-host.sh &
DESKTOP_PID=$!

echo "Agentic Wallet Server started (PID: $DESKTOP_PID)"
echo "Mobile App development server: Run './start-mobile-tool.sh' in another terminal"
echo "Browser Extension: Available in browser-bridge/packages/"
echo "API available at: http://localhost:8082"
echo ""
echo "Press Ctrl+C to stop the system"

# Wait for interrupt
trap "echo 'Stopping system...'; kill $DESKTOP_PID; exit 0" INT
wait $DESKTOP_PID
