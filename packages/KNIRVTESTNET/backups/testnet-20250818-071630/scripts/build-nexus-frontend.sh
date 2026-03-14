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

# Navigate to KNIRVNEXUS
cd ../KNIRVNEXUS

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
cp -r ../KNIRVNEXUS/.next data/knirvnexus/portal/
cp -r ../KNIRVNEXUS/public data/knirvnexus/portal/
cp ../KNIRVNEXUS/package.json data/knirvnexus/portal/
cp ../KNIRVNEXUS/next.config.ts data/knirvnexus/portal/

# Copy server.js for custom server
cp ../KNIRVNEXUS/server.js data/knirvnexus/portal/

# Copy socket compilation output
if [ -d "../KNIRVNEXUS/dist" ]; then
    cp -r ../KNIRVNEXUS/dist data/knirvnexus/portal/
fi

echo "✅ NEXUS frontend build completed successfully!"
echo "📁 Frontend available at: ./data/knirvnexus/portal/"
echo "🌐 Will be served on port 8083 (nexus gui_port)"
