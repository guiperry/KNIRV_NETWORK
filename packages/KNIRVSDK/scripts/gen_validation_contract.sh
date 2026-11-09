#!/usr/bin/env bash
# Thin-binding generator for the shared DRQ validation contract fixture.
#
# Single source of truth: packages/KNIRVSDK/core/validation_contract.json
# (the Rust SDK core). This script emits byte-identical copies into the two Go
# testdata locations that consume it, so the contract can never drift between
# KNIRVGRAPH (DRQ client) and the KNIRV_CORP backend_server (real-route test).
# Both Go tests pin the SHA-256 of their copy (validationContractSHA256 =
# 300fcea1962c1bc6b7e2c39104fafa23d7b8b511481ad92b722788d37357ff12); updating the
# canonical and re-running this generator keeps every copy and pin in sync.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CANON="${HERE}/../core/validation_contract.json"

if [[ ! -f "$CANON" ]]; then
  echo "canonical fixture not found at $CANON" >&2
  exit 1
fi

# Output 1: KNIRVGRAPH DRQ client testdata (same repo as this SDK).
GRAPH_OUT="${1:-${HERE}/../../KNIRVGRAPH/internal/app/testdata/validation_contract.json}"
# Output 2: KNIRV_CORP backend_server real-route testdata (separate repo).
CORP_OUT="${2:-/home/gperry/Documents/GitHub/KNIRV/KNIRV_CORP/packages/server/backend_server/internal/web/testdata/validation_contract.json}"

for out in "$GRAPH_OUT" "$CORP_OUT"; do
  mkdir -p "$(dirname "$out")"
  cp "$CANON" "$out"
  sha256sum "$out"
done

echo "validation_contract.json regenerated from SDK core canonical into:"
echo "  $GRAPH_OUT"
echo "  $CORP_OUT"
