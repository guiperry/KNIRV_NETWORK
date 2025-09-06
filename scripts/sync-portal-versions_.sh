#!/bin/bash

# KNIRV Portal Version Synchronization Script
# Synchronizes nexus-portal and graphchain-explorer implementations across all locations
# Performs intelligent, idempotent updates without breaking target implementations

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
PORTAL_TYPE="both"
FORCE=false
VERBOSE=false

# =============================================================================
# PORTAL LOCATIONS
# =============================================================================

# Nexus Portal locations (migrated to KNIRVTESTNET only)
declare -A NEXUS_PORTALS=(
    ["knirvtestnet"]="KNIRVTESTNET/nexus-portal"
    ["knirvnexus"]="KNIRVNEXUS"
)

# GraphChain Explorer locations (migrated to KNIRVTESTNET only)
declare -A GRAPHCHAIN_EXPLORERS=(
    ["knirvtestnet"]="KNIRVTESTNET/graphchain-explorer"
)

# GraphChain CLI binary locations
declare -A GRAPHCHAIN_CLI_BINARIES=(
    ["knirvgraph"]="KNIRVGRAPH/bin"
    ["knirvshell"]="KNIRVCLI/bin"
)

# KNIRVGRAPH build binaries locations
declare -A KNIRVGRAPH_BINARIES=(
    ["knirvgraph"]="KNIRVGRAPH/build"
    ["knirvana-rust"]="KNIRVANA/rust-client/bin"
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

detect_latest_graphchain_explorer() {
    # KNIRVGATEWAY is always considered the authoritative source for GraphChain Explorer
    # We sync FROM KNIRVGATEWAY TO other locations, not the other way around
    local authoritative_location="knirvgateway"
    local authoritative_path="$PROJECT_ROOT/${GRAPHCHAIN_EXPLORERS[$authoritative_location]}"

    log "INFO" "Using KNIRVGATEWAY as authoritative source for graphchain-explorer..." >&2

    if [[ -d "$authoritative_path" ]]; then
        # Check for key files to verify it's a valid GraphChain Explorer
        local key_files=("index.html" "js/graphchain-core.js" "css/graphchain.css")
        local file_count=0

        for file in "${key_files[@]}"; do
            local file_path="$authoritative_path/$file"
            if [[ -f "$file_path" ]]; then
                ((file_count++))
            fi
        done

        if [[ $file_count -gt 0 ]]; then
            log "INFO" "Found $authoritative_location: $file_count key files present" >&2
            log "SUCCESS" "Using authoritative graphchain-explorer: $authoritative_location" >&2
            echo "$authoritative_location"
        else
            log "WARNING" "KNIRVGATEWAY GraphChain Explorer appears incomplete, falling back..." >&2
        fi
    else
        log "WARNING" "KNIRVGATEWAY GraphChain Explorer not found, falling back..." >&2
    fi

    # Fallback to timestamp-based detection if KNIRVGATEWAY is not available
    if [[ ! -d "$authoritative_path" ]] || [[ $file_count -eq 0 ]]; then
        log "WARNING" "Falling back to timestamp-based detection..." >&2

        local latest_timestamp=0
        local latest_location=""

        for location in "${!GRAPHCHAIN_EXPLORERS[@]}"; do
            local explorer_path="$PROJECT_ROOT/${GRAPHCHAIN_EXPLORERS[$location]}"

            if [[ -d "$explorer_path" ]]; then
                # Check for key files to determine "freshness"
                local key_files=("index.html" "js/graphchain-core.js" "css/graphchain.css")
                local total_timestamp=0
                local fallback_file_count=0

                for file in "${key_files[@]}"; do
                    local file_path="$explorer_path/$file"
                    if [[ -f "$file_path" ]]; then
                        local timestamp=$(get_file_timestamp "$file_path")
                        total_timestamp=$((total_timestamp + timestamp))
                        ((fallback_file_count++))
                    fi
                done

                if [[ $fallback_file_count -gt 0 ]]; then
                    local avg_timestamp=$((total_timestamp / fallback_file_count))
                    log "INFO" "Found $location: average timestamp $avg_timestamp" >&2

                    if [[ $avg_timestamp -gt $latest_timestamp ]]; then
                        latest_timestamp="$avg_timestamp"
                        latest_location="$location"
                    fi
                fi
            else
                log "WARNING" "GraphChain explorer not found at $explorer_path" >&2
            fi
        done

        if [[ -n "$latest_location" ]]; then
            log "SUCCESS" "Fallback latest graphchain-explorer: $latest_location" >&2
            echo "$latest_location"
        else
            log "ERROR" "No valid graphchain-explorer found" >&2
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
    
    # Files to sync for KNIRVNEXUS
    local sync_patterns=(
        "src/components/auth/*"
        "src/components/dashboard/*"
        "src/components/ui/*"
        "src/hooks/*"
        "src/lib/*"
        "src/app/page.tsx"
        "src/app/layout.tsx"
        "src/app/globals.css"
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
    
    log "INFO" "Syncing to Gateway portal (Vite/React structure)"
    
    # Files to sync for Gateway portals
    local sync_patterns=(
        "src/*"
        "package.json"
        "vite.config.ts"
        "tsconfig.json"
        "tailwind.config.js"
        "index.html"
        "dashboard.html"
        "landing.html"
    )
    
    for pattern in "${sync_patterns[@]}"; do
        if [[ "$pattern" == "src/*" ]]; then
            # Recursively sync src directory
            if [[ -d "$source_path/src" ]]; then
                if [[ "$DRY_RUN" == "true" ]]; then
                    log "INFO" "[DRY RUN] Would sync src directory"
                else
                    rsync -av --exclude='node_modules' --exclude='dist' "$source_path/src/" "$target_path/src/"
                    log "SUCCESS" "Synced src directory"
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

sync_graphchain_explorer() {
    local source_location="$1"
    local target_location="$2"
    
    local source_path="$PROJECT_ROOT/${GRAPHCHAIN_EXPLORERS[$source_location]}"
    local target_path="$PROJECT_ROOT/${GRAPHCHAIN_EXPLORERS[$target_location]}"
    
    log "INFO" "Syncing graphchain-explorer from $source_location to $target_location"
    
    if [[ ! -d "$source_path" ]]; then
        log "ERROR" "Source path does not exist: $source_path"
        return 1
    fi
    
    # Backup target before sync
    if [[ -d "$target_path" ]]; then
        backup_directory "$target_path" "graphchain-explorer-${target_location}"
    fi
    
    # Create target directory if it doesn't exist
    mkdir -p "$target_path"
    
    # Sync all files except node_modules and build artifacts
    if [[ "$DRY_RUN" == "true" ]]; then
        log "INFO" "[DRY RUN] Would sync graphchain-explorer directory"
    else
        rsync -av --exclude='node_modules' --exclude='dist' --exclude='build' "$source_path/" "$target_path/"
        log "SUCCESS" "Synced graphchain-explorer directory"
    fi
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

sync_all_graphchain_explorers() {
    local latest_location
    latest_location=$(detect_latest_graphchain_explorer 2>/dev/null)

    if [[ $? -ne 0 || -z "$latest_location" ]]; then
        log "ERROR" "Failed to detect latest graphchain-explorer"
        return 1
    fi

    log "INFO" "Syncing from latest graphchain-explorer: $latest_location"

    for target_location in "${!GRAPHCHAIN_EXPLORERS[@]}"; do
        if [[ "$target_location" != "$latest_location" ]]; then
            sync_graphchain_explorer "$latest_location" "$target_location"
        fi
    done
}

# =============================================================================
# BINARY SYNC FUNCTIONS
# =============================================================================

sync_graphchain_cli_to_knirvshell() {
    log "INFO" "Syncing GraphChain CLI binary to KNIRVCLI"

    local source_dir="$PROJECT_ROOT/KNIRVGRAPH/build"
    local target_dir="$PROJECT_ROOT/KNIRVCLI/bin"
    local cli_binary="graphchain-cli"

    # Ensure source binary exists
    if [[ ! -f "$source_dir/$cli_binary" ]]; then
        log "WARNING" "GraphChain CLI binary not found at $source_dir/$cli_binary"
        log "INFO" "Building GraphChain CLI binary..."

        # Build the CLI binary to build directory
        cd "$PROJECT_ROOT/KNIRVGRAPH"
        mkdir -p build
        if ! go build -o "build/$cli_binary" ./cmd/cli/main.go; then
            log "ERROR" "Failed to build GraphChain CLI binary"
            return 1
        fi
        cd "$PROJECT_ROOT"
    fi

    # Create target directory
    mkdir -p "$target_dir"

    # Backup existing binary if it exists
    if [[ -f "$target_dir/$cli_binary" ]]; then
        local backup_file="$BACKUP_DIR/$(date +%Y%m%d_%H%M%S)_knirvshell_$cli_binary"
        mkdir -p "$(dirname "$backup_file")"
        cp "$target_dir/$cli_binary" "$backup_file"
        log "INFO" "Backed up existing $cli_binary to $backup_file"
    fi

    # Sync the binary
    if [[ "$DRY_RUN" == "true" ]]; then
        log "INFO" "[DRY RUN] Would sync: $source_dir/$cli_binary -> $target_dir/$cli_binary"
    else
        cp "$source_dir/$cli_binary" "$target_dir/$cli_binary"
        chmod +x "$target_dir/$cli_binary"
        log "SUCCESS" "Synced GraphChain CLI binary to KNIRVCLI"
    fi
}

sync_knirvgraph_binaries_to_knirvana() {
    log "INFO" "Syncing KNIRVGRAPH build binaries to KNIRVANA rust-client"

    local source_dir="$PROJECT_ROOT/KNIRVGRAPH/build"
    local target_dir="$PROJECT_ROOT/KNIRVANA/rust-client/bin"

    # Ensure source directory exists
    if [[ ! -d "$source_dir" ]]; then
        log "WARNING" "KNIRVGRAPH build directory not found at $source_dir"
        log "INFO" "Building KNIRVGRAPH binaries..."

        # Build the binaries
        cd "$PROJECT_ROOT/KNIRVGRAPH"
        if ! make build; then
            log "ERROR" "Failed to build KNIRVGRAPH binaries"
            return 1
        fi
        cd "$PROJECT_ROOT"
    fi

    # Create target directory
    mkdir -p "$target_dir"

    # Backup existing binaries
    if [[ -d "$target_dir" ]] && [[ "$(ls -A "$target_dir" 2>/dev/null)" ]]; then
        local backup_dir="$BACKUP_DIR/$(date +%Y%m%d_%H%M%S)_knirvana_binaries"
        mkdir -p "$backup_dir"
        cp -r "$target_dir"/* "$backup_dir/" 2>/dev/null || true
        log "INFO" "Backed up existing KNIRVANA binaries to $backup_dir"
    fi

    # Sync all binaries
    if [[ "$DRY_RUN" == "true" ]]; then
        log "INFO" "[DRY RUN] Would sync KNIRVGRAPH binaries from $source_dir to $target_dir"
        if [[ -d "$source_dir" ]]; then
            find "$source_dir" -type f -executable | while read -r binary; do
                local binary_name=$(basename "$binary")
                log "INFO" "[DRY RUN] Would sync: $binary -> $target_dir/$binary_name"
            done
        fi
    else
        # Copy all executable files from source to target
        if [[ -d "$source_dir" ]]; then
            find "$source_dir" -type f -executable | while read -r binary; do
                local binary_name=$(basename "$binary")
                cp "$binary" "$target_dir/$binary_name"
                chmod +x "$target_dir/$binary_name"
                log "SUCCESS" "Synced binary: $binary_name"
            done
        fi
        log "SUCCESS" "Synced KNIRVGRAPH binaries to KNIRVANA rust-client"
    fi
}

sync_all_binaries() {
    local success=true

    sync_graphchain_cli_to_knirvshell || success=false
    sync_knirvgraph_binaries_to_knirvana || success=false

    return $([[ "$success" == "true" ]] && echo 0 || echo 1)
}

show_usage() {
    cat << EOF
KNIRV Portal Version Synchronization Script

Usage: $0 [OPTIONS]

OPTIONS:
    -t, --type TYPE        Portal type to sync: nexus, graphchain, binaries, both (default: both)
    -n, --dry-run         Show what would be done without making changes
    -f, --force           Force sync even if target files are newer
    -v, --verbose         Enable verbose logging
    -h, --help            Show this help message

EXAMPLES:
    $0                           # Sync both portal types and binaries
    $0 -t nexus                  # Sync only nexus-portal
    $0 -t graphchain -n          # Dry run for graphchain-explorer only
    $0 -t binaries               # Sync only binaries (GraphChain CLI to KNIRVCLI, KNIRVGRAPH to KNIRVANA)
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
        "graphchain")
            sync_all_graphchain_explorers || success=false
            ;;
        "binaries")
            sync_all_binaries || success=false
            ;;
        "both")
            sync_all_nexus_portals || success=false
            sync_all_graphchain_explorers || success=false
            sync_all_binaries || success=false
            ;;
        *)
            log "ERROR" "Invalid portal type: $PORTAL_TYPE"
            show_usage
            exit 1
            ;;
    esac
    
    if [[ "$success" == "true" ]]; then
        log "SUCCESS" "Portal synchronization completed successfully"
        exit 0
    else
        log "ERROR" "Portal synchronization completed with errors"
        exit 1
    fi
}

# Run main function with all arguments
main "$@"
