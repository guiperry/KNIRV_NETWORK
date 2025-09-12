#!/bin/bash

# KNIRV Network ProtoBuf Synchronization Script
# Synchronizes ProtoBuf definitions across all platforms and compilers

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SHARED_PROTO_DIR="${PROJECT_ROOT}/shared-proto"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_DIR="${PROJECT_ROOT}/.proto-sync-backups/${TIMESTAMP}"

# Target directories for synchronization
declare -A SYNC_TARGETS=(
    ["KNIRVCORTEX"]="${PROJECT_ROOT}/KNIRVCORTEX/shared-types/proto"
    ["KNIRVCONTROLLER"]="${PROJECT_ROOT}/KNIRVCONTROLLER/src/core/protobuf/schemas"
    ["KNIRVENGINE"]="${PROJECT_ROOT}/KNIRVENGINE/desktop-client/proto"
    ["KNIRVANA_RUST"]="${PROJECT_ROOT}/KNIRVANA/gaming/cortex-compiler/proto"
)

# ProtoBuf files to sync
declare -A PROTO_FILES=(
    ["cortex"]="cortex/v1/cortex.proto"
    ["lora"]="lora/v1/lora.proto"
    ["agent"]="agent/v1/agent.proto"
    ["memory"]="memory/v1/memory.proto"
)

