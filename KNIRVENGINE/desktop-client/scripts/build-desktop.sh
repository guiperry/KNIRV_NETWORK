#!/bin/bash

# Build script for KNIRVENGINE Desktop Application
# This script builds both the Go backend and Electron frontend

set -e

echo "🚀 Building KNIRVENGINE Desktop Application..."

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
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

# Check if required tools are installed
check_dependencies() {
    print_status "Checking dependencies..."
    
    # Check Go
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go 1.21 or later."
        exit 1
    fi
    
    # Check Node.js
    if ! command -v node &> /dev/null; then
        print_error "Node.js is not installed. Please install Node.js 18 or later."
        exit 1
    fi
    
    # Check npm
    if ! command -v npm &> /dev/null; then
        print_error "npm is not installed. Please install npm."
        exit 1
    fi
    
    print_success "All dependencies are available"
}

# Build the Go backend
build_backend() {
    print_status "Building Go backend..."
    
    cd "$PROJECT_ROOT"
    
    # Build for current platform
    go build -o knirv-engine .
    
    if [ $? -eq 0 ]; then
        print_success "Backend built successfully"
    else
        print_error "Failed to build backend"
        exit 1
    fi
}

# Build the React frontend
build_frontend() {
    print_status "Building React frontend..."
    
    cd "$PROJECT_ROOT/gui"
    
    # Install dependencies if node_modules doesn't exist
    if [ ! -d "node_modules" ]; then
        print_status "Installing frontend dependencies..."
        npm install
    fi
    
    # Build the frontend
    npm run build
    
    if [ $? -eq 0 ]; then
        print_success "Frontend built successfully"
    else
        print_error "Failed to build frontend"
        exit 1
    fi
}

# Setup Electron
setup_electron() {
    print_status "Setting up Electron..."
    
    cd "$PROJECT_ROOT/electron"
    
    # Install Electron dependencies if node_modules doesn't exist
    if [ ! -d "node_modules" ]; then
        print_status "Installing Electron dependencies..."
        npm install
    fi
    
    print_success "Electron setup complete"
}

# Build Electron app
build_electron() {
    print_status "Building Electron application..."
    
    cd "$PROJECT_ROOT/electron"
    
    # Determine the target platform
    PLATFORM=${1:-$(uname -s | tr '[:upper:]' '[:lower:]')}
    
    case $PLATFORM in
        linux)
            npm run build:linux
            ;;
        darwin|macos)
            npm run build:mac
            ;;
        windows|win32)
            npm run build:win
            ;;
        all)
            npm run build
            ;;
        *)
            print_warning "Unknown platform: $PLATFORM. Building for current platform..."
            npm run build
            ;;
    esac
    
    if [ $? -eq 0 ]; then
        print_success "Electron application built successfully"
    else
        print_error "Failed to build Electron application"
        exit 1
    fi
}

# Development mode
dev_mode() {
    print_status "Starting development mode..."
    
    # Start backend in background
    cd "$PROJECT_ROOT"
    print_status "Starting backend server..."
    ./knirv-engine &
    BACKEND_PID=$!
    
    # Wait a moment for backend to start
    sleep 2
    
    # Start frontend dev server in background
    cd "$PROJECT_ROOT/gui"
    print_status "Starting frontend dev server..."
    npm run dev &
    FRONTEND_PID=$!
    
    # Wait a moment for frontend to start
    sleep 3
    
    # Start Electron in development mode
    cd "$PROJECT_ROOT/electron"
    print_status "Starting Electron in development mode..."
    npm run dev
    
    # Cleanup when Electron exits
    print_status "Cleaning up..."
    kill $BACKEND_PID 2>/dev/null || true
    kill $FRONTEND_PID 2>/dev/null || true
}

# Main function
main() {
    case ${1:-build} in
        dev|development)
            check_dependencies
            build_backend
            setup_electron
            dev_mode
            ;;
        build)
            check_dependencies
            build_backend
            build_frontend
            setup_electron
            build_electron ${2:-}
            ;;
        backend)
            check_dependencies
            build_backend
            ;;
        frontend)
            check_dependencies
            build_frontend
            ;;
        electron)
            check_dependencies
            setup_electron
            build_electron ${2:-}
            ;;
        clean)
            print_status "Cleaning build artifacts..."
            rm -f "$PROJECT_ROOT/knirv-engine"
            rm -f "$PROJECT_ROOT/knirv-engine.exe"
            rm -rf "$PROJECT_ROOT/gui/dist"
            rm -rf "$PROJECT_ROOT/electron/dist"
            rm -rf "$PROJECT_ROOT/electron/node_modules"
            rm -rf "$PROJECT_ROOT/gui/node_modules"
            print_success "Clean complete"
            ;;
        help|--help|-h)
            echo "Usage: $0 [command] [platform]"
            echo ""
            echo "Commands:"
            echo "  build [platform]  - Build the complete desktop application (default)"
            echo "  dev               - Start development mode with hot reload"
            echo "  backend           - Build only the Go backend"
            echo "  frontend          - Build only the React frontend"
            echo "  electron          - Build only the Electron wrapper"
            echo "  clean             - Clean all build artifacts"
            echo "  help              - Show this help message"
            echo ""
            echo "Platforms (for build command):"
            echo "  linux             - Build for Linux"
            echo "  darwin|macos      - Build for macOS"
            echo "  windows|win32     - Build for Windows"
            echo "  all               - Build for all platforms"
            echo ""
            echo "Examples:"
            echo "  $0 build linux    - Build for Linux"
            echo "  $0 dev            - Start development mode"
            echo "  $0 clean          - Clean build artifacts"
            ;;
        *)
            print_error "Unknown command: $1"
            echo "Use '$0 help' for usage information."
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"
