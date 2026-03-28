#!/bin/bash
set -e

# KNIRV Testnet Monitoring Startup Script
# This script starts the KNIRV testnet with monitoring

echo "🧪 KNIRV TESTNET - MONITORING STARTUP"
echo "====================================="

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
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
KNIRV_ROOT="$(dirname "$PROJECT_ROOT")"

print_status "Starting KNIRV testnet with monitoring..."
print_status "Project root: $PROJECT_ROOT"
print_status "KNIRV root: $KNIRV_ROOT"

# Check if monitoring stack is running (use different ports for testnet)
check_monitoring_stack() {
    print_status "Checking testnet monitoring stack status..."
    
    # For testnet, we'll use different ports to avoid conflicts
    # This would require a separate docker-compose file for testnet monitoring
    
    if [ ! -f "$PROJECT_ROOT/docker-compose.testnet-monitoring.yml" ]; then
        print_status "Creating testnet monitoring configuration..."
        
        # Create testnet-specific monitoring stack
        sed 's/9090:9090/9091:9090/g; s/3000:3000/3001:3000/g; s/5601:5601/5602:5601/g; s/9093:9093/9094:9093/g' \
            "$PROJECT_ROOT/docker-compose.monitoring.yml" > "$PROJECT_ROOT/docker-compose.testnet-monitoring.yml"
    fi
    
    if ! docker-compose -f "$PROJECT_ROOT/docker-compose.testnet-monitoring.yml" ps | grep -q "Up"; then
        print_warning "Testnet monitoring stack not running. Starting it now..."
        cd "$PROJECT_ROOT"
        docker-compose -f docker-compose.testnet-monitoring.yml up -d
        
        # Wait for services to be ready
        print_status "Waiting for testnet monitoring services to be ready..."
        sleep 30
        
        # Check Prometheus (testnet port)
        for i in {1..30}; do
            if curl -s http://localhost:9091/-/healthy >/dev/null 2>&1; then
                print_success "Testnet Prometheus is ready"
                break
            fi
            sleep 2
            if [ $i -eq 30 ]; then
                print_error "Testnet Prometheus failed to start"
                exit 1
            fi
        done
        
        # Check Grafana (testnet port)
        for i in {1..30}; do
            if curl -s http://localhost:3001/api/health >/dev/null 2>&1; then
                print_success "Testnet Grafana is ready"
                break
            fi
            sleep 2
            if [ $i -eq 30 ]; then
                print_error "Testnet Grafana failed to start"
                exit 1
            fi
        done
        
    else
        print_success "Testnet monitoring stack is already running"
    fi
}

# Start KNIRV testnet
start_testnet() {
    print_status "Starting KNIRV testnet..."
    
    cd "$KNIRV_ROOT/KNIRVTESTNET"
    
    # Use Podman for testnet as configured
    if [ -f "scripts/start-podman.sh" ]; then
        ./scripts/start-podman.sh
        print_success "KNIRV testnet started with Podman"
    elif [ -f "package.json" ]; then
        # Fallback to npm start
        npm run podman:start || npm start
        print_success "KNIRV testnet started"
    else
        print_error "Testnet startup scripts not found"
        exit 1
    fi
}

# Verify testnet services
verify_testnet_services() {
    print_status "Verifying testnet services are running..."
    
    local services=(
        "ipfs:5001:/api/v0/version"
        "knirvoracle:1317:/health"
        "testnet-gateway:10000:/"
    )
    
    local all_healthy=true
    
    for service in "${services[@]}"; do
        local name="${service%%:*}"
        local port_path="${service#*:}"
        local port="${port_path%:*}"
        local path="${port_path#*:}"
        local url="http://localhost:$port$path"
        
        print_status "Checking $name at $url..."
        
        # Give services more time to start
        for i in {1..20}; do
            if curl -s -f "$url" >/dev/null 2>&1; then
                print_success "$name is healthy"
                break
            fi
            sleep 3
            if [ $i -eq 20 ]; then
                print_warning "$name is not responding (this may be normal for testnet)"
                all_healthy=false
                break
            fi
        done
    done
    
    if [ "$all_healthy" = true ]; then
        print_success "All testnet services are healthy"
    else
        print_warning "Some testnet services may still be starting. This is normal."
    fi
}

