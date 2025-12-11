#!/bin/bash

# XION Network Monitor Integration Demo Script
# This script demonstrates how XION payment gateway integrates with the existing KNIRV Network Monitor

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NETWORK_MONITOR_DIR="$SCRIPT_DIR/../network-monitor"
CONFIG_FILE="$SCRIPT_DIR/../config/xion_network_monitor_config.json"

# Helper functions
log() {
    echo -e "${BLUE}[$(date '+%H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

error() {
    echo -e "${RED}❌ $1${NC}"
}

warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

info() {
    echo -e "${PURPLE}ℹ️  $1${NC}"
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        error "Go is not installed. Please install Go 1.19+ to continue."
        exit 1
    fi
    
    # Check if network monitor exists
    if [[ ! -d "$NETWORK_MONITOR_DIR" ]]; then
        warning "KNIRV Network Monitor not found at $NETWORK_MONITOR_DIR"
        info "The demo will show how XION integrates with the network monitor when available"
    else
        success "KNIRV Network Monitor found"
    fi
    
    # Check if configuration exists
    if [[ ! -f "$CONFIG_FILE" ]]; then
        warning "Configuration file not found: $CONFIG_FILE"
        info "Using default configuration for demo"
    else
        success "Configuration file found"
    fi
    
    success "Prerequisites check completed"
}

# Display integration overview
show_integration_overview() {
    echo ""
    echo -e "${PURPLE}╔══════════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${PURPLE}║                    XION NETWORK MONITOR INTEGRATION                          ║${NC}"
    echo -e "${PURPLE}╚══════════════════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${BLUE}🎯 INTEGRATION FEATURES:${NC}"
    echo ""
    echo -e "${GREEN}📊 Prometheus Metrics Integration${NC}"
    echo "   • xion_payments_total - Total payments processed"
    echo "   • xion_payment_flows_active - Active payment flows"
    echo "   • xion_nrv_minting_total - NRV tokens minted"
    echo "   • xion_treasury_mints_total - Treasury operations"
    echo "   • xion_gateway_uptime_seconds - Service uptime"
    echo ""
    echo -e "${GREEN}🏥 Health Monitoring${NC}"
    echo "   • Payment Gateway health checks"
    echo "   • Integration Service monitoring"
    echo "   • NRV Minting service status"
    echo "   • Treasury service health"
    echo ""
    echo -e "${GREEN}📈 Dashboard Integration${NC}"
    echo "   • Custom Go/Fyne GUI integration"
    echo "   • Grafana dashboard metrics"
    echo "   • ELK Stack log aggregation"
    echo "   • Real-time status reporting"
    echo ""
    echo -e "${GREEN}🚨 Alerting Integration${NC}"
    echo "   • Payment failure rate alerts"
    echo "   • High volume notifications"
    echo "   • Service down detection"
    echo "   • Treasury balance monitoring"
    echo ""
}

# Show network monitor status
check_network_monitor_status() {
    log "Checking KNIRV Network Monitor status..."
    
    if [[ -d "$NETWORK_MONITOR_DIR" ]]; then
        echo ""
        echo -e "${BLUE}📋 Network Monitor Components:${NC}"
        
        # Check for key components
        local components=(
            "cmd/monitor:Monitor Binary"
            "config/prometheus.yml:Prometheus Config"
            "config/grafana:Grafana Dashboards"
            "internal/monitoring:Monitoring Core"
            "scripts/start-testnet-monitoring.sh:Testnet Script"
        )
        
        for component in "${components[@]}"; do
            local path="${component%%:*}"
            local name="${component##*:}"
            
            if [[ -e "$NETWORK_MONITOR_DIR/$path" ]]; then
                success "$name"
            else
                warning "$name (not found)"
            fi
        done
        
        echo ""
        info "To start the network monitor:"
        echo "   cd $NETWORK_MONITOR_DIR"
        echo "   ./scripts/start-testnet-monitoring.sh"
        
    else
        warning "Network Monitor directory not found"
        info "XION integration is ready to connect when network monitor is available"
    fi
}

# Demonstrate XION integration features
demonstrate_integration() {
    log "Demonstrating XION Network Monitor Integration..."
    
    echo ""
    echo -e "${BLUE}🔧 XION Integration Components:${NC}"
    echo ""
    
    # Check XION integration files
    local xion_files=(
        "xion_payment_gateway.go:XION Payment Gateway"
        "xion_integration_service.go:Integration Service"
        "xion_network_monitor_integration.go:Network Monitor Integration"
        "config/xion_network_monitor_config.json:Integration Configuration"
    )
    
    for file_info in "${xion_files[@]}"; do
        local file="${file_info%%:*}"
        local name="${file_info##*:}"
        
        if [[ -f "$SCRIPT_DIR/$file" ]]; then
            success "$name"
            
            # Show key features for each file
            case "$file" in
                "xion_payment_gateway.go")
                    info "   • USDC to NRN conversion"
                    info "   • Meta Accounts support"
                    info "   • Gasless transactions"
                    ;;
                "xion_integration_service.go")
                    info "   • End-to-end payment flows"
                    info "   • NRV minting coordination"
                    info "   • Treasury integration"
                    ;;
                "xion_network_monitor_integration.go")
                    info "   • Prometheus metrics collection"
                    info "   • Health monitoring"
                    info "   • Status reporting"
                    ;;
                "config/xion_network_monitor_config.json")
                    info "   • Network monitor settings"
                    info "   • Metrics configuration"
                    info "   • Alert rules"
                    ;;
            esac
        else
            warning "$name (not found)"
        fi
    done
}

