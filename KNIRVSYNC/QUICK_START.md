# KNIRVSYNC Documentation Sync - Quick Start Guide

## ✅ Status: Ready to Use

The documentation synchronization system has been successfully implemented and is working correctly.

## Quick Commands

### From Project Root

```bash
# Scan for documentation gaps
make doc-scan

# Sync all documentation
make doc-sync

# Sync only README files
make doc-sync-readme

# Sync only API specifications
make doc-sync-api

# Generate unified OpenAPI spec
make doc-sync-unified
```

### From KNIRVSYNC Directory

```bash
cd KNIRVSYNC

# Build the tool
make build

# Scan for gaps
make scan

# Sync everything
make sync-all

# Sync specific parts
make sync-readme    # README files only
make sync-api       # API specs only
make sync-unified   # Unified API only
```

### Direct Binary Usage

```bash
cd KNIRVSYNC

# Scan only (no changes)
./bin/doc-sync -scan -config config/doc-sync-config.json

# Sync README files
./bin/doc-sync -readme -config config/doc-sync-config.json

# Sync API specs
./bin/doc-sync -api -config config/doc-sync-config.json

# Generate unified API
./bin/doc-sync -unified -config config/doc-sync-config.json

# Full sync
./bin/doc-sync -config config/doc-sync-config.json

# Verbose output
./bin/doc-sync -scan -verbose -config config/doc-sync-config.json
```

## Important Notes

### Scan Duration

**The documentation scan takes time** (typically 1-5 minutes for all services). This is normal because it:
- Scans all source code files in 15 KNIRV services
- Parses API route definitions
- Analyzes exported functions
- Compares code with documentation

**Don't worry if it seems slow** - it's thoroughly analyzing your codebase!

### Running Scans in Background

For long-running scans, use background execution:

```bash
# Option 1: Using nohup
cd KNIRVSYNC
nohup make scan > scan.log 2>&1 &

# Check progress
tail -f scan.log

# Option 2: Using screen or tmux
screen -S doc-scan
cd KNIRVSYNC && make scan
# Press Ctrl+A, D to detach
# Reattach with: screen -r doc-scan
```

### Reports Location

After running scans/syncs, check:

- **Gap Report**: `KNIRVSYNC/reports/doc-gaps-report.md` - Human-readable documentation gaps
- **Sync Report**: `KNIRVSYNC/reports/doc-sync-report.json` - Machine-readable sync results

## What Gets Synced

### README Files
- **Source**: `KNIRV*/README.md` (each service)
- **Targets**:
  - `KNIRVGATEWAY/network-website/public/documentation/[service].md`
  - `KNIRVRAMP/src/app/documentation/[service].md`

### API Specifications
- **Source**: `KNIRV*/api.yaml` (each service with API)
- **Targets**:
  - `KNIRVRAMP/src/app/api-docs/[service]-api.yaml`
  - `KNIRVGATEWAY/network-website/public/api-specs/[service]-api.yaml`

### Unified API
- **Sources**: All service `api.yaml` files
- **Target**: `KNIRVGATEWAY/config/unified-openapi.yaml`

## Typical Workflow

### 1. After Updating Documentation

```bash
# Edit your service documentation
vim KNIRVCHAIN/README.md
vim KNIRVCHAIN/api.yaml

# Sync to all locations
make doc-sync

# Commit everything
git add .
git commit -m "docs: update KNIRVCHAIN documentation"
```

### 2. Regular Gap Checks

```bash
# Weekly or after major code changes
make doc-scan

# Review the report
cat KNIRVSYNC/reports/doc-gaps-report.md

# Fix any gaps found
# Then sync again
make doc-sync
```

### 3. Before Releases

```bash
# Ensure all documentation is current
make doc-scan
make doc-sync

# Verify unified API is up to date
cat KNIRVGATEWAY/config/unified-openapi.yaml
```

## Troubleshooting

### "Scan is taking too long"
**This is normal!** Scanning 15 services with thousands of files takes time. If you want faster results:

1. Run scan in background (see above)
2. Or edit `config/doc-sync-config.json` to disable some services temporarily

### "Permission denied" errors
```bash
# Ensure directories exist and are writable
chmod +w KNIRVGATEWAY/network-website/public/documentation
chmod +w KNIRVRAMP/src/app/documentation
chmod +w KNIRVRAMP/src/app/api-docs
```

### "Config file not found"
```bash
# Make sure you're in the right directory
pwd  # Should be KNIRVSYNC or KNIRV_NETWORK

# Or use absolute path
./bin/doc-sync -scan -config /full/path/to/config/doc-sync-config.json
```

### "Module not found" errors
```bash
# Rebuild from scratch
cd KNIRVSYNC
rm -rf bin/
make build
```

## Configuration

Edit `KNIRVSYNC/config/doc-sync-config.json` to:
- Enable/disable specific services
- Add new documentation targets
- Change unified API output path
- Configure sync behavior

Example:
```json
{
  "services": [
    {
      "name": "KNIRVCHAIN",
      "path": "KNIRVCHAIN",
      "has_api": true,
      "enabled": true  // Set to false to skip
    }
  ]
}
```

## Next Steps

1. **Run your first scan**: `make doc-scan`
2. **Review gap report**: `cat KNIRVSYNC/reports/doc-gaps-report.md`
3. **Fix critical gaps**: Focus on missing API specifications first
4. **Perform full sync**: `make doc-sync`
5. **Set up automation**: Add to your CI/CD pipeline

## Getting Help

- Full documentation: `KNIRVSYNC/README.md`
- Implementation details: `KNIRVSYNC/DOC_SYNC_IMPLEMENTATION.md`
- Example output and reports in `KNIRVSYNC/reports/`

---

**Ready to use!** Start with `make doc-scan` from the project root.
