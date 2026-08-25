#!/usr/bin/env bash

# Build and launch KNIRVENGINE as a local Electron desktop application.
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
exec "$SCRIPT_DIR/start-desktop.sh"
