#!/bin/bash

# KNIRV Network Fix Synchronization Script
# Synchronizes fixes between testnet and production environments bidirectionally
# Maintains idempotency and preserves newer implementations

set -euo pipefail

# =============================================================================
# CONFIGURATION
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SYNC_CONFIG="$SCRIPT_DIR/sync-config.yaml"
SYNC_STATE_DIR="$PROJECT_ROOT/.sync-state"
BACKUP_DIR="$PROJECT_ROOT/.sync-backups"
LOG_FILE="$SYNC_STATE_DIR/sync-$(date +%Y%m%d_%H%M%S).log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default options
DRY_RUN=false
DIRECTION="both"
SERVICES="all"
FORCE=false
VERBOSE=false

# =============================================================================
# UTILITY FUNCTIONS
# =============================================================================

log() {
    local level="$1"
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    echo "[$timestamp] [$level] $message" | tee -a "$LOG_FILE"
    
    case "$level" in
        "ERROR") echo -e "${RED}❌ $message${NC}" ;;
        "SUCCESS") echo -e "${GREEN}✅ $message${NC}" ;;
        "WARNING") echo -e "${YELLOW}⚠️  $message${NC}" ;;
        "INFO") echo -e "${BLUE}ℹ️  $message${NC}" ;;
    esac
}

setup_directories() {
    mkdir -p "$SYNC_STATE_DIR" "$BACKUP_DIR"
    touch "$LOG_FILE"
}

# =============================================================================
# ENVIRONMENT DETECTION AND MAPPING
# =============================================================================

get_file_hash() {
    local file="$1"
    if [[ -f "$file" ]]; then
        sha256sum "$file" | cut -d' ' -f1
    else
        echo "missing"
    fi
}

get_file_timestamp() {
    local file="$1"
    if [[ -f "$file" ]]; then
        stat -c %Y "$file"
    else
        echo "0"
    fi
}

get_git_commit_for_file() {
    local file="$1"
    if [[ -f "$file" ]]; then
        git log -1 --format="%H %ct" -- "$file" 2>/dev/null || echo "unknown 0"
    else
        echo "unknown 0"
    fi
}

# =============================================================================
# ENVIRONMENT MAPPING
# =============================================================================

declare -A TESTNET_TO_PROD_PATHS=(
    ["KNIRVTESTNET/config"]="deployment/production-config"
    ["KNIRVTESTNET/scripts"]="scripts"
    ["KNIRVTESTNET/data/knirvgateway/nexus-portal/.env.production"]="KNIRVGATEWAY/nexus-portal/.env.production"
)

declare -A PROD_TO_TESTNET_PATHS=(
    ["deployment/production-config"]="KNIRVTESTNET/config"
    ["scripts"]="KNIRVTESTNET/scripts"
    ["KNIRVGATEWAY/nexus-portal/.env.production"]="KNIRVTESTNET/data/knirvgateway/nexus-portal/.env.production"
)

# Environment-specific transformations
transform_for_environment() {
    local file="$1"
    local source_env="$2"
    local target_env="$3"
    local temp_file=$(mktemp)
    
    cp "$file" "$temp_file"
    
    if [[ "$source_env" == "testnet" && "$target_env" == "production" ]]; then
        # Testnet → Production transformations
        sed -i 's/testnet-1/mainnet-1/g' "$temp_file"
        sed -i 's/localhost/api.knirv.com/g' "$temp_file"
        sed -i 's/TESTNET_MODE=true/TESTNET_MODE=false/g' "$temp_file"
        sed -i 's/debug_mode: true/debug_mode: false/g' "$temp_file"
        sed -i 's/validators: 1/validators: 3/g' "$temp_file"
        sed -i 's/simplified_consensus: true/simplified_consensus: false/g' "$temp_file"
    elif [[ "$source_env" == "production" && "$target_env" == "testnet" ]]; then
        # Production → Testnet transformations
        sed -i 's/mainnet-1/testnet-1/g' "$temp_file"
        sed -i 's/api\.knirv\.com/localhost/g' "$temp_file"
        sed -i 's/TESTNET_MODE=false/TESTNET_MODE=true/g' "$temp_file"
        sed -i 's/debug_mode: false/debug_mode: true/g' "$temp_file"
        sed -i 's/validators: 3/validators: 1/g' "$temp_file"
        sed -i 's/simplified_consensus: false/simplified_consensus: true/g' "$temp_file"
    fi
    
    echo "$temp_file"
}

