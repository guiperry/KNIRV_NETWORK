#!/bin/bash

# KNIRVCONTROLLER Root Consolidation Runner
# Executes the complete consolidation process with validation and error handling

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
print_status() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_header() {
    echo
    echo "=================================================="
    echo "  KNIRVCONTROLLER ROOT CONSOLIDATION"
    echo "=================================================="
    echo
}

# Check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    # Check Node.js version
    if ! command -v node &> /dev/null; then
        print_error "Node.js is not installed"
        exit 1
    fi
    
    NODE_VERSION=$(node --version | cut -d'v' -f2 | cut -d'.' -f1)
    if [ "$NODE_VERSION" -lt "20" ]; then
        print_error "Node.js 20+ required. Current version: $(node --version)"
        exit 1
    fi
    
    # Check npm
    if ! command -v npm &> /dev/null; then
        print_error "npm is not installed"
        exit 1
    fi
    
    # Check if we're in the right directory
    if [ ! -f "package.json" ] || [ ! -d "frontend" ] || [ ! -d "backend" ]; then
        print_error "Please run this script from the KNIRVCONTROLLER root directory"
        exit 1
    fi
    
    # Check for git (recommended for backup)
    if command -v git &> /dev/null; then
        if [ -d ".git" ]; then
            print_status "Git repository detected - checking for uncommitted changes..."
            if ! git diff-index --quiet HEAD --; then
                print_warning "You have uncommitted changes. Consider committing them before consolidation."
                read -p "Continue anyway? (y/N): " -n 1 -r
                echo
                if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                    print_status "Consolidation cancelled. Please commit your changes first."
                    exit 0
                fi
            fi
        fi
    fi
    
    print_success "Prerequisites check passed"
}

# Check disk space
check_disk_space() {
    print_status "Checking available disk space..."
    
    # Get available space in KB
    AVAILABLE_SPACE=$(df . | tail -1 | awk '{print $4}')
    REQUIRED_SPACE=1048576  # 1GB in KB
    
    if [ "$AVAILABLE_SPACE" -lt "$REQUIRED_SPACE" ]; then
        print_error "Insufficient disk space. At least 1GB required for consolidation."
        exit 1
    fi
    
    print_success "Sufficient disk space available"
}

# Stop any running processes
stop_running_processes() {
    print_status "Checking for running development servers..."
    
    # Check common ports
    PORTS=(3000 3001 3002 3003 5173 8080)
    FOUND_PROCESSES=false
    
    for port in "${PORTS[@]}"; do
        if lsof -i :$port &> /dev/null; then
            print_warning "Process running on port $port"
            FOUND_PROCESSES=true
        fi
    done
    
    if [ "$FOUND_PROCESSES" = true ]; then
        print_warning "Found running processes on development ports"
        read -p "Stop all processes and continue? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            for port in "${PORTS[@]}"; do
                PID=$(lsof -ti :$port)
                if [ ! -z "$PID" ]; then
                    print_status "Stopping process on port $port (PID: $PID)"
                    kill -9 $PID 2>/dev/null || true
                fi
            done
        else
            print_status "Please stop running processes manually and try again"
            exit 0
        fi
    fi
}

