#!/bin/bash

# Cleanup script for discovered agents without plugin files
# This script removes discovered agents that don't have corresponding .so or .wasm files

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default paths
DEFAULT_CONFIG_DIR="$HOME/.config/Agentic-Engine"
DEFAULT_DB_PATH="$DEFAULT_CONFIG_DIR/data/domain.db"
DEFAULT_PLUGINS_DIR="$DEFAULT_CONFIG_DIR/plugins"

# Parse command line arguments
DRY_RUN=false
CONFIG_DIR="$DEFAULT_CONFIG_DIR"

while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --config-dir)
            CONFIG_DIR="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --dry-run           Show what would be deleted without making changes"
            echo "  --config-dir DIR    Use custom config directory (default: ~/.config/Agentic-Engine)"
            echo "  --help, -h          Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0 --dry-run                    # Preview what would be deleted"
            echo "  $0                              # Actually delete discovered agents"
            echo "  $0 --config-dir /custom/path   # Use custom config directory"
            exit 0
            ;;
        *)
            echo -e "${RED}❌ Unknown option: $1${NC}"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

DB_PATH="$CONFIG_DIR/data/domain.db"
PLUGINS_DIR="$CONFIG_DIR/plugins"

echo -e "${BLUE}🧹 Agentic-Engine Agent Cleanup Script${NC}"
echo ""
echo -e "${BLUE}📂 Config directory: ${NC}$CONFIG_DIR"
echo -e "${BLUE}📂 Database path: ${NC}$DB_PATH"
echo -e "${BLUE}📂 Plugins directory: ${NC}$PLUGINS_DIR"

if [ "$DRY_RUN" = true ]; then
    echo -e "${YELLOW}🔍 DRY RUN MODE - No changes will be made${NC}"
fi

echo ""

# Check if database exists
if [ ! -f "$DB_PATH" ]; then
    echo -e "${RED}❌ Database not found: $DB_PATH${NC}"
    echo -e "${YELLOW}💡 This is normal if you haven't created any agents yet.${NC}"
    exit 0
fi

# Check if plugins directory exists
if [ ! -d "$PLUGINS_DIR" ]; then
    echo -e "${YELLOW}⚠️  Plugins directory not found: $PLUGINS_DIR${NC}"
    echo -e "${YELLOW}💡 Creating plugins directory...${NC}"
    mkdir -p "$PLUGINS_DIR"
fi

# Use the Go cleanup script
GO_SCRIPT="$(dirname "$0")/cleanup_discovered_agents.go"

if [ ! -f "$GO_SCRIPT" ]; then
    echo -e "${RED}❌ Go cleanup script not found: $GO_SCRIPT${NC}"
    echo -e "${YELLOW}💡 Please ensure cleanup_discovered_agents.go is in the same directory as this script.${NC}"
    exit 1
fi

# Check if Go is available
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed or not in PATH${NC}"
    echo -e "${YELLOW}💡 Please install Go to use this cleanup script.${NC}"
    exit 1
fi

# Run the Go cleanup script
echo -e "${BLUE}🚀 Running cleanup...${NC}"
echo ""

if [ "$DRY_RUN" = true ]; then
    go run "$GO_SCRIPT" "$DB_PATH" --dry-run
else
    go run "$GO_SCRIPT" "$DB_PATH"
fi

echo ""
echo -e "${GREEN}✅ Cleanup script completed!${NC}"

# Show remaining agents if any
if [ -f "$DB_PATH" ] && command -v sqlite3 &> /dev/null; then
    echo ""
    echo -e "${BLUE}📊 Current agent status:${NC}"
    
    # Try to show agent count (this is a simplified approach)
    # Note: chromem-go uses a different storage format, so this might not work perfectly
    echo -e "${YELLOW}💡 To see current agents, use the Agentic-Engine UI or check the database directly.${NC}"
fi

echo ""
echo -e "${BLUE}🎯 Next steps:${NC}"
echo -e "   1. Start Agentic-Engine: ${GREEN}./scripts/run_production.sh${NC}"
echo -e "   2. Create new agents using the 'Create Agent' button in the UI"
echo -e "   3. Built agents will have working terminal functionality"
