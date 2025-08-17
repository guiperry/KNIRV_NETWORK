#!/bin/bash
set -e

# KNIRV Testnet Unified Startup Script
# This script builds and starts all KNIRV testnet components in the correct order

# Get script directory and change to KNIRVTESTNET root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTNET_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$TESTNET_ROOT"

echo "🧪 KNIRV TESTNET STARTUP"
echo "======================="
echo "Starting KNIRV Decentralized Trusted Execution Network in testnet mode..."
echo ""

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Function to print colored output
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

print_step() {
    echo -e "${PURPLE}[STEP]${NC} $1"
}

# Function to check if a port is available
check_port() {
    local port=$1
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        return 1
    else
        return 0
    fi
}

# Function to wait for service to be ready
wait_for_service() {
    local name=$1
    local url=$2
    local max_attempts=30
    local attempt=1
    
    print_status "Waiting for $name to be ready at $url..."
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s -f "$url" >/dev/null 2>&1; then
            print_success "$name is ready!"
            return 0
        fi
        
        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    print_error "$name failed to start within $((max_attempts * 2)) seconds"
    return 1
}

# Function to check if process is running
check_process() {
    local pid_file=$1
    local name=$2
    
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            print_success "$name is running (PID: $pid)"
            return 0
        else
            print_warning "$name PID file exists but process is not running"
            rm -f "$pid_file"
        fi
    fi
    return 1
}

# Create necessary directories
print_step "Creating directories..."
mkdir -p logs data bin config

# Check for required dependencies
print_step "Checking dependencies..."

# Check for Go
if ! command -v go &> /dev/null; then
    print_error "Go is required but not installed. Please install Go 1.19 or later."
    exit 1
fi

# Check for Rust/Cargo
if ! command -v cargo &> /dev/null; then
    print_error "Rust/Cargo is required but not installed. Please install Rust."
    exit 1
fi

# Check for Node.js
if ! command -v node &> /dev/null; then
    print_error "Node.js is required but not installed. Please install Node.js 18 or later."
    exit 1
fi

# Check for Python3
if ! command -v python3 &> /dev/null; then
    print_error "Python3 is required but not installed."
    exit 1
fi

print_success "All dependencies are available"

# Check port availability
print_step "Checking port availability..."
ports=(1317 8090 8082 8084 8086 8888)
port_names=("KNIRV-ORACLE" "KNIRVCHAIN" "KNIRVGRAPH" "KNIRV-NEXUS" "KNIRV-ROUTER" "KNIRV-GATEWAY")

for i in "${!ports[@]}"; do
    port=${ports[$i]}
    name=${port_names[$i]}
    
    if ! check_port $port; then
        print_error "Port $port is already in use (needed for $name)"
        print_status "Please stop the service using port $port and try again"
        exit 1
    fi
done

print_success "All required ports are available"

# Build all components
print_step "Building all components..."

components=("knirvoracle" "knirvchain" "knirvgraph" "knirvnexus" "knirvrouter" "knirvgateway")

for component in "${components[@]}"; do
    print_status "Building $component..."
    if ./scripts/build-$component.sh; then
        print_success "$component built successfully"
    else
        print_error "Failed to build $component"
        exit 1
    fi
done

print_success "All components built successfully"

# Start services in order
print_step "Starting services..."

# 1. Start KNIRV-ORACLE (blockchain foundation)
print_status "Starting KNIRV-ORACLE..."
if ./scripts/start-knirvoracle.sh; then
    wait_for_service "KNIRV-ORACLE" "http://localhost:1317/health" || exit 1
else
    print_error "Failed to start KNIRV-ORACLE"
    exit 1
fi

# 2. Start KNIRVCHAIN (smart contracts and LLM validation)
print_status "Starting KNIRVCHAIN..."
if ./scripts/start-knirvchain.sh; then
    wait_for_service "KNIRVCHAIN" "http://localhost:8090/health" || exit 1
else
    print_error "Failed to start KNIRVCHAIN"
    exit 1
fi

# 3. Start KNIRVGRAPH (graph storage and DHT)
print_status "Starting KNIRVGRAPH..."
if ./scripts/start-knirvgraph.sh; then
    wait_for_service "KNIRVGRAPH" "http://localhost:8082/height" || exit 1
else
    print_error "Failed to start KNIRVGRAPH"
    exit 1
fi

# 4. Start KNIRV-NEXUS (TEE simulation)
print_status "Starting KNIRV-NEXUS..."
if ./scripts/start-knirvnexus.sh; then
    wait_for_service "KNIRV-NEXUS" "http://localhost:8084/health" || exit 1
else
    print_error "Failed to start KNIRV-NEXUS"
    exit 1
fi

# 5. Start KNIRV-ROUTER (network routing)
print_status "Starting KNIRV-ROUTER..."
if ./scripts/start-knirvrouter.sh; then
    wait_for_service "KNIRV-ROUTER" "http://localhost:8086/status" || exit 1
else
    print_error "Failed to start KNIRV-ROUTER"
    exit 1
fi

# 6. Start KNIRV-GATEWAY (API gateway)
print_status "Starting KNIRV-GATEWAY..."
if ./scripts/start-knirvgateway.sh; then
    wait_for_service "KNIRV-GATEWAY" "http://localhost:8888/gateway/health" || exit 1
else
    print_error "Failed to start KNIRV-GATEWAY"
    exit 1
fi

print_success "All services started successfully!"

# Display status
echo ""
echo "🎉 KNIRV TESTNET IS RUNNING!"
echo "============================"
echo ""
echo "Service Status:"
echo "  🔗 KNIRV-ORACLE:    http://localhost:1317"
echo "  ⛓️  KNIRVCHAIN:   http://localhost:8090"
echo "  📊 KNIRVGRAPH:    http://localhost:8082"
echo "  🔒 KNIRV-NEXUS:   http://localhost:8084 (API) / http://localhost:8083 (GUI)"
echo "  🌐 KNIRV-ROUTER:  http://localhost:8086"
echo "  🚪 KNIRV-GATEWAY: http://localhost:8888"
echo ""
echo "Testnet Endpoints:"
echo "  📋 Gateway Health:    http://localhost:8888/gateway/health"
echo "  🔍 Service Discovery: http://localhost:8888/gateway/services"
echo "  🧪 Testnet Status:    http://localhost:8888/gateway/testnet/status"
echo "  🔑 Auth Tokens:       http://localhost:8888/auth/testnet-tokens"
echo ""
echo "Testnet Features Enabled:"
echo "  ✅ Simplified authentication"
echo "  ✅ Mock LLM validation"
echo "  ✅ TEE simulation"
echo "  ✅ Local network mode"
echo "  ✅ Static service discovery"
echo "  ✅ In-memory storage"
echo ""
echo "To stop the testnet, run: ./stop-testnet.sh"
echo "To view logs, check the ./logs/ directory"
echo ""
print_success "KNIRV Testnet startup completed successfully!"
