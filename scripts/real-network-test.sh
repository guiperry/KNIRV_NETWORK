#!/bin/bash

# KNIRV Real Network Testing Script
# This script configures and runs tests against real blockchain networks

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CONFIG_DIR="$PROJECT_ROOT/integration-tests/config"
TEST_CONFIG="$CONFIG_DIR/test-config.yaml"

# Network configuration
XION_NETWORK="testnet"  # testnet or mainnet
ETHEREUM_NETWORK="goerli"  # goerli, sepolia, or mainnet
USE_REAL_FUNDS=false
DRY_RUN=false
VERBOSE=false

# Test configuration
TEST_TIMEOUT="1800s"  # 30 minutes
BRIDGE_TEST_AMOUNT="1000000"  # 1 NRN in smallest unit
MAX_GAS_PRICE="20000000000"  # 20 gwei

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

log_header() {
    echo -e "${PURPLE}╔══════════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${PURPLE}║${NC} $1 ${PURPLE}║${NC}"
    echo -e "${PURPLE}╚══════════════════════════════════════════════════════════════════════════════╝${NC}"
}

# Function to show usage
show_usage() {
    echo "KNIRV Real Network Testing Script"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Network Options:"
    echo "  --xion-network NETWORK   XION network: testnet, mainnet (default: testnet)"
    echo "  --eth-network NETWORK    Ethereum network: goerli, sepolia, mainnet (default: goerli)"
    echo "  --use-real-funds         Use real funds for testing (DANGEROUS)"
    echo "  --dry-run                Simulate tests without actual transactions"
    echo ""
    echo "Test Options:"
    echo "  --timeout DURATION       Test timeout (default: 1800s)"
    echo "  --bridge-amount AMOUNT    Bridge test amount in smallest unit (default: 1000000)"
    echo "  --max-gas-price PRICE    Maximum gas price in wei (default: 20000000000)"
    echo ""
    echo "Configuration:"
    echo "  --config FILE            Test configuration file (default: integration-tests/config/test-config.yaml)"
    echo "  --verbose                Enable verbose output"
    echo ""
    echo "Test Types:"
    echo "  --bridge-only            Run bridge tests only"
    echo "  --connectivity-only      Run connectivity tests only"
    echo "  --full-suite             Run full real network test suite"
    echo ""
    echo "Safety Options:"
    echo "  --confirm-real-funds     Confirm usage of real funds (required with --use-real-funds)"
    echo ""
    echo "Examples:"
    echo "  $0 --dry-run                           # Simulate tests"
    echo "  $0 --xion-network testnet --bridge-only # Test bridge on XION testnet"
    echo "  $0 --use-real-funds --confirm-real-funds # Use real funds (DANGEROUS)"
}

# Function to check prerequisites
check_prerequisites() {
    log_step "Checking prerequisites for real network testing..."

    # Check if test configuration exists
    if [ ! -f "$TEST_CONFIG" ]; then
        log_error "Test configuration not found: $TEST_CONFIG"
        exit 1
    fi

    # Check if KNIRV services are running
    if ! "$SCRIPT_DIR/manage-knirv.sh" health &> /dev/null; then
        log_error "KNIRV services are not running. Please start them first."
        log_info "Run: $SCRIPT_DIR/manage-knirv.sh start all"
        exit 1
    fi

    # Check network connectivity
    case $XION_NETWORK in
        "testnet")
            if ! curl -s "https://rpc.xion-testnet-1.burnt.com:443/status" > /dev/null; then
                log_error "Cannot connect to XION testnet"
                exit 1
            fi
            ;;
        "mainnet")
            if ! curl -s "https://rpc.xion.burnt.com:443/status" > /dev/null; then
                log_error "Cannot connect to XION mainnet"
                exit 1
            fi
            ;;
    esac

    log_info "Prerequisites check passed"
}

