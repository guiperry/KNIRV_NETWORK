#!/bin/bash

# KNIRV Portal Version Synchronization Script
# Synchronizes static site contents from KNIRVNEXUS/out to the corporate nexus-portal
# Clones all static files without preserving styling differences

set -euo pipefail

# =============================================================================
# CONFIGURATION
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SYNC_STATE_DIR="$PROJECT_ROOT/.portal-sync-state"
BACKUP_DIR="$PROJECT_ROOT/.portal-sync-backups"
LOG_FILE="$SYNC_STATE_DIR/portal-sync-$(date +%Y%m%d_%H%M%S).log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default options
DRY_RUN=false
FORCE=false
VERBOSE=false

# Source and target directories
SOURCE_DIR="$PROJECT_ROOT/KNIRVNEXUS/out"
TARGET_DIR="$PROJECT_ROOT/KNIRVGATEWAY/primary-website/public/nexus-portal"

# Protected files that must be preserved in target
PROTECTED_FILES=(
    "admin-gateway.config.js"
    "index.html"
    "landing.html"
    "dashboard.html"
)

# File renaming map: source -> target
# The cloned index.html from KNIRVNEXUS/out will be renamed to app.html
RENAME_MAP=(
    "index.html:app.html"
)

# =============================================================================
# UTILITY FUNCTIONS
# =============================================================================

log() {
    local level="$1"
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    echo "[$timestamp] [$level] $message" >> "$LOG_FILE"
    
    case "$level" in
        "ERROR") echo -e "${RED}❌ $message${NC}" >&2 ;;
        "SUCCESS") echo -e "${GREEN}✅ $message${NC}" >&2 ;;
        "WARNING") echo -e "${YELLOW}⚠️  $message${NC}" >&2 ;;
        "INFO") echo -e "${BLUE}ℹ️  $message${NC}" >&2 ;;
    esac
}

setup_directories() {
    mkdir -p "$SYNC_STATE_DIR" "$BACKUP_DIR"
    touch "$LOG_FILE"
}

backup_directory() {
    local dir="$1"
    local backup_name="$2"
    
    if [[ -d "$dir" ]]; then
        local backup_path="$BACKUP_DIR/$(date +%Y%m%d_%H%M%S)_${backup_name}"
        cp -r "$dir" "$backup_path"
        log "INFO" "Backed up $dir to $backup_path"
    fi
}

backup_protected_files() {
    local temp_dir="$SYNC_STATE_DIR/protected_files_$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$temp_dir"
    
    local backed_up_count=0
    for file in "${PROTECTED_FILES[@]}"; do
        local target_file="$TARGET_DIR/$file"
        if [[ -f "$target_file" ]]; then
            cp "$target_file" "$temp_dir/"
            log "INFO" "Backed up protected file: $file"
            ((backed_up_count++))
        else
            log "WARNING" "Protected file does not exist in target, skipping backup: $file"
        fi
    done
    
    log "INFO" "Backed up $backed_up_count protected file(s) to temporary directory"
    echo "$temp_dir"
}

restore_protected_files() {
    local temp_dir="$1"
    
    if [[ ! -d "$temp_dir" ]]; then
        log "WARNING" "Protected files backup directory not found: $temp_dir"
        return
    fi
    
    for file in "${PROTECTED_FILES[@]}"; do
        local backup_file="$temp_dir/$file"
        local target_file="$TARGET_DIR/$file"
        
        if [[ -f "$backup_file" ]]; then
            cp -f "$backup_file" "$target_file"
            if [[ -f "$target_file" ]]; then
                log "SUCCESS" "Restored protected file: $file"
            else
                log "ERROR" "Failed to restore protected file: $file"
            fi
        else
            log "WARNING" "Protected file not found in backup: $file"
        fi
    done
    
    rm -rf "$temp_dir"
    log "INFO" "Cleaned up temporary backup directory"
}

rename_cloned_files() {
    for mapping in "${RENAME_MAP[@]}"; do
        local source_name="${mapping%%:*}"
        local target_name="${mapping##*:}"
        local source_file="$TARGET_DIR/$source_name"
        local target_file="$TARGET_DIR/$target_name"
        
        if [[ -f "$source_file" ]]; then
            mv "$source_file" "$target_file"
            log "SUCCESS" "Renamed cloned file: $source_name -> $target_name"
        else
            log "WARNING" "Source file not found for renaming: $source_name"
        fi
    done
}

