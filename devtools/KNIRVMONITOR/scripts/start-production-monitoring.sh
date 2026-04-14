#!/bin/bash
set -e

# KNIRV Production Network Monitoring Startup Script
# This script starts the production KNIRV network with full monitoring

echo "🚀 KNIRV PRODUCTION NETWORK - MONITORING STARTUP"
echo "==============================================="

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

print_status "Starting KNIRV production network with monitoring..."
print_status "Project root: $PROJECT_ROOT"
print_status "KNIRV root: $KNIRV_ROOT"

# Check if monitoring stack is running
check_monitoring_stack() {
    print_status "Checking monitoring stack status..."
    
    if ! docker-compose -f "$PROJECT_ROOT/docker-compose.monitoring.yml" ps | grep -q "Up"; then
        print_warning "Monitoring stack not running. Starting it now..."
        cd "$PROJECT_ROOT"
        docker-compose -f docker-compose.monitoring.yml up -d
        
        # Wait for services to be ready
        print_status "Waiting for monitoring services to be ready..."
        sleep 30
        
        # Check Prometheus
        for i in {1..30}; do
            if curl -s http://localhost:9090/-/healthy >/dev/null 2>&1; then
                print_success "Prometheus is ready"
                break
            fi
            sleep 2
            if [ $i -eq 30 ]; then
                print_error "Prometheus failed to start"
                exit 1
            fi
        done
        
        # Check Grafana
        for i in {1..30}; do
            if curl -s http://localhost:3000/api/health >/dev/null 2>&1; then
                print_success "Grafana is ready"
                break
            fi
            sleep 2
            if [ $i -eq 30 ]; then
                print_error "Grafana failed to start"
                exit 1
            fi
        done
        
        # Check Elasticsearch
        for i in {1..60}; do
            if curl -s http://localhost:9200/_cluster/health >/dev/null 2>&1; then
                print_success "Elasticsearch is ready"
                break
            fi
            sleep 2
            if [ $i -eq 60 ]; then
                print_error "Elasticsearch failed to start"
                exit 1
            fi
        done
        
    else
        print_success "Monitoring stack is already running"
    fi
}

# Start IPFS for production
start_ipfs() {
    print_status "Starting IPFS for production network..."
    
    cd "$KNIRV_ROOT"
    
    if [ -f "scripts/setup-ipfs-production.sh" ]; then
        ./scripts/setup-ipfs-production.sh
    else
        print_error "IPFS setup script not found. Please run setup first."
        exit 1
    fi
    
    if [ -f "scripts/start-ipfs-production.sh" ]; then
        ./scripts/start-ipfs-production.sh
    else
        print_error "IPFS start script not found."
        exit 1
    fi
    
    # Wait for IPFS to be ready
    print_status "Waiting for IPFS to be ready..."
    for i in {1..30}; do
        if curl -s http://localhost:5001/api/v0/version >/dev/null 2>&1; then
            print_success "IPFS is ready"
            break
        fi
        sleep 2
        if [ $i -eq 30 ]; then
            print_error "IPFS failed to start"
            exit 1
        fi
    done
}

# Start KNIRV production services
start_knirv_services() {
    print_status "Starting KNIRV production services..."
    
    cd "$KNIRV_ROOT"
    
    if [ -f "scripts/manage-knirv.sh" ]; then
        # Start all KNIRV services (IPFS already started)
        ./scripts/manage-knirv.sh start knirvoracle
        sleep 3
        ./scripts/manage-knirv.sh start knirvchain
        sleep 3
        ./scripts/manage-knirv.sh start knirvgraph
        sleep 3
        ./scripts/manage-knirv.sh start knirvnexus
        sleep 3
        ./scripts/manage-knirv.sh start knirvrouter
        sleep 3
        ./scripts/manage-knirv.sh start gateway
        
        print_success "KNIRV services started"
    else
        print_error "KNIRV management script not found"
        exit 1
    fi
}

