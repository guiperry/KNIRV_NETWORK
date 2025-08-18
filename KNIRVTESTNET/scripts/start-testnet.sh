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

# Function to run smart initialization using smart-start.js logic
run_smart_initialization() {
    print_status "Running smart initialization checks..."

    # Use smart-start.js for intelligent initialization
    if [ -f "scripts/smart-start.js" ]; then
        print_status "Using smart-start.js for initialization logic..."

        # Extract just the initialization logic from smart-start.js
        node -e "
        const path = require('path');
        const fs = require('fs');

        async function runInitChecks() {
          console.log('🔧 Smart Initialization');
          console.log('======================');

          // Health check
          console.log('\\n🏥 Running health check...');
          try {
            const { checkHealth } = require('./scripts/check-health');
            const healthy = checkHealth();
            if (!healthy) {
              console.error('❌ Health check failed. Please fix issues before starting.');
              process.exit(1);
            }
            console.log('✅ Health check passed');
          } catch (error) {
            console.warn('⚠️  Health check failed to run:', error.message);
            console.log('Continuing with startup...');
          }

          // Load endpoints
          console.log('\\n🔧 Loading endpoints...');
          try {
            const { loadEndpoints } = require('./scripts/load-endpoints');
            const { endpoints, config } = loadEndpoints('testnet');
            console.log(\`✅ Loaded \${Object.keys(endpoints).length} endpoints\`);
            console.log(\`✅ Environment: \${config.DEPLOYMENT_ENV}\`);
          } catch (error) {
            console.error('❌ Failed to load endpoints:', error.message);
            process.exit(1);
          }

          // Check dependencies
          const nodeModulesPath = path.join(__dirname, 'node_modules');
          if (!fs.existsSync(nodeModulesPath)) {
            console.log('\\n📦 Dependencies need to be installed');
            console.log('Run: npm install');
            process.exit(1);
          }

          console.log('✅ Smart initialization completed successfully');
        }

        runInitChecks().catch(error => {
          console.error('❌ Smart initialization failed:', error.message);
          process.exit(1);
        });
        "

        if [ $? -eq 0 ]; then
            print_success "Smart initialization completed"
        else
            print_error "Smart initialization failed"
            exit 1
        fi
    else
        print_warning "smart-start.js not found, using basic initialization"
    fi
}

# Function to discover service port from PID file
discover_service_port() {
    local service_name=$1
    local pid_file=$2
    local default_port=$3

    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file" 2>/dev/null)
        if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
            # Get all listening ports for this process
            local ports=$(lsof -Pan -p "$pid" -i 2>/dev/null | grep LISTEN | sed 's/.*:\([0-9]*\).*/\1/' | sort -n)

            # For specific services, prefer certain ports
            case "$service_name" in
                "KNIRV-ORACLE")
                    if echo "$ports" | grep -q "^1317$"; then
                        echo "1317"
                        return 0
                    fi
                    ;;
                "KNIRV-GATEWAY")
                    if echo "$ports" | grep -q "^8888$"; then
                        echo "8888"
                        return 0
                    fi
                    ;;
                "KNIRV-ROUTER")
                    if echo "$ports" | grep -q "^8086$"; then
                        echo "8086"
                        return 0
                    fi
                    ;;
                "NANDA-ANS")
                    if echo "$ports" | grep -q "^9002$"; then
                        echo "9002"
                        return 0
                    fi
                    ;;
            esac

            # Return first available port if preferred not found
            if [ -n "$ports" ]; then
                echo "$ports" | head -1
                return 0
            fi
        fi
    fi

    # Return default port if discovery fails
    echo "$default_port"
}

# Function to wait for service to be ready with dynamic port discovery
wait_for_service() {
    local name=$1
    local default_port=$2
    local health_endpoint=$3
    local pid_file=$4
    local max_attempts=30
    local attempt=1

    print_status "Waiting for $name to be ready..."

    # Discover actual port
    local port=$(discover_service_port "$name" "$pid_file" "$default_port")
    local url="http://localhost:${port}${health_endpoint}"

    echo "Checking $name at $url"

    while [ $attempt -le $max_attempts ]; do
        if curl -s --max-time 5 "$url" >/dev/null 2>&1; then
            print_success "$name is ready on port $port!"
            return 0
        fi

        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done

    print_error "$name failed to start within timeout (checked $url)"
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

# Run smart initialization (replaces static dependency and port checking)
print_step "Smart Initialization..."
run_smart_initialization

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
    wait_for_service "KNIRV-ORACLE" "1317" "/health" "data/knirvoracle.pid" || exit 1
else
    print_error "Failed to start KNIRV-ORACLE"
    exit 1
fi

# 2. Start KNIRVCHAIN (smart contracts and LLM validation)
print_status "Starting KNIRVCHAIN..."
if ./scripts/start-knirvchain.sh; then
    wait_for_service "KNIRVCHAIN" "8090" "/health" "data/knirvchain.pid" || exit 1
else
    print_error "Failed to start KNIRVCHAIN"
    exit 1
fi

# 3. Start KNIRVGRAPH (graph storage and DHT)
print_status "Starting KNIRVGRAPH..."
if ./scripts/start-knirvgraph.sh; then
    wait_for_service "KNIRVGRAPH" "8082" "/height" "data/knirvgraph.pid" || exit 1
else
    print_error "Failed to start KNIRVGRAPH"
    exit 1
fi

# 4. Start KNIRV-NEXUS (TEE simulation)
print_status "Starting KNIRV-NEXUS..."
if ./scripts/start-knirvnexus.sh; then
    wait_for_service "KNIRV-NEXUS" "8084" "/health" "data/knirvnexus.pid" || exit 1
else
    print_error "Failed to start KNIRV-NEXUS"
    exit 1
fi

# 5. Start KNIRV-ROUTER (network routing)
print_status "Starting KNIRV-ROUTER..."
if ./scripts/start-knirvrouter.sh; then
    wait_for_service "KNIRV-ROUTER" "8086" "/status" "data/knirvrouter.pid" || exit 1
else
    print_error "Failed to start KNIRV-ROUTER"
    exit 1
fi

# 6. Start KNIRV-GATEWAY (API gateway)
print_status "Starting KNIRV-GATEWAY..."
if ./scripts/start-knirvgateway.sh; then
    wait_for_service "KNIRV-GATEWAY" "8888" "/gateway/health" "data/knirvgateway.pid" || exit 1
else
    print_error "Failed to start KNIRV-GATEWAY"
    exit 1
fi

# 7. Start NANDA ANS (Agent Registry)
print_status "Starting NANDA ANS..."
if ./scripts/start-nanda-ans.sh; then
    wait_for_service "NANDA-ANS" "9002" "/api/health" "data/nanda-ans.pid" || exit 1
else
    print_error "Failed to start NANDA ANS"
    exit 1
fi

# 8. Start Health Monitor
print_status "Starting Health Monitor..."
if ./scripts/start-health-monitor.sh; then
    wait_for_service "HEALTH-MONITOR" "10001" "/health-monitor/status" "data/health-monitor.pid" || exit 1
else
    print_error "Failed to start Health Monitor"
    exit 1
fi

print_success "All services started successfully!"

# Start KNIRVTESTNET Server using smart-start.js
print_step "Starting KNIRVTESTNET Server with Smart Logic..."

# Check if smart-start.js exists
if [ ! -f "scripts/smart-start.js" ]; then
    print_error "smart-start.js not found"
    exit 1
fi

# Use smart-start.js for intelligent server startup
print_status "Launching KNIRVTESTNET server with smart-start.js..."
print_status "This includes: health checks, endpoint loading, dependency verification, and graceful startup"

# Start the server using smart-start.js in background
if node scripts/smart-start.js &
then
    # Store the server PID
    SERVER_PID=$!
    echo $SERVER_PID > data/knirvtestnet-server.pid
    print_success "KNIRVTESTNET server started with PID $SERVER_PID"

    # Give the smart-start.js more time to complete its initialization
    print_status "Waiting for smart initialization to complete..."
    sleep 5

    # Use dynamic port discovery to check if server is responding
    if wait_for_service "KNIRVTESTNET-SERVER" "10000" "/health" "data/knirvtestnet-server.pid"; then
        print_success "KNIRVTESTNET server is ready and responding!"
    else
        print_warning "KNIRVTESTNET server may still be initializing (smart-start.js handles this)"
        print_status "Check logs for detailed startup progress"
    fi
else
    print_error "Failed to start KNIRVTESTNET server with smart-start.js"
    exit 1
fi

# Display final status
echo ""
echo "🎉 KNIRV TESTNET IS FULLY RUNNING!"
echo "=================================="
echo ""
echo "Core Services:"
echo "  🔗 KNIRV-ORACLE:    http://localhost:1317"
echo "  ⛓️  KNIRVCHAIN:   http://localhost:8090"
echo "  📊 KNIRVGRAPH:    http://localhost:8082"
echo "  🔒 KNIRV-NEXUS:   http://localhost:8084 (API) / http://localhost:8083 (GUI)"
echo "  🌐 KNIRV-ROUTER:  http://localhost:8086"
echo "  🚪 KNIRV-GATEWAY: http://localhost:8888"
echo "  🤖 NANDA ANS:     http://localhost:9002"
echo "  🏥 HEALTH MONITOR: http://localhost:10001/health-monitor"
echo ""
echo "Main Portal:"
echo "  🌐 KNIRVTESTNET:   http://localhost:10000"
echo ""
echo "Quick Access:"
echo "  📊 Status Check:     npm run testnet:status"
echo "  📋 Gateway Health:   http://localhost:8888/gateway/health"
echo "  🔍 Service Discovery: http://localhost:8888/gateway/services"
echo "  🧪 Testnet Status:   http://localhost:8888/gateway/testnet/status"
echo "  🔑 Auth Tokens:      http://localhost:8888/auth/testnet-tokens"
echo ""
echo "Management Commands:"
echo "  🛑 Stop testnet:     npm run testnet:stop"
echo "  🔄 Restart testnet:  npm run testnet:restart"
echo "  📊 View logs:        npm run logs"
echo "  🔍 List processes:   npm run services:list"
echo ""
echo "Testnet Features Enabled:"
echo "  ✅ Simplified authentication"
echo "  ✅ Mock LLM validation"
echo "  ✅ TEE simulation"
echo "  ✅ Local network mode"
echo "  ✅ Dynamic service discovery"
echo "  ✅ In-memory storage"
echo "  ✅ Smart initialization (health checks, endpoint loading)"
echo "  ✅ Dynamic port discovery"
echo "  ✅ Intelligent dependency management"
echo "  ✅ Graceful startup and shutdown"
echo ""
print_success "KNIRV Testnet startup completed successfully!"
echo ""
print_status "The testnet is now running in the background."
print_status "Use 'npm run testnet:status' to check service health."
print_status "Use 'npm run testnet:stop' to stop all services."
