#!/bin/bash

# Revolutionary KNIRVORACLE WASM Build Script
# Compiles the Rust KNIRVORACLE to WASM for embedding in KNIRVROUTER

set -e

echo "🚀 Building Revolutionary KNIRVORACLE WASM..."

# Check if wasm-pack is installed
if ! command -v wasm-pack &> /dev/null; then
    echo "❌ wasm-pack not found. Installing..."
    curl https://rustwasm.github.io/wasm-pack/installer/init.sh -sSf | sh
fi

# Check if target is added
if ! rustup target list --installed | grep -q "wasm32-unknown-unknown"; then
    echo "📦 Adding wasm32-unknown-unknown target..."
    rustup target add wasm32-unknown-unknown
fi

# Clean previous builds
echo "🧹 Cleaning previous builds..."
rm -rf pkg/
rm -rf target/wasm32-unknown-unknown/

# Build WASM package using WASM-specific configuration
echo "⚡ Building WASM package..."

# Backup original Cargo.toml and use WASM-specific one
cp Cargo.toml Cargo.toml.backup
cp Cargo-wasm.toml Cargo.toml

# Build WASM
wasm-pack build --target web

# Restore original Cargo.toml
mv Cargo.toml.backup Cargo.toml

# Check if build was successful
if [ -f "pkg/knirvchain_wasm_bg.wasm" ]; then
    echo "✅ WASM build successful!"
    echo "📦 WASM file: pkg/knirvchain_wasm_bg.wasm"
    echo "📄 TypeScript bindings: pkg/knirvchain_wasm.d.ts"
    echo "🔧 JavaScript bindings: pkg/knirvchain_wasm.js"

    # Show file sizes
    echo ""
    echo "📊 Build artifacts:"
    ls -lh pkg/knirvchain_wasm_bg.wasm
    ls -lh pkg/knirvchain_wasm.js
    ls -lh pkg/knirvchain_wasm.d.ts

    # Copy WASM file to KNIRVROUTER assets
    echo ""
    echo "📋 Copying WASM to KNIRVROUTER assets..."
    mkdir -p ../KNIRVROUTER/assets/wasm/
    cp pkg/knirvchain_wasm_bg.wasm ../KNIRVROUTER/assets/wasm/knirvchain.wasm
    cp pkg/knirvchain_wasm.js ../KNIRVROUTER/assets/wasm/knirvchain.js
    cp pkg/knirvchain_wasm.d.ts ../KNIRVROUTER/assets/wasm/knirvchain.d.ts

    echo "✅ WASM files copied to KNIRVROUTER/assets/wasm/"
    
else
    echo "❌ WASM build failed!"
    exit 1
fi

echo ""
echo "🎉 Revolutionary KNIRVORACLE WASM build complete!"
echo "🔗 Ready for embedding in KNIRVROUTER"
