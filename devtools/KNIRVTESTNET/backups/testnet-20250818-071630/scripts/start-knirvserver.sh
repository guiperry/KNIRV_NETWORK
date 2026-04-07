#!/bin/bash
set -e

echo "Starting KNIRV-NEXUS testnet node..."

# Create necessary directories
mkdir -p logs data

# Check if binaries exist
if [ ! -f "./bin/knirvserver-dve-manager" ] || [ ! -f "./bin/knirvserver-validation-core" ]; then
    echo "Error: KNIRV-NEXUS binaries not found. Please run build-knirvserver.sh first."
    exit 1
fi

# Get the correct base directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(dirname "$SCRIPT_DIR")"

# Ensure minimal config is in place
cp $BASE_DIR/data/knirvserver/knirv-nexus-minimal.yaml $BASE_DIR/data/knirvserver/knirv-nexus.yaml

# Start DVE Manager with memory limit (30MB)
echo "Starting KNIRV-NEXUS DVE Manager with 30MB memory limit..."
cd $BASE_DIR/data/knirvserver && (
    # Set memory limit for this process (30MB = 30720KB)
    ulimit -v 30720
    exec ../../bin/knirvserver-dve-manager \
        -testnet \
        -port 8084
) > ../../logs/knirvserver-dve-manager.log 2>&1 &

DVE_PID=$!
cd $BASE_DIR
echo $DVE_PID > data/knirvserver-dve-manager.pid

# Start Validation Core with memory limit (30MB)
echo "Starting KNIRV-NEXUS Validation Core with 30MB memory limit..."
cd $BASE_DIR/data/knirvserver && (
    # Set memory limit for this process (30MB = 30720KB)
    ulimit -v 30720
    exec ../../bin/knirvserver-validation-core \
        -testnet \
        -port 8085
) > ../../logs/knirvserver-validation-core.log 2>&1 &

VALIDATION_PID=$!
cd $BASE_DIR
echo $VALIDATION_PID > data/knirvserver-validation-core.pid

echo "KNIRV-NEXUS services started:"
echo "  DVE Manager PID: $(cat ./data/knirvserver-dve-manager.pid)"
echo "  Validation Core PID: $(cat ./data/knirvserver-validation-core.pid)"
echo "API endpoints:"
echo "  DVE Manager: http://localhost:8081"
echo "  Validation Core: http://localhost:8082"
echo "Testnet features:"
echo "  - Headless mode enabled"
echo "  - TEE simulation enabled"
echo "  - Mock validation responses"
echo "  - Simplified validation proofs"
echo "  - Clean database on start"
echo "Log file: ./logs/knirvserver.log"

# Wait a moment and check if processes are still running
sleep 3
dve_manager_pid=$(cat ./data/knirvserver-dve-manager.pid)
validation_core_pid=$(cat ./data/knirvserver-validation-core.pid)

if ! kill -0 $dve_manager_pid 2>/dev/null; then
    echo "Error: DVE Manager failed to start. Check logs:"
    tail -20 ./logs/knirvserver-dve-manager.log
    exit 1
fi

if ! kill -0 $validation_core_pid 2>/dev/null; then
    echo "Error: Validation Core failed to start. Check logs:"
    tail -20 ./logs/knirvserver-validation-core.log
    exit 1
fi

echo "KNIRV-NEXUS testnet services are running successfully!"