# Start the network monitor GUI for testnet
start_monitor_gui() {
    print_status "Starting KNIRV Network Monitor GUI for testnet..."
    
    cd "$PROJECT_ROOT"
    
    if [ -f "bin/knirv-network-monitor" ]; then
        print_status "Launching GUI application for testnet..."
        ./bin/knirv-network-monitor --network testnet &
        MONITOR_PID=$!
        echo $MONITOR_PID > "$PROJECT_ROOT/data/testnet-monitor.pid"
        print_success "Testnet Network Monitor GUI started (PID: $MONITOR_PID)"
    else
        print_error "Network Monitor binary not found. Please run setup first."
        exit 1
    fi
}

# Test IPFS integration
test_ipfs_integration() {
    print_status "Testing IPFS integration..."
    
    cd "$KNIRV_ROOT/KNIRVTESTNET"
    
    if [ -f "scripts/test-ipfs.sh" ]; then
        ./scripts/test-ipfs.sh
        print_success "IPFS integration test completed"
    else
        print_warning "IPFS test script not found, skipping test"
    fi
}

# Display testnet URLs
display_urls() {
    echo ""
    print_success "🎉 KNIRV Testnet with Monitoring is ready!"
    echo ""
    echo "📊 Testnet Monitoring Dashboards:"
    echo "================================="
    echo "Grafana:         http://localhost:3001 (admin/admin123)"
    echo "Prometheus:      http://localhost:9091"
    echo "Kibana:          http://localhost:5602"
    echo "AlertManager:    http://localhost:9094"
    echo ""
    echo "🧪 KNIRV Testnet Services:"
    echo "=========================="
    echo "IPFS API:        http://localhost:5001"
    echo "IPFS Gateway:    http://localhost:8080"
    echo "KNIRV Oracle:    http://localhost:1317"
    echo "Testnet Gateway: http://localhost:10000"
    echo ""
    echo "🌐 Web Applications (via Testnet Gateway):"
    echo "==========================================="
    echo "Main Portal:             http://localhost:10000"
    echo "GraphChain Explorer:     http://localhost:10000/graphchain-explorer"
    echo "Nexus Portal:            http://localhost:10000/nexus-portal"
    echo "Developer Portal:  http://localhost:10000/developer-portal"
    echo ""
    echo "🖥️  Network Monitor:"
    echo "==================="
    echo "GUI Application: Running (check desktop)"
    echo "API Endpoint:    http://localhost:8090"
    echo ""
    echo "📝 Logs and Data:"
    echo "================="
    echo "Service Logs:    ./logs/"
    echo "Testnet Logs:    ../KNIRVTESTNET/logs/"
    echo "Configuration:   ./config/"
    echo ""
    print_status "To stop testnet: ./scripts/stop-testnet-monitoring.sh"
}

# Main execution
main() {
    print_status "Starting KNIRV testnet monitoring setup..."
    
    check_monitoring_stack
    start_testnet
    verify_testnet_services
    test_ipfs_integration
    start_monitor_gui
    display_urls
    
    print_success "Testnet monitoring startup completed!"
}

# Handle script interruption
cleanup() {
    print_warning "Received interrupt signal. Cleaning up..."
    if [ -f "$PROJECT_ROOT/data/testnet-monitor.pid" ]; then
        local monitor_pid=$(cat "$PROJECT_ROOT/data/testnet-monitor.pid")
        if kill -0 "$monitor_pid" 2>/dev/null; then
            kill "$monitor_pid"
            print_status "Testnet Network Monitor GUI stopped"
        fi
        rm -f "$PROJECT_ROOT/data/testnet-monitor.pid"
    fi
    exit 0
}

trap cleanup SIGINT SIGTERM

# Run main function
main "$@"
