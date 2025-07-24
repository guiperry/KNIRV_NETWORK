#!/bin/bash

# Script to run the KNIRVROOT Payment Processor

# Set default values
PORT=${PORT:-8090}
KNIRVROOT_NODE_RPC=${KNIRVROOT_NODE_RPC:-"http://127.0.0.1:5000"}
TOKEN_SYMBOL=${TOKEN_SYMBOL:-"agent"}
TOKEN_DECIMALS=${TOKEN_DECIMALS:-6}
USD_PER_TOKEN=${USD_PER_TOKEN:-0.10}
ETH_PER_TOKEN=${ETH_PER_TOKEN:-0.00005}

# Check for required environment variables
if [ -z "$DISBURSEMENT_PRIVATE_KEY" ]; then
    echo "ERROR: DISBURSEMENT_PRIVATE_KEY environment variable is required"
    echo "Please set it with: export DISBURSEMENT_PRIVATE_KEY=your_private_key"
    exit 1
fi

# Build the payment processor
echo "Building payment processor..."
go build -o payment-processor ./payment_processor/cmd

# Run the payment processor
echo "Starting payment processor on port $PORT..."
echo "Using KNIRVROOT node at $KNIRVROOT_NODE_RPC"

./payment-processor