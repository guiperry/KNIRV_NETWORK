#!/bin/bash

# Production build and run script for KNIRVENGINE
# This script rebuilds the frontend, backend, repackages the Electron app, and runs it

set -e  # Exit on any error

echo "Starting production build process..."

# Sync environment variables
#echo "Syncing environment variables..."
#./scripts/sync-env.sh


# Rebuild frontend
echo "Building frontend..."
cd gui && npm run build

# Rebuild backend (if needed)
echo "Building backend..."
cd .. && go build -o knirv-engine .

# Repackage Electron app
echo "Packaging Electron app..."
cd electron && npm run pack:linux

# Then run
echo "Starting KNIRVENGINE desktop application..."
./dist/linux-unpacked/knirv-engine-desktop
