#!/bin/bash
set -e

echo "Building KNIRV-NEXUS Frontend for testnet..."

# Create necessary directories
mkdir -p data/knirvnexus/portal

# Check if KNIRVNEXUS directory exists
if [ ! -d "../KNIRVNEXUS" ]; then
    echo "❌ KNIRVNEXUS directory not found. Please ensure KNIRVNEXUS is in the parent directory."
    exit 1
fi

echo "Copying NEXUS source files to portal directory..."
# Copy source files to portal directory
cp -r ../KNIRVNEXUS/* data/knirvnexus/portal/ 2>/dev/null || true

# Navigate to portal directory
cd data/knirvnexus/portal

echo "Cleaning NEXUS frontend dependencies..."
rm -rf node_modules package-lock.json

echo "Installing NEXUS frontend dependencies in portal directory..."
npm install

echo "Building NEXUS frontend..."
npm run build

if [ $? -ne 0 ]; then
    echo "❌ NEXUS frontend build failed"
    exit 1
fi

echo "✅ NEXUS frontend build completed successfully!"
echo "📁 Frontend available at: ./data/knirvnexus/portal/"
echo "🌐 Will be served on port 8083 (nexus gui_port)"
echo "📦 Dependencies installed in portal directory"
