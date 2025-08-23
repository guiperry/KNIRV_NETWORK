#!/bin/bash

# CLI Synchronization Validation Script
# Validates that KNIRVCONTROLLER/cli and KNIRVSDK/cli are properly synchronized

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
CONTROLLER_CLI="$ROOT_DIR/KNIRVCONTROLLER/cli"
SDK_CLI="$ROOT_DIR/KNIRVSDK/cli"

echo -e "${BLUE}🔍 CLI Synchronization Validation${NC}"
echo "=================================="
echo "Controller CLI: $CONTROLLER_CLI"
echo "SDK CLI: $SDK_CLI"
echo ""

# Check if directories exist
if [ ! -d "$CONTROLLER_CLI" ]; then
    echo -e "${RED}❌ Controller CLI directory not found: $CONTROLLER_CLI${NC}"
    exit 1
fi

if [ ! -d "$SDK_CLI" ]; then
    echo -e "${RED}❌ SDK CLI directory not found: $SDK_CLI${NC}"
    exit 1
fi

echo -e "${GREEN}✅ CLI directories found${NC}"

# Function to calculate file hash
calculate_hash() {
    local file="$1"
    if [ -f "$file" ]; then
        sha256sum "$file" | cut -d' ' -f1
    else
        echo "FILE_NOT_FOUND"
    fi
}

# Function to validate file sync
validate_file_sync() {
    local file="$1"
    local controller_file="$CONTROLLER_CLI/$file"
    local sdk_file="$SDK_CLI/$file"
    
    local controller_hash=$(calculate_hash "$controller_file")
    local sdk_hash=$(calculate_hash "$sdk_file")
    
    if [ "$controller_hash" = "FILE_NOT_FOUND" ] && [ "$sdk_hash" = "FILE_NOT_FOUND" ]; then
        echo -e "  ${YELLOW}⚠️  $file - not found in either location${NC}"
        return 1
    elif [ "$controller_hash" = "FILE_NOT_FOUND" ]; then
        echo -e "  ${YELLOW}⚠️  $file - missing in Controller CLI${NC}"
        return 1
    elif [ "$sdk_hash" = "FILE_NOT_FOUND" ]; then
        echo -e "  ${YELLOW}⚠️  $file - missing in SDK CLI${NC}"
        return 1
    elif [ "$controller_hash" = "$sdk_hash" ]; then
        echo -e "  ${GREEN}✅ $file - in sync${NC}"
        return 0
    else
        echo -e "  ${RED}❌ $file - out of sync${NC}"
        return 1
    fi
}

# Critical files that must be in sync
CRITICAL_FILES=(
    "cmd/root.go"
    "config/config.go"
    "core/api_client.go"
    "core/event_bus.go"
    "main.go"
)

echo -e "${BLUE}🔍 Validating critical files...${NC}"
critical_errors=0
for file in "${CRITICAL_FILES[@]}"; do
    if ! validate_file_sync "$file"; then
        ((critical_errors++))
    fi
done

# Bidirectional files that should be in sync
BIDIRECTIONAL_FILES=(
    "cmd/utils.go"
    "config/defaults.go"
    "core/file_manager.go"
    "core/websocket_manager.go"
    "core/sse_client.go"
)

echo -e "\n${BLUE}🔍 Validating bidirectional files...${NC}"
bidirectional_errors=0
for file in "${BIDIRECTIONAL_FILES[@]}"; do
    if ! validate_file_sync "$file"; then
        ((bidirectional_errors++))
    fi
done

# Check Go module consistency
echo -e "\n${BLUE}🔍 Validating Go modules...${NC}"

