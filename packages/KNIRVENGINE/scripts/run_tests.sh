#!/usr/bin/env bash

# Run KNIRVENGINE's Go test suite from the project root.
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."
exec go test ./...
