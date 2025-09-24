#!/bin/bash

# KNIRVCONTROLLER to KNIRVCONTROLLER_public Sync Script
# Safely copies changed files with backup/restore functionality
# Idempotent - can be run multiple times without issues

set -euo pipefail

# Configuration
SOURCE_DIR="KNIRVCONTROLLER"
TARGET_DIR="../KNIRVCONTROLLER_public"
BACKUP_DIR="./sync_backups"
LOG_FILE="./sync_controller.log"
DRY_RUN=false
FORCE=false
VERBOSE=false
SHOW_PROGRESS=true
RESTORE=false
BACKUP_ONLY=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Progress tracking variables
TOTAL_FILES=0
PROCESSED_FILES=0
CURRENT_PHASE=""

# Progress indicator function
show_progress() {
    if [[ "$SHOW_PROGRESS" == true && "$TOTAL_FILES" -gt 0 ]]; then
        local percentage=$((PROCESSED_FILES * 100 / TOTAL_FILES))
        local bar_length=50
        local filled=$((percentage * bar_length / 100))
        local empty=$((bar_length - filled))
        
        printf "\r${CYAN}[%s]${NC} [${GREEN}%s${NC}${BLUE}%s${NC}] %d%% (%d/%d) - %s" \
            "$CURRENT_PHASE" \
            "$(printf '%*s' "$filled" | tr ' ' '=')" \
            "$(printf '%*s' "$empty" | tr ' ' '-')" \
            "$percentage" \
            "$PROCESSED_FILES" \
            "$TOTAL_FILES" \
            "$1"
    fi
}

# Clear progress line
clear_progress() {
    if [[ "$SHOW_PROGRESS" == true ]]; then
        printf "\r%*s\r" 100 ""
    fi
}

# Logging function
log() {
    local level=$1
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    # Clear progress line before logging
    clear_progress
    
    echo -e "${timestamp} [${level}] ${message}" | tee -a "$LOG_FILE"
    
    if [[ "$level" == "ERROR" ]]; then
        echo -e "${RED}${message}${NC}" >&2
    elif [[ "$level" == "WARN" ]]; then
        echo -e "${YELLOW}${message}${NC}" >&2
    elif [[ "$level" == "INFO" ]]; then
        echo -e "${GREEN}${message}${NC}"
    elif [[ "$level" == "DEBUG" && "$VERBOSE" == true ]]; then
        echo -e "${BLUE}${message}${NC}"
    elif [[ "$level" == "PROGRESS" && "$SHOW_PROGRESS" == true ]]; then
        echo -e "${PURPLE}${message}${NC}"
    fi
}

# Usage information
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Safely syncs changed files from $SOURCE_DIR to $TARGET_DIR

OPTIONS:
    -d, --dry-run    Show what would be copied without making changes
    -f, --force      Force sync even if backups exist
    -v, --verbose    Enable verbose output
    -r, --restore    Restore from the latest backup
    -b, --backup     Create backup only (no sync)
    -q, --quiet      Disable progress indicators
    -h, --help       Show this help message

FEATURES:
    • Only copies changed files (checksum comparison)
    • Creates backups before making changes
    • Idempotent - safe to run multiple times
    • Detailed logging and error handling
    • Progress indicators for long operations
    • Dry-run mode for testing

EXAMPLES:
    $0 -d          # Dry run - see what would be copied
    $0 -v          # Verbose sync
    $0 -r          # Restore from backup
    $0 -q          # Quiet mode (no progress bars)
    $0             # Normal sync operation
EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -d|--dry-run)
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
        -r|--restore)
            RESTORE=true
            shift
            ;;
        -b|--backup)
            BACKUP_ONLY=true
            shift
            ;;
        -q|--quiet)
            SHOW_PROGRESS=false
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            log "ERROR" "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Validate directories exist
validate_directories() {
    if [[ ! -d "$SOURCE_DIR" ]]; then
        log "ERROR" "Source directory '$SOURCE_DIR' does not exist"
        exit 1
    fi
    
    if [[ ! -d "$TARGET_DIR" ]]; then
        log "ERROR" "Target directory '$TARGET_DIR' does not exist"
        exit 1
    fi
    
    log "INFO" "Source: $(realpath "$SOURCE_DIR")"
    log "INFO" "Target: $(realpath "$TARGET_DIR")"
}

# Create backup directory
setup_backup_dir() {
    local timestamp=$(date '+%Y%m%d_%H%M%S')
    local current_backup="$BACKUP_DIR/backup_$timestamp"
    
    mkdir -p "$BACKUP_DIR"
    mkdir -p "$current_backup"
    
    echo "$current_backup"
}

# Calculate file checksum
get_checksum() {
    local file="$1"
    if [[ -f "$file" ]]; then
        sha256sum "$file" | cut -d' ' -f1
    else
        echo "FILE_NOT_EXIST"
    fi
}

