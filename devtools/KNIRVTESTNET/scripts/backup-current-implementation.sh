#!/bin/bash
set -e

echo "Creating backup of current KNIRVTESTNET implementation..."

# Create backup directory with timestamp
BACKUP_DIR="backups/testnet-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"

echo "Backup directory: $BACKUP_DIR"

# Backup critical files
echo "Backing up server configuration..."
cp -r server/ "$BACKUP_DIR/"

echo "Backing up scripts..."
cp -r scripts/ "$BACKUP_DIR/"

echo "Backing up data directory..."
cp -r data/ "$BACKUP_DIR/"

echo "Backing up configuration files..."
cp -r config/ "$BACKUP_DIR/"

echo "Backing up package files..."
cp package.json "$BACKUP_DIR/" 2>/dev/null || true
cp package-lock.json "$BACKUP_DIR/" 2>/dev/null || true

echo "Backing up root index.html..."
cp index.html "$BACKUP_DIR/"

echo "Backing up logs (if any)..."
cp -r logs/ "$BACKUP_DIR/" 2>/dev/null || true

# Create backup info file
cat > "$BACKUP_DIR/backup-info.txt" << EOF
KNIRVTESTNET Backup Information
===============================
Created: $(date)
Backup Type: Pre-unified-architecture
Git Commit: $(git rev-parse HEAD 2>/dev/null || echo "Not available")
Git Branch: $(git branch --show-current 2>/dev/null || echo "Not available")

Changes Made:
- Renamed data/knirvgateway to data/testnet-gateway
- Added NEXUS frontend integration
- Updated server/app.js for environment-aware routing
- Added Netlify functions testing
- Deprecated root index.html

Files Backed Up:
- server/ (Express server configuration)
- scripts/ (All build and start scripts)
- data/ (Gateway and service data)
- config/ (Configuration files)
- package.json (Dependencies)
- index.html (Root landing page)
- logs/ (Service logs if available)

Restore Instructions:
To restore this backup, run:
  rm -rf server/ scripts/ data/ config/
  cp -r $BACKUP_DIR/* ./
  
Note: This will overwrite current implementation!
EOF

echo "✅ Backup completed successfully!"
echo "📁 Backup location: $BACKUP_DIR"
echo "📄 Backup info: $BACKUP_DIR/backup-info.txt"
echo ""
echo "To restore this backup later, run:"
echo "  ./scripts/restore-backup.sh $BACKUP_DIR"
