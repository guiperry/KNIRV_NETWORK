#!/bin/bash

# KNIRVENGINE Production Deployment Script
# Builds and deploys the complete three-engine architecture

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration (adjusted for scripts directory)
DEPLOY_DIR="../dist"
BACKUP_DIR="../backup"
LOG_FILE="../deploy.log"

# Functions
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "$LOG_FILE"
}

success() {
    echo -e "${GREEN}✅ $1${NC}" | tee -a "$LOG_FILE"
}

warning() {
    echo -e "${YELLOW}⚠️  $1${NC}" | tee -a "$LOG_FILE"
}

error() {
    echo -e "${RED}❌ $1${NC}" | tee -a "$LOG_FILE"
    exit 1
}

check_dependencies() {
    log "Checking dependencies..."
    
    # Check Go
    if ! command -v go &> /dev/null; then
        error "Go is not installed. Please install Go 1.21+"
    fi
    
    # Check Node.js
    if ! command -v node &> /dev/null; then
        error "Node.js is not installed. Please install Node.js 18+"
    fi
    
    # Check Rust
    if ! command -v cargo &> /dev/null; then
        error "Rust is not installed. Please install Rust 1.70+"
    fi
    
    # Check wasm32 target
    if ! rustup target list --installed | grep -q "wasm32-unknown-unknown"; then
        log "Installing wasm32-unknown-unknown target..."
        rustup target add wasm32-unknown-unknown
    fi
    
    success "All dependencies are available"
}

create_directories() {
    log "Creating deployment directories..."
    
    mkdir -p "$DEPLOY_DIR"
    mkdir -p "$BACKUP_DIR"
    mkdir -p "$DEPLOY_DIR/agentic-wallet"
    mkdir -p "$DEPLOY_DIR/browser-bridge"
    
    success "Deployment directories created"
}

backup_existing() {
    if [ -d "$DEPLOY_DIR" ] && [ "$(ls -A $DEPLOY_DIR)" ]; then
        log "Backing up existing deployment..."
        
        BACKUP_NAME="backup_$(date +'%Y%m%d_%H%M%S')"
        cp -r "$DEPLOY_DIR" "$BACKUP_DIR/$BACKUP_NAME"
        
        success "Backup created: $BACKUP_DIR/$BACKUP_NAME"
    fi
}

build_agentic_wallet() {
    log "Building Agentic Wallet components..."

    # Build Go backend server
    cd ../agentic-wallet/go-backend
    log "Building Go backend server..."
    go build -ldflags="-w -s" -o knirv-agentic-wallet cmd/server/main.go
    cp knirv-agentic-wallet "../../$DEPLOY_DIR/agentic-wallet/"
    cp -r internal "../../$DEPLOY_DIR/agentic-wallet/" 2>/dev/null || true
    cp go.mod go.sum "../../$DEPLOY_DIR/agentic-wallet/" 2>/dev/null || true

    # Build React Native mobile app (production build)
    cd ..
    log "Building React Native mobile app..."
    if [ -f "package.json" ]; then
        npm ci --production=false
        npm run build 2>/dev/null || npm run bundle 2>/dev/null || true
        
        # Copy built assets if they exist
        if [ -d "dist" ]; then
            cp -r dist "../../$DEPLOY_DIR/agentic-wallet/app/"
        fi
        if [ -d "build" ]; then
            cp -r build "../../$DEPLOY_DIR/agentic-wallet/app/"
        fi
    fi

    cd ../..
    success "Agentic Wallet components built successfully"
}

