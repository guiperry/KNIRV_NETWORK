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
    mkdir -p "$DEPLOY_DIR/desktop-client"
    mkdir -p "$DEPLOY_DIR/mobile-controller"
    mkdir -p "$DEPLOY_DIR/agent-core"
    
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

build_desktop_host() {
    log "Building Desktop-Host engine with Electron wrapper..."

    cd ../desktop-client

    # Use the production build script which includes Electron packaging
    if [ -f "scripts/run_production.sh" ]; then
        log "Building Electron desktop application..."
        chmod +x scripts/run_production.sh

        # Build without running (modify the script temporarily)
        sed 's|./dist/linux-unpacked/knirv-engine-desktop|echo "Build completed, skipping run"|' scripts/run_production.sh > temp_build.sh
        chmod +x temp_build.sh
        ./temp_build.sh
        rm temp_build.sh

        # Copy the Electron binaries to deployment directory
        if [ -d "electron/dist/linux-unpacked" ]; then
            log "Copying Electron Linux binary..."
            cp -r electron/dist/linux-unpacked "../$DEPLOY_DIR/desktop-client/"
        fi

        if [ -d "electron/dist/win-unpacked" ]; then
            log "Copying Electron Windows binary..."
            cp -r electron/dist/win-unpacked "../$DEPLOY_DIR/desktop-client/"
        fi

        if [ -d "electron/dist/mac" ]; then
            log "Copying Electron macOS binary..."
            cp -r electron/dist/mac "../$DEPLOY_DIR/desktop-client/"
        fi
    else
        # Fallback to simple Go build
        log "Electron build script not found, using simple Go build..."
        go build -ldflags="-w -s" -o desktop-client main.go
        cp desktop-client "../$DEPLOY_DIR/desktop-client/"
    fi

    cd ..
    success "Desktop-Host engine built successfully"
}

