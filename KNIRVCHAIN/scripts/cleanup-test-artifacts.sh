#!/bin/bash

# cleanup-test-artifacts.sh
# Script to clean up test artifacts in KNIRVCHAIN

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
_DIR="$(dirname "$SCRIPT_DIR")"

echo "Cleaning up test artifacts in KNIRVCHAIN..."

cd "$_DIR"

# Clean up test databases from root directory
echo "Removing test databases from root directory..."
rm -rf test_db_* testdb_* test_chromem_* 2>/dev/null || true

# Clean up old log and coverage files from root
echo "Removing old log and coverage files from root..."
rm -f KNIRVCHAIN.log coverage.out 2>/dev/null || true

# Clean up test-reports directory (but keep the directory itself)
if [ -d "test-reports" ]; then
    echo "Cleaning test-reports directory..."
    find test-reports -name "test_*" -type d -exec rm -rf {} + 2>/dev/null || true
    find test-reports -name "*.out" -delete 2>/dev/null || true
fi

# Clean up any temporary test files
echo "Removing temporary test files..."
find . -maxdepth 1 -name "*.tmp" -delete 2>/dev/null || true
find . -maxdepth 1 -name "*.lock" -delete 2>/dev/null || true

# Clean up debug binaries
echo "Removing debug binaries..."
rm -f __debug_bin* 2>/dev/null || true

echo "Test artifact cleanup complete!"
echo "Logs are now in: logs/"
echo "Coverage reports are now in: test-reports/"
echo "Test databases are created in: test-reports/"
