#!/bin/bash
set -e

# KNIRV Testnet - Podman Installation Script
# This script installs Podman and podman-compose for local development

echo "🐳 KNIRV TESTNET - PODMAN INSTALLATION"
echo "======================================"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect operating system
detect_os() {
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        if [ -f /etc/os-release ]; then
            . /etc/os-release
            OS=$NAME
            VER=$VERSION_ID
        else
            OS="Unknown Linux"
        fi
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        OS="macOS"
    elif [[ "$OSTYPE" == "cygwin" ]] || [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "win32" ]]; then
        OS="Windows"
    else
        OS="Unknown"
    fi
}

# Install Python3 and pip if not available
install_python() {
    print_status "Checking Python3 and pip availability..."
    
    if ! command -v python3 &> /dev/null; then
        print_warning "Python3 not found. Installing..."
        case "$OS" in
            *"Ubuntu"*|*"Debian"*)
                sudo apt-get update
                sudo apt-get install -y python3 python3-pip
                ;;
            *"CentOS"*|*"Red Hat"*|*"Fedora"*)
                sudo dnf install -y python3 python3-pip
                ;;
            "macOS")
                if command -v brew &> /dev/null; then
                    brew install python3
                else
                    print_error "Homebrew not found. Please install Python3 manually."
                    exit 1
                fi
                ;;
            *)
                print_error "Unsupported OS for automatic Python3 installation: $OS"
                print_error "Please install Python3 and pip manually."
                exit 1
                ;;
        esac
    else
        print_success "Python3 is available"
    fi
    
    if ! command -v pip3 &> /dev/null; then
        print_warning "pip3 not found. Installing..."
        case "$OS" in
            *"Ubuntu"*|*"Debian"*)
                sudo apt-get install -y python3-pip
                ;;
            *"CentOS"*|*"Red Hat"*|*"Fedora"*)
                sudo dnf install -y python3-pip
                ;;
            "macOS")
                python3 -m ensurepip --upgrade
                ;;
            *)
                print_error "Please install pip3 manually for your OS: $OS"
                exit 1
                ;;
        esac
    else
        print_success "pip3 is available"
    fi
}

# Install Podman
install_podman() {
    print_status "Checking Podman installation..."
    
    if command -v podman &> /dev/null; then
        print_success "Podman is already installed"
        podman --version
        return 0
    fi
    
    print_warning "Podman not found. Installing..."
    
    case "$OS" in
        *"Ubuntu"*|*"Debian"*)
            print_status "Installing Podman on Ubuntu/Debian..."
            sudo apt-get update
            sudo apt-get install -y podman
            ;;
        *"CentOS"*|*"Red Hat"*|*"Fedora"*)
            print_status "Installing Podman on RHEL/CentOS/Fedora..."
            sudo dnf install -y podman
            ;;
        "macOS")
            print_status "Installing Podman on macOS..."
            if command -v brew &> /dev/null; then
                brew install podman
                print_status "Initializing Podman machine..."
                podman machine init
                podman machine start
            else
                print_error "Homebrew not found. Please install Homebrew first:"
                print_error "  /bin/bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
                exit 1
            fi
            ;;
        "Windows")
            print_error "Windows detected. Please install Podman Desktop manually:"
            print_error "  https://podman-desktop.io/downloads"
            exit 1
            ;;
        *)
            print_error "Unsupported OS for automatic Podman installation: $OS"
            print_error "Please install Podman manually:"
            print_error "  https://podman.io/getting-started/installation"
            exit 1
            ;;
    esac
    
    # Verify installation
    if command -v podman &> /dev/null; then
        print_success "Podman installed successfully"
        podman --version
    else
        print_error "Podman installation failed"
        exit 1
    fi
}

# Install podman-compose
install_podman_compose() {
    print_status "Installing podman-compose..."
    
    if command -v podman-compose &> /dev/null; then
        print_success "podman-compose is already installed"
        podman-compose --version
        return 0
    fi
    
    print_status "Installing podman-compose via pip3..."
    pip3 install podman-compose
    
    # Verify installation
    if command -v podman-compose &> /dev/null; then
        print_success "podman-compose installed successfully"
        podman-compose --version
    else
        print_error "podman-compose installation failed"
        print_error "You may need to add ~/.local/bin to your PATH"
        print_error "  export PATH=\$PATH:~/.local/bin"
        exit 1
    fi
}

# Configure Podman for KNIRV
configure_podman() {
    print_status "Configuring Podman for KNIRV testnet..."
    
    # Create necessary directories
    mkdir -p data/ipfs data/knirvoracle data/knirvchain data/knirvgraph data/knirvnexus data/knirvrouter logs
    
    # Set proper permissions for rootless Podman
    chmod -R 755 data config logs
    
    print_success "Podman configured for KNIRV testnet"
}

# Main installation process
main() {
    print_status "Starting Podman installation for KNIRV testnet..."
    
    # Detect operating system
    detect_os
    print_status "Detected OS: $OS"
    
    # Install Python3 and pip
    install_python
    
    # Install Podman
    install_podman
    
    # Install podman-compose
    install_podman_compose
    
    # Configure Podman for KNIRV
    configure_podman
    
    print_success "🎉 Podman installation complete!"
    echo ""
    echo "📋 Next Steps:"
    echo "=============="
    print_status "1. Start KNIRV testnet with Podman: npm run podman:start"
    print_status "2. Check container status: npm run podman:status"
    print_status "3. View logs: npm run podman:logs"
    print_status "4. Stop services: npm run podman:stop"
    echo ""
    print_status "For more information, see: PODMAN_MIGRATION.md"
}

# Run main function
main "$@"