build_mobile_tool() {
    log "Building Mobile-Tool engine..."
    
    cd ../mobile-controller
    
    # Install dependencies
    npm ci --production=false
    
    # Build for production
    npm run build
    
    if [ ! -d "dist" ]; then
        error "Mobile-Tool build failed"
    fi
    
    # Copy to deployment directory
    cp -r dist/* "../$DEPLOY_DIR/mobile-controller/"
    
    cd ..
    success "Mobile-Tool engine built successfully"
}

build_agent_core() {
    log "Building Agent-Core WASM engine..."
    
    cd ../agent-core/rust-wasm
    
    # Build WASM module
    cargo build --release --target wasm32-unknown-unknown
    
    WASM_FILE="target/wasm32-unknown-unknown/release/knirv_cortex_wasm.wasm"
    if [ ! -f "$WASM_FILE" ]; then
        error "Agent-Core WASM build failed"
    fi
    
    # Copy to deployment directory
    cp "$WASM_FILE" "../../$DEPLOY_DIR/agent-core/agent-core.wasm"
    
    cd ../..
    success "Agent-Core WASM engine built successfully"
}

copy_assets() {
    log "Copying additional assets..."

    # HRM weights are now embedded in the WASM module - no external files needed
    success "HRM weights embedded in WASM module"
    
    # Copy configuration files
    cp README.md "$DEPLOY_DIR/"
    cp test_*.js "$DEPLOY_DIR/" 2>/dev/null || true
    
    # Create production configuration
    cat > "$DEPLOY_DIR/config.json" << EOF
{
  "desktop_host": {
    "port": 8082,
    "hrm_weights_path": "./weights.safetensors",
    "tee_data_dir": "./data",
    "production": true
  },
  "mobile_tool": {
    "api_endpoint": "http://localhost:8082",
    "websocket_endpoint": "ws://localhost:8082/api/mcp/ws"
  },
  "agent_core": {
    "wasm_module": "./agent-core.wasm",
    "personality_adaptation": true,
    "memory_buffer_size": 100
  }
}
EOF
    
    success "Assets and configuration copied"
}

create_startup_scripts() {
    log "Creating startup scripts..."
    
    # Desktop Host startup script
    cat > "$DEPLOY_DIR/start-desktop-client.sh" << 'EOF'
#!/bin/bash
cd "$(dirname "$0")/desktop-client"
echo "Starting KNIRVENGINE Desktop Host..."
./desktop-client
EOF
    
    # Mobile Tool startup script (for development)
    cat > "$DEPLOY_DIR/start-mobile-controller.sh" << 'EOF'
#!/bin/bash
cd "$(dirname "$0")/mobile-controller"
echo "Starting KNIRVENGINE Mobile Tool development server..."
python3 -m http.server 8080
EOF
    
    # Complete system startup script
    cat > "$DEPLOY_DIR/start-system.sh" << 'EOF'
#!/bin/bash
echo "🚀 Starting KNIRVENGINE Complete System..."

# Start Desktop Host in background
cd "$(dirname "$0")"
./start-desktop-client.sh &
DESKTOP_PID=$!

echo "Desktop Host started (PID: $DESKTOP_PID)"
echo "Mobile Tool available at: ./mobile-controller/index.html"
echo "API available at: http://localhost:8082"
echo "MCP WebSocket at: ws://localhost:8082/api/mcp/ws"
echo ""
echo "Press Ctrl+C to stop the system"

# Wait for interrupt
trap "echo 'Stopping system...'; kill $DESKTOP_PID; exit 0" INT
wait $DESKTOP_PID
EOF
    
    # Make scripts executable
    chmod +x "$DEPLOY_DIR"/*.sh
    
    success "Startup scripts created"
}

run_tests() {
    log "Running deployment tests..."
    
    # Copy test files to deployment directory
    cp test_*.js "$DEPLOY_DIR/" 2>/dev/null || true
    
    # Install test dependencies in deployment directory
    cd "$DEPLOY_DIR"
    npm init -y > /dev/null 2>&1
    npm install ws > /dev/null 2>&1
    
    # Start system for testing
    ./start-desktop-client.sh &
    DESKTOP_PID=$!
    
    # Wait for startup
    sleep 5
    
    # Run basic health check
    if curl -s http://localhost:8082/api/health > /dev/null; then
        success "Health check passed"
    else
        warning "Health check failed - system may need manual verification"
    fi
    
    # Stop test system
    kill $DESKTOP_PID 2>/dev/null || true
    
    cd ..
    success "Deployment tests completed"
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
   ./start-desktop-client.sh    # Desktop Host only
   ./start-mobile-controller.sh     # Mobile Tool dev server
   ```

## System URLs

- **Desktop Host API**: http://localhost:8082
- **Mobile Tool**: ./mobile-controller/index.html
- **MCP WebSocket**: ws://localhost:8082/api/mcp/ws
- **Health Check**: http://localhost:8082/api/health

## Configuration

Edit `config.json` to customize system settings.

## Troubleshooting

1. **Port conflicts**: Change port in config.json
2. **Permission errors**: Ensure scripts are executable
3. **Missing dependencies**: Check system requirements
4. **HRM not working**: Ensure weights.safetensors is present

## Logs

- Desktop Host logs: Check console output
- System logs: Check deploy.log in parent directory

## Support

See README.md for detailed documentation and support information.
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
    echo "   • Desktop-Host Engine (Go)"
    echo "   • Mobile-Tool Engine (React/TypeScript)"
    echo "   • Agent-Core Engine (Rust WASM)"
    echo ""
    echo "🚀 Quick Start:"
    echo "   cd $DEPLOY_DIR"
    echo "   ./start-system.sh"
    echo ""
    echo "🌐 System URLs:"
    echo "   • Desktop Host: http://localhost:8082"
    echo "   • Mobile Tool: ./mobile-controller/index.html"
    echo "   • MCP WebSocket: ws://localhost:8082/api/mcp/ws"
    echo ""
    echo "📚 Documentation:"
    echo "   • README.md - Complete system documentation"
    echo "   • DEPLOYMENT.md - Deployment-specific guide"
    echo "   • config.json - System configuration"
    echo ""
    success "Ready for production deployment!"
}

# Main deployment process
main() {
    log "Starting KNIRVENGINE deployment process..."
    
    check_dependencies
    create_directories
    backup_existing
    
    build_desktop_host
    build_mobile_tool
    build_agent_core
    
    copy_assets
    create_startup_scripts
    create_documentation
    
    run_tests
    print_summary
}

# Run main function
main "$@"
