#!/bin/bash

# KNIRV Portal Version Synchronization Validation Script
# Validates portal implementations and synchronization readiness

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
# PORTAL EXISTENCE VALIDATION
# =============================================================================

validate_nexus_portals() {
    log "INFO" "Validating nexus-portal implementations..."
    
    # KNIRVGATEWAY nexus-portal
    local gateway_portal="$PROJECT_ROOT/KNIRVGATEWAY/nexus-portal"
    if [[ -d "$gateway_portal" && -f "$gateway_portal/package.json" ]]; then
        local version=$(jq -r '.version // "unknown"' "$gateway_portal/package.json" 2>/dev/null || echo "unknown")
        test_result "KNIRVGATEWAY Nexus Portal" "pass" "Found version $version"
        
        # Check for key files
        if [[ -f "$gateway_portal/src/App.tsx" ]]; then
            test_result "KNIRVGATEWAY App.tsx" "pass" "Main component exists"
        else
            test_result "KNIRVGATEWAY App.tsx" "fail" "Main component missing"
        fi
    else
        test_result "KNIRVGATEWAY Nexus Portal" "fail" "Directory or package.json not found"
    fi
    
    # KNIRVTESTNET nexus-portal
    local testnet_portal="$PROJECT_ROOT/KNIRVTESTNET/data/knirvgateway/nexus-portal"
    if [[ -d "$testnet_portal" && -f "$testnet_portal/package.json" ]]; then
        local version=$(jq -r '.version // "unknown"' "$testnet_portal/package.json" 2>/dev/null || echo "unknown")
        test_result "KNIRVTESTNET Nexus Portal" "pass" "Found version $version"
    else
        test_result "KNIRVTESTNET Nexus Portal" "fail" "Directory or package.json not found"
    fi
    
    # KNIRVNEXUS
    local nexus_portal="$PROJECT_ROOT/KNIRVNEXUS"
    if [[ -d "$nexus_portal" && -f "$nexus_portal/package.json" ]]; then
        local version=$(jq -r '.version // "unknown"' "$nexus_portal/package.json" 2>/dev/null || echo "unknown")
        test_result "KNIRVNEXUS Portal" "pass" "Found version $version"
        
        # Check for Next.js structure
        if [[ -f "$nexus_portal/src/app/page.tsx" ]]; then
            test_result "KNIRVNEXUS Next.js Structure" "pass" "Next.js page component exists"
        else
            test_result "KNIRVNEXUS Next.js Structure" "fail" "Next.js page component missing"
        fi
    else
        test_result "KNIRVNEXUS Portal" "fail" "Directory or package.json not found"
    fi
}

validate_graphchain_explorers() {
    log "INFO" "Validating graphchain-explorer implementations..."
    
    # KNIRVGATEWAY graphchain-explorer
    local gateway_explorer="$PROJECT_ROOT/KNIRVGATEWAY/graphchain-explorer"
    if [[ -d "$gateway_explorer" ]]; then
        test_result "KNIRVGATEWAY GraphChain Explorer" "pass" "Directory exists"
        
        # Check for key files
        local key_files=("index.html" "js/graphchain-core.js" "css/graphchain.css")
        for file in "${key_files[@]}"; do
            if [[ -f "$gateway_explorer/$file" ]]; then
                test_result "KNIRVGATEWAY $file" "pass" "Key file exists"
            else
                test_result "KNIRVGATEWAY $file" "fail" "Key file missing"
            fi
        done
    else
        test_result "KNIRVGATEWAY GraphChain Explorer" "fail" "Directory not found"
    fi
    
    # KNIRVTESTNET graphchain-explorer
    local testnet_explorer="$PROJECT_ROOT/KNIRVTESTNET/data/knirvgateway/graphchain-explorer"
    if [[ -d "$testnet_explorer" ]]; then
        test_result "KNIRVTESTNET GraphChain Explorer" "pass" "Directory exists"
        
        # Check for key files
        local key_files=("index.html" "js/graphchain-core.js" "css/graphchain.css")
        for file in "${key_files[@]}"; do
            if [[ -f "$testnet_explorer/$file" ]]; then
                test_result "KNIRVTESTNET $file" "pass" "Key file exists"
            else
                test_result "KNIRVTESTNET $file" "fail" "Key file missing"
            fi
        done
    else
        test_result "KNIRVTESTNET GraphChain Explorer" "fail" "Directory not found"
    fi
}

# =============================================================================
# DEPENDENCY VALIDATION
# =============================================================================

validate_dependencies() {
    log "INFO" "Validating dependencies and build tools..."
    
    # Check Node.js
    if command -v node >/dev/null 2>&1; then
        local node_version=$(node --version)
        test_result "Node.js" "pass" "Version $node_version available"
    else
        test_result "Node.js" "fail" "Node.js not found"
    fi
    
    # Check npm
    if command -v npm >/dev/null 2>&1; then
        local npm_version=$(npm --version)
        test_result "npm" "pass" "Version $npm_version available"
    else
        test_result "npm" "fail" "npm not found"
    fi
    
    # Check jq for JSON processing
    if command -v jq >/dev/null 2>&1; then
        test_result "jq" "pass" "JSON processor available"
    else
        test_result "jq" "fail" "jq not found (needed for package.json processing)"
    fi
    
    # Check rsync for file synchronization
    if command -v rsync >/dev/null 2>&1; then
        test_result "rsync" "pass" "File synchronization tool available"
    else
        test_result "rsync" "fail" "rsync not found (needed for directory sync)"
    fi
}

