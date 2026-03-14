#!/bin/bash

# Script to run a network of verifier nodes for testing

# Build the example
echo "Building verifier node example..."
go build -o verifier_node examples/verifier_node.go

# Start the first node (bootstrap node)
echo "Starting bootstrap node..."
./verifier_node -id=node1 -addr=127.0.0.1:9001 -wallet=verifier1 &
BOOTSTRAP_PID=$!

# Wait for bootstrap node to start
sleep 2

# Start additional nodes
echo "Starting node 2..."
./verifier_node -id=node2 -addr=127.0.0.1:9002 -wallet=verifier2 -peers='["127.0.0.1:9001"]' &
NODE2_PID=$!

echo "Starting node 3..."
./verifier_node -id=node3 -addr=127.0.0.1:9003 -wallet=verifier3 -peers='["127.0.0.1:9001"]' &
NODE3_PID=$!

# Wait for user to press Ctrl+C
echo "Verifier network running. Press Ctrl+C to stop."
trap "kill $BOOTSTRAP_PID $NODE2_PID $NODE3_PID; echo 'Nodes stopped'; exit 0" SIGINT SIGTERM

# Keep script running
while true; do
    sleep 1
done