# Compare files and return true if different
files_differ() {
    local source_file="$1"
    local target_file="$2"
    
    if [[ ! -f "$source_file" ]]; then
        return 1  # Source file doesn't exist, skip
    fi
    
    if [[ ! -f "$target_file" ]]; then
        return 0  # Target file doesn't exist, needs copying
    fi
    
    local source_checksum=$(get_checksum "$source_file")
    local target_checksum=$(get_checksum "$target_file")
    
    [[ "$source_checksum" != "$target_checksum" ]]
}

# Count total files for progress tracking
count_total_files() {
    find "$SOURCE_DIR" -type f | wc -l
}

# Find changed files with progress tracking
find_changed_files() {
    local changed_files=()
    local total_files=$(count_total_files)
    local current_file=0
    
    TOTAL_FILES=$total_files
    PROCESSED_FILES=0
    CURRENT_PHASE="SCANNING"
    
    if [[ "$DRY_RUN" != true ]]; then
        log "PROGRESS" "Scanning $total_files files for changes..."
    fi
    
    # Use find to get all files in source directory
    while IFS= read -r -d '' source_file; do
        current_file=$((current_file + 1))
        PROCESSED_FILES=$current_file
        
        # Clean the file path - remove any carriage returns or special characters
        source_file=$(echo "$source_file" | tr -d '\r' | tr -d '\n')
        
        local relative_path="${source_file#$SOURCE_DIR/}"
        local target_file="$TARGET_DIR/$relative_path"
        
        # Skip if relative path is empty or contains only special characters
        if [[ -z "$relative_path" || "$relative_path" =~ ^[[:space:]]*$ ]]; then
            continue
        fi
        
        if [[ "$DRY_RUN" != true ]]; then
            show_progress "Checking: ${relative_path:0:40}"
        fi
        
        if files_differ "$source_file" "$target_file"; then
            changed_files+=("$relative_path")
            
            if [[ "$VERBOSE" == true ]]; then
                local source_checksum=$(get_checksum "$source_file")
                local target_checksum=$(get_checksum "$target_file")
                
                if [[ ! -f "$target_file" ]]; then
                    log "DEBUG" "NEW FILE: $relative_path"
                else
                    log "DEBUG" "CHANGED: $relative_path (source:${source_checksum:0:8} target:${target_checksum:0:8})"
                fi
            fi
        fi
    done < <(find "$SOURCE_DIR" -type f -print0)
    
    if [[ "$DRY_RUN" != true ]]; then
        clear_progress
    fi
    printf '%s\n' "${changed_files[@]}"
}