# =============================================================================
# SYNCHRONIZATION FUNCTIONS
# =============================================================================

sync_static_site() {
    log "INFO" "Syncing static site from $SOURCE_DIR to $TARGET_DIR"
    
    # Check if source directory exists
    if [[ ! -d "$SOURCE_DIR" ]]; then
        log "ERROR" "Source directory does not exist: $SOURCE_DIR"
        log "INFO" "Please ensure KNIRVNEXUS has been built and the 'out' directory exists"
        return 1
    fi
    
    # Backup target before sync
    if [[ -d "$TARGET_DIR" ]]; then
        backup_directory "$TARGET_DIR" "nexus-portal-static"
    fi
    
    # Backup protected files before sync
    local protected_backup_dir=$(backup_protected_files)
    
    # Create target directory if it doesn't exist
    mkdir -p "$TARGET_DIR"
    
    # Sync all static files from source to target
    if [[ "$DRY_RUN" == "true" ]]; then
        log "INFO" "[DRY RUN] Would sync all static files from $SOURCE_DIR to $TARGET_DIR"
        log "INFO" "[DRY RUN] Protected files would be preserved: ${PROTECTED_FILES[*]}"
        if [[ -d "$SOURCE_DIR" ]]; then
            find "$SOURCE_DIR" -type f | while read -r source_file; do
                local rel_path="${source_file#$SOURCE_DIR/}"
                local target_file="$TARGET_DIR/$rel_path"
                log "INFO" "[DRY RUN] Would sync: $rel_path"
            done
        fi
    else
        # Copy all files from source to target
        if [[ -d "$SOURCE_DIR" ]]; then
            log "INFO" "Running rsync to clone static files..."
            rsync -av --delete "$SOURCE_DIR/" "$TARGET_DIR/"
            
            if [[ $? -eq 0 ]]; then
                log "SUCCESS" "Synced all static files from KNIRVNEXUS/out to GATEWAY nexus-portal"
                
                # Rename cloned files before restoring protected files
                log "INFO" "Renaming cloned files to avoid conflicts..."
                rename_cloned_files
                
                # Restore protected files after sync (overwrites any conflicting files from source)
                log "INFO" "Restoring protected files to preserve backend config and redirect logic..."
                restore_protected_files "$protected_backup_dir"
                log "SUCCESS" "Protected files preserved: ${PROTECTED_FILES[*]}"
            else
                log "ERROR" "rsync failed, restoring protected files anyway..."
                restore_protected_files "$protected_backup_dir"
                return 1
            fi
        fi
    fi
}

# =============================================================================
# USAGE AND ARGUMENT PARSING
# =============================================================================

show_usage() {
    cat << EOF
KNIRV Portal Version Synchronization Script
Syncs static site contents from KNIRVNEXUS/out to GATEWAY nexus-portal

Usage: $0 [OPTIONS]

OPTIONS:
    -n, --dry-run         Show what would be done without making changes
    -f, --force           Force sync even if target files are newer
    -v, --verbose         Enable verbose logging
    -h, --help            Show this help message

EXAMPLES:
    $0                           # Sync static site from KNIRVNEXUS to GATEWAY
    $0 -n                        # Dry run - show what would be synced
    $0 -f -v                     # Force sync with verbose output

EOF
}

parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -n|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -f|--force)
                FORCE=true
                shift
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
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
}

# =============================================================================
# MAIN EXECUTION
# =============================================================================

main() {
    parse_arguments "$@"
    setup_directories
    
    log "INFO" "Starting KNIRV Portal Version Synchronization"
    log "INFO" "Dry Run: $DRY_RUN, Force: $FORCE"
    
    local success=true
    
    sync_static_site || success=false
    
    if [[ "$success" == "true" ]]; then
        log "SUCCESS" "Static site synchronization completed successfully"
        log "INFO" "All static files copied from KNIRVNEXUS/out to the corporate nexus-portal"
        log "INFO" "Backend config and redirect logic preserved in target directory"
        exit 0
    else
        log "ERROR" "Static site synchronization completed with errors"
        exit 1
    fi
}

# Run main function with all arguments
main "$@"
