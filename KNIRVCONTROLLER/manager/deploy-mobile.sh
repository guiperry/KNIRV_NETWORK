#!/bin/bash

# KNIRV Mobile Tool - Mobile Deployment Script
# Provides multiple options for mobile testing and deployment

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# Functions
log() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

error() {
    echo -e "${RED}❌ $1${NC}"
    exit 1
}

info() {
    echo -e "${PURPLE}📱 $1${NC}"
}

# Get local IP address
get_local_ip() {
    # Try different methods to get local IP
    local ip=""
    
    # Method 1: ip route (Linux)
    if command -v ip &> /dev/null; then
        ip=$(ip route get 1.1.1.1 | grep -oP 'src \K\S+' 2>/dev/null || true)
    fi
    
    # Method 2: ifconfig (macOS/Linux)
    if [ -z "$ip" ] && command -v ifconfig &> /dev/null; then
        ip=$(ifconfig | grep -E "inet.*broadcast" | awk '{print $2}' | head -1 2>/dev/null || true)
    fi
    
    # Method 3: hostname (fallback)
    if [ -z "$ip" ] && command -v hostname &> /dev/null; then
        ip=$(hostname -I | awk '{print $1}' 2>/dev/null || true)
    fi
    
    # Default fallback
    if [ -z "$ip" ]; then
        ip="localhost"
    fi
    
    echo "$ip"
}

# Check if Capacitor is installed
check_capacitor() {
    if ! command -v npx &> /dev/null; then
        error "npx is not installed. Please install Node.js"
    fi
    
    if ! npm list @capacitor/cli &> /dev/null; then
        log "Installing Capacitor CLI..."
        npm install @capacitor/cli @capacitor/core @capacitor/android @capacitor/ios
    fi
}

# Option 1: Local Network Development Server
start_network_server() {
    log "Starting mobile-controller for local network access..."
    
    local ip=$(get_local_ip)
    
    info "Mobile Tool will be accessible at:"
    echo "  📱 Local: http://localhost:5173"
    echo "  🌐 Network: http://$ip:5173"
    echo ""
    info "To test on your mobile device:"
    echo "  1. Connect your mobile device to the same WiFi network"
    echo "  2. Open your mobile browser"
    echo "  3. Navigate to: http://$ip:5173"
    echo "  4. Add to home screen for app-like experience"
    echo ""
    warning "Make sure your firewall allows connections on port 5173"
    echo ""
    
    # Start the development server
    npm run dev
}

# Option 2: Build and serve static files
build_and_serve() {
    log "Building mobile-controller for production..."
    
    npm run build
    
    if [ ! -d "dist" ]; then
        error "Build failed - dist directory not found"
    fi
    
    success "Build completed successfully"
    
    local ip=$(get_local_ip)
    
    info "Starting static file server..."
    echo "  📱 Local: http://localhost:8080"
    echo "  🌐 Network: http://$ip:8080"
    echo ""
    
    # Start a simple HTTP server
    if command -v python3 &> /dev/null; then
        cd dist && python3 -m http.server 8080 --bind 0.0.0.0
    elif command -v python &> /dev/null; then
        cd dist && python -m SimpleHTTPServer 8080
    elif command -v npx &> /dev/null; then
        npx serve dist -l 8080 -s
    else
        error "No suitable HTTP server found. Install Python or Node.js serve package"
    fi
}

# Option 3: Capacitor Android App
build_android_app() {
    log "Building Capacitor Android app..."
    
    check_capacitor
    
    # Build the web app first
    npm run build
    
    # Initialize Capacitor if not already done
    if [ ! -f "capacitor.config.ts" ]; then
        error "Capacitor config not found. Run this script from the mobile-controller directory"
    fi
    
    # Add Android platform if not already added
    if [ ! -d "android" ]; then
        log "Adding Android platform..."
        npx cap add android
    fi
    
    # Copy web assets to native project
    log "Syncing web assets..."
    npx cap sync android
    
    # Copy web assets and update native project
    npx cap copy android
    
    success "Android app prepared successfully"
    
    info "To continue with Android development:"
    echo "  1. Install Android Studio"
    echo "  2. Open the android/ directory in Android Studio"
    echo "  3. Connect your Android device or start an emulator"
    echo "  4. Click 'Run' to install the app"
    echo ""
    echo "Or run: npx cap run android"
}

# Option 4: Capacitor iOS App
build_ios_app() {
    log "Building Capacitor iOS app..."
    
    if [[ "$OSTYPE" != "darwin"* ]]; then
        error "iOS development requires macOS"
    fi
    
    check_capacitor
    
    # Build the web app first
    npm run build
    
    # Add iOS platform if not already added
    if [ ! -d "ios" ]; then
        log "Adding iOS platform..."
        npx cap add ios
    fi
    
    # Copy web assets to native project
    log "Syncing web assets..."
    npx cap sync ios
    
    # Copy web assets and update native project
    npx cap copy ios
    
    success "iOS app prepared successfully"
    
    info "To continue with iOS development:"
    echo "  1. Install Xcode"
    echo "  2. Open the ios/ directory in Xcode"
    echo "  3. Connect your iOS device or start a simulator"
    echo "  4. Click 'Run' to install the app"
    echo ""
    echo "Or run: npx cap run ios"
}

