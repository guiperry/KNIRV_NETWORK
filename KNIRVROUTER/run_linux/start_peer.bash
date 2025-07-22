#!/bin/bash
set -x
# This script starts a peer node with an automatically assigned port

# Find the next available port starting from 5001
next_port=5001
while [ -d "../$next_port" ]; do
  next_port=$((next_port + 1))
done

echo "Starting peer on port $next_port"

# Go back to root directory
cd ../
if [ $? -ne 0 ]; then
  echo "Error: cd failed"
  exit 1
fi

# Create directory for this peer
mkdir -p $next_port
if [ $? -ne 0 ]; then
  echo "Error: mkdir failed"
  exit 1
fi

# Define the file path
file_path="constants/constants.go"

# Update the database path in constants.go
sed -i "s/\(BLOCKCHAIN_DB_PATH\s*=\s*\"\)[^\/]*\/knirvchain_peer_$next_port_db\"/\1$next_port\/knirvchain_peer_$next_port_db\"/" "$file_path"
if [ $? -ne 0 ]; then
  echo "Error: sed failed"
  exit 1
fi

cd starter
if [ $? -ne 0 ]; then
  echo "Error: cd failed"
  exit 1
fi

# Start the web GUI for this peer
go run starter.go -gui -port $next_port
if [ $? -ne 0 ]; then
  echo "Error: go run failed"
  exit 1
fi