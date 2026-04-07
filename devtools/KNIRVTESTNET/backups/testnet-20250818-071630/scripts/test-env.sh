#!/bin/bash
# Test script to verify environment loading works correctly

echo "🧪 Testing KNIRV Environment Loading"
echo "===================================="

# Test 1: Install dependencies
echo "Test 1: Installing dependencies..."
bash scripts/install-deps.sh

# Test 2: Load environment in a new shell
echo ""
echo "Test 2: Loading environment in new shell..."
bash -c "source scripts/load-env.sh && echo 'Environment loaded in new shell'"

# Test 3: Check toolchains in new shell
echo ""
echo "Test 3: Checking toolchains in new shell..."
bash -c "
source scripts/load-env.sh
echo 'Go version:' \$(go version 2>/dev/null || echo 'NOT FOUND')
echo 'Rust version:' \$(rustc --version 2>/dev/null || echo 'NOT FOUND')
echo 'Cargo version:' \$(cargo --version 2>/dev/null || echo 'NOT FOUND')
"

# Test 4: Simulate Render build process
echo ""
echo "Test 4: Simulating Render build process..."
bash -c "
echo 'Step 1: Install toolchains'
bash scripts/install-deps.sh > /dev/null 2>&1

echo 'Step 2: Load environment'
source scripts/load-env.sh > /dev/null 2>&1

echo 'Step 3: Verify Go is available'
if command -v go &> /dev/null; then
    echo '✅ Go found: \$(go version)'
else
    echo '❌ Go not found'
    exit 1
fi

echo 'Step 4: Verify Rust is available'
if command -v rustc &> /dev/null; then
    echo '✅ Rust found: \$(rustc --version)'
else
    echo '❌ Rust not found'
    exit 1
fi

echo '✅ All toolchains available in simulated build environment'
"

echo ""
echo "🎉 Environment loading test completed!"