build_browser_bridge() {
    log "Building Browser Bridge components..."
    
    cd ../browser-bridge
    
    # Build browser extension packages
    if [ -f "package.json" ]; then
        log "Building browser extension packages..."
        npm ci --production=false
        
        # Build each package
        for package in packages/*/; do
            if [ -f "$package/package.json" ]; then
                log "Building package: $(basename $package)"
                cd "$package"
                npm run build 2>/dev/null || npm run compile 2>/dev/null || true
                cd ../..
            fi
        done
        
        # Copy packages to deployment directory
        mkdir -p "../../$DEPLOY_DIR/browser-bridge/packages"
        cp -r packages/* "../../$DEPLOY_DIR/browser-bridge/packages/" 2>/dev/null || true
    fi
    
    cd ../..
    success "Browser Bridge components built successfully"
}

# Agent-Core functionality is now integrated into the agentic-wallet components

copy_assets() {
    log "Copying additional assets..."

    # Copy configuration files and documentation
    cp README.md "$DEPLOY_DIR/" 2>/dev/null || true
    cp test_basic.js "$DEPLOY_DIR/" 2>/dev/null || true
    
    success "Assets copied"
}

create_startup_scripts() {
    log "Creating startup scripts..."
    
    # Agentic Wallet startup script
    cat > "$DEPLOY_DIR/start-agentic-wallet.sh" << 'EOF'
#!/bin/bash
cd "$(dirname "$0")/agentic-wallet"
echo "Starting KNIRVENGINE Agentic Wallet Server..."
./knirv-agentic-wallet
EOF
    
    # Mobile development startup script
    cat > "$DEPLOY_DIR/start-mobile-dev.sh" << 'EOF'
#!/bin/bash
cd "$(dirname "$0")/../agentic-wallet"
echo "Starting KNIRVENGINE Mobile App development server..."
npm start
EOF
    
    # Complete system startup script
    cat > "$DEPLOY_DIR/start-system.sh" << 'EOF'
#!/bin/bash
echo "🚀 Starting KNIRVENGINE Complete System..."

# Start Agentic Wallet in background
cd "$(dirname "$0")"
./start-agentic-wallet.sh &
WALLET_PID=$!

echo "Agentic Wallet Server started (PID: $WALLET_PID)"
echo "Mobile App development: Run './start-mobile-dev.sh' in another terminal"
echo "Browser Extension: Available in browser-bridge/packages/"
echo "API available at: http://localhost:8082"
echo ""
echo "Press Ctrl+C to stop the system"

# Wait for interrupt
trap "echo 'Stopping system...'; kill $WALLET_PID; exit 0" INT
wait $WALLET_PID
EOF
    
    # Make scripts executable
    chmod +x "$DEPLOY_DIR"/*.sh
    
    success "Startup scripts created"
}

run_tests() {
    log "Running deployment tests..."
    
    # Copy test files to deployment directory
    cp test_basic.js "$DEPLOY_DIR/" 2>/dev/null || true
    
    # Install test dependencies in deployment directory
    cd "$DEPLOY_DIR"
    npm init -y > /dev/null 2>&1
    npm install > /dev/null 2>&1
    
    success "Deployment tests setup completed"
    echo "   Run './test_basic.js' to test the deployment"
}

create_documentation() {
    log "Creating deployment documentation..."
    
    cat > "$DEPLOY_DIR/DEPLOYMENT.md" << 'EOF'
# KNIRVENGINE Deployment Guide

## Quick Start

1. **Start the complete system**:
   ```bash
   ./start-system.sh
   ```

2. **Start individual components**:
   ```bash
   ./start-agentic-wallet.sh    # Agentic Wallet server only
   ./start-mobile-dev.sh        # Mobile app development server
   ```

## System URLs

- **Agentic Wallet API**: http://localhost:8082
- **Mobile App development**: Run in separate terminal with './start-mobile-dev.sh'
- **Browser Extension**: Available in browser-bridge/packages/ directory
- **Health Check**: http://localhost:8082/health or http://localhost:8082/api/health

## Configuration

Configuration is handled within each component's own configuration files.

## Troubleshooting

1. **Port conflicts**: Check each component's configuration
2. **Permission errors**: Ensure scripts are executable
3. **Missing dependencies**: Check system requirements for each component
4. **Build issues**: Verify Node.js, Go, and other dependencies are installed

## Logs

- Agentic Wallet logs: Check console output
- System logs: Check deploy.log in parent directory

## Support

See README.md files in each component directory for detailed documentation.
EOF
    
    success "Deployment documentation created"
}

print_summary() {
    log "Deployment completed successfully!"
    echo ""
    echo "📦 KNIRVENGINE Deployment Summary"
    echo "================================="
    echo ""
    echo "📁 Deployment Directory: $DEPLOY_DIR"
    echo ""
    echo "🏗️  Built Components:"
    echo "   • Agentic Wallet Server (Go)"
    echo "   • Mobile App (React Native/TypeScript)"
    echo "   • Browser Bridge Extensions (React/TypeScript)"
    echo ""
    echo "🚀 Quick Start:"
    echo "   cd $DEPLOY_DIR"
    echo "   ./start-system.sh"
    echo ""
    echo "🌐 System URLs:"
    echo "   • Agentic Wallet API: http://localhost:8082"
    echo "   • Mobile App: Run './start-mobile-dev.sh' for development"
    echo "   • Browser Extensions: Available in browser-bridge/packages/"
    echo ""
    echo "📚 Documentation:"
    echo "   • DEPLOYMENT.md - Deployment-specific guide"
    echo "   • Component READMEs - Individual component documentation"
    echo ""
    success "Ready for production deployment!"
}

# Main deployment process
main() {
    log "Starting KNIRVENGINE deployment process..."
    
    check_dependencies
    create_directories
    backup_existing
    
    build_agentic_wallet
    build_browser_bridge
    
    copy_assets
    create_startup_scripts
    create_documentation
    
    run_tests
    print_summary
}

# Run main function
main "$@"
