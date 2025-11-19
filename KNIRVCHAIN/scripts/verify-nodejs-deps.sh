#!/bin/bash

# Verification script for KNIRVCHAIN Node.js dependencies
# Checks if the manually created dependency files are in place and functional

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KNIRVORACLE_DIR="$(dirname "$SCRIPT_DIR")"

echo "🔍 Verifying KNIRVCHAIN Node.js dependencies..."

# Check axios.cjs for agent-tunnel-registry
AXIOS_FILE="$KNIRVORACLE_DIR/internal/embedded/nodejs/tunnel/agent-tunnel-registry/node_modules/axios/dist/node/axios.cjs"
echo ""
echo "📦 Checking axios dependency for agent-tunnel-registry..."

if [ -f "$AXIOS_FILE" ]; then
    echo "✅ axios.cjs exists"
    
    # Test if the file is readable and has content
    if [ -s "$AXIOS_FILE" ]; then
        echo "✅ axios.cjs has content"
        
        # Quick syntax check
        if node -c "$AXIOS_FILE" 2>/dev/null; then
            echo "✅ axios.cjs syntax is valid"
        else
            echo "❌ axios.cjs has syntax errors"
        fi
    else
        echo "❌ axios.cjs is empty"
    fi
else
    echo "❌ axios.cjs is missing"
    echo "   Expected location: $AXIOS_FILE"
fi

# Check psl.cjs for agent-payment-gateway
PSL_FILE="$KNIRVORACLE_DIR/internal/embedded/nodejs/payment/agent-payment-gateway/node_modules/psl/dist/psl.cjs"
echo ""
echo "📦 Checking psl dependency for agent-payment-gateway..."

if [ -f "$PSL_FILE" ]; then
    echo "✅ psl.cjs exists"
    
    # Test if the file is readable and has content
    if [ -s "$PSL_FILE" ]; then
        echo "✅ psl.cjs has content"
        
        # Quick syntax check
        if node -c "$PSL_FILE" 2>/dev/null; then
            echo "✅ psl.cjs syntax is valid"
        else
            echo "❌ psl.cjs has syntax errors"
        fi
    else
        echo "❌ psl.cjs is empty"
    fi
else
    echo "❌ psl.cjs is missing"
    echo "   Expected location: $PSL_FILE"
fi

# Test service startup (quick test)
echo ""
echo "🚀 Testing service startup..."

# Test tunnel registry
echo ""
echo "🔧 Testing agent-tunnel-registry..."
cd "$KNIRVORACLE_DIR/internal/embedded/nodejs/tunnel/agent-tunnel-registry"
if timeout 3s node server.js 2>&1 | grep -q "Custom Tunnel & Registry Server started"; then
    echo "✅ agent-tunnel-registry starts successfully"
else
    echo "❌ agent-tunnel-registry failed to start"
fi

# Test payment gateway
echo ""
echo "💳 Testing agent-payment-gateway..."
cd "$KNIRVORACLE_DIR/internal/embedded/nodejs/payment/agent-payment-gateway"
if timeout 3s node server.js 2>&1 | grep -q "Server running on port"; then
    echo "✅ agent-payment-gateway starts successfully"
else
    echo "❌ agent-payment-gateway failed to start"
fi

echo ""
echo "📋 Verification complete!"
echo ""
echo "💡 If any checks failed:"
echo "   1. Run: ./restore-nodejs-deps.sh"
echo "   2. If that fails, run: ./backup-nodejs-deps.sh first"
echo "   3. Check that npm install was run in the service directories"