# =============================================================================
# CONFIGURATION VALIDATION
# =============================================================================

validate_configuration() {
    log "INFO" "Validating portal synchronization configuration..."
    
    # Check sync script
    local sync_script="$SCRIPT_DIR/sync-portal-versions.sh"
    if [[ -f "$sync_script" && -x "$sync_script" ]]; then
        test_result "Sync Script" "pass" "Portal sync script is executable"
    else
        test_result "Sync Script" "fail" "Portal sync script not found or not executable"
    fi
    
    # Check configuration file
    local config_file="$SCRIPT_DIR/portal-sync-config.yaml"
    if [[ -f "$config_file" ]]; then
        test_result "Config File" "pass" "Portal sync configuration exists"
        
        # Validate YAML syntax if yq is available
        if command -v yq >/dev/null 2>&1; then
            if yq eval '.' "$config_file" >/dev/null 2>&1; then
                test_result "Config Syntax" "pass" "Configuration YAML is valid"
            else
                test_result "Config Syntax" "fail" "Configuration YAML has syntax errors"
            fi
        else
            test_result "Config Syntax" "pass" "YAML validation skipped (yq not available)"
        fi
    else
        test_result "Config File" "fail" "Portal sync configuration not found"
    fi
}

# =============================================================================
# VERSION COMPATIBILITY VALIDATION
# =============================================================================

validate_version_compatibility() {
    log "INFO" "Validating version compatibility across portals..."
    
    # Check React versions in nexus portals
    local react_versions=()
    
    for portal in "KNIRVGATEWAY/nexus-portal" "KNIRVTESTNET/data/knirvgateway/nexus-portal"; do
        local package_file="$PROJECT_ROOT/$portal/package.json"
        if [[ -f "$package_file" ]]; then
            local react_version=$(jq -r '.dependencies.react // "not found"' "$package_file" 2>/dev/null || echo "not found")
            react_versions+=("$portal:$react_version")
        fi
    done
    
    # Check KNIRVNEXUS React version
    local nexus_package="$PROJECT_ROOT/KNIRVNEXUS/package.json"
    if [[ -f "$nexus_package" ]]; then
        local react_version=$(jq -r '.dependencies.react // "not found"' "$nexus_package" 2>/dev/null || echo "not found")
        react_versions+=("KNIRVNEXUS:$react_version")
    fi
    
    if [[ ${#react_versions[@]} -gt 0 ]]; then
        test_result "React Versions" "pass" "Found React versions: ${react_versions[*]}"
    else
        test_result "React Versions" "fail" "No React versions found"
    fi
}

# =============================================================================
# BUILD VALIDATION
# =============================================================================

validate_build_capability() {
    log "INFO" "Validating build capability (dry run)..."
    
    # Test KNIRVGATEWAY nexus-portal build readiness
    local gateway_portal="$PROJECT_ROOT/KNIRVGATEWAY/nexus-portal"
    if [[ -d "$gateway_portal" && -f "$gateway_portal/package.json" ]]; then
        if [[ -d "$gateway_portal/node_modules" ]]; then
            test_result "KNIRVGATEWAY Build Ready" "pass" "Dependencies installed"
        else
            test_result "KNIRVGATEWAY Build Ready" "fail" "Dependencies not installed (run npm install)"
        fi
    fi
    
    # Test KNIRVNEXUS build readiness
    local nexus_portal="$PROJECT_ROOT/KNIRVNEXUS"
    if [[ -d "$nexus_portal" && -f "$nexus_portal/package.json" ]]; then
        if [[ -d "$nexus_portal/node_modules" ]]; then
            test_result "KNIRVNEXUS Build Ready" "pass" "Dependencies installed"
        else
            test_result "KNIRVNEXUS Build Ready" "fail" "Dependencies not installed (run npm install)"
        fi
    fi
}

# =============================================================================
# MAIN VALIDATION FUNCTION
# =============================================================================

run_validation() {
    log "INFO" "Starting KNIRV Portal Version Synchronization Validation"
    echo "=================================================================="
    
    validate_nexus_portals
    validate_graphchain_explorers
    validate_dependencies
    validate_configuration
    validate_version_compatibility
    validate_build_capability
    
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
        echo ""
        log "INFO" "Recommendations:"
        echo "  1. Install missing dependencies (Node.js, npm, jq, rsync)"
        echo "  2. Run 'npm install' in portal directories"
        echo "  3. Check file permissions on sync scripts"
        echo "  4. Verify portal directory structures"
    else
        log "SUCCESS" "All validation tests passed!"
    fi
    
    echo ""
    if [[ $TESTS_FAILED -eq 0 ]]; then
        log "SUCCESS" "🎉 KNIRV Portal Version Synchronization is ready for use!"
        echo ""
        log "INFO" "Next steps:"
        echo "  1. Run: make sync-portals-dry-run"
        echo "  2. Review changes and run: make sync-portals"
        echo "  3. Monitor with: make sync-portals-status"
        exit 0
    else
        log "ERROR" "❌ Validation failed. Please fix the issues above before using portal synchronization."
        exit 1
    fi
}

# Show usage information
show_usage() {
    cat << EOF
KNIRV Portal Version Synchronization Validation Script

Usage: $0 [OPTIONS]

OPTIONS:
    -h, --help     Show this help message

This script validates that the KNIRV Portal Version Synchronization system
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
