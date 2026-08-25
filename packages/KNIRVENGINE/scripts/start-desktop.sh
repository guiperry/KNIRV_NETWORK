#!/usr/bin/env bash

# Start the KNIRVENGINE Electron desktop application from source.
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"
go build -tags embed -o knirv-engine .
cd gui
exec npm run desktop:dev
