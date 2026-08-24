#!/bin/bash

# Build script for KNIRVENGINE Desktop Application
# This script builds native Go binaries with embedded frontend

set -e

echo "🚀 Building KNIRVENGINE Desktop Application (Native Go)..."

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

# Build the Go backend with embedded frontend
build_backend() {
    print_status "Building Go backend with embedded frontend..."

    cd "$PROJECT_ROOT"

    # Determine the target platform
    PLATFORM=${1:-$(uname -s | tr '[:upper:]' '[:lower:]')}

    case $PLATFORM in
        linux)
            GOOS=linux GOARCH=amd64 go build -tags embed -o knirv-engine-linux .
            ;;
        darwin|macos)
            GOOS=darwin GOARCH=amd64 go build -tags embed -o knirv-engine-macos .
            ;;
        windows|win32)
            GOOS=windows GOARCH=amd64 go build -tags embed -o knirv-engine-windows.exe .
            ;;
        all)
            # Build for all platforms
            GOOS=linux GOARCH=amd64 go build -tags embed -o knirv-engine-linux .
            GOOS=darwin GOARCH=amd64 go build -tags embed -o knirv-engine-macos .
            GOOS=windows GOARCH=amd64 go build -tags embed -o knirv-engine-windows.exe .
            ;;
        *)
            # Build for current platform
            go build -tags embed -o knirv-engine .
            ;;
    esac

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

# Create distribution packages
create_packages() {
    print_status "Creating distribution packages..."

    cd "$PROJECT_ROOT"

    # Create dist directory if it doesn't exist
    mkdir -p dist

    # Package binaries based on what was built
    if [ -f "knirv-engine-linux" ]; then
        print_status "Packaging Linux binary..."
        tar -czf dist/knirv-engine-linux-amd64.tar.gz knirv-engine-linux
    fi

    if [ -f "knirv-engine-macos" ]; then
        print_status "Packaging macOS binary..."
        tar -czf dist/knirv-engine-macos-amd64.tar.gz knirv-engine-macos
    fi

    if [ -f "knirv-engine-windows.exe" ]; then
        print_status "Packaging Windows binary..."
        zip -q dist/knirv-engine-windows-amd64.zip knirv-engine-windows.exe
    fi

    if [ -f "knirv-engine" ]; then
        print_status "Packaging current platform binary..."
        tar -czf dist/knirv-engine-current.tar.gz knirv-engine
    fi

    print_success "Distribution packages created in dist/"
}

# Build the four native engine executables distributed alongside desktop
# releases. macOS has separate Intel and Apple Silicon binaries.
build_platform_binaries() {
    print_status "Building cross-platform KNIRVENGINE executables..."
    cd "$PROJECT_ROOT"
    mkdir -p dist

    GOCACHE=/tmp/knirvengine-go-build CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags embed -o dist/knirv-engine-linux-amd64 .
    GOCACHE=/tmp/knirvengine-go-build CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags embed -o dist/knirv-engine-windows-amd64.exe .
    GOCACHE=/tmp/knirvengine-go-build CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -tags embed -o dist/knirv-engine-macos-amd64 .
    GOCACHE=/tmp/knirvengine-go-build CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -tags embed -o dist/knirv-engine-macos-arm64 .

    print_success "Created cross-platform executables:"
    print_success "  dist/knirv-engine-linux-amd64"
    print_success "  dist/knirv-engine-windows-amd64.exe"
    print_success "  dist/knirv-engine-macos-amd64"
    print_success "  dist/knirv-engine-macos-arm64"
}

# Development mode
dev_mode() {
    print_status "Starting development mode..."

    # Build backend for development
    cd "$PROJECT_ROOT"
    print_status "Building backend for development..."
    go build -o knirv-engine .

    # Start backend in background
    print_status "Starting backend server..."
    ./knirv-engine &
    BACKEND_PID=$!

    # Wait a moment for backend to start
    sleep 2

    # Start frontend dev server
    cd "$PROJECT_ROOT/gui"
    print_status "Starting frontend dev server..."
    print_status "Frontend will be available at http://localhost:8080"
    print_status "Backend API will be available at http://localhost:8081"
    print_status "Press Ctrl+C to stop both servers"

    # Start frontend dev server (this will block)
    npm run dev

    # Cleanup when frontend exits
    print_status "Cleaning up..."
    kill $BACKEND_PID 2>/dev/null || true
}

# Main function
main() {
    case ${1:-build} in
        dev|development)
            check_dependencies
            build_frontend
            dev_mode
            ;;
        build)
            check_dependencies
            print_status "Packaging the native desktop window..."
            cd "$PROJECT_ROOT/gui"
            npm run desktop:package
            print_success "Desktop application package created in dist/release/"
            ;;
        backend)
            check_dependencies
            build_backend ${2:-}
            ;;
        binaries|all-binaries)
            check_dependencies
            build_frontend
            build_platform_binaries
            ;;
        frontend)
            check_dependencies
            build_frontend
            ;;
        clean)
            print_status "Cleaning build artifacts..."
            rm -f "$PROJECT_ROOT/knirv-engine"*
            rm -f "$PROJECT_ROOT/knirv-engine.exe"
            rm -rf "$PROJECT_ROOT/gui/dist"
            rm -rf "$PROJECT_ROOT/dist"
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
            echo "  binaries          - Build Linux, Windows, and macOS executables into dist/"
            echo "  frontend          - Build only the React frontend"
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
