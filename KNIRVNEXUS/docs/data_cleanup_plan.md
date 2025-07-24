# Agentic Engine Data Storage Standardization Plan

## Current State Analysis

After analyzing the codebase, we've identified several critical issues with how data is currently stored in the Agentic Engine application:

1. **Agent Storage Duplication**: 
   - The agent registry uses `domain.db` in the database directory
   - There are multiple implementations for agent storage (AgentRegistry and UnifiedAgentStorage)
   - **Critical Issue**: The database and registry are separate entities when they should be the same thing
   - This creates data synchronization problems and potential inconsistencies

2. **MCP Data Organization**:
   - MCP (Multi-agent Coordination Protocol) data is scattered across multiple directories:
     - `mcp_config` in the config directory
     - `mcp_servers` in the appData root
     - `mcp-data` in the appData root
     - `mcp_logs` in the appData root
     - `mcp_monitoring` in the appData root

3. **Inconsistent Directory Structure**:
   - The application uses a mix of naming conventions (some with underscores, some with hyphens)
   - Some related data is stored in separate directories

## Directory Structure Analysis

The application currently uses the following directory structure based on `utils/appdata.go`:

```
AppDataDir/
├── data/                  # Database files (auth.db, domain.db)
├── config/                # Configuration files
│   └── mcp/               # MCP configuration
├── plugins/               # Plugin files
├── plugin-data/           # Plugin-specific data
├── mcp_servers/           # MCP server installations
├── mcp-data/              # MCP-specific data
├── mcp_logs/              # MCP server logs
├── mcp_monitoring/        # MCP monitoring data
├── logs/                  # Application logs
├── cache/                 # Cache files
│   └── workspace/         # Temporary workspace
├── backups/               # Backup files
├── quarantine/            # Quarantined files
└── security/              # Security-related files
```

## Standardization Goals

1. **Unify Agent Storage and Registry**:
   - **Primary Goal**: Make the database and registry the same entity
   - Define a single, consistent location for agent data (`data/agents.db`)
   - Eliminate duplicate storage mechanisms and consolidate to one approach
   - Ensure all agent operations use the same underlying storage

2. **Organize MCP Data**:
   - Create a unified `mcp` directory structure in the appData folder
   - Group all MCP-related data into appropriate subdirectories

3. **Establish Consistent Directory Structure**:
   - Use consistent naming conventions (prefer hyphenated names)
   - Group related data together in logical subdirectories

## Implementation Plan

### 1. Directory Structure Standardization

Define a standardized directory structure for the application:

```
AppDataDir/
├── data/                  # Database files
│   ├── auth.db            # Authentication database
│   ├── domain.db          # Domain database
│   └── agents.db          # Centralized agent database
├── config/                # Configuration files
├── plugins/               # Plugin files
│   └── data/              # Plugin-specific data (moved from plugin-data)
├── mcp/                   # All MCP-related data
│   ├── config/            # MCP configuration
│   ├── servers/           # MCP server installations
│   ├── data/              # MCP-specific data
│   ├── logs/              # MCP server logs
│   └── monitoring/        # MCP monitoring data
├── logs/                  # Application logs
├── cache/                 # Cache files
│   └── workspace/         # Temporary workspace
├── backups/               # Backup files
├── quarantine/            # Quarantined files
└── security/              # Security-related files
```

### 2. Code Updates

1. **Unify Agent Storage and Registry**:
   - Modify `agent/agent_registry.go` to use `UnifiedAgentStorage` as its backend
   - Deprecate the separate `AgentRegistry` implementation
   - Update all code that directly uses `AgentRegistry` to use the unified storage
   - Ensure a single database file (`agents.db`) is used for all agent operations

2. **Update AppData Directory Structure**:
   - Modify `utils/appdata.go` to define the standardized directory structure
   - Add a new function `GetAgentsDBPath()` that returns the path to the centralized agents database
   - Ensure consistent naming conventions

3. **Consolidate MCP Directory Functions**:
   - Update all MCP-related directory functions in `utils/appdata.go`
   - Create a new unified MCP directory structure

### 3. Documentation and Guidelines

Create clear documentation for developers on:
- Where different types of data should be stored
- Which storage mechanisms to use for different purposes
- Naming conventions and directory structure guidelines

## Detailed Implementation

### 1. AppData Directory Structure Updates

```go
// Standardized directory structure in utils/appdata.go

// GetAgentsDBPath returns the path to the centralized agents database
func GetAgentsDBPath() (string, error) {
    dbDir, err := GetDatabaseDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(dbDir, "agents.db"), nil
}

// GetMCPDir returns the base directory for all MCP-related data
func GetMCPDir() (string, error) {
    appDataDir, err := GetAppDataDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(appDataDir, "mcp"), nil
}

// GetMCPConfigDir returns the directory for MCP configurations
func GetMCPConfigDir() (string, error) {
    mcpDir, err := GetMCPDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(mcpDir, "config"), nil
}

// GetMCPServersDir returns the directory for MCP server installations
func GetMCPServersDir() (string, error) {
    mcpDir, err := GetMCPDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(mcpDir, "servers"), nil
}

// GetMCPDataDir returns the directory for MCP-specific data
func GetMCPDataDir() (string, error) {
    mcpDir, err := GetMCPDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(mcpDir, "data"), nil
}

// GetMCPLogsDir returns the directory for MCP server logs
func GetMCPLogsDir() (string, error) {
    mcpDir, err := GetMCPDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(mcpDir, "logs"), nil
}

// GetMCPMonitoringDir returns the directory for MCP monitoring data
func GetMCPMonitoringDir() (string, error) {
    mcpDir, err := GetMCPDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(mcpDir, "monitoring"), nil
}

// GetPluginDataDir returns the directory for storing plugin-specific data
func GetPluginDataDir() (string, error) {
    pluginsDir, err := GetPluginsDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(pluginsDir, "data"), nil
}
```

