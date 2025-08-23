#!/bin/bash

# KNIRV Controller Manager-Receiver Merge Runner
# This script prepares and runs the backup-and-merge process

set -e

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

# Check if we're in the right directory
if [ ! -f "package.json" ] || [ ! -d "manager" ] || [ ! -d "receiver" ] || [ ! -d "backend" ]; then
    print_error "This script must be run from the KNIRVCONTROLLER directory"
    print_error "Expected structure: KNIRVCONTROLLER/manager, KNIRVCONTROLLER/receiver, and KNIRVCONTROLLER/backend"
    exit 1
fi

print_status "Starting KNIRV Controller Manager-Receiver merge and root export process..."

# Check if Node.js is installed
if ! command -v node &> /dev/null; then
    print_error "Node.js is not installed. Please install Node.js 20+ to continue."
    exit 1
fi

# Check Node.js version
NODE_VERSION=$(node --version | cut -d'v' -f2 | cut -d'.' -f1)
if [ "$NODE_VERSION" -lt 20 ]; then
    print_error "Node.js version 20 or higher is required. Current version: $(node --version)"
    exit 1
fi

print_success "Node.js version check passed: $(node --version)"

# Check if npm is available
if ! command -v npm &> /dev/null; then
    print_error "npm is not installed. Please install npm to continue."
    exit 1
fi

# Confirm with user before proceeding
echo
print_warning "This script will:"
echo "  1. Create timestamped backups of manager, receiver, and existing root frontend"
echo "  2. Merge manager functionality into receiver"
echo "  3. Export merged application to root ./frontend/ directory"
echo "  4. Update backend configuration to serve from root frontend"
echo "  5. Update root package.json scripts for unified development"
echo "  6. Install dependencies for the unified application"
echo
read -p "Do you want to continue? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    print_status "Operation cancelled by user."
    exit 0
fi

# Make the merge script executable
chmod +x scripts/backup-and-merge-manager.js

# Run the merge script
print_status "Running backup and merge script..."
if node scripts/backup-and-merge-manager.js; then
    print_success "Merge and export completed successfully!"
    echo
    print_status "Next steps:"
    echo "  1. Review the unified application in the ./frontend/ directory"
    echo "  2. Test the unified application: npm run dev"
    echo "  3. Or build and start production: npm run build:unified && npm start"
    echo "  4. Navigate between interfaces:"
    echo "     - Receiver Interface: http://localhost:3000/"
    echo "     - Manager Interface: http://localhost:3000/manager"
    echo "     - Health Check: http://localhost:3000/health"
    echo "     - Backend API: http://localhost:3000/api"
    echo
    print_status "Backups are stored in the 'backups' directory"
    print_status "Original receiver and manager directories are preserved"
else
    print_error "Merge failed. Check the error messages above."
    print_status "Your original applications are safe - backups were created before any changes."
    exit 1
fi

# Optional: Run a quick test
echo
read -p "Would you like to run a quick test of the unified application? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    print_status "Starting unified development server for testing..."
    print_status "The application will be available at http://localhost:3000"
    print_status "Press Ctrl+C to stop the server"
    echo
    npm run dev
fi

print_success "Script completed successfully!"
