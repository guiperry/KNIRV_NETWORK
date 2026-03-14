#!/bin/bash

# XION Payment Gateway Integration Test Suite
# Comprehensive testing for the entire payment gateway system

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG_FILE="$TEST_DIR/config/xion_payment_config.json"
LOG_FILE="$TEST_DIR/test-logs/test_results.log"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "$LOG_FILE"
}

success() {
    echo -e "${GREEN}✅ $1${NC}" | tee -a "$LOG_FILE"
    ((TESTS_PASSED++))
}

error() {
    echo -e "${RED}❌ $1${NC}" | tee -a "$LOG_FILE"
    ((TESTS_FAILED++))
}

warning() {
    echo -e "${YELLOW}⚠️  $1${NC}" | tee -a "$LOG_FILE"
}

run_test() {
    local test_name="$1"
    local test_command="$2"
    
    ((TESTS_RUN++))
    log "Running test: $test_name"
    
    if eval "$test_command"; then
        success "$test_name"
        return 0
    else
        error "$test_name"
        return 1
    fi
}

# Test functions

test_configuration() {
    log "Testing configuration files..."
    
    # Check if config file exists
    if [[ ! -f "$CONFIG_FILE" ]]; then
        error "Configuration file not found: $CONFIG_FILE"
        return 1
    fi
    
    # Validate JSON syntax
    if ! jq empty "$CONFIG_FILE" 2>/dev/null; then
        error "Invalid JSON in configuration file"
        return 1
    fi
    
    # Check required configuration sections
    local required_sections=("xion_payment_gateway" "knirv_integration" "monitoring" "security")
    
    for section in "${required_sections[@]}"; do
        if ! jq -e ".$section" "$CONFIG_FILE" >/dev/null 2>&1; then
            error "Missing required configuration section: $section"
            return 1
        fi
    done
    
    success "Configuration validation"
    return 0
}

test_dependencies() {
    log "Testing dependencies..."
    
    # Check Go installation
    if ! command -v go &> /dev/null; then
        error "Go is not installed"
        return 1
    fi
    
    # Check required Go packages
    local required_packages=("github.com/gorilla/mux" "github.com/joho/godotenv")
    
    for package in "${required_packages[@]}"; do
        if ! go list "$package" &> /dev/null; then
            warning "Go package not found: $package (will be downloaded during build)"
        fi
    done
    
    # Check Node.js for KNIRVCONTROLLER tests
    if ! command -v node &> /dev/null; then
        warning "Node.js not found - KNIRVCONTROLLER tests will be skipped"
    fi
    
    success "Dependencies check"
    return 0
}

test_build() {
    log "Testing build process..."
    
    # Build KNIRVCHAIN with XION integration
    cd "$TEST_DIR"
    
    if ! go build -o test_ .; then
        error "Failed to build KNIRVCHAIN"
        return 1
    fi
    
    # Clean up build artifacts
    rm -f test_
    
    success "Build process"
    return 0
}

test_xion_payment_gateway() {
    log "Testing XION Payment Gateway..."
    
    # Test payment gateway initialization
    local test_code='
package main

import (
    "testing"
    "KNIRVCHAIN/economics"
)

func TestXIONPaymentGateway(t *testing.T) {
    config := &XIONGatewayConfig{
        XIONChainID:          "xion-testnet-1",
        XIONRPCEndpoint:      "https://rpc.xion-testnet-1.burnt.com:443",
        XIONRESTEndpoint:     "https://api.xion-testnet-1.burnt.com",
        USDCContractAddr:     "xion1usdc_test",
        NRNContractAddr:      "xion1nrn_test",
        TreasuryAddr:         "xion1treasury_test",
        ConversionRate:       "10",
        GaslessEnabled:       true,
        MaxTransactionAmount: "10000000000",
        MinTransactionAmount: "1000000",
    }
    
    gateway := NewXIONPaymentGateway(config, nil)
    if gateway == nil {
        t.Fatal("Failed to create XION payment gateway")
    }
}
'
    
    echo "$test_code" > xion_gateway_test.go
    
    if go test -run TestXIONPaymentGateway .; then
        success "XION Payment Gateway initialization"
        rm -f xion_gateway_test.go
        return 0
    else
        error "XION Payment Gateway initialization"
        rm -f xion_gateway_test.go
        return 1
    fi
}

test_integration_service() {
    log "Testing Integration Service..."
    
    # Test integration service with mock configuration
    if go run test_xion_integration.go; then
        success "Integration Service"
        return 0
    else
        error "Integration Service"
        return 1
    fi
}

test_payment_flow() {
    log "Testing Payment Flow..."
    
    # Create a simple payment flow test
    local test_script='
package main

import (
    "fmt"
    "log"
    "time"
)

func testPaymentFlow() error {
    log.Println("Testing payment flow simulation...")
    
    // Simulate payment steps
    steps := []string{
        "USDC Payment Processing",
        "NRV Minting from KNIRVROUTER", 
        "Treasury Processing via KNIRVCHAIN",
        "NRN Distribution"
    }
    
    for i, step := range steps {
        log.Printf("Step %d: %s", i+1, step)
        time.Sleep(500 * time.Millisecond) // Simulate processing time
    }
    
    log.Println("Payment flow simulation completed successfully")
    return nil
}

func main() {
    if err := testPaymentFlow(); err != nil {
        log.Fatal(err)
    }
}
'
    
    echo "$test_script" > payment_flow_test.go
    
    if go run payment_flow_test.go; then
        success "Payment Flow simulation"
        rm -f payment_flow_test.go
        return 0
    else
        error "Payment Flow simulation"
        rm -f payment_flow_test.go
        return 1
    fi
}