# Main consolidation function
run_consolidation() {
    print_status "Starting root consolidation process..."
    
    # Make scripts executable
    chmod +x scripts/*.js 2>/dev/null || true
    chmod +x scripts/*.sh 2>/dev/null || true
    
    # Step 1: Create configuration files
    print_status "Step 1: Creating unified configuration files..."
    if node scripts/create-root-configs.js; then
        print_success "Configuration files created"
    else
        print_error "Failed to create configuration files"
        exit 1
    fi
    
    # Step 2: Run main consolidation
    print_status "Step 2: Executing main consolidation..."
    if node scripts/consolidate-to-root.js; then
        print_success "Main consolidation completed"
    else
        print_error "Main consolidation failed"
        exit 1
    fi
    
    # Step 3: Install dependencies
    print_status "Step 3: Installing unified dependencies..."
    if npm install; then
        print_success "Dependencies installed"
    else
        print_error "Failed to install dependencies"
        exit 1
    fi
    
    # Step 4: Build WASM (if script exists)
    if [ -f "scripts/build-wasm.sh" ]; then
        print_status "Step 4: Building WASM components..."
        if bash scripts/build-wasm.sh; then
            print_success "WASM components built"
        else
            print_warning "WASM build failed (continuing anyway)"
        fi
    fi
    
    # Step 5: Test build
    print_status "Step 5: Testing build process..."
    if npm run build; then
        print_success "Build test successful"
    else
        print_error "Build test failed"
        exit 1
    fi
    
    print_success "Root consolidation completed successfully!"
}

# Validation function
validate_consolidation() {
    print_status "Validating consolidated structure..."
    
    # Check required files
    REQUIRED_FILES=(
        "src/main.tsx"
        "src/App.tsx"
        "src/index.css"
        "index.html"
        "vite.config.ts"
        "tsconfig.json"
        "package.json"
    )
    
    for file in "${REQUIRED_FILES[@]}"; do
        if [ ! -f "$file" ]; then
            print_error "Missing required file: $file"
            exit 1
        fi
    done
    
    # Check required directories
    REQUIRED_DIRS=(
        "src"
        "src/components"
        "src/backend"
        "dist"
    )
    
    for dir in "${REQUIRED_DIRS[@]}"; do
        if [ ! -d "$dir" ]; then
            print_error "Missing required directory: $dir"
            exit 1
        fi
    done
    
    # Test that the application starts
    print_status "Testing application startup..."
    timeout 30s npm run preview &
    PREVIEW_PID=$!
    
    sleep 10
    
    # Check if preview server is running
    if kill -0 $PREVIEW_PID 2>/dev/null; then
        print_success "Application starts successfully"
        kill $PREVIEW_PID
    else
        print_error "Application failed to start"
        exit 1
    fi
    
    print_success "Validation completed successfully"
}

# Cleanup function
cleanup_old_structure() {
    print_status "Cleaning up old structure (optional)..."
    
    read -p "Remove old frontend/, manager/, receiver/ directories? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_status "Removing old directories..."
        rm -rf frontend/ manager/ receiver/ 2>/dev/null || true
        print_success "Old directories removed"
    else
        print_status "Old directories preserved for reference"
    fi
}

# Main execution
main() {
    print_header
    
    print_status "This script will consolidate KNIRVCONTROLLER into a unified root structure"
    print_warning "This is a significant change that will modify your project structure"
    echo
    print_status "The process will:"
    echo "  1. Create comprehensive backups"
    echo "  2. Move all components to root src/ directory"
    echo "  3. Create unified configuration files"
    echo "  4. Update all import paths"
    echo "  5. Consolidate package.json dependencies"
    echo "  6. Test the consolidated structure"
    echo
    
    read -p "Do you want to continue? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_status "Consolidation cancelled by user"
        exit 0
    fi
    
    # Execute consolidation steps
    check_prerequisites
    check_disk_space
    stop_running_processes
    run_consolidation
    validate_consolidation
    cleanup_old_structure
    
    # Final success message
    echo
    print_success "🎉 KNIRVCONTROLLER root consolidation completed successfully!"
    echo
    print_status "Next steps:"
    echo "  1. Review the consolidated structure in src/"
    echo "  2. Test the application: npm start"
    echo "  3. Verify all functionality works as expected"
    echo
    print_status "Application access:"
    echo "  🌐 Frontend: http://localhost:3000/"
    echo "  🔧 API: http://localhost:3000/api (when backend is running)"
    echo
    print_status "Available commands:"
    echo "  npm start      - Start the unified application"
    echo "  npm run dev    - Development mode with hot reload"
    echo "  npm run build  - Build for production"
    echo "  npm test       - Run tests"
    echo
    
    # Offer to start the application
    read -p "Start the application now? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_status "Starting KNIRV Controller..."
        npm start
    else
        print_status "Run 'npm start' when ready to launch the application"
    fi
}

# Error handling
trap 'print_error "Script interrupted"; exit 1' INT TERM

# Run main function
main "$@"