# Option 5: PWA (Progressive Web App)
setup_pwa() {
    log "Setting up Progressive Web App (PWA)..."
    
    # Build the app
    npm run build
    
    # Create PWA manifest if it doesn't exist
    if [ ! -f "public/manifest.json" ]; then
        log "Creating PWA manifest..."
        cat > public/manifest.json << 'EOF'
{
  "name": "KNIRV Mobile Tool",
  "short_name": "KNIRV Mobile",
  "description": "Enhanced mobile client for KNIRV cognitive processing",
  "start_url": "/",
  "display": "standalone",
  "background_color": "#1e293b",
  "theme_color": "#8b5cf6",
  "orientation": "portrait",
  "icons": [
    {
      "src": "/icon-192.png",
      "sizes": "192x192",
      "type": "image/png"
    },
    {
      "src": "/icon-512.png",
      "sizes": "512x512",
      "type": "image/png"
    }
  ]
}
EOF
    fi
    
    success "PWA setup completed"
    
    info "PWA Features:"
    echo "  • Install as app on mobile devices"
    echo "  • Offline capability (when implemented)"
    echo "  • Native app-like experience"
    echo "  • Push notifications (when implemented)"
}

# Option 6: Netlify/Vercel Deployment
deploy_to_netlify() {
    log "Preparing for Netlify deployment..."
    
    # Build the app
    npm run build
    
    # Create netlify.toml if it doesn't exist
    if [ ! -f "netlify.toml" ]; then
        log "Creating Netlify configuration..."
        cat > netlify.toml << 'EOF'
[build]
  publish = "dist"
  command = "npm run build"

[[redirects]]
  from = "/*"
  to = "/index.html"
  status = 200

[build.environment]
  NODE_VERSION = "18"

[[headers]]
  for = "/*"
  [headers.values]
    X-Frame-Options = "DENY"
    X-XSS-Protection = "1; mode=block"
    X-Content-Type-Options = "nosniff"
    Referrer-Policy = "strict-origin-when-cross-origin"
EOF
    fi
    
    success "Netlify configuration created"
    
    info "To deploy to Netlify:"
    echo "  1. Install Netlify CLI: npm install -g netlify-cli"
    echo "  2. Login: netlify login"
    echo "  3. Deploy: netlify deploy --prod --dir=dist"
    echo ""
    echo "Or connect your GitHub repo to Netlify for automatic deployments"
}

# Main menu
show_menu() {
    echo ""
    echo "📱 KNIRV Mobile Tool - Deployment Options"
    echo "========================================"
    echo ""
    echo "Choose a deployment option:"
    echo ""
    echo "1) 🌐 Local Network Server (Development)"
    echo "   - Access from any device on your network"
    echo "   - Hot reload and development features"
    echo ""
    echo "2) 📦 Build & Serve (Production)"
    echo "   - Optimized production build"
    echo "   - Static file server"
    echo ""
    echo "3) 🤖 Android App (Capacitor)"
    echo "   - Native Android application"
    echo "   - Full device capabilities"
    echo ""
    echo "4) 🍎 iOS App (Capacitor)"
    echo "   - Native iOS application"
    echo "   - Requires macOS and Xcode"
    echo ""
    echo "5) 📱 Progressive Web App (PWA)"
    echo "   - Installable web app"
    echo "   - App-like experience"
    echo ""
    echo "6) ☁️  Netlify Deployment"
    echo "   - Cloud hosting"
    echo "   - Global CDN"
    echo ""
    echo "0) Exit"
    echo ""
}

# Main execution
main() {
    # Check if we're in the right directory
    if [ ! -f "package.json" ]; then
        error "Please run this script from the mobile-controller directory"
    fi
    
    # Install dependencies if needed
    if [ ! -d "node_modules" ]; then
        log "Installing dependencies..."
        npm install
    fi
    
    # Show menu if no arguments provided
    if [ $# -eq 0 ]; then
        show_menu
        read -p "Enter your choice (0-6): " choice
    else
        choice=$1
    fi
    
    case $choice in
        1)
            start_network_server
            ;;
        2)
            build_and_serve
            ;;
        3)
            build_android_app
            ;;
        4)
            build_ios_app
            ;;
        5)
            setup_pwa
            build_and_serve
            ;;
        6)
            deploy_to_netlify
            ;;
        0)
            echo "Goodbye!"
            exit 0
            ;;
        *)
            error "Invalid choice. Please select 0-6."
            ;;
    esac
}

# Run main function
main "$@"
