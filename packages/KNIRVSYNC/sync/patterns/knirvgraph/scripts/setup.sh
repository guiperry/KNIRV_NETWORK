#!/bin/bash

# Initialize testnet setup
set -e

NODES=3
CHAIN_ID="blockchain-testnet"
BASE_DIR="./testnet"

echo "Setting up testnet with $NODES nodes..."

# Clean up existing testnet
rm -rf $BASE_DIR
mkdir -p $BASE_DIR

# Initialize nodes
for i in $(seq 1 $NODES); do
    NODE_DIR="$BASE_DIR/node$i"
    mkdir -p $NODE_DIR
    
    echo "Initializing node $i..."
    tendermint init --home $NODE_DIR
    
    # Update config
    sed -i "s/moniker = \".*\"/moniker = \"node$i\"/" $NODE_DIR/config/config.toml
    sed -i "s/chain_id = \".*\"/chain_id = \"$CHAIN_ID\"/" $NODE_DIR/config/genesis.json
done

# Create genesis file
echo "Creating genesis file..."
GENESIS_FILE="$BASE_DIR/node1/config/genesis.json"

# Copy genesis to all nodes
for i in $(seq 2 $NODES); do
    cp $GENESIS_FILE $BASE_DIR/node$i/config/genesis.json
done

# Get node IDs and create persistent peers
PEERS=""
for i in $(seq 1 $NODES); do
    NODE_ID=$(tendermint show_node_id --home $BASE_DIR/node$i)
    if [ $i -eq 1 ]; then
        PEERS="$NODE_ID@localhost:$((26656 + $i - 1))"
    else
        PEERS="$PEERS,$NODE_ID@localhost:$((26656 + $i - 1))"
    fi
done

# Update persistent peers in config
for i in $(seq 1 $NODES); do
    sed -i "s/persistent_peers = \".*\"/persistent_peers = \"$PEERS\"/" $BASE_DIR/node$i/config/config.toml
    sed -i "s/laddr = \"tcp:\/\/127.0.0.1:26656\"/laddr = \"tcp:\/\/0.0.0.0:$((26656 + $i - 1))\"/" $BASE_DIR/node$i/config/config.toml
    sed -i "s/laddr = \"tcp:\/\/127.0.0.1:26657\"/laddr = \"tcp:\/\/0.0.0.0:$((26657 + $i - 1))\"/" $BASE_DIR/node$i/config/config.toml
done

echo "Testnet setup complete!"
echo "To start the testnet, run:"
for i in $(seq 1 $NODES); do
    echo "  Terminal $i: blockchain-node -home $BASE_DIR/node$i -rpc-port $((8080 + $i - 1))"
done