#!/usr/bin/env bash

# Verify the local API and sandbox service used by KNIRVENGINE desktop.
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
exec node "$SCRIPT_DIR/test-desktop-system.js"
