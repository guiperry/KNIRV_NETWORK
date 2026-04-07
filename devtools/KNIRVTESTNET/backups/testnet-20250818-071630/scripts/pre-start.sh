#!/bin/bash
# Pre-start script for KNIRV Testnet
# Ensures all toolchains are available before starting

echo "🚀 KNIRV Testnet Pre-Start Check"
echo "================================"

# Load environment
if [ -f "scripts/load-env.sh" ]; then
    source scripts/load-env.sh
fi

# Check if toolchains are available
GO_AVAILABLE=false
RUST_AVAILABLE=false

if command -v go &> /dev/null; then
    echo "✅ Go toolchain found: $(go version)"
    GO_AVAILABLE=true
else
    echo "❌ Go toolchain not found"
fi

if command -v rustc &> /dev/null; then
    echo "✅ Rust toolchain found: $(rustc --version)"
    RUST_AVAILABLE=true
else
    echo "❌ Rust toolchain not found"
fi

# Install toolchains if missing
if [ "$GO_AVAILABLE" = false ] || [ "$RUST_AVAILABLE" = false ]; then
    echo "🔧 Installing missing toolchains..."
    
    if [ -f "scripts/install-deps.sh" ]; then
        bash scripts/install-deps.sh
        
        # Reload environment after installation
        source scripts/load-env.sh
        
        # Verify installation
        if command -v go &> /dev/null; then
            echo "✅ Go toolchain now available: $(go version)"
        else
            echo "❌ Go toolchain still not found after installation"
            exit 1
        fi
        
        if command -v rustc &> /dev/null; then
            echo "✅ Rust toolchain now available: $(rustc --version)"
        else
            echo "❌ Rust toolchain still not found after installation"
            exit 1
        fi
    else
        echo "❌ install-deps.sh not found"
        exit 1
    fi
fi

echo "✅ All toolchains are available"
echo "🚀 Ready to start KNIRV Testnet"
