#!/usr/bin/env bash

# Copy local configuration into an unpacked electron-builder desktop package.
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "$SCRIPT_DIR/.." && pwd)"
ENV_FILE="$PROJECT_ROOT/.env"
RELEASE_DIR="$PROJECT_ROOT/dist/release"

if [[ ! -f "$ENV_FILE" ]]; then
    echo "Missing $ENV_FILE. Create it before packaging KNIRVENGINE." >&2
    exit 1
fi

resources_dir=""
while IFS= read -r candidate; do
    resources_dir="$candidate"
    break
done < <(find "$RELEASE_DIR" -type d -path '*/resources' -print 2>/dev/null)

if [[ -z "$resources_dir" ]]; then
    echo "No unpacked desktop package found in $RELEASE_DIR. Run scripts/deploy.sh first." >&2
    exit 1
fi

cp "$ENV_FILE" "$resources_dir/.env"
if [[ -f "$PROJECT_ROOT/utils/default.env" ]]; then
    cp "$PROJECT_ROOT/utils/default.env" "$resources_dir/default.env"
fi

echo "Copied KNIRVENGINE configuration to $resources_dir"
