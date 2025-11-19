#!/bin/bash

# Backup script for manually created Node.js dependency files
# These files are required for KNIRVCHAIN Node.js services to work properly

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KNIRVORACLE_DIR="$(dirname "$SCRIPT_DIR")"
BACKUP_DIR="$KNIRVORACLE_DIR/scripts/nodejs-deps-backup"

echo "🔄 Backing up manually created Node.js dependency files..."

# Create backup directory if it doesn't exist
mkdir -p "$BACKUP_DIR"

# Backup axios.cjs for agent-tunnel-registry
AXIOS_SOURCE="$KNIRVORACLE_DIR/internal/embedded/nodejs/tunnel/agent-tunnel-registry/node_modules/axios/dist/node/axios.cjs"
AXIOS_BACKUP="$BACKUP_DIR/axios.cjs"

if [ -f "$AXIOS_SOURCE" ]; then
    cp "$AXIOS_SOURCE" "$AXIOS_BACKUP"
    echo "✅ Backed up: axios.cjs"
else
    echo "⚠️  Warning: axios.cjs not found at $AXIOS_SOURCE"
fi

# Backup psl.cjs for agent-payment-gateway
PSL_SOURCE="$KNIRVORACLE_DIR/internal/embedded/nodejs/payment/agent-payment-gateway/node_modules/psl/dist/psl.cjs"
PSL_BACKUP="$BACKUP_DIR/psl.cjs"

if [ -f "$PSL_SOURCE" ]; then
    cp "$PSL_SOURCE" "$PSL_BACKUP"
    echo "✅ Backed up: psl.cjs"
else
    echo "⚠️  Warning: psl.cjs not found at $PSL_SOURCE"
fi

echo "📁 Backup location: $BACKUP_DIR"
echo "✅ Backup complete!"
