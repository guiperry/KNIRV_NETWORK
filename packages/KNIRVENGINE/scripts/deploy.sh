#!/usr/bin/env bash

# Build a distributable KNIRVENGINE desktop application locally.
# KNIRVENGINE is a desktop verification tool; this script only produces a
# local desktop package.
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "$SCRIPT_DIR/.." && pwd)"

for command in go node npm; do
    if ! command -v "$command" >/dev/null 2>&1; then
        echo "Missing required command: $command" >&2
        exit 1
    fi
done

if [[ -x "$SCRIPT_DIR/bundle-sandbox-tools.sh" ]]; then
    "$SCRIPT_DIR/bundle-sandbox-tools.sh"
fi

echo "Building KNIRVENGINE desktop package..."
"$SCRIPT_DIR/build-desktop.sh" build

RELEASE_DIR="$PROJECT_ROOT/dist/release"
if [[ ! -d "$RELEASE_DIR" ]]; then
    echo "Desktop package was not created at $RELEASE_DIR" >&2
    exit 1
fi

echo "Desktop package created:"
find "$RELEASE_DIR" -maxdepth 1 -type f -printf '  %f\n' | sort
