#!/bin/bash
set -e

echo "Building KNIRV-NEXUS Frontend for testnet..."

# Create necessary directories
mkdir -p data/knirvserver/portal

# Check if KNIRVSERVER directory exists
if [ ! -d "../KNIRVSERVER" ]; then
    echo "❌ KNIRVSERVER directory not found. Please ensure KNIRVSERVER is in the parent directory."
    exit 1
fi

# Navigate to KNIRVSERVER
cd ../KNIRVSERVER

echo "Cleaning NEXUS frontend dependencies..."
rm -rf node_modules package-lock.json

echo "Installing NEXUS frontend dependencies..."
npm install

echo "Building NEXUS frontend..."
npm run build

if [ $? -ne 0 ]; then
    echo "❌ NEXUS frontend build failed"
    exit 1
fi

# Copy build output to testnet
echo "Copying NEXUS frontend build to testnet..."
cd ../KNIRVTESTNET

# Copy Next.js build output
cp -r ../KNIRVSERVER/.next data/knirvserver/portal/
cp -r ../KNIRVSERVER/public data/knirvserver/portal/
cp ../KNIRVSERVER/package.json data/knirvserver/portal/
cp ../KNIRVSERVER/next.config.ts data/knirvserver/portal/

# Copy server.js for custom server
cp ../KNIRVSERVER/server.js data/knirvserver/portal/

# Copy socket compilation output
if [ -d "../KNIRVSERVER/dist" ]; then
    cp -r ../KNIRVSERVER/dist data/knirvserver/portal/
fi

echo "✅ NEXUS frontend build completed successfully!"
echo "📁 Frontend available at: ./data/knirvserver/portal/"
echo "🌐 Will be served on port 8083 (nexus gui_port)"
