#!/bin/bash
set -e

# Render-specific startup script
# This script starts only the web server components for Render deployment

echo "🚀 KNIRV TESTNET - RENDER DEPLOYMENT"
echo "===================================="

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

# Check if we're running on Render
if [ "$RENDER" != "true" ] && [ -z "$RENDER_SERVICE_ID" ]; then
    print_warning "This script is designed for Render deployment"
    print_warning "For local development, use: npm run dev"
    print_warning "For full testnet, use: npm run testnet:start"
fi

# Set environment variables for Render
export NODE_ENV=production
export TESTNET_MODE=true
export PORT=${PORT:-10000}

print_status "Starting KNIRV Testnet Web Server for Render..."
print_status "Environment: $NODE_ENV"
print_status "Port: $PORT"

# Create necessary directories
mkdir -p logs data

# Load endpoints configuration
print_status "Loading endpoint configuration..."
npm run load-endpoints:testnet

# Start the main web server
print_status "Starting Express server..."
exec node server/app.js
