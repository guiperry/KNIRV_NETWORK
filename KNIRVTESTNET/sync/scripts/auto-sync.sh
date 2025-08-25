#!/bin/bash

# KNIRV Network Automated Synchronization Script
# Synchronizes scripts and testing patterns between production and testnet

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SYNC_ROOT="$(dirname "$SCRIPT_DIR")"
TESTNET_ROOT="$(dirname "$SYNC_ROOT")"
PRODUCTION_ROOT="$(dirname "$TESTNET_ROOT")"
CONFIG_FILE="$SYNC_ROOT/sync-config.json"
OUTPUT_DIR="$SYNC_ROOT/reports"
LOG_FILE="$OUTPUT_DIR/auto-sync.log"

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
KNIRV Network Automated Synchronization Tool

Usage: $0 [OPTIONS]

OPTIONS:
    --scripts-only      Synchronize only script patterns
    --tests-only        Synchronize only test patterns
    --dry-run          Perform a dry run without making changes
    --watch            Watch for changes and auto-synchronize
    --interval DURATION Watch interval (default: 5m)
    --config FILE      Configuration file path (default: sync-config.json)
    --output DIR       Output directory for reports (default: reports)
    --verbose          Enable verbose logging
    --validate         Validate synchronization configuration
    --help             Show this help message

EXAMPLES:
    $0                          # Full synchronization
    $0 --scripts-only           # Sync only scripts
    $0 --tests-only             # Sync only tests
    $0 --dry-run                # Preview changes
    $0 --watch --interval 10m   # Watch mode with 10-minute interval
    $0 --validate               # Validate configuration

EOF
}

# Validate environment and dependencies
validate_environment() {
    log_info "Validating environment..."
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        return 1
    fi
    
    # Check required directories
    if [[ ! -d "$TESTNET_ROOT" ]]; then
        log_error "KNIRVTESTNET directory not found: $TESTNET_ROOT"
        return 1
    fi
    
    if [[ ! -d "$PRODUCTION_ROOT" ]]; then
        log_error "Production network directory not found: $PRODUCTION_ROOT"
        return 1
    fi
    
    # Check configuration file
    if [[ ! -f "$CONFIG_FILE" ]]; then
        log_error "Configuration file not found: $CONFIG_FILE"
        return 1
    fi
    
    # Create output directory
    mkdir -p "$OUTPUT_DIR"
    
    log_success "Environment validation completed"
    return 0
}

# Build the synchronization tool
build_sync_tool() {
    log_info "Building synchronization tool..."
    
    cd "$SYNC_ROOT"
    
    # Initialize Go module if it doesn't exist
    if [[ ! -f "go.mod" ]]; then
        go mod init knirv-sync
    fi
    
    # Build the sync tool
    if go build -o bin/sync ./cmd/sync; then
        log_success "Synchronization tool built successfully"
        return 0
    else
        log_error "Failed to build synchronization tool"
        return 1
    fi
}

# Validate synchronization configuration
validate_config() {
    log_info "Validating synchronization configuration..."
    
    # Check if jq is available for JSON validation
    if command -v jq &> /dev/null; then
        if jq empty "$CONFIG_FILE" 2>/dev/null; then
            log_success "Configuration file is valid JSON"
        else
            log_error "Configuration file contains invalid JSON"
            return 1
        fi
    else
        log_warning "jq not available, skipping JSON validation"
    fi
    
    # Check if required components exist in production
    local components=(
        "KNIRVORACLE"
        "KNIRVCHAIN" 
        "KNIRVGRAPH"
        "KNIRVNEXUS"
        "KNIRVROUTER"
        "KNIRVGATEWAY"
    )
    
    for component in "${components[@]}"; do
        if [[ -d "$PRODUCTION_ROOT/$component" ]]; then
            log_info "Found production component: $component"
        else
            log_warning "Production component not found: $component"
        fi
    done
    
    log_success "Configuration validation completed"
    return 0
}