# =============================================================================
# FIX DETECTION ENGINE
# =============================================================================

detect_fixes() {
    local direction="$1"
    local fixes_found=()

    # Redirect log output to stderr to avoid capturing it
    log "INFO" "Detecting fixes for direction: $direction" >&2

    case "$direction" in
        "testnet-to-prod")
            detect_testnet_fixes fixes_found
            ;;
        "prod-to-testnet")
            detect_production_fixes fixes_found
            ;;
        "both")
            detect_testnet_fixes fixes_found
            detect_production_fixes fixes_found
            ;;
    esac

    # Only output if we have fixes
    if [[ ${#fixes_found[@]} -gt 0 ]]; then
        printf '%s\n' "${fixes_found[@]}"
    fi
}

detect_testnet_fixes() {
    local -n fixes_ref=$1
    
    # Check for fixes mentioned in final-test-fixes.md
    if [[ -f "$PROJECT_ROOT/KNIRVORACLE/final-test-fixes.md" ]]; then
        log "INFO" "Found testnet fixes documentation" >&2
        fixes_ref+=("badge-attachment-fix:KNIRVORACLE/chromemDB_manager.go")
        fixes_ref+=("tunnel-registry-fix:KNIRVORACLE/tunnel_registry.go")
        fixes_ref+=("python-sdk-fix:KNIRVSDK/py/")
        fixes_ref+=("cortex-mock-fix:KNIRVCORTEX/")
        fixes_ref+=("gateway-build-fix:KNIRVGATEWAY/package.json")
    fi
    
    # Check for newer files in testnet
    for testnet_path in "${!TESTNET_TO_PROD_PATHS[@]}"; do
        local prod_path="${TESTNET_TO_PROD_PATHS[$testnet_path]}"

        if [[ -d "$PROJECT_ROOT/$testnet_path" ]]; then
            while IFS= read -r -d '' file; do
                if [[ -f "$file" ]]; then
                    local rel_file="${file#$PROJECT_ROOT/$testnet_path/}"
                    local prod_file="$PROJECT_ROOT/$prod_path/$rel_file"

                    local testnet_time=$(get_file_timestamp "$file")
                    local prod_time=$(get_file_timestamp "$prod_file")

                    if [[ $testnet_time -gt $prod_time ]]; then
                        fixes_ref+=("file-update:$testnet_path/$rel_file")
                    fi
                fi
            done < <(find "$PROJECT_ROOT/$testnet_path" -type f -print0 2>/dev/null || true)
        fi
    done
}

detect_production_fixes() {
    local -n fixes_ref=$1

    # Check for newer files in production
    for prod_path in "${!PROD_TO_TESTNET_PATHS[@]}"; do
        local testnet_path="${PROD_TO_TESTNET_PATHS[$prod_path]}"

        if [[ -d "$PROJECT_ROOT/$prod_path" ]]; then
            while IFS= read -r -d '' file; do
                if [[ -f "$file" ]]; then
                    local rel_file="${file#$PROJECT_ROOT/$prod_path/}"
                    local testnet_file="$PROJECT_ROOT/$testnet_path/$rel_file"

                    local prod_time=$(get_file_timestamp "$file")
                    local testnet_time=$(get_file_timestamp "$testnet_file")

                    if [[ $prod_time -gt $testnet_time ]]; then
                        fixes_ref+=("file-update:$prod_path/$rel_file")
                    fi
                fi
            done < <(find "$PROJECT_ROOT/$prod_path" -type f -print0 2>/dev/null || true)
        fi
    done
}

# =============================================================================
# SYNCHRONIZATION ENGINE
# =============================================================================

apply_fix() {
    local fix="$1"
    local direction="$2"

    local fix_type="${fix%%:*}"
    local fix_path="${fix#*:}"

    log "INFO" "Applying fix: $fix_type for $fix_path (direction: $direction)"

    case "$fix_type" in
        "badge-attachment-fix")
            apply_badge_attachment_fix "$fix_path" "$direction" || return 1
            ;;
        "tunnel-registry-fix")
            apply_tunnel_registry_fix "$fix_path" "$direction" || return 1
            ;;
        "python-sdk-fix")
            apply_python_sdk_fix "$fix_path" "$direction" || return 1
            ;;
        "cortex-mock-fix")
            apply_cortex_mock_fix "$fix_path" "$direction" || return 1
            ;;
        "gateway-build-fix")
            apply_gateway_build_fix "$fix_path" "$direction" || return 1
            ;;
        "file-update")
            apply_file_update "$fix_path" "$direction" || return 1
            ;;
        *)
            log "WARNING" "Unknown fix type: $fix_type"
            return 1
            ;;
    esac
    return 0
}

