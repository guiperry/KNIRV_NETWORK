# API Endpoints Audit - Phase 1.3 API Standardization

## Current API Structure Overview

### V1 API Endpoints (simple_server.go) - ACTIVE
**Base Path**: `/api/v1`

#### Core Agent Management
| Method | Endpoint | Handler | Purpose | Status |
|--------|----------|---------|---------|--------|
| POST | `/agents` | `createAgentHandler` | Create new agent | ✅ Active |
| GET | `/agents` | `getAgentsHandler` | List agents by owner | ✅ Active |
| GET | `/agents/all` | `getAllAgentsHandler` | List all agents (DB + discovered) | ✅ Active |
| POST | `/agents/sync` | `syncAgentsHandler` | Sync agents between DB and registry | ✅ Active |
| GET | `/agents/{id}` | `getAgentHandler` | Get agent by ID | ✅ Active |
| PUT | `/agents/{id}` | `updateAgentHandler` | Update agent | ✅ Active |
| DELETE | `/agents/{id}` | `deleteAgentHandler` | Delete agent | ✅ Active |
| POST | `/agents/{id}/stop` | `stopAgentHandler` | Stop agent | ✅ Active |

#### Enhanced Agent Management
| Method | Endpoint | Handler | Purpose | Status |
|--------|----------|---------|---------|--------|
| POST | `/agents/{id}/versions` | `createAgentVersionHandler` | Create agent version | ✅ Active |
| GET | `/agents/{id}/versions` | `listAgentVersionsHandler` | List agent versions | ✅ Active |
| POST | `/agents/{id}/backup` | `createAgentBackupHandler` | Create agent backup | ✅ Active |
| GET | `/agents/{id}/backups` | `listAgentBackupsHandler` | List agent backups | ✅ Active |
| POST | `/agents/{id}/restore/{backupId}` | `restoreAgentFromBackupHandler` | Restore from backup | ✅ Active |
| GET | `/agents/{id}/health` | `performAgentHealthCheckHandler` | Health check | ✅ Active |
| GET | `/agents/{id}/health/history` | `getAgentHealthHistoryHandler` | Health history | ✅ Active |
| GET | `/agents/{id}/analytics/{period}` | `generateAgentAnalyticsHandler` | Agent analytics | ✅ Active |
| POST | `/agents/{id}/rebuild` | `rebuildAgentHandler` | Rebuild agent | ✅ Active |

#### Sub-Agent Management
| Method | Endpoint | Handler | Purpose | Status |
|--------|----------|---------|---------|--------|
| POST | `/agents/{id}/sub-agents` | `spawnSubAgentHandler` | Spawn sub-agent | ✅ Active |
| GET | `/agents/{id}/sub-agents` | `getSubAgentsHandler` | List sub-agents | ✅ Active |
| DELETE | `/agents/{id}/sub-agents/{subId}` | `terminateSubAgentHandler` | Terminate sub-agent | ✅ Active |
| GET | `/agents/{id}/sub-agents/{subId}/terminal` | `getSubAgentTerminalHandler` | Sub-agent terminal | ✅ Active |
| POST | `/agents/{id}/sub-agents/{subId}/command` | `sendSubAgentCommandHandler` | Send command | ✅ Active |
| GET | `/agents/{id}/sub-agents/{subId}/logs` | `getSubAgentLogsHandler` | Sub-agent logs | ✅ Active |

#### Plugin & Template Management
| Method | Endpoint | Handler | Purpose | Status |
|--------|----------|---------|---------|--------|
| GET | `/templates` | `getAgentTemplatesHandler` | List templates | ✅ Active |
| GET | `/plugins` | `getCompiledPluginsHandler` | List plugins | ✅ Active |
| DELETE | `/plugins/{id}` | `deleteAgentPluginHandler` | Delete plugin | ✅ Active |
| GET | `/plugins/discover` | `discoverAllPluginsHandler` | Discover plugins | ✅ Active |
| POST | `/plugins/import` | `importPluginHandler` | Import plugin | ✅ Active |

#### WASM Plugin Management
| Method | Endpoint | Handler | Purpose | Status |
|--------|----------|---------|---------|--------|
| GET | `/plugins/wasm/discover` | `discoverWASMPluginsHandler` | Discover WASM plugins | ✅ Active |
| POST | `/plugins/wasm/install` | `installWASMPluginHandler` | Install WASM plugin | ✅ Active |
| POST | `/plugins/wasm/uninstall` | `uninstallWASMPluginHandler` | Uninstall WASM plugin | ✅ Active |
| GET | `/plugins/wasm/installed` | `listInstalledWASMPluginsHandler` | List installed WASM | ✅ Active |

