#!/bin/bash
set -e

echo "💰 Running Economic Loop Tests..."
echo "================================="

# Check if testnet is running
if ! curl -s http://localhost:8888/gateway/health > /dev/null 2>&1; then
    echo "❌ Error: KNIRV testnet is not running. Please start it first."
    exit 1
fi

# Initialize Go module if it doesn't exist
if [ ! -f "go.mod" ]; then
    echo "📦 Initializing Go module..."
    go mod init economic-loop-tests
    go mod tidy
fi

# Run the Go tests
echo "🧪 Executing economic loop test suite..."
go test -v -timeout=10m ./... 2>&1 | tee test_results.log

# Check test results
if [ ${PIPESTATUS[0]} -eq 0 ]; then
    echo "✅ Economic loop tests completed successfully!"
    
    # Generate summary
    TOTAL_TESTS=$(grep -c "=== RUN" test_results.log || echo "0")
    PASSED_TESTS=$(grep -c "--- PASS:" test_results.log || echo "0")
    FAILED_TESTS=$(grep -c "--- FAIL:" test_results.log || echo "0")
    
    echo ""
    echo "📊 Test Summary:"
    echo "   Total Tests: $TOTAL_TESTS"
    echo "   Passed: $PASSED_TESTS"
    echo "   Failed: $FAILED_TESTS"
    
    if [ "$FAILED_TESTS" -gt 0 ]; then
        echo "⚠️  Some tests failed. Check test_results.log for details."
        exit 1
    fi
else
    echo "❌ Economic loop tests failed!"
    echo "📋 Check test_results.log for detailed error information."
    exit 1
fi

echo "🎉 All economic loop tests passed!"