# Verify all services are running
verify_services() {
    print_status "Verifying all services are running..."
    
    local services=(
        "ipfs:5001:/api/v0/version"
        "knirvoracle:8083:/health"
        "knirvchain:8080:/health"
        "knirvgraph:8081:/height"
        "knirvnexus:8082:/health"
        "knirvrouter:5001:/status"
    )
    
    local all_healthy=true
    
    for service in "${services[@]}"; do
        local name="${service%%:*}"
        local port_path="${service#*:}"
        local port="${port_path%:*}"
        local path="${port_path#*:}"
        local url="http://localhost:$port$path"
        
        print_status "Checking $name at $url..."
        
        if curl -s -f "$url" >/dev/null 2>&1; then
            print_success "$name is healthy"
        else
            print_error "$name is not responding"
            all_healthy=false
        fi
    done
    
    if [ "$all_healthy" = true ]; then
        print_success "All KNIRV services are healthy"
    else
        print_error "Some services are not healthy. Check logs for details."
        exit 1
    fi
}

# Configure monitoring for production services
configure_monitoring() {
    print_status "Configuring monitoring for production services..."
    
    # Update Prometheus configuration to include production services
    # This would typically involve updating the prometheus.yml file
    # and reloading Prometheus configuration
    
    # Reload Prometheus configuration
    if curl -s -X POST http://localhost:9090/-/reload >/dev/null 2>&1; then
        print_success "Prometheus configuration reloaded"
    else
        print_warning "Failed to reload Prometheus configuration"
    fi
}

# Start the network monitor GUI
start_monitor_gui() {
    print_status "Starting KNIRV Network Monitor GUI..."
    
    cd "$PROJECT_ROOT"
    
    if [ -f "bin/knirv-network-monitor" ]; then
        print_status "Launching GUI application..."
        ./bin/knirv-network-monitor --network production &
        MONITOR_PID=$!
        echo $MONITOR_PID > "$PROJECT_ROOT/data/monitor.pid"
        print_success "Network Monitor GUI started (PID: $MONITOR_PID)"
    else
        print_error "Network Monitor binary not found. Please run setup first."
        exit 1
    fi
}

# Display service URLs
display_urls() {
    echo ""
    print_success "🎉 KNIRV Production Network with Monitoring is ready!"
    echo ""
    echo "📊 Monitoring Dashboards:"
    echo "========================="
    echo "Grafana:         http://localhost:3000 (admin/admin123)"
    echo "Prometheus:      http://localhost:9090"
    echo "Kibana:          http://localhost:5601"
    echo "AlertManager:    http://localhost:9093"
    echo ""
    echo "🌐 KNIRV Production Services:"
    echo "============================="
    echo "IPFS API:        http://localhost:5001"
    echo "IPFS Gateway:    http://localhost:8080"
    echo "KNIRV Oracle:    http://localhost:8083"
    echo "KNIRV Chain:     http://localhost:8080"
    echo "KNIRV Graph:     http://localhost:8081"
    echo "KNIRV Nexus:     http://localhost:8082"
    echo "KNIRV Router:    http://localhost:5001"
    echo ""
    echo "🖥️  Network Monitor:"
    echo "==================="
    echo "GUI Application: Running (check desktop)"
    echo "API Endpoint:    http://localhost:8090"
    echo ""
    echo "📝 Logs and Data:"
    echo "================="
    echo "Service Logs:    ./logs/"
    echo "Metrics Data:    Docker volumes"
    echo "Configuration:   ./config/"
    echo ""
    print_status "To stop all services: ./scripts/stop-production-monitoring.sh"
}

# Main execution
main() {
    print_status "Starting KNIRV production network monitoring setup..."
    
    check_monitoring_stack
    start_ipfs
    start_knirv_services
    verify_services
    configure_monitoring
    start_monitor_gui
    display_urls
    
    print_success "Production network monitoring startup completed!"
}

# Handle script interruption
cleanup() {
    print_warning "Received interrupt signal. Cleaning up..."
    if [ -f "$PROJECT_ROOT/data/monitor.pid" ]; then
        local monitor_pid=$(cat "$PROJECT_ROOT/data/monitor.pid")
        if kill -0 "$monitor_pid" 2>/dev/null; then
            kill "$monitor_pid"
            print_status "Network Monitor GUI stopped"
        fi
        rm -f "$PROJECT_ROOT/data/monitor.pid"
    fi
    exit 0
}

trap cleanup SIGINT SIGTERM

# Run main function
main "$@"
