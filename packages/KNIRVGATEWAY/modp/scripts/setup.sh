#!/bin/bash
# setup.sh - Setup script for ModP framework installation
# Based on: Section 7 of p_implementation.md

set -e

echo "========================================"
echo "ModP Framework Setup for KNIRVORACLE"
echo "========================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check prerequisites
check_prerequisites() {
    echo -e "${YELLOW}Checking prerequisites...${NC}"

    # Check for .NET SDK (required for P compiler)
    if ! command -v dotnet &> /dev/null; then
        echo -e "${RED}Error: .NET SDK is required but not installed.${NC}"
        echo "Install .NET SDK from: https://dotnet.microsoft.com/download"
        exit 1
    fi
    echo -e "${GREEN}✓ .NET SDK found${NC}"

    # Check for Java (required for P runtime)
    if ! command -v java &> /dev/null; then
        echo -e "${RED}Error: Java is required but not installed.${NC}"
        echo "Install Java JDK 11 or later"
        exit 1
    fi
    echo -e "${GREEN}✓ Java found${NC}"

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
        git pull origin master
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
        sudo ln -sf "$P_DIR/Bld/Drops/Release/Binaries/Pc/Pc" /usr/local/bin/pc || echo "Could not create symlink, add to PATH manually"
    fi

    echo -e "${GREEN}✓ P compiler installed${NC}"
}

# Install P runtime
install_p_runtime() {
    echo -e "${YELLOW}Installing P runtime...${NC}"

    P_DIR="${HOME}/.p-lang"

    # Build P runtime
    cd "$P_DIR"
    dotnet build -c Release ./Src/PRuntimes/PCSharpRuntime/PCSharpRuntime.csproj

    echo -e "${GREEN}✓ P runtime installed${NC}"
}

# Verify installation
verify_installation() {
    echo -e "${YELLOW}Verifying installation...${NC}"

    if command -v pc &> /dev/null; then
        echo -e "${GREEN}✓ P compiler (pc) is accessible${NC}"
        pc --version || echo "Version check not supported"
    else
        echo -e "${YELLOW}⚠ P compiler not in PATH. Add to PATH:${NC}"
        echo "export PATH=\"\$HOME/.p-lang/Bld/Drops/Release/Binaries/Pc:\$PATH\""
    fi
}

# Create project configuration
create_project_config() {
    echo -e "${YELLOW}Creating ModP project configuration...${NC}"

    MODP_DIR="$(dirname "$(dirname "$(readlink -f "$0")")")"

    # Create P project file
    cat > "$MODP_DIR/KnirvOracle.pproj" << 'EOF'
<?xml version="1.0" encoding="utf-8"?>
<Project>
  <ProjectName>KnirvOracle</ProjectName>
  <InputFiles>
    <PFile>types.p</PFile>
    <PFile>events.p</PFile>
    <PFile>machines/consensus_machine.p</PFile>
    <PFile>machines/token_machine.p</PFile>
    <PFile>machines/governance_machine.p</PFile>
    <PFile>machines/economics_machine.p</PFile>
    <PFile>machines/ibc_machine.p</PFile>
    <PFile>machines/p2p_interface.p</PFile>
    <PFile>monitors/treasury_invariant.p</PFile>
    <PFile>monitors/token_supply_invariant.p</PFile>
    <PFile>monitors/governance_invariant.p</PFile>
    <PFile>tests/app_layer.p</PFile>
    <PFile>tests/compositional_tests.p</PFile>
  </InputFiles>
  <Target>CSharp</Target>
  <OutputDir>./output</OutputDir>
</Project>
EOF

    echo -e "${GREEN}✓ Project configuration created${NC}"
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
    create_project_config
    echo ""

    echo "========================================"
    echo -e "${GREEN}ModP Framework Setup Complete!${NC}"
    echo "========================================"
    echo ""
    echo "Next steps:"
    echo "1. Add P compiler to PATH:"
    echo "   export PATH=\"\$HOME/.p-lang/Bld/Drops/Release/Binaries/Pc:\$PATH\""
    echo ""
    echo "2. Run ModP tests:"
    echo "   make test-modp"
    echo ""
    echo "3. Compile P modules:"
    echo "   cd KNIRVGATEWAY/modp && pc compile -pp:KnirvOracle.pproj"
    echo ""
}

# Run main if not sourced
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
