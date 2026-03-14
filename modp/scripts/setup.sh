#!/bin/bash
# setup.sh - Setup script for KNIRV Network ModP Framework
# Installs P language and configures the verification environment

set -e

echo "========================================"
echo "KNIRV Network ModP Framework Setup"
echo "========================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(dirname "$(readlink -f "$0")")"
MODP_DIR="$(dirname "$SCRIPT_DIR")"

# Check prerequisites
check_prerequisites() {
    echo -e "${YELLOW}Checking prerequisites...${NC}"

    # Check for .NET SDK (required for P compiler)
    if ! command -v dotnet &> /dev/null; then
        echo -e "${RED}Error: .NET SDK is required but not installed.${NC}"
        echo "Install .NET SDK from: https://dotnet.microsoft.com/download"
        exit 1
    fi
    echo -e "${GREEN}✓ .NET SDK found: $(dotnet --version)${NC}"

    # Check for Java (required for P runtime)
    if ! command -v java &> /dev/null; then
        echo -e "${RED}Error: Java is required but not installed.${NC}"
        echo "Install Java JDK 11 or later"
        exit 1
    fi
    echo -e "${GREEN}✓ Java found: $(java -version 2>&1 | head -n 1)${NC}"

    # Check for Git
    if ! command -v git &> /dev/null; then
        echo -e "${RED}Error: Git is required but not installed.${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ Git found${NC}"
}

# Install P compiler
install_p_compiler() {
    echo -e "${YELLOW}Installing P language compiler...${NC}"

    P_DIR="${HOME}/.p-lang"

    if [ -d "$P_DIR" ]; then
        echo "P language directory exists, updating..."
        cd "$P_DIR"
        git pull origin master || true
    else
        echo "Cloning P language repository..."
        git clone https://github.com/p-org/P.git "$P_DIR"
        cd "$P_DIR"
    fi

    # Build P compiler
    echo "Building P compiler..."
    cd "$P_DIR"
    dotnet build -c Release

    # Add P to PATH
    export PATH="$P_DIR/Bld/Drops/Release/Binaries/Pc:$PATH"

    # Create symlink for easy access
    if [ ! -L "/usr/local/bin/pc" ]; then
        sudo ln -sf "$P_DIR/Bld/Drops/Release/Binaries/Pc/Pc" /usr/local/bin/pc 2>/dev/null || \
            echo -e "${YELLOW}Could not create symlink, add to PATH manually${NC}"
    fi

    echo -e "${GREEN}✓ P compiler installed${NC}"
}

# Install P runtime
install_p_runtime() {
    echo -e "${YELLOW}Installing P runtime...${NC}"

    P_DIR="${HOME}/.p-lang"

    # Build P runtime
    cd "$P_DIR"
    dotnet build -c Release ./Src/PChecker/CheckerCore/CheckerCore.csproj

    echo -e "${GREEN}✓ P runtime installed${NC}"
}

# Verify installation
verify_installation() {
    echo -e "${YELLOW}Verifying installation...${NC}"

    if command -v pc &> /dev/null; then
        echo -e "${GREEN}✓ P compiler (pc) is accessible${NC}"
        pc --version 2>/dev/null || echo "Version check not supported"
    else
        echo -e "${YELLOW}⚠ P compiler not in PATH. Add to PATH:${NC}"
        echo "export PATH=\"\$HOME/.p-lang/Bld/Drops/Release/Binaries/Pc:\$PATH\""
    fi
}

# Create output directories
create_directories() {
    echo -e "${YELLOW}Creating output directories...${NC}"

    mkdir -p "$MODP_DIR/output"
    mkdir -p "$MODP_DIR/results"

    echo -e "${GREEN}✓ Directories created${NC}"
}

# Display component summary
display_summary() {
    echo ""
    echo "========================================"
    echo -e "${GREEN}KNIRV Network ModP Framework Ready!${NC}"
    echo "========================================"
    echo ""
    echo "Components modeled:"
    echo "  • KNIRVORACLE - Token, Governance, Economics, Consensus, IBC"
    echo "  • KNIRVCHAIN  - Skill Registry, LLM Registry, MCP, Node Transformation"
    echo "  • KNIRVGRAPH  - Knowledge Graph"
    echo "  • KNIRVROUTER - P2P Network, Proof of Connectivity"
    echo "  • KNIRVSERVER  - Validation, Execution Sandbox"
    echo "  • KNIRVBASE   - Base Layer"
    echo ""
    echo "Network-wide monitors:"
    echo "  • NetworkIntegrityMonitor"
    echo "  • CrossChainConsistencyMonitor"
    echo "  • TokenEconomicsMonitor"
    echo "  • GovernanceProcessMonitor"
    echo "  • ValidationIntegrityMonitor"
    echo "  • KnowledgeGraphConsistencyMonitor"
    echo ""
    echo "Next steps:"
    echo "1. Add P compiler to PATH:"
    echo "   export PATH=\"\$HOME/.p-lang/Bld/Drops/Release/Binaries/Pc:\$PATH\""
    echo ""
    echo "2. Run ModP tests:"
    echo "   make test-modp-network"
    echo ""
    echo "3. Compile P modules:"
    echo "   cd modp && pc compile -pp:KnirvNetwork.pproj"
    echo ""
}

# Main installation flow
main() {
    echo ""
    check_prerequisites
    echo ""
    install_p_compiler
    echo ""
    install_p_runtime
    echo ""
    verify_installation
    echo ""
    create_directories
    echo ""
    display_summary
}


# Run main if not sourced
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