# Show integration endpoints
show_integration_endpoints() {
    echo ""
    echo -e "${BLUE}🌐 Integration Endpoints:${NC}"
    echo ""
    echo -e "${GREEN}XION Payment Gateway:${NC}"
    echo "   • http://localhost:8080/api/payment/usdc-to-nrn"
    echo "   • http://localhost:8080/api/payment/status/{id}"
    echo "   • http://localhost:8080/api/payment/config"
    echo ""
    echo -e "${GREEN}Network Monitor Integration:${NC}"
    echo "   • http://localhost:8080/metrics (Prometheus)"
    echo "   • http://localhost:8080/health (Health Check)"
    echo "   • http://localhost:9090/api/services/xion/status"
    echo ""
    echo -e "${GREEN}Existing Network Monitor:${NC}"
    echo "   • http://localhost:9090 (Network Monitor GUI)"
    echo "   • http://localhost:9091 (Prometheus)"
    echo "   • http://localhost:3001 (Grafana)"
    echo "   • http://localhost:5601 (Kibana)"
}

# Show sample metrics
show_sample_metrics() {
    echo ""
    echo -e "${BLUE}📊 Sample XION Metrics:${NC}"
    echo ""
    
    cat << 'EOF'
# HELP xion_payments_total Total number of XION payments processed
# TYPE xion_payments_total counter
xion_payments_total 1247

# HELP xion_payment_flows_active Number of active payment flows
# TYPE xion_payment_flows_active gauge
xion_payment_flows_active 3

# HELP xion_nrv_minting_total Total number of NRV tokens minted
# TYPE xion_nrv_minting_total counter
xion_nrv_minting_total 892

# HELP xion_treasury_mints_total Total number of treasury mints processed
# TYPE xion_treasury_mints_total counter
xion_treasury_mints_total 1156

# HELP xion_gateway_uptime_seconds XION gateway uptime in seconds
# TYPE xion_gateway_uptime_seconds gauge
xion_gateway_uptime_seconds 86400
EOF
    
    echo ""
    info "These metrics are automatically collected and sent to the existing Prometheus instance"
}

# Show next steps
show_next_steps() {
    echo ""
    echo -e "${PURPLE}╔══════════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${PURPLE}║                              NEXT STEPS                                     ║${NC}"
    echo -e "${PURPLE}╚══════════════════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${BLUE}🚀 To run the full integration:${NC}"
    echo ""
    echo "1. Start the KNIRV Network Monitor:"
    echo "   cd ../network-monitor"
    echo "   ./scripts/start-testnet-monitoring.sh"
    echo ""
    echo "2. Build and run KNIRVCHAIN with XION integration:"
    echo "   go build -o knirvchain ."
    echo "   ./knirvchain"
    echo ""
    echo "3. Test the integration:"
    echo "   go run -c 'XIONNetworkMonitorDemo()' integrate_xion_with_network_monitor.go"
    echo ""
    echo "4. View in Network Monitor:"
    echo "   • Open http://localhost:9090 for the network monitor GUI"
    echo "   • Check http://localhost:3001 for Grafana dashboards"
    echo "   • Monitor http://localhost:8080/metrics for XION metrics"
    echo ""
    echo -e "${GREEN}✨ The XION payment gateway will appear as a monitored service${NC}"
    echo -e "${GREEN}   with real-time metrics, health checks, and alerting!${NC}"
}

# Main execution
main() {
    echo ""
    echo -e "${PURPLE}🚀 XION Network Monitor Integration Demo${NC}"
    echo -e "${PURPLE}=======================================${NC}"
    echo ""
    
    check_prerequisites
    show_integration_overview
    check_network_monitor_status
    demonstrate_integration
    show_integration_endpoints
    show_sample_metrics
    show_next_steps
    
    echo ""
    success "Demo completed! XION is ready to integrate with KNIRV Network Monitor."
    echo ""
}

# Run the demo
main "$@"
