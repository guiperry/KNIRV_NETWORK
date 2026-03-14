#!/bin/bash
set -e

echo "🚀 Setting up Capacitor for KNIRVCONTROLLER..."

# Color codes for output
BLUE='\033[0;34m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_status "Ensuring all dependencies are installed..."
npm install --legacy-peer-deps

print_status "Initializing Capacitor..."
npx cap init "KNIRV Controller" "com.knirv.controller" --web-dir "dist"

print_status "Adding native platforms (iOS and Android)..."
npx cap add ios
npx cap add android

print_status "Building the web app for the first time..."
npm run build

print_status "Performing initial sync with native platforms..."
npx cap sync

print_success "✅ Capacitor setup complete!"
echo "You can now run your app on a native device using:"
echo "  - npx cap open ios"
echo "  - npx cap open android"