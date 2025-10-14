-#!/bin/bash

# KNIRV Portal Version Synchronization Script
# Synchronizes nexus-portal from KNIRVNEXUS to GATEWAY only
# Performs intelligent, idempotent updates while preserving intentional styling differences

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
PORTAL_TYPE="nexus"
FORCE=false
VERBOSE=false

# =============================================================================
# PORTAL LOCATIONS
# =============================================================================

# Nexus Portal locations - only sync from KNIRVNEXUS to GATEWAY
declare -A NEXUS_PORTALS=(
    ["knirvgateway"]="KNIRVGATEWAY/primary-website/public/nexus-portal"
    ["knirvnexus"]="KNIRVNEXUS"
)


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

get_package_version() {
    local package_file="$1"
    if [[ -f "$package_file" ]]; then
        jq -r '.version // "0.0.0"' "$package_file" 2>/dev/null || echo "0.0.0"
    else
        echo "0.0.0"
    fi
}

compare_versions() {
    local version1="$1"
    local version2="$2"
    
    # Convert versions to comparable format
    local v1=$(echo "$version1" | sed 's/[^0-9.]//g')
    local v2=$(echo "$version2" | sed 's/[^0-9.]//g')
    
    if [[ "$v1" == "$v2" ]]; then
        echo "equal"
    elif printf '%s\n%s\n' "$v1" "$v2" | sort -V | head -n1 | grep -q "^$v1$"; then
        echo "older"
    else
        echo "newer"
    fi
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

# =============================================================================
# VERSION DETECTION
# =============================================================================

detect_latest_nexus_portal() {
    # KNIRVNEXUS is always considered the authoritative source
    # We sync FROM KNIRVNEXUS TO other locations, not the other way around
    local authoritative_location="knirvnexus"
    local authoritative_path="$PROJECT_ROOT/${NEXUS_PORTALS[$authoritative_location]}"
    local package_file="$authoritative_path/package.json"

    log "INFO" "Using KNIRVNEXUS as authoritative source for nexus-portal..." >&2

    if [[ -f "$package_file" ]]; then
        local version=$(get_package_version "$package_file")
        local timestamp=$(get_file_timestamp "$package_file")

        log "INFO" "Found $authoritative_location: version $version, timestamp $timestamp" >&2
        log "SUCCESS" "Using authoritative nexus-portal: $authoritative_location (version $version)" >&2
        echo "$authoritative_location"
    else
        log "ERROR" "No package.json found for authoritative source at $package_file" >&2

        # Fallback to timestamp-based detection if KNIRVNEXUS is not available
        log "WARNING" "Falling back to timestamp-based detection..." >&2

        local latest_timestamp=0
        local latest_location=""

        for location in "${!NEXUS_PORTALS[@]}"; do
            local portal_path="$PROJECT_ROOT/${NEXUS_PORTALS[$location]}"
            local fallback_package_file="$portal_path/package.json"

            if [[ -f "$fallback_package_file" ]]; then
                local timestamp=$(get_file_timestamp "$fallback_package_file")
                log "INFO" "Found $location: timestamp $timestamp" >&2

                if [[ $timestamp -gt $latest_timestamp ]]; then
                    latest_timestamp="$timestamp"
                    latest_location="$location"
                fi
            fi
        done

        if [[ -n "$latest_location" ]]; then
            log "SUCCESS" "Fallback latest nexus-portal: $latest_location" >&2
            echo "$latest_location"
        else
            log "ERROR" "No valid nexus-portal found" >&2
            return 1
        fi
    fi
}


# =============================================================================
# SYNCHRONIZATION FUNCTIONS
# =============================================================================

sync_nexus_portal() {
    local source_location="$1"
    local target_location="$2"
    
    local source_path="$PROJECT_ROOT/${NEXUS_PORTALS[$source_location]}"
    local target_path="$PROJECT_ROOT/${NEXUS_PORTALS[$target_location]}"
    
    log "INFO" "Syncing nexus-portal from $source_location to $target_location"
    
    if [[ ! -d "$source_path" ]]; then
        log "ERROR" "Source path does not exist: $source_path"
        return 1
    fi
    
    # Backup target before sync
    if [[ -d "$target_path" ]]; then
        backup_directory "$target_path" "nexus-portal-${target_location}"
    fi
    
    # Create target directory if it doesn't exist
    mkdir -p "$target_path"
    
    # Sync based on portal type
    if [[ "$target_location" == "knirvnexus" ]]; then
        sync_to_knirvnexus "$source_path" "$target_path"
    else
        sync_to_gateway_portal "$source_path" "$target_path"
    fi
}

sync_to_knirvnexus() {
    local source_path="$1"
    local target_path="$2"
    
    log "INFO" "Syncing to KNIRVNEXUS (Next.js structure)"
    
    # Files to sync for KNIRVNEXUS - exclude styling files to preserve black theme
    local sync_patterns=(
        "src/components/auth/*"
        "src/components/dashboard/*"
        "src/components/ui/*"
        "src/hooks/*"
        "src/lib/*"
        "src/app/page.tsx"
        "src/app/layout.tsx"
    )
    
    for pattern in "${sync_patterns[@]}"; do
        local source_files=("$source_path"/$pattern)
        for source_file in "${source_files[@]}"; do
            if [[ -e "$source_file" ]]; then
                local rel_path="${source_file#$source_path/}"
                local target_file="$target_path/$rel_path"
                
                # Create target directory
                mkdir -p "$(dirname "$target_file")"
                
                # Check if we should sync this file
                if should_sync_file "$source_file" "$target_file"; then
                    if [[ "$DRY_RUN" == "true" ]]; then
                        log "INFO" "[DRY RUN] Would sync: $rel_path"
                    else
                        cp "$source_file" "$target_file"
                        log "SUCCESS" "Synced: $rel_path"
                    fi
                fi
            fi
        done
    done
    
    # Handle package.json specially for KNIRVNEXUS
    sync_package_json_to_nexus "$source_path/package.json" "$target_path/package.json"
}

sync_to_gateway_portal() {
    local source_path="$1"
    local target_path="$2"
    
    log "INFO" "Syncing to Gateway portal (Vite/React structure) - preserving purple theme"
    
    # Files to sync for Gateway portals - exclude styling files to preserve purple theme
    local sync_patterns=(
        "src/*"
        "package.json"
        "vite.config.ts"
        "tsconfig.json"
        "index.html"
        "dashboard.html"
        "landing.html"
    )
    
    for pattern in "${sync_patterns[@]}"; do
        if [[ "$pattern" == "src/*" ]]; then
            # Recursively sync src directory but exclude CSS files to preserve purple theme
            if [[ -d "$source_path/src" ]]; then
                if [[ "$DRY_RUN" == "true" ]]; then
                    log "INFO" "[DRY RUN] Would sync src directory (excluding CSS files)"
                else
                    rsync -av --exclude='node_modules' --exclude='dist' --exclude='*.css' "$source_path/src/" "$target_path/src/"
                    log "SUCCESS" "Synced src directory (CSS files excluded to preserve purple theme)"
                fi
            fi
        else
            local source_file="$source_path/$pattern"
            local target_file="$target_path/$pattern"
            
            if [[ -f "$source_file" ]] && should_sync_file "$source_file" "$target_file"; then
                mkdir -p "$(dirname "$target_file")"
                if [[ "$DRY_RUN" == "true" ]]; then
                    log "INFO" "[DRY RUN] Would sync: $pattern"
                else
                    cp "$source_file" "$target_file"
                    log "SUCCESS" "Synced: $pattern"
                fi
            fi
        fi
    done
}


should_sync_file() {
    local source_file="$1"
    local target_file="$2"
    
    # Always sync if target doesn't exist
    if [[ ! -f "$target_file" ]]; then
        return 0
    fi
    
    # Skip if force is not enabled and target is newer
    if [[ "$FORCE" == "false" ]]; then
        local source_time=$(get_file_timestamp "$source_file")
        local target_time=$(get_file_timestamp "$target_file")
        
        if [[ $target_time -gt $source_time ]]; then
            log "INFO" "Skipping $(basename "$source_file") - target is newer"
            return 1
        fi
    fi
    
    # Check if files are different
    local source_hash=$(get_file_hash "$source_file")
    local target_hash=$(get_file_hash "$target_file")
    
    if [[ "$source_hash" == "$target_hash" ]]; then
        log "INFO" "Skipping $(basename "$source_file") - files are identical"
        return 1
    fi
    
    return 0
}

sync_package_json_to_nexus() {
    local source_package="$1"
    local target_package="$2"
    
    if [[ ! -f "$source_package" ]]; then
        return 0
    fi
    
    log "INFO" "Syncing package.json dependencies to KNIRVNEXUS"
    
    # Extract relevant dependencies from source
    local react_deps=$(jq -r '.dependencies | to_entries[] | select(.key | startswith("@radix-ui") or . == "react" or . == "react-dom" or . == "lucide-react" or . == "clsx" or . == "class-variance-authority" or . == "tailwind-merge") | "\(.key): \(.value)"' "$source_package" 2>/dev/null || echo "")
    
    if [[ -n "$react_deps" && "$DRY_RUN" == "false" ]]; then
        log "INFO" "Would update React dependencies in KNIRVNEXUS package.json"
        # Note: In a real implementation, we'd merge dependencies intelligently
        # For now, we'll just log what would be updated
    fi
}

# =============================================================================
# MAIN EXECUTION FUNCTIONS
# =============================================================================

sync_all_nexus_portals() {
    local latest_location
    latest_location=$(detect_latest_nexus_portal 2>/dev/null)

    if [[ $? -ne 0 || -z "$latest_location" ]]; then
        log "ERROR" "Failed to detect latest nexus-portal"
        return 1
    fi

    log "INFO" "Syncing from latest nexus-portal: $latest_location"

    for target_location in "${!NEXUS_PORTALS[@]}"; do
        if [[ "$target_location" != "$latest_location" ]]; then
            sync_nexus_portal "$latest_location" "$target_location"
        fi
    done
}


show_usage() {
    cat << EOF
KNIRV Portal Version Synchronization Script
Syncs nexus-portal from KNIRVNEXUS to GATEWAY only, preserving intentional styling differences

Usage: $0 [OPTIONS]

OPTIONS:
    -t, --type TYPE        Portal type to sync: nexus (default: nexus)
    -n, --dry-run         Show what would be done without making changes
    -f, --force           Force sync even if target files are newer
    -v, --verbose         Enable verbose logging
    -h, --help            Show this help message

EXAMPLES:
    $0                           # Sync nexus-portal from KNIRVNEXUS to GATEWAY
    $0 -t nexus                  # Sync only nexus-portal (same as default)
    $0 -n                        # Dry run - show what would be synced
    $0 -f -v                     # Force sync with verbose output

EOF
}

parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -t|--type)
                PORTAL_TYPE="$2"
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
    
    log "INFO" "Starting KNIRV Portal Version Synchronization"
    log "INFO" "Type: $PORTAL_TYPE, Dry Run: $DRY_RUN, Force: $FORCE"
    
    local success=true
    
    case "$PORTAL_TYPE" in
        "nexus")
            sync_all_nexus_portals || success=false
            ;;
        *)
            log "ERROR" "Invalid portal type: $PORTAL_TYPE (only 'nexus' is supported)"
            show_usage
            exit 1
            ;;
    esac
    
    if [[ "$success" == "true" ]]; then
        log "SUCCESS" "Portal synchronization completed successfully"
        log "INFO" "Note: Styling files were excluded to preserve intentional theme differences"
        log "INFO" "KNIRVNEXUS maintains black theme, GATEWAY maintains purple theme"
        exit 0
    else
        log "ERROR" "Portal synchronization completed with errors"
        exit 1
    fi
}

# Run main function with all arguments
main "$@"

