#!/bin/bash

# Restore script for manually created Node.js dependency files
# These files are required for KNIRVORACLE Node.js services to work properly
# Run this script after 'make sync' or whenever node_modules are cleaned

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KNIRVORACLE_DIR="$(dirname "$SCRIPT_DIR")"
BACKUP_DIR="$KNIRVORACLE_DIR/scripts/nodejs-deps-backup"

echo "🔄 Restoring manually created Node.js dependency files..."

# Check if backup directory exists
if [ ! -d "$BACKUP_DIR" ]; then
    echo "❌ Error: Backup directory not found at $BACKUP_DIR"
    echo "   Please run backup-nodejs-deps.sh first or create the files manually."
    exit 1
fi

# Restore axios.cjs for agent-tunnel-registry
AXIOS_BACKUP="$BACKUP_DIR/axios.cjs"
AXIOS_TARGET="$KNIRVORACLE_DIR/agent-tunnel-registry/node_modules/axios/dist/node/axios.cjs"

if [ -f "$AXIOS_BACKUP" ]; then
    # Create target directory if it doesn't exist
    mkdir -p "$(dirname "$AXIOS_TARGET")"
    cp "$AXIOS_BACKUP" "$AXIOS_TARGET"
    echo "✅ Restored: axios.cjs"
else
    echo "❌ Error: axios.cjs backup not found at $AXIOS_BACKUP"
    echo "   Creating from axios.js..."
    
    # Try to create from axios.js if it exists
    AXIOS_JS="$KNIRVORACLE_DIR/agent-tunnel-registry/node_modules/axios/dist/node/axios.js"
    if [ -f "$AXIOS_JS" ]; then
        mkdir -p "$(dirname "$AXIOS_TARGET")"
        cp "$AXIOS_JS" "$AXIOS_TARGET"
        echo "✅ Created axios.cjs from axios.js"
    else
        echo "❌ Error: Neither axios.cjs backup nor axios.js source found"
        echo "   Please install dependencies first: cd agent-tunnel-registry && npm install"
    fi
fi

# Restore psl.cjs for agent-payment-gateway
PSL_BACKUP="$BACKUP_DIR/psl.cjs"
PSL_TARGET="$KNIRVORACLE_DIR/agent-payment-gateway/node_modules/psl/dist/psl.cjs"

if [ -f "$PSL_BACKUP" ]; then
    # Create target directory if it doesn't exist
    mkdir -p "$(dirname "$PSL_TARGET")"
    cp "$PSL_BACKUP" "$PSL_TARGET"
    echo "✅ Restored: psl.cjs"
else
    echo "❌ Error: psl.cjs backup not found at $PSL_BACKUP"
    echo "   Creating from psl.js..."
    
    # Try to create from psl.js if it exists
    PSL_JS="$KNIRVORACLE_DIR/agent-payment-gateway/node_modules/psl/dist/psl.js"
    if [ -f "$PSL_JS" ]; then
        mkdir -p "$(dirname "$PSL_TARGET")"
        cp "$PSL_JS" "$PSL_TARGET"
        echo "✅ Created psl.cjs from psl.js"
    else
        echo "❌ Error: Neither psl.cjs backup nor psl.js source found"
        echo "   Please install dependencies first: cd agent-payment-gateway && npm install"
    fi
fi

echo "✅ Restore complete!"
echo ""
echo "📋 Next steps:"
echo "   1. Test the services: cd KNIRVORACLE && timeout 10s go run main.go --role=root"
echo "   2. If issues persist, check the service logs for dependency errors"
