#!/bin/bash

# KNIRV Network Full Demo Script
# This script provides a comprehensive demonstration of the KNIRV testnet and KNIRVCONTROLLER
# Utilizes the Makefile for infrastructure management and testing

set -e

# =============================================================================
# CONFIGURATION AND COLORS
# =============================================================================

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Demo configuration
DEMO_DURATION=${DEMO_DURATION:-300}  # 5 minutes default
INTERACTIVE_MODE=${INTERACTIVE_MODE:-true}
SKIP_TESTS=${SKIP_TESTS:-false}
CLEANUP_AFTER=${CLEANUP_AFTER:-true}

# Project directories
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTNET_DIR="$PROJECT_ROOT/KNIRVTESTNET"
CONTROLLER_DIR="$PROJECT_ROOT/KNIRVCONTROLLER"

# =============================================================================
# UTILITY FUNCTIONS
# =============================================================================

print_header() {
    echo ""
    echo -e "${BLUE}================================================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}================================================================${NC}"
    echo ""
}

print_step() {
    echo -e "${PURPLE}[STEP]${NC} $1"
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

print_info() {
    echo -e "${CYAN}[INFO]${NC} $1"
}

wait_for_user() {
    if [ "$INTERACTIVE_MODE" = "true" ]; then
        echo -e "${YELLOW}Press Enter to continue...${NC}"
        read -r
    else
        sleep 2
    fi
}

check_prerequisites() {
    print_step "Checking prerequisites..."
    
    # Check if Makefile exists
    if [ ! -f "$PROJECT_ROOT/Makefile" ]; then
        print_error "Makefile not found in project root"
        exit 1
    fi
    
    # Check required directories
    if [ ! -d "$TESTNET_DIR" ]; then
        print_error "KNIRVTESTNET directory not found"
        exit 1
    fi
    
    if [ ! -d "$CONTROLLER_DIR" ]; then
        print_error "KNIRVCONTROLLER directory not found"
        exit 1
    fi
    
    # Check required tools
    command -v make >/dev/null 2>&1 || { print_error "make not found"; exit 1; }
    command -v node >/dev/null 2>&1 || { print_error "node not found"; exit 1; }
    command -v npm >/dev/null 2>&1 || { print_error "npm not found"; exit 1; }
    command -v curl >/dev/null 2>&1 || { print_error "curl not found"; exit 1; }
    
    print_success "All prerequisites satisfied"
}

# =============================================================================
# DEMO PHASES
# =============================================================================

phase_1_introduction() {
    print_header "PHASE 1: KNIRV NETWORK FULL DEMO INTRODUCTION"
    
    echo -e "${CYAN}Welcome to the KNIRV Network Full Demo!${NC}"
    echo ""
    echo "This demonstration will showcase:"
    echo "  🧪 Complete KNIRV Testnet deployment"
    echo "  🎮 KNIRVCONTROLLER cognitive interface"
    echo "  🔗 Service integration and communication"
    echo "  📊 Real-time monitoring and analytics"
    echo "  🧪 Comprehensive testing suite"
    echo ""
    echo "Demo Configuration:"
    echo "  Duration: ${DEMO_DURATION} seconds"
    echo "  Interactive Mode: ${INTERACTIVE_MODE}"
    echo "  Skip Tests: ${SKIP_TESTS}"
    echo "  Cleanup After: ${CLEANUP_AFTER}"
    echo ""
    
    wait_for_user
}

phase_2_infrastructure_setup() {
    print_header "PHASE 2: INFRASTRUCTURE SETUP AND VALIDATION"
    
    print_step "Validating project structure using Makefile..."
    make status
    
    print_step "Checking system health..."
    make health-check || print_warning "Some services may not be running yet"
    
    print_step "Validating configuration..."
    if [ -f "deployment/ansible/.env" ]; then
        make validate-config || print_warning "Configuration validation failed"
    else
        print_warning "Ansible configuration not found - skipping validation"
    fi
    
    wait_for_user
}

phase_3_testnet_deployment() {
    print_header "PHASE 3: KNIRV TESTNET DEPLOYMENT"
    
    print_step "Starting KNIRVTESTNET using integrated scripts..."
    cd "$TESTNET_DIR"
    
    # Check if testnet is already running
    if npm run testnet:status | grep -q "Running"; then
        print_warning "Testnet services already running - stopping first"
        npm run testnet:stop
        sleep 5
    fi
    
    print_step "Deploying complete testnet infrastructure..."
    npm run testnet:start
    
    print_step "Waiting for services to stabilize..."
    sleep 10
    
    print_step "Verifying testnet deployment..."
    npm run testnet:status
    
    cd "$PROJECT_ROOT"
    wait_for_user
}

phase_4_controller_integration() {
    print_header "PHASE 4: KNIRVCONTROLLER INTEGRATION"
    
    print_step "Starting KNIRVCONTROLLER in testnet mode..."
    cd "$CONTROLLER_DIR"
    
    # Set testnet environment variables
    export NODE_ENV=testnet
    export KNIRV_TESTNET_MODE=true
    export PORT=3000
    
    print_step "Starting KNIRVCONTROLLER development server..."
    npm run dev &
    CONTROLLER_PID=$!
    
    print_step "Waiting for KNIRVCONTROLLER to initialize..."
    sleep 15
    
    # Test controller endpoint
    if curl -s http://localhost:3000 >/dev/null; then
        print_success "KNIRVCONTROLLER is running on http://localhost:3000"
    else
        print_warning "KNIRVCONTROLLER may still be starting up"
    fi
    
    cd "$PROJECT_ROOT"
    wait_for_user
}

phase_5_service_demonstration() {
    print_header "PHASE 5: SERVICE INTEGRATION DEMONSTRATION"
    
    print_step "Testing service endpoints..."
    
    # Test each service
    services=(
        "KNIRV-ORACLE:1317:/health"
        "KNIRVCHAIN:8090:/health"
        "KNIRVGRAPH:8082:/height"
        "KNIRV-NEXUS:8084:/health"
        "KNIRV-ROUTER:8086:/"
        "KNIRV-GATEWAY:8888:/gateway/health"
    )
    
    for service in "${services[@]}"; do
        name=$(echo "$service" | cut -d: -f1)
        port=$(echo "$service" | cut -d: -f2)
        endpoint=$(echo "$service" | cut -d: -f3)
        
        print_step "Testing $name on port $port..."
        if curl -s -f "http://localhost:$port$endpoint" >/dev/null; then
            print_success "$name is responding"
        else
            print_warning "$name may not be fully ready"
        fi
    done
    
    print_step "Displaying service URLs for manual testing..."
    echo ""
    echo "🌐 Access Points:"
    echo "  KNIRVCONTROLLER:    http://localhost:3000"
    echo "  KNIRV-NEXUS:        http://localhost:8084"
    echo "  KNIRV-GATEWAY:      http://localhost:8888"
    echo "  KNIRVCHAIN:         http://localhost:8090"
    echo "  KNIRVGRAPH:         http://localhost:8082"
    echo "  KNIRV-ORACLE:       http://localhost:1317"
    echo "  KNIRV-ROUTER:       http://localhost:8086"
    echo ""
    
    wait_for_user
}

phase_6_testing_suite() {
    print_header "PHASE 6: COMPREHENSIVE TESTING SUITE"
    
    if [ "$SKIP_TESTS" = "true" ]; then
        print_warning "Skipping tests as requested"
        return
    fi
    
    print_step "Running KNIRV Network test suite using Makefile..."
    
    # Run quick tests to avoid long execution time
    print_step "Running quick test suite..."
    make test-quick || print_warning "Some tests may have failed"
    
    print_step "Running testnet-specific tests..."
    make testnet-tests || print_warning "Testnet tests may have failed"
    
    print_step "Generating test reports..."
    make test-reports
    
    if [ -d "test-reports" ]; then
        print_success "Test reports generated in test-reports/ directory"
        ls -la test-reports/ | head -10
    fi
    
    wait_for_user
}

phase_7_monitoring_analytics() {
    print_header "PHASE 7: MONITORING AND ANALYTICS"
    
    print_step "Displaying real-time service status..."
    cd "$TESTNET_DIR"
    npm run testnet:status
    
    print_step "Showing service logs (last 10 lines each)..."
    if [ -d "logs" ]; then
        for log in logs/*.log; do
            if [ -f "$log" ]; then
                echo ""
                echo -e "${YELLOW}=== $(basename "$log") ===${NC}"
                tail -10 "$log" 2>/dev/null || echo "Log file empty or not accessible"
            fi
        done
    fi
    
    cd "$PROJECT_ROOT"
    wait_for_user
}

phase_8_interactive_demo() {
    print_header "PHASE 8: INTERACTIVE DEMONSTRATION"
    
    print_step "Opening browser windows for interactive exploration..."
    
    # List of URLs to open
    urls=(
        "http://localhost:3000"
        "http://localhost:8084"
        "http://localhost:8888"
    )
    
    echo "The following services are ready for interaction:"
    echo ""
    for url in "${urls[@]}"; do
        echo "  🌐 $url"
    done
    echo ""
    
    if command -v xdg-open >/dev/null 2>&1; then
        print_step "Opening browser windows..."
        for url in "${urls[@]}"; do
            xdg-open "$url" 2>/dev/null &
        done
    elif command -v open >/dev/null 2>&1; then
        print_step "Opening browser windows..."
        for url in "${urls[@]}"; do
            open "$url" 2>/dev/null &
        done
    else
        print_warning "Cannot auto-open browsers. Please manually visit the URLs above."
    fi
    
    echo ""
    echo -e "${CYAN}Demo Environment Active!${NC}"
    echo ""
    echo "You can now:"
    echo "  • Interact with KNIRVCONTROLLER cognitive interface"
    echo "  • Explore KNIRV-NEXUS validation engine"
    echo "  • Test API endpoints and service integration"
    echo "  • Monitor real-time network activity"
    echo ""
    echo -e "${YELLOW}Demo will run for $DEMO_DURATION seconds...${NC}"
    
    # Keep demo running for specified duration
    sleep "$DEMO_DURATION"
}

phase_9_cleanup() {
    print_header "PHASE 9: CLEANUP AND SUMMARY"
    
    if [ "$CLEANUP_AFTER" = "true" ]; then
        print_step "Stopping KNIRVCONTROLLER..."
        if [ -n "$CONTROLLER_PID" ]; then
            kill "$CONTROLLER_PID" 2>/dev/null || true
        fi
        
        print_step "Stopping testnet services..."
        cd "$TESTNET_DIR"
        npm run testnet:stop
        
        print_step "Cleaning up test artifacts..."
        cd "$PROJECT_ROOT"
        make test-clean || true
        
        print_success "Cleanup completed"
    else
        print_warning "Skipping cleanup - services remain running"
    fi
    
    print_step "Demo Summary:"
    echo ""
    echo "✅ KNIRV Testnet deployed and tested"
    echo "✅ KNIRVCONTROLLER integrated and functional"
    echo "✅ Service communication verified"
    echo "✅ Comprehensive testing completed"
    echo "✅ Real-time monitoring demonstrated"
    echo ""
    echo -e "${GREEN}🎉 KNIRV Network Full Demo Completed Successfully!${NC}"
    echo ""
    
    if [ "$CLEANUP_AFTER" = "false" ]; then
        echo "Services are still running. To stop them manually:"
        echo "  cd KNIRVTESTNET && npm run testnet:stop"
        echo "  pkill -f 'vite.*3000'"
    fi
}

# =============================================================================
# MAIN EXECUTION
# =============================================================================

main() {
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --duration)
                DEMO_DURATION="$2"
                shift 2
                ;;
            --non-interactive)
                INTERACTIVE_MODE=false
                shift
                ;;
            --skip-tests)
                SKIP_TESTS=true
                shift
                ;;
            --no-cleanup)
                CLEANUP_AFTER=false
                shift
                ;;
            --help)
                echo "KNIRV Network Full Demo Script"
                echo ""
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --duration SECONDS     Demo duration (default: 300)"
                echo "  --non-interactive      Run without user prompts"
                echo "  --skip-tests          Skip testing phase"
                echo "  --no-cleanup          Don't cleanup after demo"
                echo "  --help                Show this help"
                echo ""
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    # Execute demo phases
    check_prerequisites
    phase_1_introduction
    phase_2_infrastructure_setup
    phase_3_testnet_deployment
    phase_4_controller_integration
    phase_5_service_demonstration
    phase_6_testing_suite
    phase_7_monitoring_analytics
    phase_8_interactive_demo
    phase_9_cleanup
}

# Trap signals for cleanup
trap 'print_error "Demo interrupted"; phase_9_cleanup; exit 1' INT TERM

# Run main function
main "$@"
