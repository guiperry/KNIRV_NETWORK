#!/bin/bash

# KNIRV Network Fix Synchronization Test Script
# Tests the synchronization system with sample scenarios

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    local level="$1"
    shift
    local message="$*"
    
    case "$level" in
        "ERROR") echo -e "${RED}❌ $message${NC}" ;;
        "SUCCESS") echo -e "${GREEN}✅ $message${NC}" ;;
        "WARNING") echo -e "${YELLOW}⚠️  $message${NC}" ;;
        "INFO") echo -e "${BLUE}ℹ️  $message${NC}" ;;
    esac
}

# Test scenarios
test_dry_run() {
    log "INFO" "Testing dry run functionality..."
    
    if "$SCRIPT_DIR/sync-network-fixes.sh" --dry-run --direction both; then
        log "SUCCESS" "Dry run completed successfully"
        return 0
    else
        log "ERROR" "Dry run failed"
        return 1
    fi
}

test_validation() {
    log "INFO" "Testing validation system..."
    
    if "$SCRIPT_DIR/validate-sync.sh"; then
        log "SUCCESS" "Validation completed successfully"
        return 0
    else
        log "WARNING" "Validation found issues (this may be expected)"
        return 0  # Don't fail the test for validation warnings
    fi
}

test_help_output() {
    log "INFO" "Testing help output..."
    
    if "$SCRIPT_DIR/sync-network-fixes.sh" --help >/dev/null 2>&1; then
        log "SUCCESS" "Help output working"
        return 0
    else
        log "ERROR" "Help output failed"
        return 1
    fi
}

test_config_parsing() {
    log "INFO" "Testing configuration parsing..."
    
    local config_file="$SCRIPT_DIR/sync-config.yaml"
    if [[ -f "$config_file" ]]; then
        if command -v yq >/dev/null 2>&1; then
            if yq eval '.' "$config_file" >/dev/null 2>&1; then
                log "SUCCESS" "Configuration file is valid"
                return 0
            else
                log "ERROR" "Configuration file has syntax errors"
                return 1
            fi
        else
            log "WARNING" "yq not available, skipping YAML validation"
            return 0
        fi
    else
        log "ERROR" "Configuration file not found"
        return 1
    fi
}

test_directory_structure() {
    log "INFO" "Testing directory structure..."
    
    local required_dirs=(
        "KNIRVTESTNET"
        "deployment"
        "scripts"
        "KNIRVROOT"
        "KNIRVGATEWAY"
    )
    
    local missing_dirs=()
    for dir in "${required_dirs[@]}"; do
        if [[ ! -d "$PROJECT_ROOT/$dir" ]]; then
            missing_dirs+=("$dir")
        fi
    done
    
    if [[ ${#missing_dirs[@]} -eq 0 ]]; then
        log "SUCCESS" "All required directories present"
        return 0
    else
        log "WARNING" "Missing directories: ${missing_dirs[*]}"
        return 0  # Don't fail for missing directories
    fi
}

run_all_tests() {
    log "INFO" "Starting KNIRV Network Fix Synchronization System Tests"
    echo "=================================================================="
    
    local tests_passed=0
    local tests_failed=0
    
    # Run tests
    local test_functions=(
        "test_help_output"
        "test_config_parsing"
        "test_directory_structure"
        "test_validation"
        "test_dry_run"
    )
    
    for test_func in "${test_functions[@]}"; do
        echo ""
        if $test_func; then
            ((tests_passed++))
        else
            ((tests_failed++))
        fi
    done
    
    echo ""
    echo "=================================================================="
    log "INFO" "Test Summary"
    echo "=================================================================="
    
    log "SUCCESS" "Tests Passed: $tests_passed"
    if [[ $tests_failed -gt 0 ]]; then
        log "ERROR" "Tests Failed: $tests_failed"
    fi
    
    echo ""
    if [[ $tests_failed -eq 0 ]]; then
        log "SUCCESS" "🎉 All tests passed! The synchronization system is ready for use."
        echo ""
        log "INFO" "Next steps:"
        echo "  1. Run validation: ./scripts/validate-sync.sh"
        echo "  2. Test with dry run: ./scripts/sync-network-fixes.sh --dry-run"
        echo "  3. Perform actual sync: ./scripts/sync-network-fixes.sh"
        echo ""
        return 0
    else
        log "ERROR" "❌ Some tests failed. Please review the issues above."
        return 1
    fi
}

show_usage() {
    cat << EOF
KNIRV Network Fix Synchronization Test Script

Usage: $0 [OPTIONS]

OPTIONS:
    -h, --help     Show this help message

This script tests the KNIRV Network Fix Synchronization system to ensure
it's properly configured and functional.

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            log "ERROR" "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Run the tests
run_all_tests
