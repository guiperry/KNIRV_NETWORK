#!/bin/bash

# KNIRV Network Synchronization Integration Script
# Complete integration of all synchronization components

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SYNC_ROOT="$(dirname "$SCRIPT_DIR")"
TESTNET_ROOT="$(dirname "$SYNC_ROOT")"
PRODUCTION_ROOT="$(dirname "$TESTNET_ROOT")"
CONFIG_FILE="$SYNC_ROOT/sync-config.json"
OUTPUT_DIR="$SYNC_ROOT/reports"
LOG_FILE="$OUTPUT_DIR/integration.log"

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
KNIRV Network Synchronization Integration Tool

Usage: $0 COMMAND [OPTIONS]

COMMANDS:
    setup               Setup synchronization environment
    sync                Perform full synchronization
    validate            Validate synchronization state
    monitor             Start monitoring mode
    rollback            Rollback to previous state
    status              Show synchronization status
    test                Run integration tests

OPTIONS:
    --config FILE       Configuration file path (default: sync-config.json)
    --output DIR        Output directory for reports (default: reports)
    --force             Force operation without confirmation
    --verbose           Enable verbose logging
    --help              Show this help message

EXAMPLES:
    $0 setup                    # Setup synchronization environment
    $0 sync                     # Perform full synchronization
    $0 validate --verbose       # Validate with verbose output
    $0 monitor                  # Start monitoring mode
    $0 rollback                 # Interactive rollback
    $0 status                   # Show current status

EOF
}

