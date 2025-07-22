#!/bin/bash

# Build script for KNIRVCHAIN

echo "Building KNIRVCHAIN..."
go build -o KNIRVCHAIN

if [ $? -eq 0 ]; then
    echo "Build successful! Binary created: KNIRVCHAIN"
    echo ""
    echo "Usage:"
    echo "  ./KNIRVCHAIN             # Start the desktop GUI (default)"
    echo "  ./KNIRVCHAIN -webgui     # Start the web GUI"
    echo "  ./KNIRVCHAIN -port 5001  # Start the web GUI on a custom port"
    echo "  ./KNIRVCHAIN -chain      # Start in blockchain mode"
    echo "  ./KNIRVCHAIN -wallet     # Start in wallet mode"
    echo ""
    echo "Making the binary executable..."
    chmod +x KNIRVCHAIN
    echo "Done!"
else
    echo "Build failed!"
fi