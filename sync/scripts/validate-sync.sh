#!/bin/bash

# KNIRV Network Synchronization Validation Script
# Validates the current state of synchronization between production and testnet

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SYNC_ROOT="$(dirname "$SCRIPT_DIR")"
TESTNET_ROOT="$(dirname "$SYNC_ROOT")"
PRODUCTION_ROOT="$(dirname "$TESTNET_ROOT")"
CONFIG_FILE="$SYNC_ROOT/sync-config.json"
OUTPUT_DIR="$SYNC_ROOT/reports"
LOG_FILE="$OUTPUT_DIR/validation.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOG_FILE"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
}

# Print usage information
print_usage() {
    cat << EOF
KNIRV Network Synchronization Validation Tool

Usage: $0 [OPTIONS]

OPTIONS:
    --component NAME    Validate specific component only
    --pattern NAME      Validate specific pattern only
    --scripts-only      Validate only script patterns
    --tests-only        Validate only test patterns
    --detailed          Show detailed validation results
    --fix               Attempt to fix validation issues
    --config FILE       Configuration file path (default: sync-config.json)
    --output DIR        Output directory for reports (default: reports)
    --help              Show this help message

EXAMPLES:
    $0                              # Full validation
    $0 --component KNIRVORACLE      # Validate specific component
    $0 --scripts-only               # Validate only scripts
    $0 --detailed                   # Show detailed results
    $0 --fix                        # Fix validation issues

EOF
}

# Validate a specific component
validate_component() {
    local component_name="$1"
    local component_path="$PRODUCTION_ROOT/$component_name"
    local testnet_path="$TESTNET_ROOT/sync/patterns/$(echo "$component_name" | tr '[:upper:]' '[:lower:]')"
    
    log_info "Validating component: $component_name"
    
    if [[ ! -d "$component_path" ]]; then
        log_error "Production component not found: $component_path"
        return 1
    fi
    
    local issues=0
    
    # Check for script files
    if [[ -d "$component_path/scripts" ]]; then
        local script_count
        script_count=$(find "$component_path/scripts" -name "*.sh" | wc -l)
        log_info "Found $script_count script files in production"
        
        if [[ -d "$testnet_path/scripts" ]]; then
            local testnet_script_count
            testnet_script_count=$(find "$testnet_path/scripts" -name "*.sh" | wc -l)
            log_info "Found $testnet_script_count script files in testnet"
            
            if [[ $script_count -gt $testnet_script_count ]]; then
                log_warning "Missing $(($script_count - $testnet_script_count)) script files in testnet"
                issues=$((issues + 1))
            fi
        else
            log_warning "No testnet scripts directory found"
            issues=$((issues + 1))
        fi
    fi
    
    # Check for test files
    local test_patterns=("*_test.go" "*.test.js" "*test*.rs")
    for pattern in "${test_patterns[@]}"; do
        local test_count
        test_count=$(find "$component_path" -name "$pattern" | wc -l)
        
        if [[ $test_count -gt 0 ]]; then
            log_info "Found $test_count test files matching $pattern"
            
            if [[ -d "$testnet_path/tests" ]]; then
                local testnet_test_count
                testnet_test_count=$(find "$testnet_path/tests" -name "$pattern" | wc -l)
                
                if [[ $test_count -gt $testnet_test_count ]]; then
                    log_warning "Missing $(($test_count - $testnet_test_count)) test files matching $pattern"
                    issues=$((issues + 1))
                fi
            else
                log_warning "No testnet tests directory found"
                issues=$((issues + 1))
            fi
        fi
    done
    
    if [[ $issues -eq 0 ]]; then
        log_success "Component $component_name validation passed"
        return 0
    else
        log_error "Component $component_name has $issues validation issues"
        return 1
    fi
}

# Validate script patterns
validate_scripts() {
    log_info "Validating script patterns..."
    
    local components=("KNIRVORACLE" "KNIRVCHAIN" "KNIRVGRAPH" "KNIRVNEXUS" "KNIRVROUTER" "KNIRVGATEWAY")
    local total_issues=0
    
    for component in "${components[@]}"; do
        if ! validate_component "$component"; then
            total_issues=$((total_issues + 1))
        fi
    done
    
    if [[ $total_issues -eq 0 ]]; then
        log_success "All script patterns validated successfully"
        return 0
    else
        log_error "Script validation found issues in $total_issues components"
        return 1
    fi
}

# Validate test patterns
validate_tests() {
    log_info "Validating test patterns..."
    
    local components=("KNIRVORACLE" "KNIRVCHAIN" "KNIRVGRAPH" "KNIRVNEXUS" "KNIRVROUTER" "KNIRVGATEWAY")
    local total_issues=0
    
    for component in "${components[@]}"; do
        local component_path="$PRODUCTION_ROOT/$component"
        local testnet_path="$TESTNET_ROOT/sync/patterns/$(echo "$component" | tr '[:upper:]' '[:lower:]')"
        
        if [[ ! -d "$component_path" ]]; then
            continue
        fi
        
        log_info "Checking test patterns for $component"
        
        # Check for various test file types
        local test_files
        test_files=$(find "$component_path" -name "*test*" -type f | wc -l)
        
        if [[ $test_files -gt 0 ]]; then
            log_info "Found $test_files test files in $component"
            
            if [[ ! -d "$testnet_path/tests" ]]; then
                log_warning "No testnet test directory for $component"
                total_issues=$((total_issues + 1))
            fi
        fi
    done
    
    if [[ $total_issues -eq 0 ]]; then
        log_success "All test patterns validated successfully"
        return 0
    else
        log_error "Test validation found issues in $total_issues components"
        return 1
    fi
}

