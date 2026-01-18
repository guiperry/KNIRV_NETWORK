#!/bin/sh
# run-tests-simple.sh - Simplified script to debug syntax errors

set -e

SCRIPT_DIR="$(dirname "$(readlink -f "$0")")"
MODP_DIR="$(dirname "$SCRIPT_DIR")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Configuration
OUTPUT_DIR="$MODP_DIR/output"
RESULTS_DIR="$MODP_DIR/results"
LOG_FILE="$RESULTS_DIR/test-$(date +%Y%m%d-%H%M%S).log"

# Initialize
PC_COMPILER="dotnet $HOME/.p-lang/Bld/Drops/Release/Binaries/net8.0/p.dll"

initialize() {
    echo -e "${BLUE}Initializing KNIRV Network ModP test environment...${NC}"

    mkdir -p "$OUTPUT_DIR"
    mkdir -p "$RESULTS_DIR"

    # Check P compiler
    if ! [ -f "$HOME/.p-lang/Bld/Drops/Release/Binaries/net8.0/p.dll" ]; then
        echo -e "${RED}Error: P compiler (p.dll) not found at $HOME/.p-lang/Bld/Drops/Release/Binaries/net8.0/p.dll.${NC}"
        echo "Run './scripts/setup.sh' to install the P framework."
        exit 1
    fi

    echo -e "${GREEN}✓ P compiler found${NC}"
}

# Compile P modules
compile_modules() {
    echo -e "${BLUE}Compiling KNIRV Network P modules...${NC}"

    cd "$MODP_DIR"

    # Compile all P files
    $PC_COMPILER compile -pp KnirvNetwork.pproj 2>&1 | tee -a "$LOG_FILE"

    if [ ${PIPESTATUS[0]} -eq 0 ]; then
        echo -e "${GREEN}✓ Compilation successful${NC}"
    else
        echo -e "${RED}✗ Compilation failed${NC}"
        exit 1
    fi
}

main() {
    initialize
    compile_modules
}

main "$@"
