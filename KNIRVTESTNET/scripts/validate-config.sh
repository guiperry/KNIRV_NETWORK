#!/bin/bash

# KNIRV Testnet Configuration Validation Script
# Validates that all services are properly configured for testnet mode

# Get script directory and change to KNIRVTESTNET root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTNET_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$TESTNET_ROOT"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# Configuration
VERBOSE=false
FIX_ISSUES=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -f|--fix)
            FIX_ISSUES=true
            shift
            ;;
        -h|--help)
            echo "KNIRV Testnet Configuration Validation"
            echo ""
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  -v, --verbose         Show detailed validation information"
            echo "  -f, --fix            Attempt to fix configuration issues"
            echo "  -h, --help           Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                   Basic configuration validation"
            echo "  $0 --verbose         Detailed validation output"
            echo "  $0 --fix             Validate and fix issues"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[⚠]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1"
}

print_header() {
    echo -e "${PURPLE}$1${NC}"
}

print_verbose() {
    if [ "$VERBOSE" = true ]; then
        echo -e "${BLUE}[VERBOSE]${NC} $1"
    fi
}

# Validation counters
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0
WARNINGS=0

# Function to increment check counters
check_passed() {
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    PASSED_CHECKS=$((PASSED_CHECKS + 1))
    print_success "$1"
}

check_failed() {
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
    print_error "$1"
}

check_warning() {
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    WARNINGS=$((WARNINGS + 1))
    print_warning "$1"
}

# Function to check if a file exists
check_file_exists() {
    local file=$1
    local description=$2
    
    if [ -f "$file" ]; then
        check_passed "$description exists: $file"
        return 0
    else
        check_failed "$description missing: $file"
        return 1
    fi
}

# Function to check if a directory exists
check_dir_exists() {
    local dir=$1
    local description=$2
    
    if [ -d "$dir" ]; then
        check_passed "$description exists: $dir"
        return 0
    else
        check_failed "$description missing: $dir"
        return 1
    fi
}

# Function to check environment variable in file
check_env_var() {
    local file=$1
    local var=$2
    local expected_value=$3
    local description=$4
    
    if [ -f "$file" ]; then
        local actual_value=$(grep "^$var=" "$file" 2>/dev/null | cut -d'=' -f2)
        if [ "$actual_value" = "$expected_value" ]; then
            check_passed "$description: $var=$actual_value"
            return 0
        else
            check_failed "$description: $var expected '$expected_value', got '$actual_value'"
            return 1
        fi
    else
        check_failed "$description: Configuration file not found: $file"
        return 1
    fi
}

# Function to check JSON configuration
check_json_config() {
    local file=$1
    local key=$2
    local expected_value=$3
    local description=$4
    
    if [ -f "$file" ]; then
        if command -v jq >/dev/null 2>&1; then
            local actual_value=$(jq -r ".$key" "$file" 2>/dev/null)
            if [ "$actual_value" = "$expected_value" ]; then
                check_passed "$description: $key=$actual_value"
                return 0
            else
                check_failed "$description: $key expected '$expected_value', got '$actual_value'"
                return 1
            fi
        else
            check_warning "$description: jq not available, skipping JSON validation"
            return 1
        fi
    else
        check_failed "$description: Configuration file not found: $file"
        return 1
    fi
}

# Function to validate port availability
check_port_available() {
    local port=$1
    local service=$2
    
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        check_warning "Port $port is in use (needed for $service)"
        return 1
    else
        check_passed "Port $port is available for $service"
        return 0
    fi
}

# Function to check binary exists
check_binary_exists() {
    local binary=$1
    local description=$2
    
    if [ -f "$binary" ]; then
        check_passed "$description binary exists: $binary"
        return 0
    else
        check_failed "$description binary missing: $binary"
        return 1
    fi
}

