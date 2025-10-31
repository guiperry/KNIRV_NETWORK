# KNIRV-NEXUS Configuration Migration Guide

## Overview

This document describes the configuration consolidation and XDG Base Directory migration for KNIRV-NEXUS.

## Changes Made

### 1. Configuration Files Consolidated

Previously, there were three separate YAML configuration files:
- `/KNIRVNEXUS/knirv-nexus.yaml` (root - original)
- `/KNIRVNEXUS/config/knirv-nexus.yaml` (config directory)
- `/KNIRVNEXUS/backend/config/knirv-nexus.yaml` (backend duplicate)

**Now:**
- **Primary Config**: `/KNIRVNEXUS/config/knirv-nexus.yaml` (authoritative)
- **Backend Config**: `/KNIRVNEXUS/backend/config/knirv-nexus.yaml` (synced copy)
- **Root Config**: `/KNIRVNEXUS/knirv-nexus.yaml` (reference only - can be ignored)

All three files now have identical comprehensive configuration including all services, settings, and paths.

### 2. XDG Base Directory Compliance

All data directories now use the XDG Base Directory Specification:

**Old Behavior (Relative Paths):**
```
./data/nexus.db
./logs/nexus.log
./workspaces/
./projects/
./images/
./reports/
```

These created folders in the current working directory, cluttering the project root.

**New Behavior (XDG Base Directory):**
```
~/.local/share/knirvnexus/backend_server/data/nexus.db
~/.local/share/knirvnexus/backend_server/logs/nexus.log
~/.local/share/knirvnexus/backend_server/workspaces/
~/.local/share/knirvnexus/backend_server/projects/
~/.local/share/knirvnexus/backend_server/images/
~/.local/share/knirvnexus/backend_server/reports/
```

All data is now stored in the user's home directory, following Linux standards.

### 3. Configuration Updates

#### config.go Changes

**New Functions:**
- `getAppDataDir()` - Gets/creates XDG data directory
- `expandPath()` - Expands ~ paths to home directory
- `expandPaths()` - Expands all paths in config and creates directories

**Updated setDefaults():**
- Now calls `getAppDataDir()` to get the app data directory
- Uses `filepath.Join()` to construct paths in XDG directory
- All paths now default to user's home directory instead of relative paths

**Updated Load():**
- Calls `config.expandPaths()` after unmarshaling to expand ~ paths
- Automatically creates all necessary directories

#### Security Changes

- Default `security.auth_required` changed to `false` (for testnet mode)
- Default `security.tls_enabled` changed to `false` (for development)
- These can be overridden in config or environment variables

### 4. File References Updated

All code now properly references the config files:
- Backend loads from: `config/knirv-nexus.yaml`
- Backend respects command-line `--config` flag for custom paths
- Wrapper frontend loads separate `nexus.yaml` config

## Migration Guide

### For Users Upgrading

1. **No Action Required** - The configuration system is backward compatible
   - Existing relative paths in custom configs will still work
   - The config loader will automatically expand ~ paths

2. **Recommended** - Update your config file to use XDG paths:
   ```yaml
   # Instead of:
   database:
     path: ./data/nexus.db

   # Use:
   database:
     path: ~/.local/share/knirvnexus/backend_server/data/nexus.db
   ```

3. **Optional** - Migrate existing data:
   ```bash
   # Move old data to new location
   mkdir -p ~/.local/share/knirvnexus/backend_server
   mv ./data ~/.local/share/knirvnexus/backend_server/
   mv ./logs ~/.local/share/knirvnexus/backend_server/
   mv ./workspaces ~/.local/share/knirvnexus/backend_server/
   mv ./projects ~/.local/share/knirvnexus/backend_server/
   mv ./images ~/.local/share/knirvnexus/backend_server/
   ```

### For Developers

1. **Configuration Loading**:
   ```go
   // Config is loaded with expanded paths
   cfg, err := config.Load()
   // cfg.Database.Path is already expanded: /home/user/.local/share/knirvnexus/backend_server/data/nexus.db
   // cfg.CDE.WorkspaceRoot is already expanded and directory created
   ```

2. **Custom Configs**:
   - Use `~` in config files for home directory expansion
   - All paths with `~` are automatically expanded during loading
   - Directories are automatically created

3. **Environment Variables**:
   ```bash
   # Override config values with environment variables
   KNIRV_DATABASE_PATH=~/.local/share/custom/path/nexus.db
   KNIRV_CDE_WORKSPACE_ROOT=~/.local/share/custom/workspaces
   ```

## Directory Structure

After running KNIRV-NEXUS, you'll see:

```
~/.local/share/knirvnexus/backend_server/
├── data/
│   ├── nexus.db
│   └── backups/
├── logs/
│   └── nexus.log
├── images/
│   └── (CDE base images)
├── workspaces/
│   └── (CDE environments)
├── projects/
│   └── (User projects)
└── reports/
    └── (Generated reports)
```

## Configuration File Locations

1. **Load Order** (in `config/config.go`):
   ```
   1. Command line --config flag
   2. ./config/knirv-nexus.yaml
   3. ./knirv-nexus.yaml
   ```

2. **Environment Variables**:
   - Prefix: `KNIRV_`
   - Automatically parsed (e.g., `KNIRV_API_PORT=9000`)

3. **Defaults**:
   - Hardcoded in `setDefaults()` function
   - Can be overridden by config files or environment variables

## Troubleshooting

### Directories Still Being Created in Project Root

**Cause**: Old configuration or relative paths still being used

**Solution**:
1. Check which config file is being used: Look for log message `"Passing config file to backend: ..."`
2. Ensure you're using `./config/knirv-nexus.yaml`
3. Verify paths use `~` prefix for home directory expansion

### Permission Errors on ~/.local/share

**Cause**: Insufficient permissions to create directories

**Solution**:
```bash
# Ensure correct permissions
mkdir -p ~/.local/share/knirvnexus/backend_server
chmod 755 ~/.local/share/knirvnexus/backend_server
```

### Custom Config Not Being Used

**Solution**:
```bash
# Pass config explicitly
./backend_server --config ./config/knirv-nexus.yaml

# Or set environment variable
KNIRV_DATABASE_PATH=~/.local/share/custom/nexus.db ./backend_server
```

## Reference

- XDG Base Directory Specification: https://specifications.freedesktop.org/basedir-spec/
- Configuration file: `/KNIRVNEXUS/config/knirv-nexus.yaml`
- Config loader: `/KNIRVNEXUS/backend/internal/config/config.go`