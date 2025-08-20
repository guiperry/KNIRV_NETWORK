#!/bin/bash

# KNIRV Testnet - Axios Corruption Fix Script
# This script fixes the corrupted axios installation that's missing dist/ directory

set -e

echo "🔧 KNIRV Testnet - Fixing Axios Corruption"
echo "=========================================="

# Change to KNIRVTESTNET root directory
cd "$(dirname "$0")/.."

echo "📍 Current directory: $(pwd)"
echo ""

# Function to check axios installation
check_axios() {
    echo "🔍 Checking axios installation..."
    
    if [ ! -d "node_modules/axios" ]; then
        echo "❌ axios not found in node_modules"
        return 1
    fi
    
    if [ ! -f "node_modules/axios/dist/node/axios.cjs" ]; then
        echo "❌ axios dist/node/axios.cjs missing (corrupted installation)"
        return 1
    fi
    
    echo "✅ axios installation appears healthy"
    return 0
}

# Function to fix axios corruption
fix_axios_corruption() {
    echo "🧹 Fixing axios corruption..."
    
    # Remove corrupted axios
    echo "🗑️  Removing corrupted axios installation..."
    rm -rf node_modules/axios
    
    # Clear npm cache for axios specifically
    echo "🗑️  Clearing axios from npm cache..."
    npm cache clean --force
    
    # Reinstall axios with known working version
    echo "📦 Reinstalling axios with stable version..."
    npm install axios@1.6.8 --save-exact
    
    # Verify the fix
    if check_axios; then
        echo "✅ Axios corruption fixed successfully!"
        return 0
    else
        echo "❌ Axios fix failed, trying alternative approach..."
        return 1
    fi
}

# Function to try alternative axios fix
alternative_axios_fix() {
    echo "🔄 Trying alternative axios fix..."
    
    # Remove entire node_modules and package-lock.json
    echo "🗑️  Removing all node_modules and package-lock.json..."
    rm -rf node_modules package-lock.json
    
    # Clear npm cache completely
    echo "🗑️  Clearing npm cache completely..."
    npm cache clean --force
    npm cache verify
    
    # Reinstall all dependencies
    echo "📦 Reinstalling all dependencies..."
    npm install
    
    # Verify the fix
    if check_axios; then
        echo "✅ Alternative axios fix successful!"
        return 0
    else
        echo "❌ Alternative axios fix failed"
        return 1
    fi
}

# Function to test axios functionality
test_axios() {
    echo "🧪 Testing axios functionality..."
    
    # Create a simple test script
    cat > /tmp/test-axios.js << 'EOF'
try {
    const axios = require('axios');
    console.log('✅ axios require() successful');
    console.log('📦 axios version:', axios.VERSION || 'unknown');
    process.exit(0);
} catch (error) {
    console.error('❌ axios require() failed:', error.message);
    process.exit(1);
}
EOF
    
    # Run the test
    if node /tmp/test-axios.js; then
        echo "✅ Axios functionality test passed!"
        rm -f /tmp/test-axios.js
        return 0
    else
        echo "❌ Axios functionality test failed!"
        rm -f /tmp/test-axios.js
        return 1
    fi
}

# Main execution
main() {
    echo "Starting axios corruption fix process..."
    echo ""

    # Check if we're already in a fix process to prevent recursion
    if [ "$KNIRV_AXIOS_FIX_IN_PROGRESS" = "true" ]; then
        echo "⚠️  Axios fix already in progress, preventing recursion"
        echo "✅ Skipping axios fix to avoid infinite loop"
        exit 0
    fi

    # Set flag to prevent recursion
    export KNIRV_AXIOS_FIX_IN_PROGRESS=true

    # Check current status
    if check_axios && test_axios; then
        echo "✅ Axios is already working correctly!"
        unset KNIRV_AXIOS_FIX_IN_PROGRESS
        exit 0
    fi
    
    echo ""
    echo "🔧 Axios corruption detected, attempting fix..."
    echo ""
    
    # Try primary fix
    if fix_axios_corruption && test_axios; then
        echo ""
        echo "🎉 Axios corruption fixed successfully!"
        unset KNIRV_AXIOS_FIX_IN_PROGRESS
        exit 0
    fi

    echo ""
    echo "🔄 Primary fix failed, trying alternative approach..."
    echo ""

    # Try alternative fix
    if alternative_axios_fix && test_axios; then
        echo ""
        echo "🎉 Axios corruption fixed with alternative method!"
        unset KNIRV_AXIOS_FIX_IN_PROGRESS
        exit 0
    fi

    echo ""
    echo "❌ All axios fix attempts failed!"
    echo ""
    echo "🔧 Manual troubleshooting steps:"
    echo "1. Check Node.js version: node --version"
    echo "2. Check npm version: npm --version"
    echo "3. Try manual reinstall: rm -rf node_modules package-lock.json && npm install"
    echo "4. Check for permission issues"
    echo "5. Consider using a different axios version: npm install axios@1.6.8"

    unset KNIRV_AXIOS_FIX_IN_PROGRESS
    exit 1
}

# Run main function
main "$@"