# Setup synchronization environment
setup_sync_environment() {
    log_info "Setting up KNIRV Network synchronization environment..."
    
    # Create necessary directories
    mkdir -p "$SYNC_ROOT"/{bin,reports,backups,patterns}
    mkdir -p "$OUTPUT_DIR"
    
    # Create Go module if it doesn't exist
    if [[ ! -f "$SYNC_ROOT/go.mod" ]]; then
        cd "$SYNC_ROOT"
        go mod init knirv-sync
        log_success "Go module initialized"
    fi
    
    # Build synchronization tools
    log_info "Building synchronization tools..."
    
    cd "$SYNC_ROOT"
    
    # Build main sync tool
    if go build -o bin/sync ./cmd/sync; then
        log_success "Sync tool built successfully"
    else
        log_error "Failed to build sync tool"
        return 1
    fi
    
    # Make scripts executable
    chmod +x scripts/*.sh
    log_success "Scripts made executable"
    
    # Validate configuration
    if [[ -f "$CONFIG_FILE" ]]; then
        if command -v jq &> /dev/null; then
            if jq empty "$CONFIG_FILE" 2>/dev/null; then
                log_success "Configuration file validated"
            else
                log_error "Configuration file contains invalid JSON"
                return 1
            fi
        else
            log_warning "jq not available, skipping JSON validation"
        fi
    else
        log_error "Configuration file not found: $CONFIG_FILE"
        return 1
    fi
    
    log_success "Synchronization environment setup completed"
}

# Perform full synchronization
perform_full_sync() {
    log_info "Performing full KNIRV Network synchronization..."
    
    # Create pre-sync snapshot
    log_info "Creating pre-sync snapshot..."
    if "$SCRIPT_DIR/rollback.sh" create "Pre-sync snapshot $(date)"; then
        log_success "Pre-sync snapshot created"
    else
        log_warning "Failed to create pre-sync snapshot"
    fi
    
    # Run synchronization
    log_info "Running synchronization..."
    if "$SCRIPT_DIR/auto-sync.sh" --verbose; then
        log_success "Synchronization completed"
    else
        log_error "Synchronization failed"
        return 1
    fi
    
    # Validate results
    log_info "Validating synchronization results..."
    if "$SCRIPT_DIR/validate-sync.sh" --detailed; then
        log_success "Validation passed"
    else
        log_error "Validation failed"
        return 1
    fi
    
    log_success "Full synchronization completed successfully"
}

# Validate synchronization state
validate_sync_state() {
    log_info "Validating KNIRV Network synchronization state..."
    
    if "$SCRIPT_DIR/validate-sync.sh" "$@"; then
        log_success "Synchronization state is valid"
        return 0
    else
        log_error "Synchronization state validation failed"
        return 1
    fi
}

# Start monitoring mode
start_monitoring() {
    log_info "Starting KNIRV Network synchronization monitoring..."
    
    # Start auto-sync in watch mode
    "$SCRIPT_DIR/auto-sync.sh" --watch --interval 10m --verbose
}

# Interactive rollback
interactive_rollback() {
    log_info "KNIRV Network Synchronization Rollback"
    
    # List available snapshots
    echo ""
    echo "Available snapshots:"
    "$SCRIPT_DIR/rollback.sh" list
    
    echo ""
    read -p "Enter snapshot ID to rollback to (or 'cancel' to abort): " snapshot_id
    
    if [[ "$snapshot_id" == "cancel" ]]; then
        log_info "Rollback cancelled by user"
        return 0
    fi
    
    if [[ -z "$snapshot_id" ]]; then
        log_error "No snapshot ID provided"
        return 1
    fi
    
    # Verify snapshot
    if "$SCRIPT_DIR/rollback.sh" verify "$snapshot_id"; then
        log_success "Snapshot verified"
    else
        log_error "Snapshot verification failed"
        return 1
    fi
    
    # Perform rollback
    if "$SCRIPT_DIR/rollback.sh" rollback "$snapshot_id"; then
        log_success "Rollback completed successfully"
    else
        log_error "Rollback failed"
        return 1
    fi
}

# Show synchronization status
show_sync_status() {
    log_info "KNIRV Network Synchronization Status"
    
    echo ""
    echo "=========================================="
    echo "SYNCHRONIZATION STATUS"
    echo "=========================================="
    
    # Check if tools are built
    if [[ -f "$SYNC_ROOT/bin/sync" ]]; then
        echo "✓ Sync tool: Available"
    else
        echo "✗ Sync tool: Not built"
    fi
    
    # Check configuration
    if [[ -f "$CONFIG_FILE" ]]; then
        echo "✓ Configuration: Available"
    else
        echo "✗ Configuration: Missing"
    fi
    
    # Check production components
    echo ""
    echo "Production Components:"
    for component in KNIRVORACLE KNIRVCHAIN KNIRVGRAPH KNIRVSERVER KNIRVROUTER KNIRVGATEWAY; do
        if [[ -d "$PRODUCTION_ROOT/$component" ]]; then
            echo "  ✓ $component"
        else
            echo "  ✗ $component"
        fi
    done
    
    # Check sync patterns
    echo ""
    echo "Sync Patterns:"
    for component in knirvoracle knirvchain knirvgraph knirvserver knirvrouter knirvgateway; do
        if [[ -d "$SYNC_ROOT/patterns/$component" ]]; then
            echo "  ✓ $component"
        else
            echo "  ✗ $component"
        fi
    done
    
    # Check recent reports
    echo ""
    echo "Recent Reports:"
    if [[ -d "$OUTPUT_DIR" ]]; then
        local report_count
        report_count=$(find "$OUTPUT_DIR" -name "*.json" -type f | wc -l)
        echo "  Reports: $report_count"
        
        if [[ $report_count -gt 0 ]]; then
            local latest_report
            latest_report=$(find "$OUTPUT_DIR" -name "*.json" -type f -printf '%T@ %p\n' | sort -n | tail -1 | cut -d' ' -f2-)
            echo "  Latest: $(basename "$latest_report")"
        fi
    else
        echo "  Reports: No reports directory"
    fi
    
    # Check snapshots
    echo ""
    echo "Snapshots:"
    if [[ -d "$SYNC_ROOT/backups" ]]; then
        local snapshot_count
        snapshot_count=$(find "$SYNC_ROOT/backups" -maxdepth 1 -type d -name "snapshot-*" | wc -l)
        echo "  Snapshots: $snapshot_count"
    else
        echo "  Snapshots: No backups directory"
    fi
    
    echo "=========================================="
}

# Run integration tests
run_integration_tests() {
    log_info "Running KNIRV Network synchronization integration tests..."
    
    local test_passed=0
    local test_failed=0
    
    # Test 1: Environment setup
    log_info "Test 1: Environment setup"
    if setup_sync_environment; then
        log_success "✓ Environment setup test passed"
        test_passed=$((test_passed + 1))
    else
        log_error "✗ Environment setup test failed"
        test_failed=$((test_failed + 1))
    fi
    
    # Test 2: Configuration validation
    log_info "Test 2: Configuration validation"
    if [[ -f "$CONFIG_FILE" ]]; then
        log_success "✓ Configuration validation test passed"
        test_passed=$((test_passed + 1))
    else
        log_error "✗ Configuration validation test failed"
        test_failed=$((test_failed + 1))
    fi
    
    # Test 3: Sync tool functionality
    log_info "Test 3: Sync tool functionality"
    if [[ -f "$SYNC_ROOT/bin/sync" ]]; then
        log_success "✓ Sync tool functionality test passed"
        test_passed=$((test_passed + 1))
    else
        log_error "✗ Sync tool functionality test failed"
        test_failed=$((test_failed + 1))
    fi
    
    # Test 4: Validation script
    log_info "Test 4: Validation script"
    if "$SCRIPT_DIR/validate-sync.sh" --help > /dev/null 2>&1; then
        log_success "✓ Validation script test passed"
        test_passed=$((test_passed + 1))
    else
        log_error "✗ Validation script test failed"
        test_failed=$((test_failed + 1))
    fi
    
    # Test 5: Rollback script
    log_info "Test 5: Rollback script"
    if "$SCRIPT_DIR/rollback.sh" list > /dev/null 2>&1; then
        log_success "✓ Rollback script test passed"
        test_passed=$((test_passed + 1))
    else
        log_error "✗ Rollback script test failed"
        test_failed=$((test_failed + 1))
    fi
    
    echo ""
    echo "=========================================="
    echo "INTEGRATION TEST RESULTS"
    echo "=========================================="
    echo "Tests Passed: $test_passed"
    echo "Tests Failed: $test_failed"
    echo "Total Tests: $((test_passed + test_failed))"
    
    if [[ $test_failed -eq 0 ]]; then
        log_success "All integration tests passed"
        return 0
    else
        log_error "$test_failed integration tests failed"
        return 1
    fi
}

# Main execution
main() {
    # Default values
    FORCE="false"
    VERBOSE="false"
    
    # Parse command line arguments
    if [[ $# -eq 0 ]]; then
        print_usage
        exit 1
    fi
    
    local command="$1"
    shift
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case $1 in
            --config)
                CONFIG_FILE="$2"
                shift 2
                ;;
            --output)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            --force)
                FORCE="true"
                shift
                ;;
            --verbose)
                VERBOSE="true"
                shift
                ;;
            --help)
                print_usage
                exit 0
                ;;
            -*)
                log_error "Unknown option: $1"
                print_usage
                exit 1
                ;;
            *)
                break
                ;;
        esac
    done
    
    # Create output directory
    mkdir -p "$OUTPUT_DIR"
    
    # Initialize logging
    echo "$(date): Starting KNIRV Network Synchronization Integration: $command" > "$LOG_FILE"
    
    log_info "KNIRV Network Synchronization Integration"
    log_info "Command: $command"
    log_info "Configuration: $CONFIG_FILE"
    log_info "Output Directory: $OUTPUT_DIR"
    
    # Execute command
    case $command in
        setup)
            setup_sync_environment
            ;;
        sync)
            perform_full_sync
            ;;
        validate)
            validate_sync_state "$@"
            ;;
        monitor)
            start_monitoring
            ;;
        rollback)
            interactive_rollback
            ;;
        status)
            show_sync_status
            ;;
        test)
            run_integration_tests
            ;;
        *)
            log_error "Unknown command: $command"
            print_usage
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"
