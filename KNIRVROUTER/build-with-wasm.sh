#!/bin/bash

# Build script for KNIRVROUTER with optional WASM support
set -e

echo "🚀 Building KNIRVROUTER with Revolutionary WASM Support..."

# Check if WASM support should be enabled
ENABLE_WASM=${1:-"false"}

if [ "$ENABLE_WASM" = "true" ] || [ "$ENABLE_WASM" = "1" ]; then
    echo "✅ WASM support ENABLED"
    
    # Note: Wasmer-go dependency will be installed automatically if needed
    echo "📦 WASM loader will use fallback implementation if wasmer-go is not available"
    
    # Build with WASM support (using headless mode)
    echo "⚡ Building KNIRVROUTER with WASM support..."
    go build -tags "wasmloader,headless" -o knirvrouter-wasm .
    
    if [ -f "knirvrouter-wasm" ]; then
        echo "✅ KNIRVROUTER with WASM support built successfully!"
        echo "📦 Binary: knirvrouter-wasm"
        echo ""
        echo "🚀 To run with WASM support:"
        echo "   KNIRV_ENABLE_WASM=true ./knirvrouter-wasm chain --port=5000 --miners_address=test_miner"
        echo ""
        echo "🔗 WASM endpoints will be available at:"
        echo "   http://localhost:8082/wasm/invoke"
        echo "   http://localhost:8082/wasm/status"
        echo ""
        
        # Show file size
        ls -lh knirvrouter-wasm
    else
        echo "❌ Build failed!"
        exit 1
    fi
    
else
    echo "ℹ️  WASM support DISABLED"
    
    # Build without WASM support (fallback)
    echo "⚡ Building KNIRVROUTER without WASM support..."
    go build -o knirvrouter .
    
    if [ -f "knirvrouter" ]; then
        echo "✅ KNIRVROUTER built successfully!"
        echo "📦 Binary: knirvrouter"
        echo ""
        echo "🚀 To run:"
        echo "   ./knirvrouter chain --port=5000 --miners_address=test_miner"
        echo ""
        echo "🔗 Embedded chain endpoints will be available at:"
        echo "   http://localhost:8081/embedded-chain/invoke"
        echo ""
        echo "ℹ️  To enable WASM support, run:"
        echo "   ./build-with-wasm.sh true"
        echo ""
        
        # Show file size
        ls -lh knirvrouter
    else
        echo "❌ Build failed!"
        exit 1
    fi
fi

echo ""
echo "🎉 Build complete!"