apply_badge_attachment_fix() {
    local file_path="$1"
    local direction="$2"

    # This fix improves ChromeDB query logic for badge attachments
    log "INFO" "Applying badge attachment fix to $file_path (direction: $direction)"

    if [[ "$direction" == "testnet-to-prod" || "$direction" == "both" ]]; then
        # Copy the improved ChromeDB query logic from testnet to production
        local source="$PROJECT_ROOT/$file_path"
        local target="$PROJECT_ROOT/KNIRVORACLE/chromemDB_manager.go"

        log "INFO" "Source: $source, Target: $target"

        if [[ -f "$source" ]]; then
            if ! backup_file "$target"; then
                log "ERROR" "Failed to backup target file for badge attachment fix"
                return 1
            fi
            if [[ "$DRY_RUN" == "true" ]]; then
                log "INFO" "[DRY RUN] Would apply badge attachment fix to $target"
            else
                if cp "$source" "$target" 2>/dev/null; then
                    log "SUCCESS" "Badge attachment fix applied to production"
                else
                    log "ERROR" "Failed to copy badge attachment fix to production"
                    return 1
                fi
            fi
        else
            log "WARNING" "Source file not found for badge attachment fix: $source"
            return 1
        fi
    else
        log "INFO" "Skipping badge attachment fix for direction: $direction"
    fi
    return 0
}

apply_tunnel_registry_fix() {
    local file_path="$1"
    local direction="$2"

    log "INFO" "Applying tunnel registry fix to $file_path (direction: $direction)"

    if [[ "$direction" == "testnet-to-prod" || "$direction" == "both" ]]; then
        local source="$PROJECT_ROOT/$file_path"
        local target="$PROJECT_ROOT/KNIRVORACLE/tunnel_registry.go"

        log "INFO" "Source: $source, Target: $target"

        if [[ -f "$source" ]]; then
            # Backup target file only if it exists (it might be a new file)
            if [[ -f "$target" ]]; then
                if ! backup_file "$target"; then
                    log "ERROR" "Failed to backup target file for tunnel registry fix"
                    return 1
                fi
            else
                log "INFO" "Target file doesn't exist - creating new tunnel registry Go wrapper"
            fi

            if [[ "$DRY_RUN" == "true" ]]; then
                if [[ -f "$target" ]]; then
                    log "INFO" "[DRY RUN] Would update tunnel registry fix to $target"
                else
                    log "INFO" "[DRY RUN] Would create new tunnel registry Go wrapper at $target"
                fi
            else
                if cp "$source" "$target" 2>/dev/null; then
                    if [[ -f "$target" ]]; then
                        log "SUCCESS" "Tunnel registry fix applied to production"
                    else
                        log "SUCCESS" "New tunnel registry Go wrapper created"
                    fi
                else
                    log "ERROR" "Failed to copy tunnel registry fix to production"
                    return 1
                fi
            fi
        else
            log "WARNING" "Source file not found for tunnel registry fix: $source"
            return 1
        fi
    else
        log "INFO" "Skipping tunnel registry fix for direction: $direction"
    fi
    return 0
}