# Create backup of target files with progress tracking
create_backup() {
    local backup_dir="$1"
    shift
    local changed_files=("$@")
    local total_files=${#changed_files[@]}
    local current_file=0
    
    TOTAL_FILES=$total_files
    PROCESSED_FILES=0
    CURRENT_PHASE="BACKUP"
    
    if [[ $total_files -gt 0 && "$DRY_RUN" != true ]]; then
        log "PROGRESS" "Creating backup of $total_files files..."
    fi
    
    for file_path in "${changed_files[@]}"; do
        current_file=$((current_file + 1))
        PROCESSED_FILES=$current_file
        
        local target_file="$TARGET_DIR/$file_path"
        
        if [[ -f "$target_file" ]]; then
            local backup_file="$backup_dir/$file_path"
            local backup_dir_path=$(dirname "$backup_file")
            
            if [[ "$DRY_RUN" != true ]]; then
                show_progress "Backing up: ${file_path:0:40}"
            fi
            
            mkdir -p "$backup_dir_path"
            cp "$target_file" "$backup_file"
            
            if [[ "$VERBOSE" == true ]]; then
                log "DEBUG" "Backed up: $file_path"
            fi
        fi
    done
    
    if [[ "$DRY_RUN" != true ]]; then
        clear_progress
    fi
    
    # Save backup metadata
    echo "Backup created: $(date)" > "$backup_dir/backup_metadata.txt"
    echo "Source: $SOURCE_DIR" >> "$backup_dir/backup_metadata.txt"
    echo "Target: $TARGET_DIR" >> "$backup_dir/backup_metadata.txt"
    printf '%s\n' "${changed_files[@]}" > "$backup_dir/changed_files.txt"
}

# Restore from backup with progress tracking
restore_backup() {
    local latest_backup=$(ls -dt "$BACKUP_DIR"/backup_* 2>/dev/null | head -1)
    
    if [[ -z "$latest_backup" ]]; then
        log "ERROR" "No backups found in $BACKUP_DIR"
        exit 1
    fi
    
    log "INFO" "Restoring from backup: $latest_backup"
    
    if [[ ! -f "$latest_backup/changed_files.txt" ]]; then
        log "ERROR" "Backup metadata missing, cannot restore"
        exit 1
    fi
    
    # Read changed files from backup
    mapfile -t changed_files < "$latest_backup/changed_files.txt"
    local total_files=${#changed_files[@]}
    local current_file=0
    
    TOTAL_FILES=$total_files
    PROCESSED_FILES=0
    CURRENT_PHASE="RESTORE"
    
    if [[ $total_files -gt 0 && "$DRY_RUN" != true ]]; then
        log "PROGRESS" "Restoring $total_files files from backup..."
    fi
    
    for file_path in "${changed_files[@]}"; do
        current_file=$((current_file + 1))
        PROCESSED_FILES=$current_file
        
        local backup_file="$latest_backup/$file_path"
        local target_file="$TARGET_DIR/$file_path"
        
        if [[ "$DRY_RUN" != true ]]; then
            show_progress "Restoring: ${file_path:0:40}"
        fi
        
        if [[ -f "$backup_file" ]]; then
            if [[ "$DRY_RUN" == true ]]; then
                log "INFO" "[DRY RUN] Would restore: $file_path"
            else
                local target_dir=$(dirname "$target_file")
                mkdir -p "$target_dir"
                cp "$backup_file" "$target_file"
                if [[ "$VERBOSE" == true ]]; then
                    log "DEBUG" "Restored: $file_path"
                fi
            fi
        fi
    done
    
    if [[ "$DRY_RUN" != true ]]; then
        if [[ "$DRY_RUN" != true ]]; then
            clear_progress
        fi
    fi
    log "INFO" "Restore completed successfully"
}

# Copy changed files with progress tracking
copy_changed_files() {
    local changed_files=("$@")
    local total_files=${#changed_files[@]}
    local current_file=0
    
    TOTAL_FILES=$total_files
    PROCESSED_FILES=0
    CURRENT_PHASE="COPYING"
    
    if [[ $total_files -gt 0 && "$DRY_RUN" != true ]]; then
        log "PROGRESS" "Copying $total_files changed files..."
    fi
    
    for file_path in "${changed_files[@]}"; do
        current_file=$((current_file + 1))
        PROCESSED_FILES=$current_file
        
        # Clean the file path - remove any carriage returns or special characters
        file_path=$(echo "$file_path" | tr -d '\r' | tr -d '\n')
        
        # Skip if file path is empty or contains only special characters
        if [[ -z "$file_path" || "$file_path" =~ ^[[:space:]]*$ ]]; then
            continue
        fi
        
        local source_file="$SOURCE_DIR/$file_path"
        local target_file="$TARGET_DIR/$file_path"
        local target_dir=$(dirname "$target_file")
        
        # Validate that source file exists and is readable
        if [[ ! -f "$source_file" || ! -r "$source_file" ]]; then
            log "WARN" "Source file not found or not readable: $file_path"
            continue
        fi
        
        if [[ "$DRY_RUN" != true ]]; then
            show_progress "Copying: ${file_path:0:40}"
        fi
        
        if [[ "$DRY_RUN" == true ]]; then
            log "INFO" "[DRY RUN] Would copy: $file_path"
        else
            mkdir -p "$target_dir"
            cp "$source_file" "$target_file"
            if [[ "$VERBOSE" == true ]]; then
                log "DEBUG" "Copied: $file_path"
            fi
        fi
    done
    
    clear_progress
}

# Main sync function
sync_files() {
    log "INFO" "Starting sync operation"
    
    # Find changed files
    mapfile -t changed_files < <(find_changed_files)
    
    if [[ ${#changed_files[@]} -eq 0 ]]; then
        log "INFO" "No changes detected - directories are already in sync"
        return 0
    fi
    
    log "INFO" "Found ${#changed_files[@]} file(s) to sync"
    
    # Create backup
    local backup_dir
    if [[ "$DRY_RUN" != true && "$BACKUP_ONLY" != true ]]; then
        backup_dir=$(setup_backup_dir)
        create_backup "$backup_dir" "${changed_files[@]}"
    fi
    
    # Copy files
    if [[ "$BACKUP_ONLY" != true ]]; then
        copy_changed_files "${changed_files[@]}"
    fi
    
    if [[ "$DRY_RUN" == true ]]; then
        log "INFO" "Dry run completed - no changes made"
    elif [[ "$BACKUP_ONLY" == true ]]; then
        log "INFO" "Backup completed - no files copied"
    else
        log "INFO" "Sync completed successfully"
        log "INFO" "Backup available at: $backup_dir"
    fi
}

# Main execution
main() {
    log "INFO" "KNIRVCONTROLLER to KNIRVCONTROLLER_public Sync Script"
    log "INFO" "==================================================="
    
    validate_directories
    
    if [[ "${RESTORE:-false}" == true ]]; then
        restore_backup
    elif [[ "${BACKUP_ONLY:-false}" == true ]]; then
        mapfile -t changed_files < <(find_changed_files)
        if [[ ${#changed_files[@]} -gt 0 ]]; then
            backup_dir=$(setup_backup_dir)
            create_backup "$backup_dir" "${changed_files[@]}"
        else
            log "INFO" "No changes detected - backup not needed"
        fi
    else
        sync_files
    fi
    
    log "INFO" "Operation completed"
}

# Trap signals for clean exit
trap 'log "ERROR" "Script interrupted"; exit 1' INT TERM

# Run main function
main "$@"