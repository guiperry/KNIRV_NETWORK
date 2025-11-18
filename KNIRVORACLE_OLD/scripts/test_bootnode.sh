#!/bin/bash

# Test script for bootnode functionality

# Set up colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== KNIRVORACLE Bootnode Test Suite ===${NC}"
echo "This script will test the bootnode functionality"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed${NC}"
    exit 1
fi

# Check if Node.js is installed (for registry service)
if ! command -v node &> /dev/null; then
    echo -e "${RED}Error: Node.js is not installed${NC}"
    exit 1
fi

# Check if required Node.js packages are installed
echo -e "${YELLOW}Checking Node.js dependencies...${NC}"
if ! npm list express &> /dev/null || ! npm list stun &> /dev/null; then
    echo -e "${YELLOW}Installing required Node.js packages...${NC}"
    npm install express stun
fi

# Run the unit tests
echo -e "${YELLOW}Running bootnode unit tests...${NC}"
go test -v -run "^Test(Bootnode|DiscoverPublicAddress)" ./...

# Start the registry service in the background
echo -e "${YELLOW}Starting registry service...${NC}"
node registry-service.js &
REGISTRY_PID=$!

# Wait for the registry service to start
sleep 2

# Start a bootnode in the background
echo -e "${YELLOW}Starting bootnode...${NC}"
go run . --bootnode --p2p.port=5050 &
BOOTNODE_PID=$!

# Wait for the bootnode to start
sleep 5

# Start a client node in the background
echo -e "${YELLOW}Starting client node...${NC}"
go run . --p2p.port=5051 &
CLIENT_PID=$!

# Wait for the client node to start
sleep 5

# Test if the bootnode is registered with the registry
echo -e "${YELLOW}Testing bootnode registration...${NC}"
REGISTRY_RESPONSE=$(curl -s http://localhost:3003/nodes)
if [[ $REGISTRY_RESPONSE == *"5050"* ]]; then
    echo -e "${GREEN}Bootnode successfully registered with registry${NC}"
else
    echo -e "${RED}Bootnode registration failed${NC}"
    echo "Registry response: $REGISTRY_RESPONSE"
fi

# Clean up
echo -e "${YELLOW}Cleaning up...${NC}"
kill $CLIENT_PID
kill $BOOTNODE_PID
kill $REGISTRY_PID

echo -e "${GREEN}Test completed${NC}"