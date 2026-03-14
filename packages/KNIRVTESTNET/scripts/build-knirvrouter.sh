#!/bin/bash
set -e

echo "Building KNIRV-ROUTER for testnet..."

# Check if KNIRV-ROUTER is currently running
ROUTER_WAS_RUNNING=false
if [ -f "data/knirvrouter.pid" ]; then
    ROUTER_PID=$(cat data/knirvrouter.pid 2>/dev/null || echo "")
    if [ -n "$ROUTER_PID" ] && kill -0 "$ROUTER_PID" 2>/dev/null; then
        echo "⚠️  KNIRV-ROUTER is currently running (PID: $ROUTER_PID)"
        echo "Stopping KNIRV-ROUTER to update binary..."
        ROUTER_WAS_RUNNING=true

        # Use kill_knirv.sh to properly stop the router
        if [ -f "scripts/kill_knirv.sh" ]; then
            echo "Using kill_knirv.sh to stop router..."
            # Kill only router-related processes
            bash scripts/kill_knirv.sh --service router --quiet 2>/dev/null || {
                echo "kill_knirv.sh failed, using manual kill..."
                kill "$ROUTER_PID" 2>/dev/null || true
                sleep 2
                # Force kill if still running
                if kill -0 "$ROUTER_PID" 2>/dev/null; then
                    kill -9 "$ROUTER_PID" 2>/dev/null || true
                    sleep 1
                fi
            }
        else
            # Fallback to manual kill
            kill "$ROUTER_PID" 2>/dev/null || true
            sleep 2
            if kill -0 "$ROUTER_PID" 2>/dev/null; then
                kill -9 "$ROUTER_PID" 2>/dev/null || true
                sleep 1
            fi
        fi

        # Clean up PID file
        rm -f data/knirvrouter.pid
        echo "✅ KNIRV-ROUTER stopped successfully"
    fi
fi

# Ensure target directory exists
mkdir -p bin

# Now update the binary
echo "Updating KNIRV-ROUTER binary..."

# Check if we have a pre-built binary
if [ -f "../packages/KNIRVROUTER/bin/knirvrouter-headless" ]; then
    echo "Using existing KNIRV-ROUTER headless binary..."
    cp ../packages/KNIRVROUTER/bin/knirvrouter-headless bin/knirvrouter
    chmod +x bin/knirvrouter
    echo "✅ Copied existing KNIRV-ROUTER headless binary"
else
    echo "Building KNIRV-ROUTER from source..."
    cd ../packages/KNIRVROUTER
    go mod tidy
    go build -tags headless -o knirvrouter-headless ./main_headless.go
    cp knirvrouter-headless ../../KNIRVTESTNET/bin/knirvrouter
    chmod +x ../../KNIRVTESTNET/bin/knirvrouter
    cd ../../KNIRVTESTNET
    echo "✅ Built and copied KNIRV-ROUTER headless binary"
fi

# Create testnet data directories
echo "Setting up testnet data directories..."
mkdir -p data/knirvrouter

# Create testnet-specific configuration
echo "Creating testnet configuration..."
cat > data/knirvrouter/test.env << 'EOF'
# Testnet Configuration for KNIRV-ROUTER
TESTNET_MODE=true
LOCAL_NETWORK_MODE=true
MOCK_NRN_MINTING=true
SIMPLIFIED_CONSENSUS=true
DISABLE_XION_BRIDGE=true

# Basic Configuration
PORT=5001
MINERS_ADDRESS=KNIRVROUTER_Testnet_Miner
BLOCKCHAIN_NAME=KNIRVROUTER_TESTNET
CURRENCY_NAME=NRN
HEX_PREFIX=0x
ADDRESS_PREFIX=KNIRVROUTER-

# Testnet-specific settings
TESTNET_CHAIN_ID=knirvrouter-testnet-1
TESTNET_VALIDATORS=3
TESTNET_INITIAL_NRN=1000000000000

# Mining Configuration
MINING_DIFFICULTY=1
MINING_REWARD=100
DECIMAL=8

# Consensus Configuration
CONSENSUS_PAUSE_TIME=30

# Status Messages
SUCCESS=true
FAILED=false
PENDING=pending
TXN_VERIFICATION_SUCCESS=verified
TXN_VERIFICATION_FAILURE=failed
BLOCKCHAIN_STATUS=active

# Peer Configuration (empty for local testnet)
PEER_ADDRESSES=

# Installation
INSTALL_COMPLETE=true
EOF

# Copy configuration to config directory as well
cp data/knirvrouter/test.env config/knirvrouter-testnet.env

# If router was running before, restart it
if [ "$ROUTER_WAS_RUNNING" = true ]; then
    echo "Restarting KNIRV-ROUTER..."
    if [ -f "scripts/start-knirvrouter.sh" ]; then
        bash scripts/start-knirvrouter.sh
        if [ $? -eq 0 ]; then
            echo "✅ KNIRV-ROUTER restarted successfully after build"
        else
            echo "⚠️  KNIRV-ROUTER restart failed, you may need to start it manually"
        fi
    else
        echo "⚠️  start-knirvrouter.sh not found, please start router manually"
    fi
else
    echo "ℹ️  KNIRV-ROUTER was not running before build, not restarting"
fi

echo "KNIRV-ROUTER testnet build completed successfully!"
