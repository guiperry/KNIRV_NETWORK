#!/bin/bash
set -e

echo "=========================================="
echo "KNIRV-TESTNET: Network Shutdown"
echo "=========================================="

# Already in KNIRVTESTNET directory

# Function to stop service by PID file
stop_service() {
    local pid_file=$1
    local name=$2
    
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            echo "🛑 Stopping $name (PID: $pid)..."
            kill "$pid"
            sleep 2
            if kill -0 "$pid" 2>/dev/null; then
                echo "💀 Force killing $name..."
                kill -9 "$pid"
            fi
        fi
        rm -f "$pid_file"
        echo "✅ $name stopped."
    else
        echo "⚠️  $name PID file not found."
    fi
}

# Stop all services in reverse order
echo "Stopping all services..."

stop_service "./data/knirvgateway.pid" "KNIRV-GATEWAY"
stop_service "./data/knirvrouter.pid" "KNIRV-ROUTER"
stop_service "./data/knirvnexus-2.pid" "KNIRV-NEXUS-2"
stop_service "./data/knirvnexus-1.pid" "KNIRV-NEXUS-1"
stop_service "./data/knirvgraph.pid" "KNIRVGRAPH"
stop_service "./data/knirvchain.pid" "KNIRVCHAIN"
stop_service "./data/knirvroot.pid" "KNIRV-ROOT"
stop_service "./data/ipfs.pid" "IPFS"

echo ""
echo "=========================================="
echo "🏁 All services stopped successfully!"
echo "=========================================="
echo "📋 Logs preserved in: ./logs/"
echo "💾 Data preserved in: ./data/"
echo ""
echo "To restart: ./scripts/start-testnet.sh"
