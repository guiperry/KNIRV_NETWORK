#!/bin/bash
set -e

# KNIRV Testnet Dependency Installation Script
# Installs Go, Rust, and other required toolchains for building KNIRV components

echo "🔧 KNIRV Testnet Dependency Installation"
echo "========================================"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
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

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    armv7l) ARCH="armv6l" ;;
    *) print_error "Unsupported architecture: $ARCH"; exit 1 ;;
esac

print_status "Detected OS: $OS, Architecture: $ARCH"

# Create directories
mkdir -p ~/.local/bin
mkdir -p ~/.local/go
mkdir -p ~/.local/rust

# Add to PATH if not already there
if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
    export PATH="$HOME/.local/bin:$PATH"
    echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
fi

# Install Go
install_go() {
    print_status "Installing Go toolchain..."
    
    # Check if Go is already installed
    if command -v go &> /dev/null; then
        GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
        print_success "Go is already installed: $GO_VERSION"
        return 0
    fi
    
    # Download and install Go
    GO_VERSION="1.23.4"
    GO_TARBALL="go${GO_VERSION}.${OS}-${ARCH}.tar.gz"
    GO_URL="https://golang.org/dl/${GO_TARBALL}"
    
    print_status "Downloading Go ${GO_VERSION} for ${OS}-${ARCH}..."
    
    if command -v wget &> /dev/null; then
        wget -q "$GO_URL" -O "/tmp/${GO_TARBALL}"
    elif command -v curl &> /dev/null; then
        curl -sL "$GO_URL" -o "/tmp/${GO_TARBALL}"
    else
        print_error "Neither wget nor curl found. Please install one of them."
        exit 1
    fi
    
    print_status "Extracting Go..."
    tar -C ~/.local -xzf "/tmp/${GO_TARBALL}"
    
    # Create symlink
    ln -sf ~/.local/go/bin/go ~/.local/bin/go
    ln -sf ~/.local/go/bin/gofmt ~/.local/bin/gofmt
    
    # Set GOROOT and GOPATH
    export GOROOT="$HOME/.local/go"
    export GOPATH="$HOME/.local/go-workspace"
    export PATH="$GOROOT/bin:$PATH"
    
    echo 'export GOROOT="$HOME/.local/go"' >> ~/.bashrc
    echo 'export GOPATH="$HOME/.local/go-workspace"' >> ~/.bashrc
    echo 'export PATH="$GOROOT/bin:$PATH"' >> ~/.bashrc
    
    # Cleanup
    rm -f "/tmp/${GO_TARBALL}"
    
    print_success "Go ${GO_VERSION} installed successfully"
}

# Install Rust
install_rust() {
    print_status "Installing Rust toolchain..."
    
    # Check if Rust is already installed
    if command -v rustc &> /dev/null; then
        RUST_VERSION=$(rustc --version | awk '{print $2}')
        print_success "Rust is already installed: $RUST_VERSION"
        return 0
    fi
    
    print_status "Downloading and installing Rust..."
    
    # Download rustup installer
    if command -v curl &> /dev/null; then
        curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --no-modify-path
    else
        print_error "curl is required to install Rust"
        exit 1
    fi
    
    # Source cargo environment
    source ~/.cargo/env
    
    # Create symlinks
    ln -sf ~/.cargo/bin/rustc ~/.local/bin/rustc
    ln -sf ~/.cargo/bin/cargo ~/.local/bin/cargo
    ln -sf ~/.cargo/bin/rustup ~/.local/bin/rustup
    
    # Add to PATH
    export PATH="$HOME/.cargo/bin:$PATH"
    echo 'export PATH="$HOME/.cargo/bin:$PATH"' >> ~/.bashrc
    
    print_success "Rust installed successfully"
}

# Install Node.js (if not present)
install_nodejs() {
    print_status "Checking Node.js installation..."

    if command -v node &> /dev/null; then
        NODE_VERSION=$(node --version)
        print_success "Node.js is already installed: $NODE_VERSION"
        return 0
    fi

    print_status "Node.js not found..."

    # Try to install Node.js using NodeSource repository (for Ubuntu/Debian)
    if command -v apt-get &> /dev/null; then
        if sudo -n true 2>/dev/null; then
            print_status "Installing Node.js via NodeSource..."
            curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
            sudo apt-get install -y nodejs
            print_success "Node.js installed"
        else
            print_warning "Node.js not found and no sudo access for installation"
            print_warning "Assuming Node.js is available in the environment"
        fi
    else
        print_warning "Node.js not found and automatic installation not supported for this OS"
        print_warning "Assuming Node.js is available in the environment"
    fi
}