# Function to configure test environment for real networks
configure_real_network_environment() {
    log_step "Configuring real network test environment..."

    # Create temporary config with real network settings
    local temp_config="/tmp/knirv-real-network-config.yaml"
    
    # Copy base config
    cp "$TEST_CONFIG" "$temp_config"

    # Update configuration for real networks
    cat >> "$temp_config" << EOF

# Real Network Testing Override
real_network_testing:
  enabled: true
  xion:
    network: "$XION_NETWORK"
    rpc_url: "$(get_xion_rpc_url)"
    chain_id: "$(get_xion_chain_id)"
  ethereum:
    network: "$ETHEREUM_NETWORK"
    rpc_url: "$(get_ethereum_rpc_url)"
    chain_id: $(get_ethereum_chain_id)
  
  test_parameters:
    bridge_amount: "$BRIDGE_TEST_AMOUNT"
    max_gas_price: "$MAX_GAS_PRICE"
    use_real_funds: $USE_REAL_FUNDS
    dry_run: $DRY_RUN
    timeout: "$TEST_TIMEOUT"
EOF

    export KNIRV_TEST_CONFIG="$temp_config"
    export KNIRV_REAL_NETWORK_TESTING=true

    log_info "Real network environment configured"
    log_info "XION Network: $XION_NETWORK"
    log_info "Ethereum Network: $ETHEREUM_NETWORK"
    log_info "Use Real Funds: $USE_REAL_FUNDS"
    log_info "Dry Run: $DRY_RUN"
}

# Function to get XION RPC URL
get_xion_rpc_url() {
    case $XION_NETWORK in
        "testnet")
            echo "https://rpc.xion-testnet-1.burnt.com:443"
            ;;
        "mainnet")
            echo "https://rpc.xion.burnt.com:443"
            ;;
    esac
}

# Function to get XION chain ID
get_xion_chain_id() {
    case $XION_NETWORK in
        "testnet")
            echo "xion-testnet-1"
            ;;
        "mainnet")
            echo "xion-mainnet-1"
            ;;
    esac
}

# Function to get Ethereum RPC URL
get_ethereum_rpc_url() {
    case $ETHEREUM_NETWORK in
        "goerli")
            echo "https://goerli.infura.io/v3/YOUR_PROJECT_ID"
            ;;
        "sepolia")
            echo "https://sepolia.infura.io/v3/YOUR_PROJECT_ID"
            ;;
        "mainnet")
            echo "https://mainnet.infura.io/v3/YOUR_PROJECT_ID"
            ;;
    esac
}

# Function to get Ethereum chain ID
get_ethereum_chain_id() {
    case $ETHEREUM_NETWORK in
        "goerli")
            echo "5"
            ;;
        "sepolia")
            echo "11155111"
            ;;
        "mainnet")
            echo "1"
            ;;
    esac
}

# Function to run bridge tests
run_bridge_tests() {
    log_step "Running real network bridge tests..."

    if [ "$USE_REAL_FUNDS" = true ]; then
        log_warn "⚠️  USING REAL FUNDS FOR BRIDGE TESTING ⚠️"
        log_warn "This will perform actual blockchain transactions"
        
        if [ "$CONFIRM_REAL_FUNDS" != true ]; then
            log_error "Real funds usage not confirmed. Use --confirm-real-funds flag."
            exit 1
        fi
        
        read -p "Are you sure you want to proceed with real funds? (yes/no): " confirm
        if [ "$confirm" != "yes" ]; then
            log_info "Bridge testing cancelled"
            exit 0
        fi
    fi

    # Run bridge-specific tests
    cd "$PROJECT_ROOT/integration-tests"
    
    local test_args="-v -timeout $TEST_TIMEOUT -run TestBridge"
    if [ "$VERBOSE" = true ]; then
        test_args="$test_args -v"
    fi

    if go test $test_args; then
        log_info "Bridge tests completed successfully"
    else
        log_error "Bridge tests failed"
        return 1
    fi
}

