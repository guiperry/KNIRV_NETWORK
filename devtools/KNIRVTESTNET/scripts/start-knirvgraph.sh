#!/bin/bash
set -e

echo "Starting KNIRVGRAPH testnet node..."

# Create necessary directories
mkdir -p logs data

# Ensure proper permissions on data directory (only create/fix if missing or wrong)
if [ ! -d "data/knirvgraph" ]; then
    mkdir -p data/knirvgraph
    chmod 755 data/knirvgraph
    echo "Created new data directory: data/knirvgraph"
else
    # Permissions check - only fix if needed
    current_perms=$(stat -c %a data/knirvgraph 2>/dev/null || echo "unknown")
    if [ "$current_perms" != "755" ]; then
        chmod 755 data/knirvgraph
        echo "Fixed permissions on data/knirvgraph"
    fi
    
    # Check for corruption indicators (Badger/Ristretto leave these on unclean shutdown)
    if [ -f "data/knirvgraph/LOCK" ]; then
        # Check if the process holding the lock is actually running
        if [ -f "data/knirvgraph/LOCK" ] && ! fuser data/knirvgraph/LOCK >/dev/null 2>&1; then
            echo "Cleaning up stale LOCK file (process not running)"
            rm -f data/knirvgraph/LOCK
        fi
    fi
    
    # Only clean corrupted files if manifest is missing (indicates incomplete shutdown)
    if [ -d "data/knirvgraph" ] && [ ! -f "data/knirvgraph/MANIFEST" ]; then
        if [ -f "data/knirvgraph/000001.vlog" ] || [ -f "data/knirvgraph/000002.vlog" ]; then
            echo "Database appears corrupted (missing MANIFEST). Cleaning up corrupted files..."
            rm -f data/knirvgraph/*.vlog data/knirvgraph/*.sst 2>/dev/null || true
        fi
    fi
fi

chmod 755 data
echo "✅ Data directory ready"

# Check if binary exists
if [ ! -f "./bin/knirvgraph" ]; then
    echo "Error: KNIRVGRAPH binary not found. Please run build-knirvgraph.sh first."
    exit 1
fi

# Start KNIRVGRAPH in testnet mode with in-memory storage (no memory limit due to Go 1.23.3 compatibility)
echo "Starting KNIRVGRAPH with testnet features and resource optimizations..."
(
    # Note: Memory will be managed by Go runtime and system limits (Go 1.23.3 compatibility)
    exec ./bin/knirvgraph \
        --testnet \
        --memory \
        --populate \
        --max-nodes 250 \
        --rpc-port 8082 \
        --home ./data/knirvgraph
) > ./logs/knirvgraph.log 2>&1 &

echo $! > ./data/knirvgraph.pid
echo "KNIRVGRAPH testnet started with PID $(cat ./data/knirvgraph.pid)"
echo "API endpoint: http://localhost:8082"
echo "Testnet features:"
echo "  - In-memory storage enabled"
echo "  - Pre-populated test data"
echo "  - Real DHT implementation"
echo "  - Full graph operations"
echo "Resource optimizations:"
echo "  - Reduced max nodes: 250 (50% reduction from default 500)"
echo "  - In-memory storage (faster, reduced disk I/O)"
echo "  - No memory limit (Go 1.23.3 runtime managed)"
echo "Log file: ./logs/knirvgraph.log"

# Wait a moment and check if process is still running
sleep 3
if ! kill -0 $(cat ./data/knirvgraph.pid) 2>/dev/null; then
    echo "Error: KNIRVGRAPH failed to start. Check logs:"
    tail -20 ./logs/knirvgraph.log
    exit 1
fi

echo "KNIRVGRAPH testnet is running successfully!"
