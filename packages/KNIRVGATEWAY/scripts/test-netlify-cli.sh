#!/bin/bash

# Quick test script for netlify-cli functionality
# This script tests if netlify-cli is working without the full health check

echo "🧪 Testing netlify-cli functionality..."
echo "======================================"

# Change to KNIRVORACLE directory
cd "$(dirname "$0")/.."

echo "📍 Current directory: $(pwd)"
echo ""

# Test 1: Check if netlify binary exists
echo "🔍 Test 1: Checking if netlify binary exists..."
if [ -f "node_modules/.bin/netlify" ]; then
    echo "✅ netlify binary found at node_modules/.bin/netlify"
else
    echo "❌ netlify binary not found"
    echo "💡 Try running: npm install"
    exit 1
fi
echo ""

# Test 2: Test netlify version with timeout
echo "🔍 Test 2: Testing netlify version command..."
if timeout 30s npx netlify --version; then
    echo "✅ netlify version command successful"
else
    echo "❌ netlify version command failed or timed out"
    echo "💡 Try running: ./scripts/fix-netlify-cli.sh"
    exit 1
fi
echo ""

# Test 3: Test netlify help (quick version)
echo "🔍 Test 3: Testing netlify help command..."
if timeout 15s npx netlify --help | head -5; then
    echo "✅ netlify help command successful"
else
    echo "⚠️  netlify help command failed or timed out (non-critical)"
fi
echo ""

echo "🎉 netlify-cli tests completed!"
echo ""
echo "💡 If tests failed, run: ./scripts/fix-netlify-cli.sh"
