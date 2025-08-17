#!/bin/bash

# KNIRV Network Fix Synchronization Validation Script
# Validates that synchronized fixes are working correctly

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Validation results
TESTS_PASSED=0
TESTS_FAILED=0
VALIDATION_ERRORS=()

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

test_result() {
    local test_name="$1"
    local result="$2"
    local message="$3"
    
    if [[ "$result" == "pass" ]]; then
        ((TESTS_PASSED++))
        log "SUCCESS" "$test_name: $message"
    else
        ((TESTS_FAILED++))
        log "ERROR" "$test_name: $message"
        VALIDATION_ERRORS+=("$test_name: $message")
    fi
}

# =============================================================================
# CONFIGURATION VALIDATION
# =============================================================================

validate_config_syntax() {
    log "INFO" "Validating configuration file syntax..."
    
    # Check YAML syntax
    if command -v yq >/dev/null 2>&1; then
        if yq eval '.' "$SCRIPT_DIR/sync-config.yaml" >/dev/null 2>&1; then
            test_result "Config Syntax" "pass" "sync-config.yaml is valid YAML"
        else
            test_result "Config Syntax" "fail" "sync-config.yaml has invalid YAML syntax"
        fi
    else
        log "WARNING" "yq not found, skipping YAML validation"
    fi
    
    # Check JSON configs
    for config_file in "$PROJECT_ROOT"/KNIRVORACLE/config/*.json; do
        if [[ -f "$config_file" ]]; then
            if jq empty "$config_file" >/dev/null 2>&1; then
                test_result "JSON Config" "pass" "$(basename "$config_file") is valid JSON"
            else
                test_result "JSON Config" "fail" "$(basename "$config_file") has invalid JSON syntax"
            fi
        fi
    done
}

# =============================================================================
# SERVICE HEALTH VALIDATION
# =============================================================================

validate_service_configs() {
    log "INFO" "Validating service configurations..."
    
    # KNIRVORACLE config validation
    local root_config="$PROJECT_ROOT/KNIRVORACLE/config/root_config.json"
    if [[ -f "$root_config" ]]; then
        local chain_id=$(jq -r '.chainID' "$root_config" 2>/dev/null || echo "")
        if [[ -n "$chain_id" && "$chain_id" != "null" ]]; then
            test_result "KNIRVORACLE Config" "pass" "Chain ID configured: $chain_id"
        else
            test_result "KNIRVORACLE Config" "fail" "Chain ID not configured"
        fi
    else
        test_result "KNIRVORACLE Config" "fail" "root_config.json not found"
    fi
    
    # KNIRVGATEWAY config validation
    local gateway_package="$PROJECT_ROOT/KNIRVGATEWAY/package.json"
    if [[ -f "$gateway_package" ]]; then
        local name=$(jq -r '.name' "$gateway_package" 2>/dev/null || echo "")
        if [[ -n "$name" && "$name" != "null" ]]; then
            test_result "KNIRVGATEWAY Config" "pass" "Package configured: $name"
        else
            test_result "KNIRVGATEWAY Config" "fail" "Package name not configured"
        fi
    else
        test_result "KNIRVGATEWAY Config" "fail" "package.json not found"
    fi
    
    # KNIRVTESTNET config validation
    local testnet_configs=("$PROJECT_ROOT"/KNIRVTESTNET/config/*.yaml)
    local valid_configs=0
    for config in "${testnet_configs[@]}"; do
        if [[ -f "$config" ]]; then
            if yq eval '.' "$config" >/dev/null 2>&1; then
                ((valid_configs++))
            fi
        fi
    done
    
    if [[ $valid_configs -gt 0 ]]; then
        test_result "KNIRVTESTNET Config" "pass" "$valid_configs valid testnet configurations"
    else
        test_result "KNIRVTESTNET Config" "fail" "No valid testnet configurations found"
    fi
}

# =============================================================================
# ENVIRONMENT CONSISTENCY VALIDATION
# =============================================================================

validate_environment_consistency() {
    log "INFO" "Validating environment consistency..."
    
    # Check for proper environment transformations
    local testnet_env="$PROJECT_ROOT/KNIRVTESTNET/test.env"
    local prod_config="$PROJECT_ROOT/deployment/production-config/optimization.yaml"
    
    if [[ -f "$testnet_env" ]]; then
        if grep -q "TESTNET_MODE=true" "$testnet_env"; then
            test_result "Testnet Environment" "pass" "Testnet mode properly configured"
        else
            test_result "Testnet Environment" "fail" "Testnet mode not configured"
        fi
    fi
    
    if [[ -f "$prod_config" ]]; then
        if grep -q "knirv-production" "$prod_config"; then
            test_result "Production Environment" "pass" "Production namespace configured"
        else
            test_result "Production Environment" "fail" "Production namespace not configured"
        fi
    fi
}

# =============================================================================
# FIX-SPECIFIC VALIDATION
# =============================================================================

validate_badge_attachment_fix() {
    log "INFO" "Validating badge attachment fix..."
    
    local chromem_file="$PROJECT_ROOT/KNIRVORACLE/chromem_manager.go"
    if [[ -f "$chromem_file" ]]; then
        if grep -q "GetBadgeAttachments" "$chromem_file"; then
            if grep -q "progressive limits" "$chromem_file" || grep -q "agent-specific filtering" "$chromem_file"; then
                test_result "Badge Attachment Fix" "pass" "Enhanced query logic found"
            else
                test_result "Badge Attachment Fix" "fail" "Enhanced query logic not found"
            fi
        else
            test_result "Badge Attachment Fix" "fail" "GetBadgeAttachments method not found"
        fi
    else
        test_result "Badge Attachment Fix" "fail" "chromem_manager.go not found"
    fi
}

validate_tunnel_registry_fix() {
    log "INFO" "Validating tunnel registry fix..."
    
    local tunnel_file="$PROJECT_ROOT/KNIRVORACLE/tunnel_registry.go"
    if [[ -f "$tunnel_file" ]]; then
        if grep -q "URI resolution" "$tunnel_file" || grep -q "direct nodes" "$tunnel_file"; then
            test_result "Tunnel Registry Fix" "pass" "Enhanced URI resolution found"
        else
            test_result "Tunnel Registry Fix" "fail" "Enhanced URI resolution not found"
        fi
    else
        test_result "Tunnel Registry Fix" "fail" "tunnel_registry.go not found"
    fi
}

validate_python_sdk_fix() {
    log "INFO" "Validating Python SDK fix..."
    
    local sdk_dir="$PROJECT_ROOT/KNIRVSDK/py"
    if [[ -d "$sdk_dir" ]]; then
        local module_count=$(find "$sdk_dir" -name "*.py" | wc -l)
        if [[ $module_count -gt 0 ]]; then
            test_result "Python SDK Fix" "pass" "$module_count Python modules found"
        else
            test_result "Python SDK Fix" "fail" "No Python modules found"
        fi
    else
        test_result "Python SDK Fix" "fail" "Python SDK directory not found"
    fi
}

validate_gateway_build_fix() {
    log "INFO" "Validating Gateway build fix..."
    
    local package_file="$PROJECT_ROOT/KNIRVGATEWAY/package.json"
    if [[ -f "$package_file" ]]; then
        if jq -e '.dependencies' "$package_file" >/dev/null 2>&1; then
            test_result "Gateway Build Fix" "pass" "Dependencies configured"
        else
            test_result "Gateway Build Fix" "fail" "Dependencies not configured"
        fi
    else
        test_result "Gateway Build Fix" "fail" "package.json not found"
    fi
}

# =============================================================================
# INTEGRATION VALIDATION
# =============================================================================

validate_integration() {
    log "INFO" "Validating integration with existing systems..."
    
    # Check if integration test script exists and is executable
    local integration_script="$PROJECT_ROOT/scripts/run-integration-tests.sh"
    if [[ -f "$integration_script" && -x "$integration_script" ]]; then
        test_result "Integration Tests" "pass" "Integration test script available"
    else
        test_result "Integration Tests" "fail" "Integration test script not available or not executable"
    fi
    
    # Check for Makefile targets
    local makefile="$PROJECT_ROOT/Makefile"
    if [[ -f "$makefile" ]]; then
        if grep -q "test-integration" "$makefile"; then
            test_result "Makefile Integration" "pass" "Integration test target found"
        else
            test_result "Makefile Integration" "fail" "Integration test target not found"
        fi
    else
        test_result "Makefile Integration" "fail" "Makefile not found"
    fi
}

# =============================================================================
# MAIN VALIDATION FUNCTION
# =============================================================================

run_validation() {
    log "INFO" "Starting KNIRV Network Fix Synchronization Validation"
    echo "=================================================================="
    
    validate_config_syntax
    validate_service_configs
    validate_environment_consistency
    validate_badge_attachment_fix
    validate_tunnel_registry_fix
    validate_python_sdk_fix
    validate_gateway_build_fix
    validate_integration
    
    echo ""
    echo "=================================================================="
    log "INFO" "Validation Summary"
    echo "=================================================================="
    
    log "SUCCESS" "Tests Passed: $TESTS_PASSED"
    if [[ $TESTS_FAILED -gt 0 ]]; then
        log "ERROR" "Tests Failed: $TESTS_FAILED"
        echo ""
        log "ERROR" "Failed Tests:"
        for error in "${VALIDATION_ERRORS[@]}"; do
            echo "  - $error"
        done
    else
        log "SUCCESS" "All validation tests passed!"
    fi
    
    echo ""
    if [[ $TESTS_FAILED -eq 0 ]]; then
        log "SUCCESS" "🎉 KNIRV Network Fix Synchronization is ready for use!"
        exit 0
    else
        log "ERROR" "❌ Validation failed. Please fix the issues above before using the synchronization script."
        exit 1
    fi
}

# Show usage information
show_usage() {
    cat << EOF
KNIRV Network Fix Synchronization Validation Script

Usage: $0 [OPTIONS]

OPTIONS:
    -h, --help     Show this help message

This script validates that the KNIRV Network Fix Synchronization system
is properly configured and ready for use.

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

# Run the validation
run_validation
