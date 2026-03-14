#!/bin/bash

# Run the setup script to create necessary directories
# ./scripts/setup.sh # setup.sh just creates dirs, which we do below anyway

echo "Cleaning up any existing database files..."
rm -rf database database_reflection
mkdir -p database database_reflection
echo "Database directories created."

# Define cleanup function
cleanup() {
    echo "Shutting down nodes..."
    # Check if PIDs exist before killing
    if kill -0 $MAIN_PID 2>/dev/null; then
        echo "Sending TERM signal to Main Node (PID: $MAIN_PID)..."
        kill -TERM $MAIN_PID
    fi
    if kill -0 $REFLECTION_PID 2>/dev/null; then
        echo "Sending TERM signal to Reflection Node (PID: $REFLECTION_PID)..."
        kill -TERM $REFLECTION_PID
    fi
    # Wait a moment for graceful shutdown
    wait $MAIN_PID $REFLECTION_PID 2>/dev/null
    echo "Shutdown complete."
    exit 0 # Exit script after cleanup
}

# Trap Ctrl+C (INT signal) and TERM signal
trap cleanup INT TERM

# Start the root blockchain node in the background
echo "Starting Root blockchain node on port 5000..."
go run . -miners_address=testaddress65166fcb6516cb -port=5000 -p2p.port=7000 -database_path=database/agent.db --reflect http://127.0.0.1:5001 &
MAIN_PID=$!
echo "Main Node started with PID: $MAIN_PID"

echo "Giving main node a few seconds to initialize..."
sleep 5 # Add a 5-second pause

# Wait for the main node to start
echo "Waiting for main node (5000) to become healthy..."
HEALTH_URL_MAIN="http://127.0.0.1:5000/health"
MAX_WAIT=120 # Wait max 120 seconds
WAIT_INTERVAL=3 # Check more frequently
ELAPSED=0

while [ $ELAPSED -lt $MAX_WAIT ]; do
    # CORRECTED: Use GET request, discard output, check exit code
    if curl --silent --fail -o /dev/null $HEALTH_URL_MAIN; then
        echo "Main node is healthy."
        break
    fi
    # Check if the process died unexpectedly
    if ! kill -0 $MAIN_PID 2>/dev/null; then
        echo "Error: Main node process (PID: $MAIN_PID) terminated unexpectedly."
        cleanup # Attempt cleanup before exiting
        exit 1
    fi
    echo "Main node not ready yet, waiting..."
    sleep $WAIT_INTERVAL
    ELAPSED=$((ELAPSED + WAIT_INTERVAL))
done

if [ $ELAPSED -ge $MAX_WAIT ]; then
    echo "Error: Main node did not become healthy within $MAX_WAIT seconds."
    cleanup # Attempt cleanup before exiting
    exit 1
fi

# Start the reflection node directly in the background
echo "Starting Reflection node on port 5001..."
go run . -miners_address=testReflection65166fcb6516cb -port=5001 -p2p.port=7001 -database_path=database_reflection/agent_reflection.db --reflect http://127.0.0.1:5000 &
REFLECTION_PID=$!
echo "Reflection Node started with PID: $REFLECTION_PID"

# Wait for the reflection node to start
echo "Waiting for reflection node (5001) to become healthy..."
HEALTH_URL_REFLECTION="http://127.0.0.1:5001/health" # CORRECTED: Use the right URL variable
ELAPSED=0

while [ $ELAPSED -lt $MAX_WAIT ]; do
    # CORRECTED: Use GET request, correct URL, discard output, check exit code
    if curl --silent --fail -o /dev/null $HEALTH_URL_REFLECTION; then
        echo "Reflection node is healthy."
        break
    fi
     # Check if the process died unexpectedly
    if ! kill -0 $REFLECTION_PID 2>/dev/null; then
        echo "Error: Reflection node process (PID: $REFLECTION_PID) terminated unexpectedly."
        cleanup # Attempt cleanup before exiting
        exit 1
    fi
    echo "Reflection node not ready yet, waiting..."
    sleep $WAIT_INTERVAL
    ELAPSED=$((ELAPSED + WAIT_INTERVAL))
done

if [ $ELAPSED -ge $MAX_WAIT ]; then
    echo "Error: Reflection node did not become healthy within $MAX_WAIT seconds."
    cleanup # Attempt cleanup before exiting
    exit 1
fi

# Wait indefinitely for the background processes (or until Ctrl+C)
echo "Blockchain network is running (Main: $MAIN_PID, Reflection: $REFLECTION_PID). Press Ctrl+C to stop all nodes."
wait # Wait for background PIDs or signals

# Note: The 'wait' command without arguments waits for all background jobs
# of the current shell. The trap will handle the specific PIDs upon INT/TERM.
