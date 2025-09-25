#!/bin/bash

# KNIRV Network Rollback and Recovery Script
# Manages snapshots and rollback operations for synchronization

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SYNC_ROOT="$(dirname "$SCRIPT_DIR")"
TESTNET_ROOT="$(dirname "$SYNC_ROOT")"
BACKUP_DIR="$SYNC_ROOT/backups"
LOG_FILE="$SYNC_ROOT/reports/rollback.log"

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
KNIRV Network Rollback and Recovery Tool

Usage: $0 COMMAND [OPTIONS]

COMMANDS:
    create DESCRIPTION      Create a new snapshot
    list                   List all available snapshots
    rollback SNAPSHOT_ID   Rollback to a specific snapshot
    delete SNAPSHOT_ID     Delete a specific snapshot
    cleanup DAYS           Delete snapshots older than DAYS
    verify SNAPSHOT_ID     Verify snapshot integrity

OPTIONS:
    --backup-dir DIR       Backup directory (default: sync/backups)
    --force               Force operation without confirmation
    --verbose             Enable verbose logging
    --help                Show this help message

EXAMPLES:
    $0 create "Before major sync update"
    $0 list
    $0 rollback snapshot-20240825-143022
    $0 delete snapshot-20240825-143022
    $0 cleanup 30
    $0 verify snapshot-20240825-143022

EOF
}

# Create a new snapshot
create_snapshot() {
    local description="$1"
    
    if [[ -z "$description" ]]; then
        log_error "Description is required for snapshot creation"
        return 1
    fi
    
    log_info "Creating snapshot: $description"
    
    # Generate snapshot ID
    local snapshot_id="snapshot-$(date +%Y%m%d-%H%M%S)"
    local snapshot_path="$BACKUP_DIR/$snapshot_id"
    
    # Create backup directory
    mkdir -p "$snapshot_path"
    
    # Backup sync patterns
    if [[ -d "$TESTNET_ROOT/sync/patterns" ]]; then
        log_info "Backing up sync patterns..."
        cp -r "$TESTNET_ROOT/sync/patterns" "$snapshot_path/"
    fi
    
    # Backup scripts
    if [[ -d "$TESTNET_ROOT/scripts" ]]; then
        log_info "Backing up scripts..."
        cp -r "$TESTNET_ROOT/scripts" "$snapshot_path/"
    fi
    
    # Backup tests
    if [[ -d "$TESTNET_ROOT/tests" ]]; then
        log_info "Backing up tests..."
        cp -r "$TESTNET_ROOT/tests" "$snapshot_path/"
    fi
    
    # Create metadata
    local metadata_file="$snapshot_path/metadata.json"
    cat > "$metadata_file" << EOF
{
  "id": "$snapshot_id",
  "timestamp": "$(date -Iseconds)",
  "description": "$description",
  "components": ["patterns", "scripts", "tests"],
  "backup_path": "$snapshot_path",
  "size": $(du -sb "$snapshot_path" | cut -f1),
  "checksum": "$(find "$snapshot_path" -type f -exec md5sum {} \; | md5sum | cut -d' ' -f1)"
}
EOF
    
    log_success "Snapshot created: $snapshot_id"
    echo "Snapshot ID: $snapshot_id"
    echo "Path: $snapshot_path"
    echo "Size: $(du -sh "$snapshot_path" | cut -f1)"
}