### 2. Unified Agent Storage Implementation

```go
// Update in main.go to use the unified agent storage approach
agentsDBPath, err := utils.GetAgentsDBPath()
if err != nil {
    log.Fatalf("Failed to get agents database path: %v", err)
}

// Create the unified agent storage as the single source of truth
unifiedStorage, err := agent.NewUnifiedAgentStorage(agentsDBPath)
if err != nil {
    log.Fatalf("Failed to create unified agent storage: %v", err)
}

// Create registry adapter that uses the unified storage
// This maintains compatibility with existing code that expects an AgentRegistry
registryAdapter := agent.NewAgentRegistryAdapter(unifiedStorage)

// Use the adapter anywhere the old registry was used
// This ensures all operations use the same underlying storage
```

### 3. Agent Registry Refactoring

```go
// In agent/agent_registry.go

// AgentRegistry now becomes a thin wrapper around UnifiedAgentStorage
type AgentRegistry struct {
    storage *UnifiedAgentStorage
}

// NewAgentRegistry creates a new agent registry backed by UnifiedAgentStorage
func NewAgentRegistry(dbPath string) (*AgentRegistry, error) {
    // Create the unified storage
    storage, err := NewUnifiedAgentStorage(dbPath)
    if err != nil {
        return nil, fmt.Errorf("failed to create unified storage: %v", err)
    }
    
    return &AgentRegistry{
        storage: storage,
    }, nil
}

// RegisterAgent now uses the unified storage
func (r *AgentRegistry) RegisterAgent(agentID string, config map[string]interface{}) error {
    return r.storage.RegisterAgentConfig(context.Background(), agentID, config)
}

// GetAgent now uses the unified storage
func (r *AgentRegistry) GetAgent(agentID string) (map[string]interface{}, error) {
    return r.storage.GetAgentConfig(context.Background(), agentID)
}

// Other methods similarly delegate to the unified storage
```

## Data Storage Guidelines

### Agent Data

- **Single Source of Truth**: `UnifiedAgentStorage` is the only storage mechanism for agent data
- **Database Location**: Always use `utils.GetAgentsDBPath()` to get the path to `agents.db`
- **Access Patterns**:
  - Direct: Use `UnifiedAgentStorage` methods for new code
  - Legacy: Use `AgentRegistry` which now internally uses `UnifiedAgentStorage`
  - Repository: Use `AgentRepositoryAdapter` for repository-pattern operations
- **Important**: Never create separate storage mechanisms for agents

### MCP Data

- **Base Directory**: All MCP data should be stored under the directory returned by `utils.GetMCPDir()`
- **Subdirectories**:
  - Configuration: `utils.GetMCPConfigDir()`
  - Server installations: `utils.GetMCPServersDir()`
  - Data: `utils.GetMCPDataDir()`
  - Logs: `utils.GetMCPLogsDir()`
  - Monitoring: `utils.GetMCPMonitoringDir()`

### Plugin Data

- **Plugin Files**: Store in the directory returned by `utils.GetPluginsDir()`
- **Plugin-specific Data**: Store in the directory returned by `utils.GetPluginDataDir()`

## Naming Conventions

1. **Directory Names**:
   - Use hyphenated lowercase names (e.g., `plugin-data` not `plugin_data`)
   - Avoid underscores in directory names

2. **Function Names**:
   - Use camelCase for function names
   - Use descriptive names that clearly indicate the purpose

3. **Database Files**:
   - Use lowercase names with `.db` extension
   - Use descriptive names that indicate the content (e.g., `agents.db`, `auth.db`)

## Implementation Timeline

1. **Code Updates**: 1 week
   - Update `utils/appdata.go` with standardized directory functions
   - Update agent storage code to use the centralized approach
   - Update MCP-related code to use the unified directory structure

2. **Testing**: 3 days
   - Verify all functionality works with the standardized structure
   - Ensure no regressions in existing features

3. **Documentation**: 2 days
   - Create developer guidelines for data storage
   - Update existing documentation to reflect the standardized approach

## Conclusion

This standardization plan addresses the critical issue of database and registry separation in the Agentic Engine application. By unifying the agent storage into a single source of truth and establishing clear guidelines for data organization, we can eliminate data synchronization problems and ensure consistency throughout the application.

The key benefits of this approach include:

1. **Elimination of Data Duplication**: By making the database and registry the same entity, we prevent inconsistencies and synchronization issues.

2. **Simplified Architecture**: Developers only need to understand one storage mechanism rather than multiple overlapping systems.

3. **Improved Maintainability**: A consistent directory structure and clear guidelines make the codebase easier to maintain and extend.

4. **Better Performance**: Eliminating duplicate storage operations can improve application performance.

By implementing these changes, we'll create a more robust foundation for the Agentic Engine application that will support its continued growth and development.