#### ADK Agent System (Plugin-based)
| Method | Endpoint | Handler | Purpose | Status |
|--------|----------|---------|---------|--------|
| GET | `/adk/agents` | `discoverAgentsHandler` | Discover ADK agents | ✅ Active |
| POST | `/adk/agents/activate` | `activateAgentHandler` | Activate ADK agent | ✅ Active |
| POST | `/adk/agents/deactivate` | `deactivateAgentHandler` | Deactivate ADK agent | ✅ Active |
| GET | `/adk/agents/capabilities` | `getAgentCapabilitiesHandler` | Get capabilities | ✅ Active |
| GET | `/adk/agents/schema` | `getAgentSchemaHandler` | Get schema | ✅ Active |
| POST | `/adk/agents/inference` | `processInferenceHandler` | Process inference | ✅ Active |
| POST | `/adk/agents/memory` | `setAgentMemoryHandler` | Set memory | ✅ Active |
| GET | `/adk/agents/memory` | `getAgentMemoryHandler` | Get memory | ✅ Active |
| GET | `/adk/agents/detailed` | `getAvailableAgentsDetailedHandler` | Detailed agent info | ✅ Active |

#### Target System
| Method | Endpoint | Handler | Purpose | Status |
|--------|----------|---------|---------|--------|
| GET | `/targets` | `getTargetsHandler` | List targets | ✅ Active |

#### Terminal Management
| Method | Endpoint | Handler | Purpose | Status |
|--------|----------|---------|---------|--------|
| POST | `/terminal/create` | `createTerminalHandler` | Create terminal | ✅ Active |
| POST | `/terminal/write` | `writeToTerminalHandler` | Write to terminal | ✅ Active |
| GET | `/terminal/read` | `readFromTerminalHandler` | Read from terminal | ✅ Active |
| POST | `/terminal/resize` | `resizeTerminalHandler` | Resize terminal | ✅ Active |
| POST | `/terminal/close` | `closeTerminalHandler` | Close terminal | ✅ Active |
| GET | `/terminal/ws` | `terminalWebSocketHandler` | Terminal WebSocket | ✅ Active |
| GET | `/terminal/logs` | `getTerminalLogsHandler` | Terminal logs | ✅ Active |

#### WebSocket Endpoints
| Method | Endpoint | Handler | Purpose | Status |
|--------|----------|---------|---------|--------|
| GET | `/ws` | `mainWebSocketHandler` | Main WebSocket | ✅ Active |
| GET | `/desktop/secure-ws` | `desktopSecureWebSocketHandler` | Desktop WebSocket | ✅ Active |

#### Settings & Configuration
| Method | Endpoint | Handler | Purpose | Status |
|--------|----------|---------|---------|--------|
| POST | `/settings/api-keys` | `handleAPIKeys` | Manage API keys | ✅ Active |
| GET | `/inference/models` | `handleInferenceModels` | List models | ✅ Active |
| POST | `/inference/moa/{type}` | `handleMOASettings` | MOA settings | ✅ Active |

#### AI Error Analysis
| Method | Endpoint | Handler | Purpose | Status |
|--------|----------|---------|---------|--------|
| POST | `/inference/analyze-error` | `handleAnalyzeError` | Analyze errors | ✅ Active |
| POST | `/inference/chat-error` | `handleChatError` | Chat error analysis | ✅ Active |

#### Debug Endpoints
| Method | Endpoint | Handler | Purpose | Status |
|--------|----------|---------|---------|--------|
| POST | `/debug/toggle-demo-data` | `handleToggleDemoData` | Toggle demo data | ✅ Active |
| GET | `/debug/demo-data-status` | `handleDemoDataStatus` | Demo data status | ✅ Active |
| POST | `/debug/clear-all-agents` | `handleClearAllAgents` | Clear all agents | ✅ Active |

#### Capabilities
| Method | Endpoint | Handler | Purpose | Status |
|--------|----------|---------|---------|--------|
| GET | `/capabilities` | `handleListCapabilities` | List capabilities | ✅ Active |
| GET | `/capabilities/mcp` | `handleListMCPCapabilities` | List MCP capabilities | ✅ Active |

#### User Management
| Method | Endpoint | Handler | Purpose | Status |
|--------|----------|---------|---------|--------|
| GET | `/users` | `handleListUsers` | List users | ✅ Active |
| GET | `/users/{id}` | `handleGetUser` | Get user | ✅ Active |
| POST | `/users` | `handleCreateUser` | Create user | ✅ Active |
| PUT | `/users/{id}` | `handleUpdateUser` | Update user | ✅ Active |
| DELETE | `/users/{id}` | `handleDeleteUser` | Delete user | ✅ Active |

