#!/bin/bash
# KNIRVHASHER Encoder Flight Integration — E2E Pipeline Test
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$HERE/../../.." && pwd)"
ENCODER_DIR="$ROOT/packages/KNIRVHASHER/pipeline/2_DATA_ENCODER"

echo "=== KNIRVHASHER Encoder Flight Integration ==="

# 1. Create temp workspace
WORKSPACE=$(mktemp -d)
trap "rm -rf $WORKSPACE" EXIT

ARROW_DIR="$WORKSPACE/arrows"
OUTPUT_DIR="$WORKSPACE/output"
mkdir -p "$ARROW_DIR" "$OUTPUT_DIR"

# 2. Run the encoder integration test against a real KNIRVBASE instance
echo ""
echo "--- Phase 1: Unit Tests (encoder + writer) ---"
cd "$ENCODER_DIR"
go test -v -count=1 -run "TestLoadArrowBatch|TestRetryWithBackoff|TestWaitForCollectionReady|TestNRVEncoder_Run|TestNRVEncoder_Skips|TestSecurityRecordToSlotVector" ./internal/encoder/ 2>&1 | tail -30

echo ""
echo "--- Phase 2: Writer Unit Tests ---"
go test -v -count=1 -run "TestNRVWriter" ./internal/writer/ 2>&1 | tail -10

echo ""
echo "--- Phase 3: Integration Tests (real KNIRVBASE) ---"
go test -v -count=1 -run "TestEncoderIntegration" ./tests/ 2>&1 | tail -30

# 3. Verify all tests passed
echo ""
echo "--- Summary ---"
cd "$ENCODER_DIR"
go test ./internal/encoder/ ./internal/writer/ ./tests/ 2>&1 | grep -E "^(ok|FAIL)"

echo ""
echo "=== KNIRVHASHER Encoder Flight Integration Complete ==="
