# KNIRVCHAIN Node.js Dependencies Backup & Restoration System

## Overview

This system manages manually created Node.js dependency files that are required for KNIRVCHAIN's embedded Node.js services to function properly. These files are lost whenever `node_modules` directories are cleaned or when running `make sync`.

## Critical Files Managed

The following files are manually created and must be preserved:

1. **`KNIRVCHAIN/pkg/embedded/nodejs/tunnel/agent-tunnel-registry/node_modules/axios/dist/node/axios.cjs`**
   - **Purpose**: CommonJS wrapper for axios ES module
   - **Issue**: axios v1.x is ES module only, but KNIRVCHAIN requires CommonJS
   - **Solution**: Downgraded to axios v0.27.2 + manual CJS file

2. **`KNIRVCHAIN/agent-payment-gateway/node_modules/psl/dist/psl.cjs`**
   - **Purpose**: CommonJS wrapper for psl ES module  
   - **Issue**: psl v1.15.0 is ES module only, but tough-cookie requires CommonJS
   - **Solution**: Manual CJS file created from psl.js

## Scripts

### `backup-nodejs-deps.sh`
- **Purpose**: Creates backups of the manually created dependency files
- **Location**: `KNIRVCHAIN/scripts/backup-nodejs-deps.sh`
- **Backup Location**: `KNIRVCHAIN/scripts/nodejs-deps-backup/`
- **Usage**: `./backup-nodejs-deps.sh`

### `restore-nodejs-deps.sh`
- **Purpose**: Restores the manually created dependency files from backup
- **Location**: `KNIRVCHAIN/scripts/restore-nodejs-deps.sh`
- **Usage**: `./restore-nodejs-deps.sh`
- **Fallback**: If backup doesn't exist, attempts to create from .js files

## Automatic Integration

### Make Sync Integration
The restoration script is automatically called after `make sync` operations:
- **Hook Location**: `scripts/sync-network-fixes.sh` (post-sync hook)
- **Trigger**: Any successful sync operation (except dry-runs)
- **Behavior**: Automatically restores Node.js dependencies after sync

## Usage Scenarios

### 1. After `make sync`
```bash
# The restoration happens automatically
make sync-testnet-to-prod
# Node.js dependencies are automatically restored
```

### 2. Manual Backup Before Cleaning
```bash
cd KNIRVCHAIN/scripts
./backup-nodejs-deps.sh
# Clean node_modules...
./restore-nodejs-deps.sh
```

### 3. Fresh Setup
```bash
# Install dependencies first
cd KNIRVCHAIN/pkg/embedded/nodejs/tunnel/agent-tunnel-registry && npm install
cd ../../payment/agent-payment-gateway && npm install

# Then restore the manual files
cd ../../../../../scripts
./restore-nodejs-deps.sh
```

## Troubleshooting

### Error: "axios.cjs backup not found"
1. Check if `node_modules/axios/dist/node/axios.js` exists
2. If yes, the script will create `.cjs` from `.js` automatically
3. If no, run `npm install` in the service directory first

### Error: "psl.cjs backup not found"  
1. Check if `node_modules/psl/dist/psl.js` exists
2. If yes, the script will create `.cjs` from `.js` automatically
3. If no, run `npm install` in the service directory first

### Services Still Failing After Restoration
1. Verify the files exist:
   ```bash
   ls -la KNIRVCHAIN/agent-tunnel-registry/node_modules/axios/dist/node/axios.cjs
   ls -la KNIRVCHAIN/agent-payment-gateway/node_modules/psl/dist/psl.cjs
   ```

2. Test the services individually:
   ```bash
   cd KNIRVCHAIN/agent-tunnel-registry && timeout 5s node server.js
   cd ../agent-payment-gateway && timeout 5s node server.js
   ```

3. Check for other dependency issues in the logs

## Maintenance

### Regular Backup
It's recommended to run backup before major changes:
```bash
cd KNIRVCHAIN/scripts
./backup-nodejs-deps.sh
```

### Verify Backup Integrity
```bash
ls -la KNIRVCHAIN/scripts/nodejs-deps-backup/
# Should show: axios.cjs and psl.cjs
```

## Technical Details

### Why These Files Are Needed
- **axios**: KNIRVCHAIN uses `require('axios')` but axios v1.x is ES module only
- **psl**: tough-cookie dependency uses `require('psl')` but psl v1.15.0 is ES module only

### Alternative Solutions Considered
1. **Upgrade to ES modules**: Would require major KNIRVCHAIN refactoring
2. **Use different packages**: Would require changing service implementations
3. **Manual CJS wrappers**: ✅ **Current solution** - minimal impact, preserves functionality

## Integration with KNIRV Network

This system is integrated with the broader KNIRV Network synchronization system:
- Automatic restoration after sync operations
- Logging integrated with sync system logs
- Backup preservation across environment synchronizations
- Compatible with all sync directions (testnet ↔ production)
