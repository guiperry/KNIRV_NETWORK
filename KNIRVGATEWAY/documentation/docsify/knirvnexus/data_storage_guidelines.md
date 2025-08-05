

---

**Source**: KNIRVNEXUS/docs/data_storage_guidelines.md

# Data Storage Guidelines for Agentic Engine

## Overview

This document outlines the standardized data storage architecture implemented in Agentic Engine. The new system provides a unified, organized approach to data management across all components.

## Directory Structure

### Base Directory

All application data is stored in OS-specific directories:

- **Linux**: `~/.config/Agentic-Engine/`
- **Windows**: `%APPDATA%\Agentic-Engine\`
- **macOS**: `~/Library/Application Support/Agentic-Engine/`

### Standardized Subdirectories

```
~/.config/Agentic-Engine/
├── data/                    # Core application databases
│   ├── agents.db/          # Centralized agent storage (chromem-go)
│   ├── auth.db             # Authentication database
│   ├── domain.db/          # Domain-specific data (chromem-go)
│   └── inference_engine.db # Inference engine data
├── mcp/                    # Model Context Protocol data
│   ├── config/            # MCP server configurations
│   ├── data/              # MCP-specific data files
│   ├── logs/              # MCP server logs
│   ├── monitoring/        # MCP monitoring data
│   └── servers/           # MCP server installations
├── plugins/               # Plugin system
│   └── data/              # Plugin-specific data
├── templates/             # Agent templates
├── downloads/             # Downloaded files
├── logs/                  # Application logs
├── security/              # Security-related data
├── quarantine/            # Quarantined files
├── analytics/             # Analytics data
├── backups/               # Backup files
├── health/                # Health check data
└── versions/              # Version management
```

## Key Changes from Previous Architecture

### 1. Unified Agent Storage

**Before**: Separate `AgentRegistry` and `UnifiedAgentStorage` systems
**After**: `AgentRegistry` now uses `UnifiedAgentStorage` as its backend

- All agent data stored in `data/agents.db/`
- Eliminates duplication between storage systems
- Maintains backward compatibility

### 2. Centralized MCP Structure

**Before**: Scattered MCP directories (`mcp_servers`, `mcp-data`, `mcp_logs`, etc.)
**After**: Unified under `mcp/` directory

- `mcp/config/` - Server configurations
- `mcp/data/` - MCP-specific data
- `mcp/logs/` - Server logs
- `mcp/monitoring/` - Monitoring data
- `mcp/servers/` - Server installations

### 3. Consistent Naming Conventions

**Before**: Mixed naming (underscores and hyphens)
**After**: Consistent hyphenated names for directories

- `plugin-data` → `plugins/data`
- `mcp_servers` → `mcp/servers`
- `mcp-data` → `mcp/data`

## API Functions

### Core Directory Functions

```go
// Base directories
utils.GetAppDataDir() (string, error)
utils.GetDatabaseDir() (string, error)

// Agent storage
utils.GetAgentsDBPath() (string, error)

// MCP directories
utils.GetMCPDir() (string, error)
utils.GetMCPConfigDir() (string, error)
utils.GetMCPServersDir() (string, error)
utils.GetMCPDataDir() (string, error)
utils.GetMCPLogsDir() (string, error)
utils.GetMCPMonitoringDir() (string, error)

// Plugin directories
utils.GetPluginsDir() (string, error)
utils.GetPluginDataDir() (string, error)

// Other directories
utils.GetTemplatesDir() (string, error)
utils.GetDownloadsDir() (string, error)
utils.GetLogsDir() (string, error)
utils.GetSecurityDir() (string, error)
utils.GetQuarantineDir() (string, error)
```

### Directory Creation

```go
// Ensures all required directories exist
utils.EnsureAppDataDirs() error

// Create specific directory if it doesn't exist
utils.EnsureDir(path string) error
```

## Usage Guidelines

### For Developers

1. **Always use utility functions** instead of hardcoded paths
2. **Call `EnsureAppDataDirs()`** during application initialization
3. **Use `GetAgentsDBPath()`** for all agent storage operations
4. **Use MCP-specific functions** for MCP-related operations

### Example Usage

```go
// Initialize agent storage
agentsDBPath, err := utils.GetAgentsDBPath()
if err != nil {
    return fmt.Errorf("failed to get agents DB path: %w", err)
}
registry, err := agent.NewAgentRegistry(agentsDBPath)

// Initialize MCP configuration
configDir, err := utils.GetMCPConfigDir()
if err != nil {
    return fmt.Errorf("failed to get MCP config dir: %w", err)
}
```

## Migration Notes

### Automatic Migration

The system automatically creates the new directory structure when the application starts. Legacy directories may coexist temporarily.

### Data Migration

- Agent data is automatically migrated to the unified storage system
- MCP services use the new directory structure immediately
- No manual migration steps required

## Benefits

1. **Consistency**: Unified naming and organization across all components
2. **Maintainability**: Clear separation of concerns and data types
3. **Scalability**: Organized structure supports future growth
4. **Cross-platform**: OS-appropriate data storage locations
5. **Backward Compatibility**: Existing code continues to work

## Implementation Status

✅ **Completed**:
- Directory structure standardization
- Agent storage unification
- MCP directory consolidation
- API function implementation
- Testing and validation

## Related Files

- `utils/appdata.go` - Core directory management functions
- `agent/agent_registry.go` - Unified agent storage implementation
- `agent/unified_agent_storage.go` - Centralized agent storage
- `api/mcp_*.go` - MCP services using new structure
- `docs/data_cleanup_plan.md` - Original implementation plan
- `docs/data_cleanup_worklog.md` - Implementation progress log


---

<div class="footer-links">


© 2025 KNIRV Network
</div>
