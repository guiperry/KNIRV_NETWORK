#!/bin/bash

# Start real KNIRVCONTROLLER with testnet configuration

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTNET_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"
PROJECT_ROOT="$(dirname "$TESTNET_ROOT")"
CONTROLLER_DIR="$PROJECT_ROOT/KNIRVCONTROLLER"

if [ ! -d "$CONTROLLER_DIR" ]; then
    echo "KNIRVCONTROLLER not found, using demo service"
    exit 1
fi

cd "$CONTROLLER_DIR"

# Set environment variables for testnet
export NODE_ENV=testnet
export KNIRV_TESTNET_MODE=true
export KNIRV_CONFIG_FILE="$TESTNET_ROOT/config/knirvcontroller/testnet-config.json"
export PORT=8088
export API_PORT=8089

# Start the controller
if [ -f "dist/index.js" ]; then
    echo "Starting built KNIRVCONTROLLER..."
    node dist/index.js
elif [ -f "package.json" ] && grep -q "dev" package.json; then
    echo "Starting KNIRVCONTROLLER in development mode..."
    npm run dev
else
    echo "Starting KNIRVCONTROLLER with npm start..."
    npm start
fi