apply_python_sdk_fix() {
    local file_path="$1"
    local direction="$2"

    log "INFO" "Applying Python SDK fix to $file_path"

    if [[ "$direction" == "testnet-to-prod" ]]; then
        local source="$PROJECT_ROOT/$file_path"
        local target="$PROJECT_ROOT/KNIRVSDK/py/"

        if [[ -d "$source" ]]; then
            backup_file "$target"
            if [[ "$DRY_RUN" == "true" ]]; then
                log "INFO" "[DRY RUN] Would sync Python SDK modules to $target"
            else
                rsync -av "$source/" "$target/"
                log "SUCCESS" "Python SDK fix applied to production"
            fi
        fi
    fi
}

apply_cortex_mock_fix() {
    local file_path="$1"
    local direction="$2"

    log "INFO" "Applying CORTEX mock fix to $file_path"

    if [[ "$direction" == "testnet-to-prod" ]]; then
        local source="$PROJECT_ROOT/$file_path"
        local target="$PROJECT_ROOT/KNIRVCORTEX/"

        if [[ -d "$source" ]]; then
            backup_file "$target"
            if [[ "$DRY_RUN" == "true" ]]; then
                log "INFO" "[DRY RUN] Would sync CORTEX mock implementations to $target"
            else
                rsync -av "$source/" "$target/"
                log "SUCCESS" "CORTEX mock fix applied to production"
            fi
        fi
    fi
}

apply_gateway_build_fix() {
    local file_path="$1"
    local direction="$2"

    log "INFO" "Applying Gateway build fix to $file_path"

    if [[ "$direction" == "testnet-to-prod" ]]; then
        local source="$PROJECT_ROOT/$file_path"
        local target="$PROJECT_ROOT/KNIRVGATEWAY/package.json"

        if [[ -f "$source" ]]; then
            backup_file "$target"
            if [[ "$DRY_RUN" == "true" ]]; then
                log "INFO" "[DRY RUN] Would apply Gateway build fix to $target"
            else
                cp "$source" "$target"
                log "SUCCESS" "Gateway build fix applied to production"
            fi
        fi
    fi
}

apply_file_update() {
    local file_path="$1"
    local direction="$2"
    
    local source_file=""
    local target_file=""
    local source_env=""
    local target_env=""
    
    if [[ "$direction" == "testnet-to-prod" ]]; then
        source_file="$PROJECT_ROOT/$file_path"
        # Map testnet path to production path
        for testnet_path in "${!TESTNET_TO_PROD_PATHS[@]}"; do
            if [[ "$file_path" == "$testnet_path"* ]]; then
                local rel_path="${file_path#$testnet_path/}"
                target_file="$PROJECT_ROOT/${TESTNET_TO_PROD_PATHS[$testnet_path]}/$rel_path"
                break
            fi
        done
        source_env="testnet"
        target_env="production"
    elif [[ "$direction" == "prod-to-testnet" ]]; then
        source_file="$PROJECT_ROOT/$file_path"
        # Map production path to testnet path
        for prod_path in "${!PROD_TO_TESTNET_PATHS[@]}"; do
            if [[ "$file_path" == "$prod_path"* ]]; then
                local rel_path="${file_path#$prod_path/}"
                target_file="$PROJECT_ROOT/${PROD_TO_TESTNET_PATHS[$prod_path]}/$rel_path"
                break
            fi
        done
        source_env="production"
        target_env="testnet"
    fi
    
    if [[ -n "$target_file" && -f "$source_file" ]]; then
        backup_file "$target_file"
        
        # Transform file for target environment
        local transformed_file=$(transform_for_environment "$source_file" "$source_env" "$target_env")
        
        # Ensure target directory exists
        mkdir -p "$(dirname "$target_file")"
        
        if [[ "$DRY_RUN" == "true" ]]; then
            log "INFO" "[DRY RUN] Would copy $source_file to $target_file"
        else
            cp "$transformed_file" "$target_file"
            log "SUCCESS" "File updated: $target_file"
        fi
        
        # Clean up temporary file
        rm -f "$transformed_file"
    fi
}