# List all snapshots
list_snapshots() {
    log_info "Listing available snapshots..."
    
    if [[ ! -d "$BACKUP_DIR" ]]; then
        log_warning "No backup directory found"
        return 0
    fi
    
    local snapshots=()
    while IFS= read -r -d '' snapshot_dir; do
        snapshots+=("$snapshot_dir")
    done < <(find "$BACKUP_DIR" -maxdepth 1 -type d -name "snapshot-*" -print0 | sort -z)
    
    if [[ ${#snapshots[@]} -eq 0 ]]; then
        log_warning "No snapshots found"
        return 0
    fi
    
    echo ""
    echo "Available Snapshots:"
    echo "==================="
    
    for snapshot_dir in "${snapshots[@]}"; do
        local snapshot_id
        snapshot_id=$(basename "$snapshot_dir")
        local metadata_file="$snapshot_dir/metadata.json"
        
        if [[ -f "$metadata_file" ]]; then
            local description timestamp size
            description=$(jq -r '.description' "$metadata_file" 2>/dev/null || echo "No description")
            timestamp=$(jq -r '.timestamp' "$metadata_file" 2>/dev/null || echo "Unknown")
            size=$(du -sh "$snapshot_dir" | cut -f1)
            
            echo "ID: $snapshot_id"
            echo "  Description: $description"
            echo "  Created: $timestamp"
            echo "  Size: $size"
            echo ""
        else
            echo "ID: $snapshot_id (metadata missing)"
            echo ""
        fi
    done
}

# Rollback to a specific snapshot
rollback_snapshot() {
    local snapshot_id="$1"
    
    if [[ -z "$snapshot_id" ]]; then
        log_error "Snapshot ID is required for rollback"
        return 1
    fi
    
    local snapshot_path="$BACKUP_DIR/$snapshot_id"
    
    if [[ ! -d "$snapshot_path" ]]; then
        log_error "Snapshot not found: $snapshot_id"
        return 1
    fi
    
    log_info "Rolling back to snapshot: $snapshot_id"
    
    # Verify snapshot integrity
    if ! verify_snapshot "$snapshot_id"; then
        log_error "Snapshot integrity check failed"
        return 1
    fi
    
    # Create pre-rollback backup
    local pre_rollback_description="Pre-rollback backup before $snapshot_id"
    log_info "Creating pre-rollback backup..."
    create_snapshot "$pre_rollback_description"
    
    # Confirm rollback unless forced
    if [[ "$FORCE" != "true" ]]; then
        echo ""
        echo "WARNING: This will replace current sync state with snapshot $snapshot_id"
        read -p "Are you sure you want to continue? (y/N): " -n 1 -r
        echo ""
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Rollback cancelled by user"
            return 0
        fi
    fi
    
    local files_restored=0
    
    # Restore sync patterns
    if [[ -d "$snapshot_path/patterns" ]]; then
        log_info "Restoring sync patterns..."
        rm -rf "$TESTNET_ROOT/sync/patterns"
        cp -r "$snapshot_path/patterns" "$TESTNET_ROOT/sync/"
        files_restored=$((files_restored + $(find "$snapshot_path/patterns" -type f | wc -l)))
    fi
    
    # Restore scripts
    if [[ -d "$snapshot_path/scripts" ]]; then
        log_info "Restoring scripts..."
        rm -rf "$TESTNET_ROOT/scripts"
        cp -r "$snapshot_path/scripts" "$TESTNET_ROOT/"
        files_restored=$((files_restored + $(find "$snapshot_path/scripts" -type f | wc -l)))
    fi
    
    # Restore tests
    if [[ -d "$snapshot_path/tests" ]]; then
        log_info "Restoring tests..."
        rm -rf "$TESTNET_ROOT/tests"
        cp -r "$snapshot_path/tests" "$TESTNET_ROOT/"
        files_restored=$((files_restored + $(find "$snapshot_path/tests" -type f | wc -l)))
    fi
    
    log_success "Rollback completed successfully"
    echo "Files restored: $files_restored"
}

# Delete a specific snapshot
delete_snapshot() {
    local snapshot_id="$1"
    
    if [[ -z "$snapshot_id" ]]; then
        log_error "Snapshot ID is required for deletion"
        return 1
    fi
    
    local snapshot_path="$BACKUP_DIR/$snapshot_id"
    
    if [[ ! -d "$snapshot_path" ]]; then
        log_error "Snapshot not found: $snapshot_id"
        return 1
    fi
    
    # Confirm deletion unless forced
    if [[ "$FORCE" != "true" ]]; then
        echo ""
        echo "WARNING: This will permanently delete snapshot $snapshot_id"
        read -p "Are you sure you want to continue? (y/N): " -n 1 -r
        echo ""
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Deletion cancelled by user"
            return 0
        fi
    fi
    
    log_info "Deleting snapshot: $snapshot_id"
    rm -rf "$snapshot_path"
    log_success "Snapshot deleted: $snapshot_id"
}

# Cleanup old snapshots
cleanup_snapshots() {
    local days="$1"
    
    if [[ -z "$days" ]]; then
        log_error "Number of days is required for cleanup"
        return 1
    fi
    
    if ! [[ "$days" =~ ^[0-9]+$ ]]; then
        log_error "Days must be a positive number"
        return 1
    fi
    
    log_info "Cleaning up snapshots older than $days days..."
    
    if [[ ! -d "$BACKUP_DIR" ]]; then
        log_warning "No backup directory found"
        return 0
    fi
    
    local deleted=0
    local cutoff_date
    cutoff_date=$(date -d "$days days ago" +%s)
    
    while IFS= read -r -d '' snapshot_dir; do
        local snapshot_id
        snapshot_id=$(basename "$snapshot_dir")
        
        # Extract date from snapshot ID (format: snapshot-YYYYMMDD-HHMMSS)
        if [[ $snapshot_id =~ snapshot-([0-9]{8})-([0-9]{6}) ]]; then
            local date_part="${BASH_REMATCH[1]}"
            local time_part="${BASH_REMATCH[2]}"
            local snapshot_date
            snapshot_date=$(date -d "${date_part:0:4}-${date_part:4:2}-${date_part:6:2} ${time_part:0:2}:${time_part:2:2}:${time_part:4:2}" +%s)
            
            if [[ $snapshot_date -lt $cutoff_date ]]; then
                log_info "Deleting old snapshot: $snapshot_id"
                rm -rf "$snapshot_dir"
                deleted=$((deleted + 1))
            fi
        fi
    done < <(find "$BACKUP_DIR" -maxdepth 1 -type d -name "snapshot-*" -print0)
    
    log_success "Cleaned up $deleted old snapshots"
}

# Verify snapshot integrity
verify_snapshot() {
    local snapshot_id="$1"
    
    if [[ -z "$snapshot_id" ]]; then
        log_error "Snapshot ID is required for verification"
        return 1
    fi
    
    local snapshot_path="$BACKUP_DIR/$snapshot_id"
    local metadata_file="$snapshot_path/metadata.json"
    
    if [[ ! -d "$snapshot_path" ]]; then
        log_error "Snapshot not found: $snapshot_id"
        return 1
    fi
    
    if [[ ! -f "$metadata_file" ]]; then
        log_error "Snapshot metadata not found: $snapshot_id"
        return 1
    fi
    
    log_info "Verifying snapshot integrity: $snapshot_id"
    
    # Check if jq is available
    if ! command -v jq &> /dev/null; then
        log_warning "jq not available, skipping detailed verification"
        log_success "Basic verification passed"
        return 0
    fi
    
    # Verify metadata
    local expected_size current_size expected_checksum current_checksum
    expected_size=$(jq -r '.size' "$metadata_file")
    current_size=$(du -sb "$snapshot_path" | cut -f1)
    expected_checksum=$(jq -r '.checksum' "$metadata_file")
    current_checksum=$(find "$snapshot_path" -type f -exec md5sum {} \; | md5sum | cut -d' ' -f1)
    
    if [[ "$expected_size" != "$current_size" ]]; then
        log_error "Size mismatch: expected $expected_size, got $current_size"
        return 1
    fi
    
    if [[ "$expected_checksum" != "$current_checksum" ]]; then
        log_error "Checksum mismatch: expected $expected_checksum, got $current_checksum"
        return 1
    fi
    
    log_success "Snapshot integrity verified: $snapshot_id"
    return 0
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
            --backup-dir)
                BACKUP_DIR="$2"
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
    
    # Create backup and reports directories
    mkdir -p "$BACKUP_DIR"
    mkdir -p "$(dirname "$LOG_FILE")"
    
    # Initialize logging
    echo "$(date): Starting KNIRV Network Rollback Operation: $command" > "$LOG_FILE"
    
    log_info "KNIRV Network Rollback and Recovery Tool"
    log_info "Command: $command"
    log_info "Backup Directory: $BACKUP_DIR"
    
    # Execute command
    case $command in
        create)
            if [[ $# -eq 0 ]]; then
                log_error "Description is required for create command"
                exit 1
            fi
            create_snapshot "$*"
            ;;
        list)
            list_snapshots
            ;;
        rollback)
            if [[ $# -eq 0 ]]; then
                log_error "Snapshot ID is required for rollback command"
                exit 1
            fi
            rollback_snapshot "$1"
            ;;
        delete)
            if [[ $# -eq 0 ]]; then
                log_error "Snapshot ID is required for delete command"
                exit 1
            fi
            delete_snapshot "$1"
            ;;
        cleanup)
            if [[ $# -eq 0 ]]; then
                log_error "Number of days is required for cleanup command"
                exit 1
            fi
            cleanup_snapshots "$1"
            ;;
        verify)
            if [[ $# -eq 0 ]]; then
                log_error "Snapshot ID is required for verify command"
                exit 1
            fi
            verify_snapshot "$1"
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
