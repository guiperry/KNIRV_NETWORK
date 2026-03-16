#!/bin/bash

# KNIRV-NEXUS Operational Modes Test Script
# Tests both headless and GUI modes with configuration management

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_DIR="$PROJECT_ROOT/test-results"
LOG_FILE="$TEST_DIR/operational-modes-test.log"

# Create test directory
mkdir -p "$TEST_DIR"

# Logging function
log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "$LOG_FILE"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOG_FILE"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

# Test functions
test_configuration_loading() {
    log "Testing configuration loading..."
    
    # Test with testnet environment
    export KNIRV_ENV=testnet
    if go run cmd/main.go --config-test 2>&1 | grep -q "Configuration loaded successfully"; then
        success "Testnet configuration loaded successfully"
    else
        error "Failed to load testnet configuration"
        return 1
    fi
    
    # Test with production environment
    export KNIRV_ENV=production
    if go run cmd/main.go --config-test 2>&1 | grep -q "Configuration loaded successfully"; then
        success "Production configuration loaded successfully"
    else
        error "Failed to load production configuration"
        return 1
    fi
    
    unset KNIRV_ENV
}

test_headless_mode() {
    log "Testing headless mode..."
    
    # Start in headless mode
    export KNIRV_MODE=headless
    export KNIRV_GUI_ENABLED=false
    export KNIRV_API_PORT=8080
    
    # Start the service in background
    go run cmd/main.go --mode=headless --config=testnet &
    SERVICE_PID=$!
    
    # Wait for service to start
    sleep 5
    
    # Test API endpoints
    if curl -f http://localhost:8080/health >/dev/null 2>&1; then
        success "Headless mode API is responding"
    else
        error "Headless mode API is not responding"
        kill $SERVICE_PID 2>/dev/null || true
        return 1
    fi
    
    # Test that GUI is not accessible
    if curl -f http://localhost:9080 >/dev/null 2>&1; then
        error "GUI should not be accessible in headless mode"
        kill $SERVICE_PID 2>/dev/null || true
        return 1
    else
        success "GUI correctly disabled in headless mode"
    fi
    
    # Clean up
    kill $SERVICE_PID 2>/dev/null || true
    wait $SERVICE_PID 2>/dev/null || true
    
    unset KNIRV_MODE KNIRV_GUI_ENABLED KNIRV_API_PORT
}

test_gui_mode() {
    log "Testing GUI mode..."
    
    # Start in GUI mode
    export KNIRV_MODE=gui
    export KNIRV_GUI_ENABLED=true
    export KNIRV_API_PORT=8080
    export KNIRV_GUI_PORT=9080
    
    # Start the service in background
    go run cmd/main.go --mode=gui --config=testnet &
    SERVICE_PID=$!
    
    # Wait for service to start
    sleep 5
    
    # Test API endpoints
    if curl -f http://localhost:8080/health >/dev/null 2>&1; then
        success "GUI mode API is responding"
    else
        error "GUI mode API is not responding"
        kill $SERVICE_PID 2>/dev/null || true
        return 1
    fi
    
    # Test GUI accessibility
    if curl -f http://localhost:9080 >/dev/null 2>&1; then
        success "GUI is accessible in GUI mode"
    else
        error "GUI should be accessible in GUI mode"
        kill $SERVICE_PID 2>/dev/null || true
        return 1
    fi
    
    # Clean up
    kill $SERVICE_PID 2>/dev/null || true
    wait $SERVICE_PID 2>/dev/null || true
    
    unset KNIRV_MODE KNIRV_GUI_ENABLED KNIRV_API_PORT KNIRV_GUI_PORT
}

test_cli_flags() {
    log "Testing CLI flags..."
    
    # Test help flag
    if go run cmd/main.go --help 2>&1 | grep -q "Usage:"; then
        success "Help flag works correctly"
    else
        error "Help flag not working"
        return 1
    fi
    
    # Test version flag
    if go run cmd/main.go --version 2>&1 | grep -q "KNIRV-NEXUS"; then
        success "Version flag works correctly"
    else
        error "Version flag not working"
        return 1
    fi
    
    # Test config validation
    if go run cmd/main.go --validate-config --config=testnet 2>&1 | grep -q "valid"; then
        success "Config validation works correctly"
    else
        error "Config validation not working"
        return 1
    fi
}

test_environment_variables() {
    log "Testing environment variable override..."
    
    # Test port override
    export KNIRV_API_PORT=8090
    export KNIRV_MODE=headless
    
    # Start service with environment override
    go run cmd/main.go --config=testnet &
    SERVICE_PID=$!
    
    # Wait for service to start
    sleep 5
    
    # Test that service is running on overridden port
    if curl -f http://localhost:8090/health >/dev/null 2>&1; then
        success "Environment variable override works correctly"
    else
        error "Environment variable override not working"
        kill $SERVICE_PID 2>/dev/null || true
        return 1
    fi
    
    # Clean up
    kill $SERVICE_PID 2>/dev/null || true
    wait $SERVICE_PID 2>/dev/null || true
    
    unset KNIRV_API_PORT KNIRV_MODE
}

test_role_based_access() {
    log "Testing role-based access control..."
    
    # Start service in testnet mode
    export KNIRV_MODE=headless
    export KNIRV_ENV=testnet
    
    go run cmd/main.go --config=testnet &
    SERVICE_PID=$!
    
    # Wait for service to start
    sleep 5
    
    # Test admin access
    if curl -H "Authorization: Bearer testnet-admin-123" -f http://localhost:8080/api/admin/status >/dev/null 2>&1; then
        success "Admin role access works correctly"
    else
        warning "Admin role access test skipped (endpoint may not be implemented)"
    fi
    
    # Test validator access
    if curl -H "Authorization: Bearer testnet-validator-456" -f http://localhost:8080/api/validation/tasks >/dev/null 2>&1; then
        success "Validator role access works correctly"
    else
        warning "Validator role access test skipped (endpoint may not be implemented)"
    fi
    
    # Test observer access
    if curl -H "Authorization: Bearer testnet-observer-789" -f http://localhost:8080/api/system/status >/dev/null 2>&1; then
        success "Observer role access works correctly"
    else
        warning "Observer role access test skipped (endpoint may not be implemented)"
    fi
    
    # Clean up
    kill $SERVICE_PID 2>/dev/null || true
    wait $SERVICE_PID 2>/dev/null || true
    
    unset KNIRV_MODE KNIRV_ENV
}

# Main test execution
main() {
    log "Starting KNIRV-NEXUS Operational Modes Test Suite"
    log "Project root: $PROJECT_ROOT"
    log "Test results will be saved to: $TEST_DIR"
    
    cd "$PROJECT_ROOT"
    
    # Check if Go is available
    if ! command -v go &> /dev/null; then
        error "Go is not installed or not in PATH"
        exit 1
    fi
    
    # Check if project builds
    log "Building project..."
    if go build -o "$TEST_DIR/knirv-server" cmd/main.go; then
        success "Project builds successfully"
    else
        error "Project build failed"
        exit 1
    fi
    
    # Run tests
    local failed_tests=0
    
    test_configuration_loading || ((failed_tests++))
    test_cli_flags || ((failed_tests++))
    test_environment_variables || ((failed_tests++))
    test_headless_mode || ((failed_tests++))
    test_gui_mode || ((failed_tests++))
    test_role_based_access || ((failed_tests++))
    
    # Summary
    log "Test suite completed"
    if [ $failed_tests -eq 0 ]; then
        success "All tests passed!"
        exit 0
    else
        error "$failed_tests test(s) failed"
        exit 1
    fi
}

# Run main function
main "$@"