#### Security Monitoring
| Method | Endpoint | Handler | Purpose | Status |
|--------|----------|---------|---------|--------|
| GET | `/security/status` | `handleSecurityStatus` | Security status | ✅ Active |
| GET | `/security/events` | `handleSecurityEvents` | Security events | ✅ Active |
| GET | `/security/shield/status` | `handleSHIELDStatus` | SHIELD status | ✅ Active |
| POST | `/security/shield/agents/{agentId}/monitor` | `handleStartAgentMonitoring` | Start monitoring | ✅ Active |
| POST | `/security/shield/agents/{agentId}/verify` | `handleVerifyAgentIntegrity` | Verify integrity | ✅ Active |

#### System Control
| Method | Endpoint | Handler | Purpose | Status |
|--------|----------|---------|---------|--------|
| POST | `/api/shutdown` | `handleShutdownRequest` | Shutdown system | ✅ Active |

### V2 API Endpoints (unified_agent_api.go) - INACTIVE (NOT REGISTERED)
**Base Path**: `/api/v2` - ❌ **NOT REGISTERED IN MAIN SERVER**

#### Core CRUD Operations
| Method | Endpoint | Handler | Purpose | Status |
|--------|----------|---------|---------|--------|
| GET | `/agents` | `handleListAgents` | List unified agents | ❌ Not Registered |
| POST | `/agents` | `handleCreateAgent` | Create unified agent | ❌ Not Registered |
| GET | `/agents/{id}` | `handleGetAgent` | Get unified agent | ❌ Not Registered |
| PUT | `/agents/{id}` | `handleUpdateAgent` | Update unified agent | ❌ Not Registered |
| DELETE | `/agents/{id}` | `handleDeleteAgent` | Delete unified agent | ❌ Not Registered |

#### Discovery Operations
| Method | Endpoint | Handler | Purpose | Status |
|--------|----------|---------|---------|--------|
| GET | `/agents/discover` | `handleDiscoverAgents` | Discover agents | ❌ Not Registered |
| POST | `/agents/register` | `handleRegisterAgent` | Register agent | ❌ Not Registered |

#### Configuration Operations
| Method | Endpoint | Handler | Purpose | Status |
|--------|----------|---------|---------|--------|
| GET | `/agents/{id}/config` | `handleGetAgentConfig` | Get agent config | ❌ Not Registered |
| PUT | `/agents/{id}/config` | `handleUpdateAgentConfig` | Update agent config | ❌ Not Registered |

#### Search Operations
| Method | Endpoint | Handler | Purpose | Status |
|--------|----------|---------|---------|--------|
| GET | `/agents/search` | `handleSearchAgents` | Search agents | ❌ Not Registered |

#### Status Operations
| Method | Endpoint | Handler | Purpose | Status |
|--------|----------|---------|---------|--------|
| POST | `/agents/{id}/activate` | `handleActivateAgent` | Activate agent | ❌ Not Registered |
| POST | `/agents/{id}/deactivate` | `handleDeactivateAgent` | Deactivate agent | ❌ Not Registered |

#### Filter Operations
| Method | Endpoint | Handler | Purpose | Status |
|--------|----------|---------|---------|--------|
| GET | `/agents/by-type/{type}` | `handleGetAgentsByType` | Get agents by type | ❌ Not Registered |
| GET | `/agents/by-status/{status}` | `handleGetAgentsByStatus` | Get agents by status | ❌ Not Registered |
| GET | `/agents/by-build-target/{target}` | `handleGetAgentsByBuildTarget` | Get agents by build target | ❌ Not Registered |

## Frontend API Expectations vs Backend Reality

### ❌ CRITICAL MISMATCH: Frontend calls `/api/v1/unified-agents/*` but backend provides `/api/v1/agents/*`

**Frontend Service (agentService.js) Calls**:
- `GET /api/v1/unified-agents` → **404 NOT FOUND**
- `POST /api/v1/unified-agents` → **404 NOT FOUND**  
- `GET /api/v1/unified-agents/{id}` → **404 NOT FOUND**
- `PUT /api/v1/unified-agents/{id}` → **404 NOT FOUND**
- `DELETE /api/v1/unified-agents/{id}` → **404 NOT FOUND**

**Backend Reality**:
- `GET /api/v1/agents` ✅ EXISTS
- `POST /api/v1/agents` ✅ EXISTS
- `GET /api/v1/agents/{id}` ✅ EXISTS  
- `PUT /api/v1/agents/{id}` ✅ EXISTS
- `DELETE /api/v1/agents/{id}` ✅ EXISTS

## Summary

- **V1 API**: 80+ endpoints active and registered
- **V2 API**: 15+ endpoints defined but NOT registered in main server
- **Frontend**: Expects `/unified-agents` paths that don't exist
- **Backend**: Provides `/agents` paths that frontend doesn't call

**Next Steps**: Update frontend to use existing `/api/v1/agents/*` endpoints and implement standardized response format.