# Fix validation issues
fix_issues() {
    log_info "Attempting to fix validation issues..."
    
    # Run synchronization to fix issues
    if [[ -f "$SYNC_ROOT/scripts/auto-sync.sh" ]]; then
        log_info "Running auto-sync to fix issues..."
        if "$SYNC_ROOT/scripts/auto-sync.sh" --dry-run; then
            log_info "Dry run successful, running actual sync..."
            "$SYNC_ROOT/scripts/auto-sync.sh"
            log_success "Synchronization completed"
        else
            log_error "Dry run failed, not proceeding with sync"
            return 1
        fi
    else
        log_error "Auto-sync script not found"
        return 1
    fi
}

# Generate detailed validation report
generate_detailed_report() {
    log_info "Generating detailed validation report..."
    
    local report_file="$OUTPUT_DIR/validation-report-$(date +%Y%m%d-%H%M%S).txt"
    
    {
        echo "KNIRV Network Synchronization Validation Report"
        echo "Generated: $(date)"
        echo "=============================================="
        echo ""
        
        echo "PRODUCTION COMPONENTS:"
        for component in KNIRVORACLE KNIRVCHAIN KNIRVGRAPH KNIRVNEXUS KNIRVROUTER KNIRVGATEWAY; do
            if [[ -d "$PRODUCTION_ROOT/$component" ]]; then
                echo "✓ $component"
                
                # Count scripts
                if [[ -d "$PRODUCTION_ROOT/$component/scripts" ]]; then
                    local script_count
                    script_count=$(find "$PRODUCTION_ROOT/$component/scripts" -name "*.sh" | wc -l)
                    echo "  Scripts: $script_count"
                fi
                
                # Count tests
                local test_count
                test_count=$(find "$PRODUCTION_ROOT/$component" -name "*test*" -type f | wc -l)
                echo "  Tests: $test_count"
            else
                echo "✗ $component (not found)"
            fi
        done
        
        echo ""
        echo "TESTNET SYNC PATTERNS:"
        for component in knirvoracle knirvchain knirvgraph knirvnexus knirvrouter knirvgateway; do
            local testnet_path="$TESTNET_ROOT/sync/patterns/$component"
            if [[ -d "$testnet_path" ]]; then
                echo "✓ $component"
                
                if [[ -d "$testnet_path/scripts" ]]; then
                    local script_count
                    script_count=$(find "$testnet_path/scripts" -name "*.sh" | wc -l)
                    echo "  Scripts: $script_count"
                fi
                
                if [[ -d "$testnet_path/tests" ]]; then
                    local test_count
                    test_count=$(find "$testnet_path/tests" -name "*test*" -type f | wc -l)
                    echo "  Tests: $test_count"
                fi
            else
                echo "✗ $component (not synchronized)"
            fi
        done
        
    } > "$report_file"
    
    log_success "Detailed report generated: $report_file"
    
    if [[ "$DETAILED" == "true" ]]; then
        cat "$report_file"
    fi
}

# Main execution
main() {
    # Default values
    COMPONENT=""
    PATTERN=""
    SCRIPTS_ONLY="false"
    TESTS_ONLY="false"
    DETAILED="false"
    FIX="false"
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --component)
                COMPONENT="$2"
                shift 2
                ;;
            --pattern)
                PATTERN="$2"
                shift 2
                ;;
            --scripts-only)
                SCRIPTS_ONLY="true"
                shift
                ;;
            --tests-only)
                TESTS_ONLY="true"
                shift
                ;;
            --detailed)
                DETAILED="true"
                shift
                ;;
            --fix)
                FIX="true"
                shift
                ;;
            --config)
                CONFIG_FILE="$2"
                shift 2
                ;;
            --output)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            --help)
                print_usage
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                print_usage
                exit 1
                ;;
        esac
    done
    
    # Create output directory
    mkdir -p "$OUTPUT_DIR"
    
    # Initialize logging
    echo "$(date): Starting KNIRV Network Synchronization Validation" > "$LOG_FILE"
    
    log_info "KNIRV Network Synchronization Validation"
    log_info "Testnet Root: $TESTNET_ROOT"
    log_info "Production Root: $PRODUCTION_ROOT"
    log_info "Configuration: $CONFIG_FILE"
    
    local validation_passed=true
    
    # Validate specific component if requested
    if [[ -n "$COMPONENT" ]]; then
        if ! validate_component "$COMPONENT"; then
            validation_passed=false
        fi
    else
        # Validate scripts if not tests-only
        if [[ "$TESTS_ONLY" != "true" ]]; then
            if ! validate_scripts; then
                validation_passed=false
            fi
        fi
        
        # Validate tests if not scripts-only
        if [[ "$SCRIPTS_ONLY" != "true" ]]; then
            if ! validate_tests; then
                validation_passed=false
            fi
        fi
    fi
    
    # Generate detailed report if requested
    if [[ "$DETAILED" == "true" ]]; then
        generate_detailed_report
    fi
    
    # Fix issues if requested
    if [[ "$FIX" == "true" && "$validation_passed" == "false" ]]; then
        fix_issues
    fi
    
    if [[ "$validation_passed" == "true" ]]; then
        log_success "All validation checks passed"
        exit 0
    else
        log_error "Validation failed"
        exit 1
    fi
}

# Execute main function
main "$@"
