#!/usr/bin/env bash

# Start the local browser development workflow (Go API plus Vite frontend).
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
exec "$SCRIPT_DIR/start-browser.sh"
