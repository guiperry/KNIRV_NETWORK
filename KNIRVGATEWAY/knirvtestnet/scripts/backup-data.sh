#!/bin/bash
set -e

echo "=========================================="
echo "KNIRV-TESTNET: Data Backup"
echo "=========================================="

# Already in KNIRVTESTNET directory

BACKUP_DIR="./backups/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

echo "📦 Creating backup in $BACKUP_DIR..."

# Backup blockchain data
echo "💾 Backing up blockchain data..."
if [ -d "data/knirvroot" ]; then
    cp -r data/knirvroot "$BACKUP_DIR/"
    echo "  ✅ KNIRV-ROOT data backed up"
fi

if [ -d "data/knirvchain" ]; then
    cp -r data/knirvchain "$BACKUP_DIR/"
    echo "  ✅ KNIRVCHAIN data backed up"
fi

if [ -d "data/knirvgraph" ]; then
    cp -r data/knirvgraph "$BACKUP_DIR/"
    echo "  ✅ KNIRVGRAPH data backed up"
fi

if [ -d "data/knirvnexus" ]; then
    cp -r data/knirvnexus "$BACKUP_DIR/"
    echo "  ✅ KNIRV-NEXUS data backed up"
fi

if [ -d "data/ipfs" ]; then
    cp -r data/ipfs "$BACKUP_DIR/"
    echo "  ✅ IPFS data backed up"
fi

# Backup configurations
echo "⚙️  Backing up configurations..."
cp -r config "$BACKUP_DIR/"
echo "  ✅ Configuration files backed up"

# Backup logs
echo "📋 Backing up logs..."
cp -r logs "$BACKUP_DIR/"
echo "  ✅ Log files backed up"

# Create archive
echo "🗜️  Creating compressed archive..."
tar -czf "$BACKUP_DIR.tar.gz" -C backups "$(basename $BACKUP_DIR)"
rm -rf "$BACKUP_DIR"

# Calculate backup size
BACKUP_SIZE=$(du -sh "$BACKUP_DIR.tar.gz" | cut -f1)

echo ""
echo "=========================================="
echo "✅ Backup completed successfully!"
echo "=========================================="
echo "📦 Backup file: $BACKUP_DIR.tar.gz"
echo "📏 Backup size: $BACKUP_SIZE"
echo ""
echo "To restore:"
echo "  1. Stop testnet: ./scripts/stop-testnet.sh"
echo "  2. Extract: tar -xzf $BACKUP_DIR.tar.gz -C backups/"
echo "  3. Restore: cp -r backups/$(basename $BACKUP_DIR)/* ./"
echo "  4. Start testnet: ./scripts/start-testnet.sh"
