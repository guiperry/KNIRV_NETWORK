#!/bin/bash

# Documentation Synchronization Script for KNIRV Network
# Syncs documentation across the monorepo and generates unified API specs

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KNIRVSYNC_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
NETWORK_ROOT="$(cd "$KNIRVSYNC_ROOT/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Build the doc-sync binary
build_binary() {
    log_info "Building doc-sync binary..."
    cd "$KNIRVSYNC_ROOT"

    if ! go build -o bin/doc-sync ./cmd/doc-sync/main.go; then
        log_error "Failed to build doc-sync binary"
        exit 1
    fi

    log_success "Binary built successfully: $KNIRVSYNC_ROOT/bin/doc-sync"
}

# Scan for documentation gaps
scan_gaps() {
    log_info "Scanning for documentation gaps..."
    cd "$KNIRVSYNC_ROOT"

    ./bin/doc-sync -scan -config config/doc-sync-config.json

    if [ -f "reports/doc-gaps-report.md" ]; then
        log_success "Gap report generated: reports/doc-gaps-report.md"

        # Display summary
        gap_count=$(grep -c "^##" reports/doc-gaps-report.md || echo "0")
        log_info "Services with gaps: $gap_count"
    fi
}

# Sync README files
sync_readmes() {
    log_info "Synchronizing README files..."
    cd "$KNIRVSYNC_ROOT"

    ./bin/doc-sync -readme -config config/doc-sync-config.json

    log_success "README synchronization completed"
}

# Sync API specifications
sync_api_specs() {
    log_info "Synchronizing API specifications..."
    cd "$KNIRVSYNC_ROOT"

    ./bin/doc-sync -api -config config/doc-sync-config.json

    log_success "API spec synchronization completed"
}

# Generate unified API spec
generate_unified() {
    log_info "Generating unified OpenAPI specification..."
    cd "$KNIRVSYNC_ROOT"

    ./bin/doc-sync -unified -config config/doc-sync-config.json

    log_success "Unified API spec generated"
}

# Sync all documentation
sync_all() {
    log_info "Performing complete documentation synchronization..."
    cd "$KNIRVSYNC_ROOT"

    ./bin/doc-sync -config config/doc-sync-config.json

    log_success "Complete synchronization finished"
}

# Display usage
usage() {
    cat << EOF
Documentation Synchronization Tool for KNIRV Network

Usage: $0 [COMMAND] [OPTIONS]

Commands:
    build       Build the doc-sync binary
    scan        Scan for documentation gaps only
    readme      Sync README files to central documentation
    api         Sync API specifications to central locations
    unified     Generate unified OpenAPI specification
    all         Perform complete documentation synchronization (default)
    help        Display this help message

Options:
    -v, --verbose    Enable verbose output
    -c, --config     Specify custom configuration file

Examples:
    $0 build         # Build the binary
    $0 scan          # Scan for gaps
    $0 all           # Sync everything
    $0 readme        # Sync only READMEs
    $0 api           # Sync only API specs

EOF
}

# Main script logic
main() {
    local command="${1:-all}"

    # Ensure we're in the right directory
    if [ ! -d "$KNIRVSYNC_ROOT/config" ]; then
        log_error "KNIRVSYNC directory structure not found"
        exit 1
    fi

    case "$command" in
        build)
            build_binary
            ;;
        scan)
            if [ ! -f "$KNIRVSYNC_ROOT/bin/doc-sync" ]; then
                log_warning "Binary not found, building first..."
                build_binary
            fi
            scan_gaps
            ;;
        readme)
            if [ ! -f "$KNIRVSYNC_ROOT/bin/doc-sync" ]; then
                log_warning "Binary not found, building first..."
                build_binary
            fi
            sync_readmes
            ;;
        api)
            if [ ! -f "$KNIRVSYNC_ROOT/bin/doc-sync" ]; then
                log_warning "Binary not found, building first..."
                build_binary
            fi
            sync_api_specs
            ;;
        unified)
            if [ ! -f "$KNIRVSYNC_ROOT/bin/doc-sync" ]; then
                log_warning "Binary not found, building first..."
                build_binary
            fi
            generate_unified
            ;;
        all)
            if [ ! -f "$KNIRVSYNC_ROOT/bin/doc-sync" ]; then
                log_warning "Binary not found, building first..."
                build_binary
            fi
            sync_all
            ;;
        help|--help|-h)
            usage
            ;;
        *)
            log_error "Unknown command: $command"
            usage
            exit 1
            ;;
    esac
}

# Run main
main "$@"