# Check if both have valid go.mod files
if [ -f "$CONTROLLER_CLI/go.mod" ] && [ -f "$SDK_CLI/go.mod" ]; then
    echo -e "  ${GREEN}✅ Both CLIs have go.mod files${NC}"
    
    # Check if they can build
    echo -e "  ${BLUE}🔨 Testing Controller CLI build...${NC}"
    if (cd "$CONTROLLER_CLI" && go build -o /tmp/controller-cli-test main.go); then
        echo -e "  ${GREEN}✅ Controller CLI builds successfully${NC}"
        rm -f /tmp/controller-cli-test
    else
        echo -e "  ${RED}❌ Controller CLI build failed${NC}"
        ((critical_errors++))
    fi
    
    echo -e "  ${BLUE}🔨 Testing SDK CLI build...${NC}"
    if (cd "$SDK_CLI" && go build -o /tmp/sdk-cli-test main.go); then
        echo -e "  ${GREEN}✅ SDK CLI builds successfully${NC}"
        rm -f /tmp/sdk-cli-test
    else
        echo -e "  ${RED}❌ SDK CLI build failed${NC}"
        ((critical_errors++))
    fi
else
    echo -e "  ${RED}❌ Missing go.mod files${NC}"
    ((critical_errors++))
fi

# Check version tracking
echo -e "\n${BLUE}🔍 Checking version tracking...${NC}"
VERSION_FILE="$ROOT_DIR/KNIRVCONTROLLER/cli-sync-version.json"
if [ -f "$VERSION_FILE" ]; then
    echo -e "  ${GREEN}✅ Version tracking file exists${NC}"
    
    # Check if it's valid JSON
    if jq empty "$VERSION_FILE" 2>/dev/null; then
        echo -e "  ${GREEN}✅ Version tracking file is valid JSON${NC}"
        
        # Show last sync time
        last_sync=$(jq -r '.timestamp' "$VERSION_FILE" 2>/dev/null || echo "unknown")
        echo -e "  ${BLUE}📅 Last sync: $last_sync${NC}"
    else
        echo -e "  ${RED}❌ Version tracking file is invalid JSON${NC}"
        ((bidirectional_errors++))
    fi
else
    echo -e "  ${YELLOW}⚠️  Version tracking file not found${NC}"
    echo -e "  ${BLUE}💡 Run 'node scripts/cli-sync.js sync' to create it${NC}"
fi

# Check for common sync issues
echo -e "\n${BLUE}🔍 Checking for common sync issues...${NC}"

# Check for conflicting imports
echo -e "  ${BLUE}🔍 Checking for import conflicts...${NC}"
controller_imports=$(find "$CONTROLLER_CLI" -name "*.go" -exec grep -l "github.com/guiperry/KNIRV_NETWORK" {} \; 2>/dev/null | wc -l)
sdk_imports=$(find "$SDK_CLI" -name "*.go" -exec grep -l "github.com/guiperry/KNIRV_NETWORK" {} \; 2>/dev/null | wc -l)

if [ "$controller_imports" -gt 0 ] && [ "$sdk_imports" -gt 0 ]; then
    echo -e "  ${GREEN}✅ Both CLIs use consistent import paths${NC}"
else
    echo -e "  ${YELLOW}⚠️  Import path usage differs between CLIs${NC}"
fi

# Summary
echo -e "\n${BLUE}📊 Validation Summary${NC}"
echo "===================="
echo "Critical errors: $critical_errors"
echo "Bidirectional errors: $bidirectional_errors"
total_errors=$((critical_errors + bidirectional_errors))

if [ $total_errors -eq 0 ]; then
    echo -e "${GREEN}🎉 All validations passed! CLIs are properly synchronized.${NC}"
    exit 0
elif [ $critical_errors -eq 0 ]; then
    echo -e "${YELLOW}⚠️  Minor sync issues found. Consider running synchronization.${NC}"
    echo -e "${BLUE}💡 Run: node scripts/cli-sync.js sync${NC}"
    exit 1
else
    echo -e "${RED}❌ Critical sync issues found! Synchronization required.${NC}"
    echo -e "${BLUE}💡 Run: node scripts/cli-sync.js sync${NC}"
    exit 2
fi