# Install Python3 (if not present)
install_python() {
    print_status "Checking Python3 installation..."
    
    if command -v python3 &> /dev/null; then
        PYTHON_VERSION=$(python3 --version)
        print_success "Python3 is already installed: $PYTHON_VERSION"
        return 0
    fi
    
    print_warning "Python3 not found. Please install Python3 manually."
}

# Install build essentials
install_build_essentials() {
    print_status "Checking build essentials..."

    # Check if we have a C compiler
    if command -v gcc &> /dev/null || command -v clang &> /dev/null; then
        print_success "C compiler found"
        return 0
    fi

    print_status "Installing build essentials..."

    # Try to install without sudo first (for containerized environments)
    if command -v apt-get &> /dev/null; then
        if sudo -n apt-get update 2>/dev/null && sudo -n apt-get install -y build-essential pkg-config libssl-dev 2>/dev/null; then
            print_success "Build essentials installed"
        else
            print_warning "Could not install build essentials (no sudo access)"
            print_warning "Assuming build tools are available in the environment"
        fi
    elif command -v yum &> /dev/null; then
        if sudo -n yum groupinstall -y "Development Tools" 2>/dev/null && sudo -n yum install -y openssl-devel 2>/dev/null; then
            print_success "Build essentials installed"
        else
            print_warning "Could not install build essentials (no sudo access)"
            print_warning "Assuming build tools are available in the environment"
        fi
    else
        print_warning "Could not install build essentials automatically"
        print_warning "Assuming build tools are available in the environment"
    fi
}

# Main installation process
main() {
    print_status "Starting dependency installation for KNIRV Testnet..."
    
    # Install build essentials first (if on Linux)
    if [[ "$OS" == "linux" ]]; then
        install_build_essentials
    fi
    
    # Install core toolchains
    install_go
    install_rust
    install_nodejs
    install_python
    
    # Refresh environment and verify installations
    print_status "Refreshing environment and verifying installations..."

    # Source updated environment
    if [ -f ~/.bashrc ]; then
        source ~/.bashrc 2>/dev/null || true
    fi

    # Explicitly export PATH with all toolchain locations
    export PATH="$HOME/.local/bin:$HOME/.cargo/bin:$HOME/.local/go/bin:$PATH"
    export GOROOT="$HOME/.local/go"
    export GOPATH="$HOME/.local/go-workspace"

    # Also source cargo environment if it exists
    if [ -f ~/.cargo/env ]; then
        source ~/.cargo/env 2>/dev/null || true
    fi
    
    echo ""
    echo "🔍 Installation Verification:"
    echo "============================="
    
    if command -v go &> /dev/null; then
        print_success "Go: $(go version)"
    else
        print_error "Go installation failed"
    fi
    
    if command -v rustc &> /dev/null; then
        print_success "Rust: $(rustc --version)"
    else
        print_error "Rust installation failed"
    fi
    
    if command -v cargo &> /dev/null; then
        print_success "Cargo: $(cargo --version)"
    else
        print_error "Cargo installation failed"
    fi
    
    if command -v node &> /dev/null; then
        print_success "Node.js: $(node --version)"
    else
        print_warning "Node.js not found"
    fi
    
    if command -v python3 &> /dev/null; then
        print_success "Python3: $(python3 --version)"
    else
        print_warning "Python3 not found"
    fi
    
    echo ""
    print_success "Dependency installation completed!"

    # Create environment file for current session
    print_status "Creating environment file for current session..."
    cat > ~/.knirv-env << EOF
# KNIRV Testnet Environment Variables
export PATH="$HOME/.local/bin:$HOME/.cargo/bin:$HOME/.local/go/bin:\$PATH"
export GOROOT="$HOME/.local/go"
export GOPATH="$HOME/.local/go-workspace"
EOF

    # Source the environment file
    source ~/.knirv-env

    print_status "Environment refreshed for current session"
    print_status "Toolchains are now available in PATH"

    # Install Node.js dependencies and fix axios if needed
    print_status "Installing Node.js dependencies..."
    if [ -f "package.json" ]; then
        npm install || {
            print_warning "npm install failed, checking for axios corruption..."
            if [ -f "scripts/fix-axios-corruption.sh" ]; then
                print_status "Running axios corruption fix..."
                chmod +x scripts/fix-axios-corruption.sh
                ./scripts/fix-axios-corruption.sh || {
                    print_warning "Axios fix script failed, trying manual fix..."
                    npm install axios@1.6.8 --save-exact
                }
            else
                print_warning "Axios fix script not found, trying manual fix..."
                npm install axios@1.6.8 --save-exact
            fi

            # Retry npm install after axios fix
            npm install || print_error "npm install failed even after axios fix"
        }
        print_success "Node.js dependencies installed"
    else
        print_warning "package.json not found, skipping npm install"
    fi

    print_status "You can now run 'npm start' to build and start the KNIRV testnet"
}

# Run main function
main "$@"