backup_file() {
    local file="$1"
    if [[ -f "$file" ]]; then
        local backup_path="$BACKUP_DIR/$(date +%Y%m%d_%H%M%S)_$(basename "$file")"
        if cp "$file" "$backup_path" 2>/dev/null; then
            log "INFO" "Backed up $file to $backup_path"
            return 0
        else
            log "ERROR" "Failed to backup $file to $backup_path"
            return 1
        fi
    else
        log "INFO" "No existing file to backup: $file (this is normal for new files)"
    fi
    return 0
}

# =============================================================================
# MAIN EXECUTION
# =============================================================================

show_usage() {
    cat << EOF
KNIRV Network Fix Synchronization Script

Usage: $0 [OPTIONS]

OPTIONS:
    -d, --direction DIRECTION   Sync direction: testnet-to-prod, prod-to-testnet, both (default: both)
    -s, --services SERVICES     Services to sync: all, knirvoracle, knirvchain, etc. (default: all)
    -n, --dry-run              Show what would be done without making changes
    -f, --force                Force sync even if target is newer
    -v, --verbose              Enable verbose logging
    -h, --help                 Show this help message

EXAMPLES:
    $0                                    # Sync all fixes in both directions
    $0 -d testnet-to-prod                # Sync testnet fixes to production
    $0 -d prod-to-testnet -n             # Dry run: show what would be synced from prod to testnet
    $0 -s knirvoracle -v                   # Sync only KNIRVORACLE fixes with verbose output

EOF
}

parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -d|--direction)
                DIRECTION="$2"
                shift 2
                ;;
            -s|--services)
                SERVICES="$2"
                shift 2
                ;;
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

main() {
    parse_arguments "$@"
    setup_directories
    
    log "INFO" "Starting KNIRV Network Fix Synchronization"
    log "INFO" "Direction: $DIRECTION, Services: $SERVICES, Dry Run: $DRY_RUN"
    
    # Detect fixes
    local fixes_output
    fixes_output=$(detect_fixes "$DIRECTION")

    if [[ -z "$fixes_output" ]]; then
        log "INFO" "No fixes detected for synchronization"
        exit 0
    fi

    # Convert output to array
    local fixes=()
    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            fixes+=("$line")
        fi
    done <<< "$fixes_output"

    log "INFO" "Found ${#fixes[@]} fixes to synchronize"

    # Apply fixes
    local applied=0
    local failed=0

    for fix in "${fixes[@]}"; do
        if [[ -n "$fix" ]]; then
            # Temporarily disable strict error handling for individual fix application
            set +e
            if apply_fix "$fix" "$DIRECTION"; then
                ((applied++))
            else
                ((failed++))
                log "ERROR" "Failed to apply fix: $fix"
            fi
            # Re-enable strict error handling
            set -e
        fi
    done
    
    log "SUCCESS" "Synchronization completed: $applied applied, $failed failed"

    # Post-sync hook: Restore Node.js dependencies for KNIRVORACLE
    if [[ ! "$DRY_RUN" == "true" ]]; then
        log "INFO" "Running post-sync hook: Restoring KNIRVORACLE Node.js dependencies"
        if [[ -f "$PROJECT_ROOT/KNIRVORACLE/scripts/restore-nodejs-deps.sh" ]]; then
            cd "$PROJECT_ROOT/KNIRVORACLE/scripts"
            if ./restore-nodejs-deps.sh; then
                log "SUCCESS" "Node.js dependencies restored successfully"
            else
                log "WARNING" "Node.js dependency restoration failed - manual intervention may be required"
            fi
        else
            log "WARNING" "Node.js dependency restoration script not found"
        fi
    fi

    if [[ $failed -gt 0 ]]; then
        exit 1
    fi
}

# Run main function with all arguments
main "$@"