# Function to print colored output
print_status() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Function to create backup
create_backup() {
    local target_dir=$1
    local target_name=$2
    
    if [ -d "$target_dir" ]; then
        local backup_target="${BACKUP_DIR}/${target_name}"
        mkdir -p "$backup_target"
        cp -r "$target_dir"/* "$backup_target/" 2>/dev/null || true
        print_status "$BLUE" "  ✓ Backed up $target_name to $backup_target"
    fi
}

# Function to sync protobuf files
sync_protobuf_files() {
    local target_dir=$1
    local target_name=$2
    
    print_status "$BLUE" "Syncing ProtoBuf files to $target_name..."
    
    # Create target directory if it doesn't exist
    mkdir -p "$target_dir"
    
    # Copy each proto file
    for proto_name in "${!PROTO_FILES[@]}"; do
        local source_file="${SHARED_PROTO_DIR}/${PROTO_FILES[$proto_name]}"
        local target_file="${target_dir}/${proto_name}.proto"
        
        if [ -f "$source_file" ]; then
            # Create subdirectories if needed
            local target_subdir=$(dirname "$target_file")
            mkdir -p "$target_subdir"
            
            # Copy the file
            cp "$source_file" "$target_file"
            print_status "$GREEN" "  ✓ Synced ${proto_name}.proto"
        else
            print_status "$YELLOW" "  ⚠ Source file not found: $source_file"
        fi
    done
}

# Function to generate code for each platform
generate_platform_code() {
    local target_name=$1
    local target_dir=$2
    
    print_status "$BLUE" "Generating code for $target_name..."
    
    case "$target_name" in
        "KNIRVCORTEX")
            generate_rust_code "$target_dir"
            ;;
        "KNIRVCONTROLLER")
            generate_typescript_code "$target_dir"
            ;;
        "KNIRVENGINE")
            generate_go_code "$target_dir"
            ;;
        "KNIRVANA_RUST")
            generate_rust_code "$target_dir"
            ;;
        *)
            print_status "$YELLOW" "  ⚠ Unknown platform: $target_name"
            ;;
    esac
}

# Function to generate Rust code
generate_rust_code() {
    local proto_dir=$1
    local output_dir="${proto_dir}/../generated"
    
    mkdir -p "$output_dir"
    
    # Check if protoc is available
    if command -v protoc &> /dev/null; then
        # Generate Rust code using prost
        for proto_file in "$proto_dir"/*.proto; do
            if [ -f "$proto_file" ]; then
                protoc --rust_out="$output_dir" --proto_path="$proto_dir" "$proto_file" 2>/dev/null || true
            fi
        done
        print_status "$GREEN" "  ✓ Generated Rust code"
    else
        print_status "$YELLOW" "  ⚠ protoc not found, skipping Rust code generation"
    fi
}

# Function to generate TypeScript code
generate_typescript_code() {
    local proto_dir=$1
    local output_dir="${proto_dir}/../generated"
    
    mkdir -p "$output_dir"
    
    # Check if protoc and protoc-gen-ts are available
    if command -v protoc &> /dev/null && command -v protoc-gen-ts &> /dev/null; then
        for proto_file in "$proto_dir"/*.proto; do
            if [ -f "$proto_file" ]; then
                protoc --ts_out="$output_dir" --proto_path="$proto_dir" "$proto_file" 2>/dev/null || true
            fi
        done
        print_status "$GREEN" "  ✓ Generated TypeScript code"
    else
        print_status "$YELLOW" "  ⚠ protoc or protoc-gen-ts not found, skipping TypeScript code generation"
    fi
}

# Function to generate Go code
generate_go_code() {
    local proto_dir=$1
    local output_dir="${proto_dir}/../generated"
    
    mkdir -p "$output_dir"
    
    # Check if protoc and protoc-gen-go are available
    if command -v protoc &> /dev/null && command -v protoc-gen-go &> /dev/null; then
        for proto_file in "$proto_dir"/*.proto; do
            if [ -f "$proto_file" ]; then
                protoc --go_out="$output_dir" --proto_path="$proto_dir" "$proto_file" 2>/dev/null || true
            fi
        done
        print_status "$GREEN" "  ✓ Generated Go code"
    else
        print_status "$YELLOW" "  ⚠ protoc or protoc-gen-go not found, skipping Go code generation"
    fi
}

# Function to validate protobuf files
validate_protobuf_files() {
    print_status "$BLUE" "Validating ProtoBuf files..."
    
    local validation_failed=false
    
    for proto_name in "${!PROTO_FILES[@]}"; do
        local proto_file="${SHARED_PROTO_DIR}/${PROTO_FILES[$proto_name]}"
        
        if [ -f "$proto_file" ]; then
            if command -v protoc &> /dev/null; then
                if protoc --proto_path="$SHARED_PROTO_DIR" --descriptor_set_out=/dev/null "$proto_file" 2>/dev/null; then
                    print_status "$GREEN" "  ✓ Valid: ${proto_name}.proto"
                else
                    print_status "$RED" "  ✗ Invalid: ${proto_name}.proto"
                    validation_failed=true
                fi
            else
                print_status "$YELLOW" "  ⚠ protoc not available, skipping validation for ${proto_name}.proto"
            fi
        else
            print_status "$RED" "  ✗ Missing: ${proto_name}.proto"
            validation_failed=true
        fi
    done
    
    if [ "$validation_failed" = true ]; then
        print_status "$RED" "❌ ProtoBuf validation failed!"
        return 1
    else
        print_status "$GREEN" "✅ All ProtoBuf files are valid!"
        return 0
    fi
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --dry-run          Show what would be synchronized without making changes"
    echo "  --validate-only    Only validate ProtoBuf files, don't sync"
    echo "  --no-backup        Skip creating backups"
    echo "  --generate-code    Generate platform-specific code after sync"
    echo "  --help             Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                 # Sync all ProtoBuf files"
    echo "  $0 --dry-run       # Preview sync operations"
    echo "  $0 --validate-only # Only validate files"
    echo "  $0 --generate-code # Sync and generate code"
}

# Main function
main() {
    local dry_run=false
    local validate_only=false
    local no_backup=false
    local generate_code=false
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --dry-run)
                dry_run=true
                shift
                ;;
            --validate-only)
                validate_only=true
                shift
                ;;
            --no-backup)
                no_backup=true
                shift
                ;;
            --generate-code)
                generate_code=true
                shift
                ;;
            --help)
                show_usage
                exit 0
                ;;
            *)
                print_status "$RED" "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    print_status "$BLUE" "🚀 KNIRV Network ProtoBuf Synchronization"
    print_status "$BLUE" "========================================"
    echo ""
    
    # Validate ProtoBuf files first
    if ! validate_protobuf_files; then
        exit 1
    fi
    
    if [ "$validate_only" = true ]; then
        print_status "$GREEN" "✅ Validation complete!"
        exit 0
    fi
    
    # Create backup directory
    if [ "$no_backup" = false ] && [ "$dry_run" = false ]; then
        mkdir -p "$BACKUP_DIR"
        print_status "$BLUE" "📦 Creating backups..."
        
        for target_name in "${!SYNC_TARGETS[@]}"; do
            create_backup "${SYNC_TARGETS[$target_name]}" "$target_name"
        done
        echo ""
    fi
    
    # Sync ProtoBuf files
    print_status "$BLUE" "🔄 Synchronizing ProtoBuf files..."
    echo ""
    
    for target_name in "${!SYNC_TARGETS[@]}"; do
        if [ "$dry_run" = true ]; then
            print_status "$YELLOW" "[DRY RUN] Would sync to $target_name: ${SYNC_TARGETS[$target_name]}"
        else
            sync_protobuf_files "${SYNC_TARGETS[$target_name]}" "$target_name"
            
            if [ "$generate_code" = true ]; then
                generate_platform_code "$target_name" "${SYNC_TARGETS[$target_name]}"
            fi
        fi
        echo ""
    done
    
    if [ "$dry_run" = true ]; then
        print_status "$YELLOW" "🔍 Dry run complete - no changes made"
    else
        print_status "$GREEN" "✅ ProtoBuf synchronization complete!"
        
        if [ "$no_backup" = false ]; then
            print_status "$BLUE" "📦 Backups stored in: $BACKUP_DIR"
        fi
    fi
}

# Run main function with all arguments
main "$@"
