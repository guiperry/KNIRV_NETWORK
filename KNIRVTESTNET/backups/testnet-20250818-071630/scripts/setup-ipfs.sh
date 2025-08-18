#!/bin/bash
set -e

echo "Setting up IPFS node for testnet..."

# Already in KNIRVTESTNET directory

# Check if IPFS is installed
if ! command -v ipfs &> /dev/null; then
    echo "IPFS not found. Installing IPFS..."
    
    # Download and install IPFS
    wget https://dist.ipfs.tech/kubo/v0.24.0/kubo_v0.24.0_linux-amd64.tar.gz
    tar -xzf kubo_v0.24.0_linux-amd64.tar.gz
    sudo mv kubo/ipfs /usr/local/bin/
    rm -rf kubo kubo_v0.24.0_linux-amd64.tar.gz
fi

# Set IPFS path for testnet
export IPFS_PATH=./data/ipfs

# Initialize IPFS if not already done
if [ ! -d "./data/ipfs" ]; then
    echo "Initializing IPFS..."
    ipfs init
fi

# Configure for testnet
ipfs config Addresses.API /ip4/0.0.0.0/tcp/5001
ipfs config Addresses.Gateway /ip4/0.0.0.0/tcp/8080
ipfs config --json API.HTTPHeaders.Access-Control-Allow-Origin '["*"]'
ipfs config --json API.HTTPHeaders.Access-Control-Allow-Methods '["PUT", "POST"]'

# Start IPFS daemon
echo "Starting IPFS daemon..."
ipfs daemon > ./logs/ipfs.log 2>&1 &
echo $! > ./data/ipfs.pid

echo "IPFS node started with PID $(cat ./data/ipfs.pid)"
echo "API endpoint: http://localhost:5001"
echo "Gateway endpoint: http://localhost:8080"
