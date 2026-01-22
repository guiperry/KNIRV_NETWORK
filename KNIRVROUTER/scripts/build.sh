#!/bin/bash

# Build script for KNIRVROUTER
# This script should be run from the scripts directory

echo "Building KNIRVROUTER..."

# Build all versions
echo "Building main GUI version..."
go build -o ../bin/knirvrouter ../cmd/knirvrouter/

if [ $? -eq 0 ]; then
    echo "Build successful! Binary created: ../bin/knirvrouter"
    echo ""
    echo "Usage:"
    echo "  ../bin/knirvrouter             # Start the desktop GUI (default)"
    echo "  ../bin/knirvrouter -chain      # Start in blockchain mode"
    echo "  ../bin/knirvrouter -wallet     # Start in wallet mode"
    echo ""
    echo "Making the binary executable..."
    chmod +x ../bin/knirvrouter
    echo "Done!"
else
    echo "Build failed!"
fi