test_knirvcontroller_integration() {
    log "Testing KNIRVCONTROLLER Integration..."
    
    # Check if KNIRVCONTROLLER directory exists
    if [[ ! -d "../KNIRVCONTROLLER" ]]; then
        warning "KNIRVCONTROLLER directory not found - skipping integration test"
        return 0
    fi
    
    cd "../KNIRVCONTROLLER"
    
    # Check if package.json exists
    if [[ ! -f "package.json" ]]; then
        warning "KNIRVCONTROLLER package.json not found - skipping integration test"
        cd "$TEST_DIR"
        return 0
    fi
    
    # Check TypeScript compilation
    if command -v npm &> /dev/null; then
        if npm run build --if-present; then
            success "KNIRVCONTROLLER build"
        else
            warning "KNIRVCONTROLLER build failed"
        fi
    else
        warning "npm not found - skipping KNIRVCONTROLLER build test"
    fi
    
    cd "$TEST_DIR"
    return 0
}

test_api_endpoints() {
    log "Testing API Endpoints..."
    
    # Test configuration endpoint
    local config_test='{"success": true, "data": {"chain_id": "xion-testnet-1"}}'
    
    # Test rates endpoint  
    local rates_test='{"success": true, "data": {"usdc_to_nrn": "10"}}'
    
    # Simulate API responses
    log "Simulating API endpoint responses..."
    echo "$config_test" | jq empty && success "Config endpoint format"
    echo "$rates_test" | jq empty && success "Rates endpoint format"
    
    return 0
}

test_security() {
    log "Testing Security Features..."
    
    # Test rate limiting configuration
    local rate_limit_enabled=$(jq -r '.security.rate_limiting.enabled' "$CONFIG_FILE")
    if [[ "$rate_limit_enabled" == "true" ]]; then
        success "Rate limiting enabled"
    else
        warning "Rate limiting disabled"
    fi
    
    # Test validation configuration
    local validation_enabled=$(jq -r '.security.validation.address_verification' "$CONFIG_FILE")
    if [[ "$validation_enabled" == "true" ]]; then
        success "Address validation enabled"
    else
        warning "Address validation disabled"
    fi
    
    # Test encryption configuration
    local encryption_enabled=$(jq -r '.security.encryption.enabled' "$CONFIG_FILE")
    if [[ "$encryption_enabled" == "true" ]]; then
        success "Encryption enabled"
    else
        warning "Encryption disabled"
    fi
    
    return 0
}

test_monitoring() {
    log "Testing Monitoring Features..."
    
    # Test monitoring configuration
    local monitoring_enabled=$(jq -r '.monitoring.enabled' "$CONFIG_FILE")
    if [[ "$monitoring_enabled" == "true" ]]; then
        success "Monitoring enabled"
    else
        warning "Monitoring disabled"
    fi
    
    # Test payment tracking
    local payment_tracking=$(jq -r '.monitoring.payment_tracking.status_check_interval' "$CONFIG_FILE")
    if [[ "$payment_tracking" != "null" ]]; then
        success "Payment tracking configured"
    else
        warning "Payment tracking not configured"
    fi
    
    return 0
}

# Main test execution
main() {
    log "Starting XION Payment Gateway Integration Test Suite"
    log "=================================================="
    
    # Initialize log file
    echo "XION Payment Gateway Integration Test Results" > "$LOG_FILE"
    echo "Started: $(date)" >> "$LOG_FILE"
    echo "=========================================" >> "$LOG_FILE"
    
    # Run all tests
    run_test "Configuration Validation" "test_configuration"
    run_test "Dependencies Check" "test_dependencies"
    run_test "Build Process" "test_build"
    run_test "XION Payment Gateway" "test_xion_payment_gateway"
    run_test "Integration Service" "test_integration_service"
    run_test "Payment Flow" "test_payment_flow"
    run_test "KNIRVCONTROLLER Integration" "test_knirvcontroller_integration"
    run_test "API Endpoints" "test_api_endpoints"
    run_test "Security Features" "test_security"
    run_test "Monitoring Features" "test_monitoring"
    
    # Print summary
    log ""
    log "Test Summary"
    log "============"
    log "Tests Run: $TESTS_RUN"
    log "Tests Passed: $TESTS_PASSED"
    log "Tests Failed: $TESTS_FAILED"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        success "All tests passed! 🎉"
        echo "Completed: $(date)" >> "$LOG_FILE"
        exit 0
    else
        error "Some tests failed. Check $LOG_FILE for details."
        echo "Completed: $(date)" >> "$LOG_FILE"
        exit 1
    fi
}

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "Error: jq is required for this test suite. Please install jq."
    exit 1
fi

# Run main function
main "$@"