# Main validation function
perform_validation() {
    print_header "🔍 KNIRV TESTNET CONFIGURATION VALIDATION"
    print_header "=========================================="
    echo ""
    
    # 1. Check directory structure
    print_status "Checking directory structure..."
    check_dir_exists "bin" "Binary directory"
    check_dir_exists "config" "Configuration directory"
    check_dir_exists "data" "Data directory"
    check_dir_exists "logs" "Logs directory"
    check_dir_exists "scripts" "Scripts directory"
    echo ""
    
    # 2. Check required binaries
    print_status "Checking required binaries..."
    check_binary_exists "bin/knirvroot" "KNIRV-ROOT"
    check_binary_exists "bin/knirvchain" "KNIRVCHAIN"
    check_binary_exists "bin/knirvgraph" "KNIRVGRAPH"
    check_binary_exists "bin/knirvnexus" "KNIRV-NEXUS"
    check_binary_exists "bin/knirvrouter" "KNIRV-ROUTER"
    echo ""
    
    # 3. Check KNIRV-ROOT configuration
    print_status "Validating KNIRV-ROOT configuration..."
    check_file_exists "config/knirvroot-testnet-config.json" "KNIRV-ROOT testnet config"
    if [ -f "config/knirvroot-testnet-config.json" ]; then
        check_json_config "config/knirvroot-testnet-config.json" "testnet.enabled" "true" "KNIRV-ROOT testnet mode"
        check_json_config "config/knirvroot-testnet-config.json" "testnet.chain_id" "knirv-testnet-1" "KNIRV-ROOT chain ID"
        check_json_config "config/knirvroot-testnet-config.json" "port" "1317" "KNIRV-ROOT port"
    fi
    echo ""
    
    # 4. Check KNIRVCHAIN configuration
    print_status "Validating KNIRVCHAIN configuration..."
    check_file_exists "config/knirvchain-testnet-config.toml" "KNIRVCHAIN testnet config"
    if [ -f "config/knirvchain-testnet-config.toml" ]; then
        # Check TOML format for testnet.enabled = true
        local toml_enabled=$(grep -A 10 "^\[testnet\]" "config/knirvchain-testnet-config.toml" | grep "enabled" | cut -d'=' -f2 | tr -d ' ')
        if [ "$toml_enabled" = "true" ]; then
            check_passed "KNIRVCHAIN testnet mode: enabled=true"
        else
            check_failed "KNIRVCHAIN testnet mode: enabled expected 'true', got '$toml_enabled'"
        fi
    fi
    echo ""
    
    # 5. Check KNIRVGRAPH configuration
    print_status "Validating KNIRVGRAPH configuration..."
    check_file_exists "config/knirvgraph-testnet-config.yaml" "KNIRVGRAPH testnet config"
    echo ""
    
    # 6. Check KNIRV-NEXUS configuration
    print_status "Validating KNIRV-NEXUS configuration..."
    check_file_exists "config/knirvnexus-testnet-config.yaml" "KNIRV-NEXUS testnet config"
    echo ""
    
    # 7. Check KNIRV-ROUTER configuration
    print_status "Validating KNIRV-ROUTER configuration..."
    check_file_exists "config/knirvrouter-testnet.env" "KNIRV-ROUTER testnet config"
    if [ -f "config/knirvrouter-testnet.env" ]; then
        check_env_var "config/knirvrouter-testnet.env" "TESTNET_MODE" "true" "KNIRV-ROUTER testnet mode"
        check_env_var "config/knirvrouter-testnet.env" "PORT" "5001" "KNIRV-ROUTER port"
    fi
    echo ""
    
    # 8. Check KNIRV-GATEWAY configuration
    print_status "Validating KNIRV-GATEWAY configuration..."
    check_dir_exists "data/knirvgateway" "KNIRV-GATEWAY data directory"
    check_file_exists "data/knirvgateway/.env" "KNIRV-GATEWAY environment config"
    if [ -f "data/knirvgateway/.env" ]; then
        check_env_var "data/knirvgateway/.env" "TESTNET_MODE" "true" "KNIRV-GATEWAY testnet mode"
        check_env_var "data/knirvgateway/.env" "GATEWAY_PORT" "8888" "KNIRV-GATEWAY port"
    fi
    echo ""
    
    # 9. Check port availability
    print_status "Checking port availability..."
    check_port_available 1317 "KNIRV-ROOT"
    check_port_available 8080 "KNIRVCHAIN"
    check_port_available 8081 "KNIRVGRAPH"
    check_port_available 8082 "KNIRV-NEXUS"
    check_port_available 5001 "KNIRV-ROUTER"
    check_port_available 8888 "KNIRV-GATEWAY"
    echo ""
    
    # 10. Check system dependencies
    print_status "Checking system dependencies..."
    if command -v go >/dev/null 2>&1; then
        check_passed "Go is installed: $(go version | cut -d' ' -f3)"
    else
        check_failed "Go is not installed"
    fi
    
    if command -v cargo >/dev/null 2>&1; then
        check_passed "Rust/Cargo is installed: $(cargo --version | cut -d' ' -f2)"
    else
        check_failed "Rust/Cargo is not installed"
    fi
    
    if command -v node >/dev/null 2>&1; then
        check_passed "Node.js is installed: $(node --version)"
    else
        check_failed "Node.js is not installed"
    fi
    
    if command -v python3 >/dev/null 2>&1; then
        check_passed "Python3 is installed: $(python3 --version | cut -d' ' -f2)"
    else
        check_failed "Python3 is not installed"
    fi
    echo ""
    
    # Summary
    print_header "📊 VALIDATION SUMMARY"
    print_header "===================="
    echo "Total checks: $TOTAL_CHECKS"
    echo "Passed: $PASSED_CHECKS"
    echo "Failed: $FAILED_CHECKS"
    echo "Warnings: $WARNINGS"
    echo ""
    
    if [ $FAILED_CHECKS -eq 0 ]; then
        if [ $WARNINGS -eq 0 ]; then
            print_success "All configuration checks passed! ✨"
            print_status "The testnet is ready to start."
        else
            print_warning "Configuration is mostly valid with $WARNINGS warnings."
            print_status "The testnet should start but may have issues."
        fi
    else
        print_error "Configuration validation failed with $FAILED_CHECKS errors."
        print_status "Please fix the issues before starting the testnet."
        
        if [ "$FIX_ISSUES" = true ]; then
            echo ""
            print_status "Attempting to fix issues..."
            # Add fix logic here if needed
            print_warning "Automatic fixes not yet implemented."
        fi
    fi
    
    echo ""
    print_status "To start the testnet: ./start-testnet.sh"
    print_status "To check service health: ./health-check.sh"
    
    # Return appropriate exit code
    if [ $FAILED_CHECKS -eq 0 ]; then
        exit 0
    else
        exit 1
    fi
}

# Main execution
perform_validation
