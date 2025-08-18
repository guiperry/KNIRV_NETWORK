#!/bin/bash
set -e

if [ $# -eq 0 ]; then
    echo "Usage: $0 <backup-directory>"
    echo ""
    echo "Available backups:"
    ls -la backups/ 2>/dev/null || echo "No backups found"
    exit 1
fi

BACKUP_DIR="$1"

if [ ! -d "$BACKUP_DIR" ]; then
    echo "Error: Backup directory '$BACKUP_DIR' not found"
    exit 1
fi

echo "🔄 Restoring KNIRVTESTNET from backup: $BACKUP_DIR"
echo ""

# Show backup info if available
if [ -f "$BACKUP_DIR/backup-info.txt" ]; then
    echo "📄 Backup Information:"
    cat "$BACKUP_DIR/backup-info.txt"
    echo ""
fi

# Confirm restoration
read -p "Are you sure you want to restore this backup? This will overwrite current implementation. (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Restoration cancelled."
    exit 0
fi

echo "Creating backup of current state before restoration..."
./scripts/backup-current-implementation.sh

echo "Restoring files from backup..."

# Remove current directories
echo "Removing current implementation..."
rm -rf server/ scripts/ data/ config/ 2>/dev/null || true

# Restore from backup
echo "Restoring server configuration..."
cp -r "$BACKUP_DIR/server/" ./ 2>/dev/null || true

echo "Restoring scripts..."
cp -r "$BACKUP_DIR/scripts/" ./ 2>/dev/null || true

echo "Restoring data directory..."
cp -r "$BACKUP_DIR/data/" ./ 2>/dev/null || true

echo "Restoring configuration files..."
cp -r "$BACKUP_DIR/config/" ./ 2>/dev/null || true

echo "Restoring package files..."
cp "$BACKUP_DIR/package.json" ./ 2>/dev/null || true
cp "$BACKUP_DIR/package-lock.json" ./ 2>/dev/null || true

echo "Restoring root index.html..."
cp "$BACKUP_DIR/index.html" ./ 2>/dev/null || true

echo "Restoring logs..."
cp -r "$BACKUP_DIR/logs/" ./ 2>/dev/null || true

# Make scripts executable
chmod +x scripts/*.sh 2>/dev/null || true

echo ""
echo "✅ Restoration completed successfully!"
echo "🔄 KNIRVTESTNET has been restored from backup: $BACKUP_DIR"
echo ""
echo "Next steps:"
echo "1. Review the restored configuration"
echo "2. Run './scripts/build-all.sh' if needed"
echo "3. Run './scripts/start-testnet.sh' to start services"
