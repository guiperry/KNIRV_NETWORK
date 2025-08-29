# MCP Server Integration Architecture

## Overview

This document outlines the architecture for integrating Model Context Protocol (MCP) servers into the KNIRVENGINE, enabling automatic discovery, installation, and management of MCP servers from the official repository.

## Architecture Components

### 1. MCP Server Registry Service

**Location**: `api/mcp_registry_service.go`

**Responsibilities**:
- Fetch and parse MCP server metadata from GitHub repository
- Categorize servers by type (TypeScript/Python, functionality)
- Cache server information locally
- Provide search and filtering capabilities

**Key Features**:
- Periodic sync with GitHub repository
- Server metadata extraction from README files
- Dependency analysis (npm/uvx requirements)
- Version tracking and updates

### 2. MCP Server Installation Manager

**Location**: `api/mcp_installation_service.go`

**Responsibilities**:
- Handle installation of MCP servers via npm/uvx
- Manage dependencies and environment setup
- Track installation status and versions
- Handle uninstallation and cleanup

**Installation Types**:
- **TypeScript Servers**: Use `npx` for execution
- **Python Servers**: Use `uvx` for installation and execution
- **Custom Servers**: Support for custom installation scripts

### 3. MCP Server Lifecycle Manager

**Location**: `api/mcp_lifecycle_service.go`

**Responsibilities**:
- Start/stop/restart MCP servers
- Process monitoring and health checks
- Log management and debugging
- Resource usage tracking

**Process Management**:
- Spawn server processes with proper configuration
- Monitor server health via MCP protocol
- Handle server crashes and auto-restart
- Manage server ports and communication

### 4. MCP Configuration Manager

**Location**: `api/mcp_config_service.go`

**Responsibilities**:
- Store and manage server configurations
- Handle environment variables and CLI arguments
- Manage server-specific settings
- Configuration validation and templates

### 5. Enhanced Capabilities System

**Database Schema Extensions**:
```go
type MCPServer struct {
    ID              string            `json:"id"`
    Name            string            `json:"name"`
    Description     string            `json:"description"`
    Type            string            `json:"type"` // "typescript" | "python"
    Category        string            `json:"category"`
    Repository      string            `json:"repository"`
    InstallCommand  string            `json:"install_command"`
    RunCommand      string            `json:"run_command"`
    Status          string            `json:"status"` // "available" | "installing" | "installed" | "running" | "error"
    Version         string            `json:"version"`
    Dependencies    []string          `json:"dependencies"`
    Configuration   map[string]string `json:"configuration"`
    OwnerID         *int64            `json:"owner_id"`
    CreatedAt       time.Time         `json:"created_at"`
    UpdatedAt       time.Time         `json:"updated_at"`
}

type MCPCapability struct {
    ID          string    `json:"id"`
    ServerID    string    `json:"server_id"`
    Name        string    `json:"name"`
    Type        string    `json:"type"` // "tool" | "resource" | "prompt"
    Description string    `json:"description"`
    Schema      string    `json:"schema"`
    Enabled     bool      `json:"enabled"`
    CreatedAt   time.Time `json:"created_at"`
}
```

### 6. Frontend Integration

**New Components**:
- `MCPServerBrowser.jsx`: Browse and search available MCP servers
- `MCPServerManager.jsx`: Manage installed servers
- `MCPServerInstaller.jsx`: Handle server installation process
- `MCPCapabilityConfig.jsx`: Configure server capabilities

**Enhanced Existing Components**:
- Extend `CapabilityStore.jsx` to include MCP servers
- Update `MCPCapabilityManager.jsx` for better server management
- Enhance capability selection in agent configuration

## Data Flow

### 1. Server Discovery
```
GitHub Repository → Registry Service → Local Cache → Frontend Display
```

### 2. Server Installation
```
User Selection → Installation Service → Package Manager → Process Manager → Capability Registration
```

### 3. Server Execution
```
Agent Request → Capability Router → MCP Server → Response Processing → Agent Response
```

## Security Considerations

### 1. Sandboxing
- Run MCP servers in isolated processes
- Limit file system access
- Network access controls
- Resource usage limits

### 2. Configuration Validation
- Validate server configurations
- Sanitize environment variables
- Secure credential management
- Permission-based access control

### 3. Process Security
- Non-privileged execution
- Process isolation
- Secure communication channels
- Audit logging

## Implementation Phases

### Phase 1: Core Infrastructure
1. MCP Registry Service
2. Basic Installation Manager
3. Database schema updates
4. Basic frontend components

### Phase 2: Advanced Features
1. Lifecycle management
2. Configuration management
3. Enhanced UI components
4. Monitoring and logging

### Phase 3: Security & Production
1. Sandboxing implementation
2. Security hardening
3. Performance optimization
4. Production deployment features

## API Endpoints

### MCP Registry
- `GET /api/v1/mcp/servers` - List available servers
- `GET /api/v1/mcp/servers/:id` - Get server details
- `POST /api/v1/mcp/servers/sync` - Sync with GitHub repository

### MCP Installation
- `POST /api/v1/mcp/servers/:id/install` - Install server
- `DELETE /api/v1/mcp/servers/:id/uninstall` - Uninstall server
- `GET /api/v1/mcp/servers/:id/status` - Get installation status

### MCP Lifecycle
- `POST /api/v1/mcp/servers/:id/start` - Start server
- `POST /api/v1/mcp/servers/:id/stop` - Stop server
- `POST /api/v1/mcp/servers/:id/restart` - Restart server
- `GET /api/v1/mcp/servers/:id/health` - Health check

### MCP Configuration
- `GET /api/v1/mcp/servers/:id/config` - Get configuration
- `PUT /api/v1/mcp/servers/:id/config` - Update configuration
- `GET /api/v1/mcp/servers/:id/capabilities` - List server capabilities

## Integration Points

### 1. Agent System Integration
- Register MCP capabilities as agent tools
- Route capability requests to appropriate servers
- Handle server responses and errors
- Capability discovery and registration

### 2. Workflow Integration
- Include MCP capabilities in workflow orchestration
- Handle server dependencies in workflows
- Manage server lifecycle during workflow execution

### 3. Monitoring Integration
- Server health monitoring
- Performance metrics collection
- Error tracking and alerting
- Usage analytics

## Configuration Examples

### TypeScript Server Configuration
```json
{
  "id": "filesystem-server",
  "name": "Filesystem Server",
  "type": "typescript",
  "install_command": "npx -y @modelcontextprotocol/server-filesystem",
  "run_command": "npx @modelcontextprotocol/server-filesystem /allowed/path",
  "environment": {
    "NODE_ENV": "production"
  },
  "arguments": ["/allowed/path"]
}
```

### Python Server Configuration
```json
{
  "id": "git-server",
  "name": "Git Server",
  "type": "python",
  "install_command": "uvx mcp-server-git",
  "run_command": "uvx mcp-server-git --repository /path/to/repo",
  "environment": {
    "PYTHONPATH": "/custom/path"
  },
  "arguments": ["--repository", "/path/to/repo"]
}
```
