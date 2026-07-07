#!/bin/bash
set -e

# KNIRV Production Network - IPFS Setup Script
# This script sets up IPFS for the production KNIRV network

echo "🌐 KNIRV PRODUCTION NETWORK - IPFS SETUP"
echo "========================================"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Configuration
IPFS_VERSION="v0.24.0"
IPFS_PATH="${IPFS_PATH:-./data/ipfs-production}"
IPFS_API_PORT="${IPFS_API_PORT:-5001}"
IPFS_GATEWAY_PORT="${IPFS_GATEWAY_PORT:-8080}"
IPFS_SWARM_PORT="${IPFS_SWARM_PORT:-4001}"
ENVIRONMENT="${ENVIRONMENT:-production}"

print_status "IPFS Configuration:"
print_status "  Version: $IPFS_VERSION"
print_status "  Data Path: $IPFS_PATH"
print_status "  API Port: $IPFS_API_PORT"
print_status "  Gateway Port: $IPFS_GATEWAY_PORT"
print_status "  Swarm Port: $IPFS_SWARM_PORT"
print_status "  Environment: $ENVIRONMENT"

# Check if IPFS is already installed
if command -v ipfs &> /dev/null; then
    INSTALLED_VERSION=$(ipfs version --number)
    print_success "IPFS already installed (Version: $INSTALLED_VERSION)"
else
    print_status "Installing IPFS $IPFS_VERSION..."
    
    # Download and install IPFS
    IPFS_DIST_URL="https://dist.ipfs.tech/kubo/$IPFS_VERSION/kubo_${IPFS_VERSION}_linux-amd64.tar.gz"
    
    print_status "Downloading IPFS from $IPFS_DIST_URL"
    wget -q "$IPFS_DIST_URL" -O /tmp/kubo.tar.gz
    
    print_status "Extracting IPFS..."
    cd /tmp
    tar -xzf kubo.tar.gz
    
    print_status "Installing IPFS binary..."
    sudo mv kubo/ipfs /usr/local/bin/
    rm -rf kubo kubo.tar.gz
    
    print_success "IPFS $IPFS_VERSION installed successfully"
fi

# Set IPFS path for production
export IPFS_PATH="$IPFS_PATH"

# Create data directory
mkdir -p "$IPFS_PATH"

# Initialize IPFS if not already done
if [ ! -f "$IPFS_PATH/config" ]; then
    print_status "Initializing IPFS for KNIRV production network..."
    ipfs init --profile server
    print_success "IPFS initialized with server profile"
else
    print_status "IPFS already initialized"
fi

# Configure IPFS for KNIRV production network
print_status "Configuring IPFS for KNIRV production network..."

# API and Gateway addresses
ipfs config Addresses.API "/ip4/0.0.0.0/tcp/$IPFS_API_PORT"
ipfs config Addresses.Gateway "/ip4/0.0.0.0/tcp/$IPFS_GATEWAY_PORT"

# Swarm addresses
ipfs config --json Addresses.Swarm '[
  "/ip4/0.0.0.0/tcp/'$IPFS_SWARM_PORT'",
  "/ip6/::/tcp/'$IPFS_SWARM_PORT'",
  "/ip4/0.0.0.0/udp/'$IPFS_SWARM_PORT'/quic",
  "/ip6/::/udp/'$IPFS_SWARM_PORT'/quic"
]'

# CORS configuration for KNIRV web applications
ipfs config --json API.HTTPHeaders.Access-Control-Allow-Origin '["*"]'
ipfs config --json API.HTTPHeaders.Access-Control-Allow-Methods '["PUT", "POST", "GET", "DELETE"]'
ipfs config --json API.HTTPHeaders.Access-Control-Allow-Headers '["Authorization", "Content-Type"]'

# Connection management for production
ipfs config --json Swarm.ConnMgr.HighWater 200
ipfs config --json Swarm.ConnMgr.LowWater 100
ipfs config --json Swarm.ConnMgr.GracePeriod '"30s"'

# Resource management
ipfs config --json Datastore.StorageMax '"10GB"'
ipfs config --json Datastore.GCPeriod '"1h"'

# Network configuration for KNIRV
ipfs config --json Discovery.MDNS.Enabled true
ipfs config --json Routing.Type '"dhtclient"'

# Security settings for production
ipfs config --json Gateway.PublicGateways 'null'
ipfs config --json Gateway.NoFetch false

# KNIRV-specific configuration
ipfs config --json Experimental.FilestoreEnabled true
ipfs config --json Experimental.UrlstoreEnabled true
ipfs config --json Experimental.GraphsyncEnabled true

# Set custom agent version for KNIRV network identification
KNIRV_AGENT_VERSION="knirv-production-$(date +%Y%m%d)"
ipfs config --json Version.AgentVersion "\"kubo/$IPFS_VERSION/$KNIRV_AGENT_VERSION\""

print_success "IPFS configured for KNIRV production network"

# Create systemd service for production deployment
if [ "$ENVIRONMENT" = "production" ] && [ -d "/etc/systemd/system" ]; then
    print_status "Creating systemd service for IPFS..."
    
    sudo tee /etc/systemd/system/ipfs-knirv.service > /dev/null <<EOF
[Unit]
Description=IPFS daemon for KNIRV Network
After=network.target

[Service]
Type=notify
User=$USER
Environment=IPFS_PATH=$IPFS_PATH
ExecStart=/usr/local/bin/ipfs daemon --enable-gc --migrate
Restart=on-failure
RestartSec=10
KillSignal=SIGINT

[Install]
WantedBy=multi-user.target
EOF

    sudo systemctl daemon-reload
    sudo systemctl enable ipfs-knirv
    print_success "Systemd service created and enabled"
fi

# Create startup script
print_status "Creating IPFS startup script..."
cat > ./scripts/start-ipfs-production.sh << 'EOF'
#!/bin/bash
set -e

# KNIRV Production IPFS Startup Script

IPFS_PATH="${IPFS_PATH:-./data/ipfs-production}"
export IPFS_PATH

echo "🌐 Starting IPFS for KNIRV Production Network..."
echo "IPFS Path: $IPFS_PATH"

# Check if IPFS is configured
if [ ! -f "$IPFS_PATH/config" ]; then
    echo "❌ IPFS not configured. Run setup-ipfs-production.sh first."
    exit 1
fi

# Start IPFS daemon
echo "🚀 Starting IPFS daemon..."
ipfs daemon --enable-gc --migrate > ./logs/ipfs-production.log 2>&1 &
IPFS_PID=$!

echo $IPFS_PID > ./data/ipfs-production.pid
echo "✅ IPFS started with PID $IPFS_PID"
echo "📊 API: http://localhost:5001"
echo "🌐 Gateway: http://localhost:8080"
echo "🔗 Swarm: tcp://localhost:4001"
echo "📝 Logs: ./logs/ipfs-production.log"
EOF

chmod +x ./scripts/start-ipfs-production.sh

print_success "IPFS setup completed for KNIRV production network!"
print_status "Next steps:"
print_status "1. Start IPFS: ./scripts/start-ipfs-production.sh"
print_status "2. Test IPFS: curl http://localhost:$IPFS_API_PORT/api/v0/version"
print_status "3. Integrate with KNIRV services"

echo ""
print_status "IPFS Endpoints:"
print_status "  API: http://localhost:$IPFS_API_PORT"
print_status "  Gateway: http://localhost:$IPFS_GATEWAY_PORT"
print_status "  Swarm: tcp://localhost:$IPFS_SWARM_PORT"
