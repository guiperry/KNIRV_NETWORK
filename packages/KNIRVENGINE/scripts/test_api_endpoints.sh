#!/usr/bin/env bash

# Smoke-test the API served by the running KNIRVENGINE desktop application.
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
exec node "$SCRIPT_DIR/test-desktop-api.js"
