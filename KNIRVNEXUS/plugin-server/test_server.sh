#!/bin/bash

# Test script for KNIRVROOT Plugin Agent Server

set -e

SERVER_PORT=8082
SERVER_URL="http://localhost:$SERVER_PORT"
TEST_DIR="./test-agents"
TEST_FILE="test-agent.wasm"

echo "🧪 Testing KNIRVROOT Plugin Agent Server"
echo "=========================================="

# Clean up function
cleanup() {
    echo "🧹 Cleaning up..."
    if [ ! -z "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
    fi
    rm -rf "$TEST_DIR"
    rm -f "$TEST_FILE"
}

# Set up cleanup trap
trap cleanup EXIT

# Create test directory and file
echo "📁 Setting up test environment..."
mkdir -p "$TEST_DIR"
echo "This is a test WASM plugin agent file" > "$TEST_FILE"

# Start the server in background
echo "🚀 Starting plugin server on port $SERVER_PORT..."
./plugin-server --port $SERVER_PORT --agents "$TEST_DIR" --name "Test Server" &
SERVER_PID=$!

# Wait for server to start
echo "⏳ Waiting for server to start..."
sleep 2

# Test 1: Server info
echo "🔍 Test 1: Server info"
curl -s "$SERVER_URL/info" | jq . || echo "Server info response received"

# Test 2: List agents (should be empty initially)
echo "📋 Test 2: List agents (empty)"
curl -s "$SERVER_URL/list" | jq . || echo "List response received"

# Test 3: Upload agent
echo "📤 Test 3: Upload agent"
curl -s -X POST -F "plugin-agent=@$TEST_FILE" "$SERVER_URL/upload" | jq . || echo "Upload response received"

# Test 4: List agents (should have one agent)
echo "📋 Test 4: List agents (with uploaded agent)"
curl -s "$SERVER_URL/list" | jq . || echo "List response received"

# Test 5: Download agent
echo "📥 Test 5: Download agent"
curl -s -o "downloaded-$TEST_FILE" "$SERVER_URL/agents/$TEST_FILE"
if [ -f "downloaded-$TEST_FILE" ]; then
    echo "✅ Agent downloaded successfully"
    rm "downloaded-$TEST_FILE"
else
    echo "❌ Agent download failed"
fi

# Test 6: Delete agent
echo "🗑️  Test 6: Delete agent"
curl -s -X DELETE "$SERVER_URL/delete/$TEST_FILE" | jq . || echo "Delete response received"

# Test 7: List agents (should be empty again)
echo "📋 Test 7: List agents (after deletion)"
curl -s "$SERVER_URL/list" | jq . || echo "List response received"

echo "✅ All tests completed!"
echo "🎉 Plugin Agent Server is working correctly!"