# Function to run connectivity tests
run_connectivity_tests() {
    log_step "Running real network connectivity tests..."

    cd "$PROJECT_ROOT/integration-tests"
    
    local test_args="-v -timeout $TEST_TIMEOUT -run TestConnectivity"
    if [ "$VERBOSE" = true ]; then
        test_args="$test_args -v"
    fi

    if go test $test_args; then
        log_info "Connectivity tests completed successfully"
    else
        log_error "Connectivity tests failed"
        return 1
    fi
}

# Function to run full real network test suite
run_full_test_suite() {
    log_step "Running full real network test suite..."

    # Use the production integration test
    cd "$PROJECT_ROOT/integration-tests"
    
    local test_args="-v -timeout $TEST_TIMEOUT -run TestProductionDeploymentIntegration"
    if [ "$VERBOSE" = true ]; then
        test_args="$test_args -v"
    fi

    if go test $test_args; then
        log_info "Full test suite completed successfully"
    else
        log_error "Full test suite failed"
        return 1
    fi
}

# Function to cleanup
cleanup() {
    log_step "Cleaning up real network test environment..."
    
    # Remove temporary config
    if [ -n "$KNIRV_TEST_CONFIG" ] && [ -f "$KNIRV_TEST_CONFIG" ]; then
        rm -f "$KNIRV_TEST_CONFIG"
    fi
    
    # Unset environment variables
    unset KNIRV_TEST_CONFIG
    unset KNIRV_REAL_NETWORK_TESTING
    
    log_info "Cleanup completed"
}

# Parse command line arguments
BRIDGE_ONLY=false
CONNECTIVITY_ONLY=false
FULL_SUITE=false
CONFIRM_REAL_FUNDS=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --xion-network)
            XION_NETWORK="$2"
            shift 2
            ;;
        --eth-network)
            ETHEREUM_NETWORK="$2"
            shift 2
            ;;
        --use-real-funds)
            USE_REAL_FUNDS=true
            shift
            ;;
        --confirm-real-funds)
            CONFIRM_REAL_FUNDS=true
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --timeout)
            TEST_TIMEOUT="$2"
            shift 2
            ;;
        --bridge-amount)
            BRIDGE_TEST_AMOUNT="$2"
            shift 2
            ;;
        --max-gas-price)
            MAX_GAS_PRICE="$2"
            shift 2
            ;;
        --config)
            TEST_CONFIG="$2"
            shift 2
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --bridge-only)
            BRIDGE_ONLY=true
            shift
            ;;
        --connectivity-only)
            CONNECTIVITY_ONLY=true
            shift
            ;;
        --full-suite)
            FULL_SUITE=true
            shift
            ;;
        --help)
            show_usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Default to full suite if no specific test type selected
if [ "$BRIDGE_ONLY" = false ] && [ "$CONNECTIVITY_ONLY" = false ] && [ "$FULL_SUITE" = false ]; then
    FULL_SUITE=true
fi

# Set trap for cleanup
trap cleanup EXIT

# Main execution
main() {
    log_header "KNIRV Real Network Testing"

    # Safety check for real funds
    if [ "$USE_REAL_FUNDS" = true ]; then
        log_warn "⚠️  REAL FUNDS MODE ENABLED ⚠️"
        log_warn "This will use real cryptocurrency for testing"
        log_warn "Make sure you understand the risks and costs involved"
    fi

    check_prerequisites
    configure_real_network_environment

    local test_result=0

    # Run selected tests
    if [ "$BRIDGE_ONLY" = true ]; then
        run_bridge_tests || test_result=$?
    elif [ "$CONNECTIVITY_ONLY" = true ]; then
        run_connectivity_tests || test_result=$?
    elif [ "$FULL_SUITE" = true ]; then
        run_full_test_suite || test_result=$?
    fi

    if [ $test_result -eq 0 ]; then
        log_header "Real Network Testing Completed Successfully"
        log_info "All tests passed on real networks"
    else
        log_header "Real Network Testing Failed"
        log_error "Some tests failed on real networks"
        exit $test_result
    fi
}

# Run main function
main "$@"