# Run synchronization
run_sync() {
    local args=()
    
    # Add configuration and paths
    args+=(--config "$CONFIG_FILE")
    args+=(--testnet "$TESTNET_ROOT")
    args+=(--production "$PRODUCTION_ROOT")
    args+=(--output "$OUTPUT_DIR")
    
    # Add user-specified options
    if [[ "$SCRIPTS_ONLY" == "true" ]]; then
        args+=(--scripts-only)
    fi
    
    if [[ "$TESTS_ONLY" == "true" ]]; then
        args+=(--tests-only)
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        args+=(--dry-run)
    fi
    
    if [[ "$WATCH" == "true" ]]; then
        args+=(--watch)
        args+=(--interval "$INTERVAL")
    fi
    
    if [[ "$VERBOSE" == "true" ]]; then
        args+=(--verbose)
    fi
    
    log_info "Starting synchronization with args: ${args[*]}"
    
    # Run the sync tool
    if "$SYNC_ROOT/bin/sync" "${args[@]}"; then
        log_success "Synchronization completed successfully"
        return 0
    else
        log_error "Synchronization failed"
        return 1
    fi
}

# Generate synchronization summary
generate_summary() {
    log_info "Generating synchronization summary..."
    
    local latest_report
    latest_report=$(find "$OUTPUT_DIR" -name "sync-report-*.json" -type f -printf '%T@ %p\n' | sort -n | tail -1 | cut -d' ' -f2-)
    
    if [[ -n "$latest_report" && -f "$latest_report" ]]; then
        log_info "Latest report: $latest_report"
        
        if command -v jq &> /dev/null; then
            local total_patterns successful failed
            total_patterns=$(jq -r '.total_patterns' "$latest_report")
            successful=$(jq -r '.successful' "$latest_report")
            failed=$(jq -r '.failed' "$latest_report")
            
            echo ""
            echo "=========================================="
            echo "SYNCHRONIZATION SUMMARY"
            echo "=========================================="
            echo "Total Patterns: $total_patterns"
            echo "Successful: $successful"
            echo "Failed: $failed"
            echo "Report: $latest_report"
            echo "=========================================="
        fi
    else
        log_warning "No synchronization reports found"
    fi
}

# Main execution
main() {
    # Default values
    SCRIPTS_ONLY="false"
    TESTS_ONLY="false"
    DRY_RUN="false"
    WATCH="false"
    INTERVAL="5m"
    VERBOSE="false"
    VALIDATE_ONLY="false"
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --scripts-only)
                SCRIPTS_ONLY="true"
                shift
                ;;
            --tests-only)
                TESTS_ONLY="true"
                shift
                ;;
            --dry-run)
                DRY_RUN="true"
                shift
                ;;
            --watch)
                WATCH="true"
                shift
                ;;
            --interval)
                INTERVAL="$2"
                shift 2
                ;;
            --config)
                CONFIG_FILE="$2"
                shift 2
                ;;
            --output)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            --verbose)
                VERBOSE="true"
                shift
                ;;
            --validate)
                VALIDATE_ONLY="true"
                shift
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
    
    # Initialize logging
    echo "$(date): Starting KNIRV Network Synchronization" > "$LOG_FILE"
    
    log_info "KNIRV Network Automated Synchronization"
    log_info "Testnet Root: $TESTNET_ROOT"
    log_info "Production Root: $PRODUCTION_ROOT"
    log_info "Configuration: $CONFIG_FILE"
    log_info "Output Directory: $OUTPUT_DIR"
    
    # Validate environment
    if ! validate_environment; then
        exit 1
    fi
    
    # Validate configuration
    if ! validate_config; then
        exit 1
    fi
    
    # Exit if only validation was requested
    if [[ "$VALIDATE_ONLY" == "true" ]]; then
        log_success "Validation completed successfully"
        exit 0
    fi
    
    # Build sync tool
    if ! build_sync_tool; then
        exit 1
    fi
    
    # Run synchronization
    if ! run_sync; then
        exit 1
    fi
    
    # Generate summary
    generate_summary
    
    log_success "KNIRV Network synchronization completed"
}

# Execute main function
main "$@"
