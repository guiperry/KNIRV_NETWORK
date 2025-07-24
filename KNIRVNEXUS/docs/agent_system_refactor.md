# Agent System Refactoring Proposal

## Primary Objective

The primary objective of this refactoring is to systematically eliminate all redundancies in the agent system while ensuring we don't introduce new ones in the process. The current system has evolved organically, resulting in duplicated functionality, overlapping responsibilities, and inconsistent interfaces. This refactoring aims to consolidate related functionality, establish clear component boundaries, and create a more maintainable architecture without adding new layers of complexity or duplication.go mod tidy


## Current System Analysis

After analyzing the agent creation, discovery, loading, and listing system in the Agentic Engine, I've identified several redundancies and areas for improvement. This document outlines the current architecture, identifies issues, and proposes a comprehensive refactoring plan that eliminates these redundancies while maintaining or enhancing functionality.

### Current Architecture Overview

The agent system is spread across multiple components:

1. **Agent Registry (`agent/agent_registry.go`)**
   - A thin wrapper around UnifiedAgentStorage
   - Provides basic CRUD operations for agent configurations

2. **Unified Agent Storage (`agent/unified_agent_storage.go`)**
   - Core storage system using chromem-go for agent persistence
   - Includes multiple adapter patterns to maintain compatibility with different interfaces
   - Handles agent configuration, metadata, and plugin information

3. **Agent Builder (`agent/agent_builder.go`)**
   - Manages agent creation and template processing
   - Builds plugins from templates
   - Registers agents with the registry

4. **Enhanced Agent Manager (`agent/enhanced_agent_manager.go`)**
   - Provides advanced agent management capabilities
   - Handles versioning, backups, health checks, and analytics
   - Operates on top of the registry and builder

5. **Agent Inferencer (`agentify/agent_inferencer.go`)**
   - Manages agent execution and inference
   - Handles agent activation, session management, and inference processing
   - Includes auto-registration of discovered agents

6. **Agent Plugin Loader (`agentify/agent_plugin_loader.go`)**
   - Loads and manages native plugin agents
   - Discovers available plugins
   - Handles plugin metadata extraction

7. **Agent WASM Loader (`agentify/agent_wasm_loader.go`)**
   - Loads and manages WebAssembly-based agents
   - Discovers available WASM agents
   - Provides WASM runtime integration

### Identified Redundancies and Issues

1. **Duplicate Agent Discovery Logic**
   - Both `AgentPluginLoader.DiscoverPlugins()` and `AgentInferencer.ListAvailableAgents()` perform similar discovery operations
   - `AgentInferencer.SyncDiscoveredAgentsToRegistry()` duplicates functionality in `autoRegisterAgents()`

2. **Multiple Storage Abstractions**
   - `AgentRegistry` is now just a thin wrapper around `UnifiedAgentStorage`
   - Multiple adapter patterns (`AgentRegistryAdapter`, `AgentRepositoryAdapter`, `UnifiedAgentStorageAdapter`) create unnecessary complexity

3. **Inconsistent Agent Representation**
   - Multiple agent data structures (`UnifiedAgent`, `Agent`, `AgentConfig`, `AgentInfo`) with overlapping fields
   - Conversion between these structures adds complexity and potential for errors

4. **Redundant Registration Mechanisms**
   - Agent registration happens in multiple places (Builder, Inferencer)
   - Duplicate code for extracting agent metadata from filenames

5. **Complex Initialization Chains**
   - Multiple constructors with different parameters (`NewAgentRegistry`, `NewAgentBuilderWithStorage`, etc.)
   - Dependency injection is inconsistent across components

6. **Separate Plugin and WASM Loading Logic**
   - Similar but separate code paths for plugin and WASM agents
   - Duplicate discovery, loading, and management logic

7. **Scattered Agent Lifecycle Management**
   - Agent activation, deactivation, and session management are separate from storage and discovery
   - No unified approach to agent lifecycle events

## Proposed Refactoring

### 1. Unified Agent Core System

Create a single, unified agent core system that consolidates storage, discovery, and basic operations:

```go
// agent/core/agent_core.go
package core

// UnifiedAgent represents the complete agent data model
type UnifiedAgent struct {
    // Core fields
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Type        string    `json:"type"`
    Version     string    `json:"version"`
    Description string    `json:"description"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
    
    // Configuration
    Config      map[string]interface{} `json:"config"`
    
    // Runtime information
    BuildTarget string   `json:"build_target"` // "plugin", "wasm", etc.
    PluginPath  string   `json:"plugin_path,omitempty"`
    Status      string   `json:"status"`
    
    // Metadata
    Collection   string   `json:"collection"`
    ImageURL     string   `json:"image_url,omitempty"`
    Capabilities []string `json:"capabilities"`
    TargetTypes  []string `json:"target_types"`
    Tags         []string `json:"tags,omitempty"`
    
    // Security
    OwnerID     int64               `json:"owner_id"`
    APIKeys     map[string]string   `json:"api_keys,omitempty"`
    Permissions map[string]bool     `json:"permissions,omitempty"`
    
    // Terminal configuration
    DefaultTerminalConfig *TerminalConfig `json:"default_terminal_config,omitempty"`
}

// TerminalConfig represents terminal configuration for an agent
type TerminalConfig struct {
    DefaultRows    int               `json:"default_rows"`
    DefaultCols    int               `json:"default_cols"`
    FontSize       int               `json:"font_size"`
    FontFamily     string            `json:"font_family"`
    Theme          string            `json:"theme"`
    ScrollbackSize int               `json:"scrollback_size"`
    AutoOpen       bool              `json:"auto_open"`
    CustomCSS      string            `json:"custom_css,omitempty"`
    CustomOptions  map[string]string `json:"custom_options,omitempty"`
}

// AgentCoreService provides a unified interface for agent operations
type AgentCoreService interface {
    // Core CRUD operations
    CreateAgent(ctx context.Context, agent *UnifiedAgent) error
    GetAgent(ctx context.Context, id string) (*UnifiedAgent, error)
    UpdateAgent(ctx context.Context, agent *UnifiedAgent) error
    DeleteAgent(ctx context.Context, id string) error
    ListAgents(ctx context.Context, filter map[string]interface{}) ([]*UnifiedAgent, error)
    
    // Discovery operations
    DiscoverAgents(ctx context.Context) ([]*UnifiedAgent, error)
    RegisterDiscoveredAgent(ctx context.Context, agentPath string) (*UnifiedAgent, error)
    
    // Configuration operations
    GetAgentConfig(ctx context.Context, id string) (map[string]interface{}, error)
    UpdateAgentConfig(ctx context.Context, id string, config map[string]interface{}) error
    
    // Search operations
    SearchAgents(ctx context.Context, query string, limit int) ([]*UnifiedAgent, error)
    
    // Lifecycle hooks
    OnAgentCreated(agent *UnifiedAgent)
    OnAgentUpdated(agent *UnifiedAgent)
    OnAgentDeleted(id string)
}
```

### 2. Unified Agent Runtime System

Create a unified runtime system that handles both plugin and WASM agents:

```go
// agentify/runtime/agent_runtime.go
package runtime

// TerminalSession represents an active terminal session
type TerminalSession struct {
    ID           string    `json:"id"`
    AgentID      string    `json:"agent_id"`
    SessionID    string    `json:"session_id"`
    Rows         int       `json:"rows"`
    Cols         int       `json:"cols"`
    CreatedAt    time.Time `json:"created_at"`
    LastActivity time.Time `json:"last_activity"`
    PID          int       `json:"pid,omitempty"`
    Status       string    `json:"status"` // "active", "idle", "closed"
    BufferSize   int       `json:"buffer_size"`
    OutputBuffer []byte    `json:"output_buffer,omitempty"`
}

// TerminalManager handles terminal operations across all agents
type TerminalManager interface {
    // Terminal lifecycle
    CreateTerminal(ctx context.Context, agentID, sessionID string, rows, cols int) (*TerminalSession, error)
    ResizeTerminal(ctx context.Context, terminalID string, rows, cols int) error
    CloseTerminal(ctx context.Context, terminalID string) error
    
    // Terminal I/O
    WriteToTerminal(ctx context.Context, terminalID string, data []byte) error
    ReadFromTerminal(ctx context.Context, terminalID string) ([]byte, error)
    
    // Terminal management
    ListTerminals(ctx context.Context, sessionID string) ([]*TerminalSession, error)
    GetTerminalSession(ctx context.Context, terminalID string) (*TerminalSession, error)
    
    // Terminal events
    SubscribeToTerminalOutput(ctx context.Context, terminalID string) (<-chan []byte, error)
    UnsubscribeFromTerminalOutput(ctx context.Context, terminalID string, subID string) error
}

// AgentRuntime represents the unified runtime for all agent types
type AgentRuntime interface {
    // Lifecycle management
    Initialize(config map[string]interface{}) error
    Start() error
    Stop() error
    
    // Inference processing
    ProcessInference(ctx context.Context, request *InferenceRequest) (*InferenceResponse, error)
    
    // Capabilities and schema
    GetCapabilities() *AgentCapabilities
    GetSchema() *AgentSchema
    
    // Memory management
    GetMemory(key string) (interface{}, error)
    SetMemory(key string, value interface{}) error
    
    // Terminal management
    CreateTerminal(rows, cols int) (string, error)
    ResizeTerminal(terminalID string, rows, cols int) error
    WriteToTerminal(terminalID string, data []byte) error
    ReadFromTerminal(terminalID string) ([]byte, error)
    CloseTerminal(terminalID string) error
    
    // Tool execution
    CallTool(ctx context.Context, toolName string, params map[string]interface{}) (interface{}, error)
}

// AgentRuntimeManager manages agent runtimes
type AgentRuntimeManager interface {
    // Runtime management
    LoadAgent(ctx context.Context, agentID string, config map[string]interface{}) (AgentRuntime, error)
    UnloadAgent(ctx context.Context, agentID string) error
    GetAgentRuntime(agentID string) (AgentRuntime, error)
    
    // Session management
    CreateSession(ctx context.Context, agentID string) (string, error)
    GetSession(sessionID string) (string, error) // Returns agentID
    CloseSession(ctx context.Context, sessionID string) error
    
    // Terminal management
    GetTerminalManager() TerminalManager
    
    // Discovery
    DiscoverRuntimeAgents(ctx context.Context) (map[string]string, error) // Returns map[agentID]buildTarget
}
```

### 3. Agent Builder System

Refactor the agent builder to focus solely on agent creation and template processing:

```go
// agent/builder/agent_builder.go
package builder

// AgentBuilder handles agent creation from templates
type AgentBuilder interface {
    // Template management
    ListTemplates() ([]string, error)
    GetTemplateDetails(templateName string) (*TemplateDetails, error)
    ValidateTemplate(templateName string) error
    
    // Build operations
    BuildAgent(config *BuildConfig) (*BuildResult, error)
    GetBuildStatus(buildID string) (*BuildStatus, error)
    CancelBuild(buildID string) error
    
    // Post-build operations
    RegisterBuiltAgent(ctx context.Context, buildResult *BuildResult) (string, error)
}

// BuildConfig represents the configuration for building an agent
type BuildConfig struct {
    AgentID      string                 `json:"agent_id,omitempty"`
    Name         string                 `json:"name"`
    Type         string                 `json:"type"`
    Template     string                 `json:"template"`
    BuildTarget  string                 `json:"build_target"` // "plugin", "wasm", etc.
    Version      string                 `json:"version"`
    Description  string                 `json:"description,omitempty"`
    Instruction  string                 `json:"instruction,omitempty"`
    Capabilities []string               `json:"capabilities,omitempty"`
    TargetTypes  []string               `json:"target_types,omitempty"`
    ExtraParams  map[string]interface{} `json:"extra_params,omitempty"`
}

// BuildResult represents the result of a build operation
type BuildResult struct {
    BuildID      string                 `json:"build_id"`
    AgentID      string                 `json:"agent_id"`
    Status       string                 `json:"status"`
    OutputPath   string                 `json:"output_path,omitempty"`
    BuildTarget  string                 `json:"build_target"`
    Config       map[string]interface{} `json:"config"`
    StartedAt    time.Time              `json:"started_at"`
    CompletedAt  time.Time              `json:"completed_at,omitempty"`
    ErrorMessage string                 `json:"error_message,omitempty"`
    LogOutput    []string               `json:"log_output,omitempty"`
}
```

### 4. Agent Management System

Create a dedicated management system for advanced agent operations:

```go
// agent/manager/agent_manager.go
package manager

// AgentManager provides advanced agent management capabilities
type AgentManager interface {
    // Version management
    CreateVersion(ctx context.Context, agentID string, version string, changelog string) (*AgentVersion, error)
    ListVersions(ctx context.Context, agentID string) ([]*AgentVersion, error)
    SwitchVersion(ctx context.Context, agentID string, version string) error
    
    // Backup management
    CreateBackup(ctx context.Context, agentID string, description string) (*AgentBackup, error)
    ListBackups(ctx context.Context, agentID string) ([]*AgentBackup, error)
    RestoreBackup(ctx context.Context, backupID string) error
    
    // Health management
    PerformHealthCheck(ctx context.Context, agentID string) (*AgentHealth, error)
    GetHealthHistory(ctx context.Context, agentID string) ([]*AgentHealth, error)
    
    // Analytics
    GenerateAnalytics(ctx context.Context, agentID string, period string) (*AgentAnalytics, error)
    GetAnalyticsHistory(ctx context.Context, agentID string) ([]*AgentAnalytics, error)
    
    // Deployment management
    DeployAgent(ctx context.Context, agentID string, environment string) (*Deployment, error)
    ListDeployments(ctx context.Context, agentID string) ([]*Deployment, error)
    UndeployAgent(ctx context.Context, deploymentID string) error
}
```

### 5. Unified Agent Service

Create a high-level service that integrates all components:

```go
// agent/service/agent_service.go
package service

// AgentService provides a unified interface for all agent operations
type AgentService struct {
    core    core.AgentCoreService
    runtime runtime.AgentRuntimeManager
    builder builder.AgentBuilder
    manager manager.AgentManager
}

// NewAgentService creates a new agent service
func NewAgentService(config *ServiceConfig) (*AgentService, error) {
    // Initialize components with proper dependency injection
    coreService, err := core.NewAgentCoreService(config.DBPath)
    if err != nil {
        return nil, err
    }
    
    runtimeManager, err := runtime.NewAgentRuntimeManager(config.PluginsDir)
    if err != nil {
        return nil, err
    }
    
    agentBuilder, err := builder.NewAgentBuilder(config.TemplatesDir, config.OutputDir)
    if err != nil {
        return nil, err
    }
    
    agentManager, err := manager.NewAgentManager(config.DataDir)
    if err != nil {
        return nil, err
    }
    
    // Connect components
    runtimeManager.SetCoreService(coreService)
    agentBuilder.SetCoreService(coreService)
    agentManager.SetCoreService(coreService)
    
    return &AgentService{
        core:    coreService,
        runtime: runtimeManager,
        builder: agentBuilder,
        manager: agentManager,
    }, nil
}

// Expose unified methods that delegate to appropriate components
// ...
```

## Terminal Management System

The terminal management system is a critical component that enables interactive communication with agents. The refactored architecture includes a comprehensive terminal management system that spans both backend and frontend.

### Backend Terminal Management

```go
// agentify/terminal/terminal_manager.go
package terminal

// TerminalImplementation defines the interface for terminal backend implementations
type TerminalImplementation interface {
    // Lifecycle methods
    Create(rows, cols int, env map[string]string) (string, error)
    Resize(id string, rows, cols int) error
    Write(id string, data []byte) error
    Read(id string) ([]byte, error)
    Close(id string) error
    
    // Status methods
    IsAlive(id string) bool
    GetPID(id string) int
    GetStats(id string) *TerminalStats
}

// TerminalStats provides statistics about a terminal session
type TerminalStats struct {
    BytesRead       int64     `json:"bytes_read"`
    BytesWritten    int64     `json:"bytes_written"`
    LastActivity    time.Time `json:"last_activity"`
    StartTime       time.Time `json:"start_time"`
    CommandCount    int       `json:"command_count"`
    MemoryUsage     int64     `json:"memory_usage"`
    CPUUsage        float64   `json:"cpu_usage"`
    ExitCode        int       `json:"exit_code,omitempty"`
    HasExited       bool      `json:"has_exited"`
}

// TerminalManagerImpl implements the TerminalManager interface
type TerminalManagerImpl struct {
    terminals       map[string]*ManagedTerminal
    implementations map[string]TerminalImplementation
    eventBus        *EventBus
    mutex           sync.RWMutex
}

// ManagedTerminal represents a terminal managed by the terminal manager
type ManagedTerminal struct {
    Session      *TerminalSession
    Implementation string
    EventChannel chan []byte
    Subscribers  map[string]chan []byte
}

// NewTerminalManager creates a new terminal manager
func NewTerminalManager() *TerminalManagerImpl {
    manager := &TerminalManagerImpl{
        terminals:       make(map[string]*ManagedTerminal),
        implementations: make(map[string]TerminalImplementation),
        eventBus:        NewEventBus(),
    }
    
    // Register default implementations
    manager.RegisterImplementation("pty", NewPtyTerminal())
    manager.RegisterImplementation("wasm", NewWasmTerminal())
    manager.RegisterImplementation("docker", NewDockerTerminal())
    
    return manager
}

// RegisterImplementation registers a terminal implementation
func (m *TerminalManagerImpl) RegisterImplementation(name string, impl TerminalImplementation) {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    m.implementations[name] = impl
}
```

### Frontend Terminal Integration

#### Agent Terminal Component

```typescript
// gui/src/components/terminal/AgentTerminal.tsx
import React, { useEffect, useRef, useState } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebLinksAddon } from '@xterm/addon-web-links';
import { WebglAddon } from '@xterm/addon-webgl';
import { SearchAddon } from '@xterm/addon-search';
import '@xterm/xterm/css/xterm.css';
import { useWebSocket } from '../../hooks/useWebSocket';
import { useAgentContext } from '../../contexts/AgentContext';
import { useErrorInferencer } from '../../hooks/useErrorInferencer';

interface AgentTerminalProps {
  terminalId: string;
  sessionId: string;
  agentId: string;
  initialRows?: number;
  initialCols?: number;
  theme?: 'light' | 'dark' | 'custom';
  customTheme?: any;
  onReady?: (terminal: Terminal) => void;
  onClose?: () => void;
  onError?: (error: Error) => void;
  compact?: boolean;
}

export const AgentTerminal: React.FC<AgentTerminalProps> = ({
  terminalId,
  sessionId,
  agentId,
  initialRows = 24,
  initialCols = 80,
  theme = 'dark',
  customTheme,
  onReady,
  onClose,
  onError,
  compact = false,
}) => {
  const terminalRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const [isReady, setIsReady] = useState(false);
  const { agentService } = useAgentContext();
  const { analyzeError } = useErrorInferencer();
  
  // WebSocket connection for real-time terminal updates
  const { connected, message, send, error: wsError } = useWebSocket(
    `ws://localhost:8080/api/terminal/${terminalId}/ws`
  );

  // Handle WebSocket errors
  useEffect(() => {
    if (wsError) {
      console.error('Terminal WebSocket error:', wsError);
      if (onError) {
        onError(new Error(`Terminal connection error: ${wsError.message}`));
      }
      
      // Send to error inferencer
      analyzeError({
        source: 'agent_terminal',
        agentId,
        sessionId,
        terminalId,
        error: wsError.message,
        context: {
          component: 'AgentTerminal',
          connectionType: 'WebSocket'
        }
      });
    }
  }, [wsError]);

  useEffect(() => {
    // Initialize terminal
    if (terminalRef.current && !xtermRef.current) {
      const terminal = new Terminal({
        rows: compact ? 12 : initialRows,
        cols: compact ? 60 : initialCols,
        theme: getThemeConfig(theme, customTheme),
        cursorBlink: true,
        scrollback: 5000,
        fontFamily: 'Menlo, Monaco, "Courier New", monospace',
        fontSize: compact ? 12 : 14,
      });
      
      // Add addons
      const fitAddon = new FitAddon();
      terminal.loadAddon(fitAddon);
      terminal.loadAddon(new WebLinksAddon());
      
      // Only add WebGL in full mode for performance
      if (!compact) {
        try {
          terminal.loadAddon(new WebglAddon());
        } catch (error) {
          console.warn('WebGL addon failed to load:', error);
        }
      }
      
      terminal.loadAddon(new SearchAddon());
      
      // Store refs
      xtermRef.current = terminal;
      fitAddonRef.current = fitAddon;
      
      // Open terminal
      terminal.open(terminalRef.current);
      fitAddon.fit();
      
      // Handle user input
      terminal.onData((data) => {
        try {
          if (connected) {
            send(data);
          } else {
            // Send via REST API if WebSocket is not connected
            agentService.writeToTerminal(terminalId, data);
          }
        } catch (error) {
          console.error('Failed to send terminal data:', error);
          if (error instanceof Error) {
            analyzeError({
              source: 'agent_terminal',
              agentId,
              sessionId,
              terminalId,
              error: error.message,
              context: {
                component: 'AgentTerminal',
                action: 'sendData'
              }
            });
          }
        }
      });
      
      // Handle resize
      const handleResize = () => {
        if (fitAddonRef.current && xtermRef.current) {
          fitAddonRef.current.fit();
          const { rows, cols } = xtermRef.current;
          agentService.resizeTerminal(terminalId, rows, cols)
            .catch(error => {
              console.error('Failed to resize terminal:', error);
              analyzeError({
                source: 'agent_terminal',
                agentId,
                sessionId,
                terminalId,
                error: error.message,
                context: {
                  component: 'AgentTerminal',
                  action: 'resize',
                  dimensions: { rows, cols }
                }
              });
            });
        }
      };
      
      window.addEventListener('resize', handleResize);
      
      // Notify parent component
      setIsReady(true);
      if (onReady) {
        onReady(terminal);
      }
      
      return () => {
        window.removeEventListener('resize', handleResize);
        terminal.dispose();
        if (onClose) {
          onClose();
        }
      };
    }
  }, [terminalRef.current]);
  
  // Handle WebSocket messages
  useEffect(() => {
    if (message && xtermRef.current) {
      xtermRef.current.write(message);
    }
  }, [message]);
  
  // Poll for terminal data if WebSocket is not connected
  useEffect(() => {
    if (!connected && isReady) {
      const interval = setInterval(async () => {
        try {
          const data = await agentService.readFromTerminal(terminalId);
          if (data && xtermRef.current) {
            xtermRef.current.write(data);
          }
        } catch (error) {
          console.error('Failed to read terminal data:', error);
          if (error instanceof Error) {
            analyzeError({
              source: 'agent_terminal',
              agentId,
              sessionId,
              terminalId,
              error: error.message,
              context: {
                component: 'AgentTerminal',
                action: 'pollData'
              }
            });
          }
        }
      }, 500);
      
      return () => clearInterval(interval);
    }
  }, [connected, isReady, terminalId]);
  
  return (
    <div className={`agent-terminal ${compact ? 'agent-terminal-compact' : ''}`} 
         ref={terminalRef} 
         style={{ 
           height: '100%', 
           width: '100%',
           borderRadius: compact ? '4px' : '0',
           overflow: 'hidden'
         }} 
    />
  );
};

// Helper function to get theme configuration
function getThemeConfig(theme: string, customTheme?: any) {
  switch (theme) {
    case 'light':
      return {
        background: '#ffffff',
        foreground: '#000000',
        cursor: '#333333',
        // ... other light theme colors
      };
    case 'dark':
      return {
        background: '#1e1e1e',
        foreground: '#f0f0f0',
        cursor: '#ffffff',
        // ... other dark theme colors
      };
    case 'custom':
      return customTheme || {};
    default:
      return {};
  }
}
```

#### Agent Card with Integrated Terminal

```typescript
// gui/src/components/agent/AgentCard.tsx
import React, { useState, useEffect } from 'react';
import { Card, CardHeader, CardContent, CardActions, IconButton, Button, Collapse, Typography, Box } from '@mui/material';
import { Maximize2, Minimize2, Terminal as TerminalIcon, Play, Pause, Settings, X, ChevronDown, ChevronUp } from 'lucide-react';
import { AgentTerminal } from '../terminal/AgentTerminal';
import { useAgentContext } from '../../contexts/AgentContext';
import { useErrorInferencer } from '../../hooks/useErrorInferencer';

interface AgentCardProps {
  agent: UnifiedAgent;
  onActivate?: (agentId: string) => void;
  onDeactivate?: (agentId: string) => void;
  onSettings?: (agentId: string) => void;
  onRemove?: (agentId: string) => void;
  active?: boolean;
}

export const AgentCard: React.FC<AgentCardProps> = ({
  agent,
  onActivate,
  onDeactivate,
  onSettings,
  onRemove,
  active = false,
}) => {
  const [expanded, setExpanded] = useState(false);
  const [terminalVisible, setTerminalVisible] = useState(false);
  const [terminalExpanded, setTerminalExpanded] = useState(false);
  const [terminalId, setTerminalId] = useState<string | null>(null);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const { agentService } = useAgentContext();
  const { analyzeError } = useErrorInferencer();

  // Initialize terminal when agent becomes active
  useEffect(() => {
    if (active && agent.DefaultTerminalConfig?.AutoOpen && !terminalId) {
      handleShowTerminal();
    }
  }, [active, agent]);

  // Clean up terminal when component unmounts
  useEffect(() => {
    return () => {
      if (terminalId) {
        agentService.closeTerminal(terminalId)
          .catch(error => {
            console.error('Failed to close terminal:', error);
            analyzeError({
              source: 'agent_card',
              agentId: agent.ID,
              sessionId: sessionId || undefined,
              error: error.message,
              context: {
                component: 'AgentCard',
                action: 'closeTerminal'
              }
            });
          });
      }
    };
  }, [terminalId]);

  const handleActivate = () => {
    if (onActivate) {
      onActivate(agent.ID);
    }
    
    // Create a session for this agent
    agentService.createSession(agent.ID)
      .then(newSessionId => {
        setSessionId(newSessionId);
        // Auto-open terminal if configured
        if (agent.DefaultTerminalConfig?.AutoOpen) {
          handleShowTerminal();
        }
      })
      .catch(error => {
        console.error('Failed to create agent session:', error);
        analyzeError({
          source: 'agent_card',
          agentId: agent.ID,
          error: error.message,
          context: {
            component: 'AgentCard',
            action: 'createSession'
          }
        });
      });
  };

  const handleDeactivate = () => {
    if (onDeactivate) {
      onDeactivate(agent.ID);
    }
    
    // Close the session
    if (sessionId) {
      agentService.closeSession(sessionId)
        .catch(error => {
          console.error('Failed to close agent session:', error);
          analyzeError({
            source: 'agent_card',
            agentId: agent.ID,
            sessionId,
            error: error.message,
            context: {
              component: 'AgentCard',
              action: 'closeSession'
            }
          });
        });
      setSessionId(null);
    }
    
    // Close the terminal
    if (terminalId) {
      agentService.closeTerminal(terminalId)
        .catch(error => {
          console.error('Failed to close terminal:', error);
          analyzeError({
            source: 'agent_card',
            agentId: agent.ID,
            sessionId,
            terminalId,
            error: error.message,
            context: {
              component: 'AgentCard',
              action: 'closeTerminal'
            }
          });
        });
      setTerminalId(null);
    }
    
    setTerminalVisible(false);
    setTerminalExpanded(false);
  };

  const handleShowTerminal = () => {
    if (!sessionId) {
      console.error('Cannot show terminal: No active session');
      return;
    }
    
    // Get terminal config
    const config = agent.DefaultTerminalConfig || {
      DefaultRows: 24,
      DefaultCols: 80,
      FontSize: 14,
      Theme: 'dark'
    };
    
    // Create a terminal
    agentService.createTerminal(agent.ID, sessionId, config.DefaultRows, config.DefaultCols)
      .then(newTerminalId => {
        setTerminalId(newTerminalId);
        setTerminalVisible(true);
      })
      .catch(error => {
        console.error('Failed to create terminal:', error);
        analyzeError({
          source: 'agent_card',
          agentId: agent.ID,
          sessionId,
          error: error.message,
          context: {
            component: 'AgentCard',
            action: 'createTerminal'
          }
        });
      });
  };

  const handleHideTerminal = () => {
    setTerminalVisible(false);
    
    // Don't close the terminal, just hide it
    // This allows the terminal state to be preserved
  };

  const handleExpandTerminal = () => {
    setTerminalExpanded(!terminalExpanded);
  };

  return (
    <Card className={`agent-card ${active ? 'agent-card-active' : ''}`}>
      <CardHeader
        title={agent.Name}
        subheader={`v${agent.Version} - ${agent.Type}`}
        action={
          <Box sx={{ display: 'flex' }}>
            {active && (
              <IconButton onClick={handleExpandTerminal} size="small">
                {terminalExpanded ? <Minimize2 size={18} /> : <Maximize2 size={18} />}
              </IconButton>
            )}
            <IconButton onClick={() => setExpanded(!expanded)} size="small">
              {expanded ? <ChevronUp size={18} /> : <ChevronDown size={18} />}
            </IconButton>
          </Box>
        }
      />
      
      {/* Terminal area - shown in compact mode when not expanded */}
      {active && terminalVisible && !terminalExpanded && terminalId && sessionId && (
        <Box sx={{ height: '150px', position: 'relative' }}>
          <AgentTerminal
            terminalId={terminalId}
            sessionId={sessionId}
            agentId={agent.ID}
            compact={true}
            theme={agent.DefaultTerminalConfig?.Theme as any || 'dark'}
            onError={(error) => {
              analyzeError({
                source: 'agent_terminal',
                agentId: agent.ID,
                sessionId,
                terminalId,
                error: error.message,
                context: {
                  component: 'AgentTerminal',
                  mode: 'compact'
                }
              });
            }}
          />
        </Box>
      )}
      
      {/* Expanded terminal modal */}
      {active && terminalVisible && terminalExpanded && terminalId && sessionId && (
        <div className="terminal-modal">
          <div className="terminal-modal-header">
            <Typography variant="h6">{agent.Name} Terminal</Typography>
            <IconButton onClick={handleExpandTerminal}>
              <Minimize2 size={18} />
            </IconButton>
          </div>
          <div className="terminal-modal-content">
            <AgentTerminal
              terminalId={terminalId}
              sessionId={sessionId}
              agentId={agent.ID}
              theme={agent.DefaultTerminalConfig?.Theme as any || 'dark'}
              initialRows={agent.DefaultTerminalConfig?.DefaultRows || 24}
              initialCols={agent.DefaultTerminalConfig?.DefaultCols || 80}
              onError={(error) => {
                analyzeError({
                  source: 'agent_terminal',
                  agentId: agent.ID,
                  sessionId,
                  terminalId,
                  error: error.message,
                  context: {
                    component: 'AgentTerminal',
                    mode: 'expanded'
                  }
                });
              }}
            />
          </div>
        </div>
      )}
      
      <Collapse in={expanded}>
        <CardContent>
          <Typography variant="body2" color="text.secondary">
            {agent.Description || 'No description available.'}
          </Typography>
          
          <Box sx={{ mt: 2 }}>
            <Typography variant="subtitle2">Capabilities:</Typography>
            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, mt: 0.5 }}>
              {agent.Capabilities.map((capability, index) => (
                <Box key={index} sx={{ 
                  bgcolor: 'primary.light', 
                  color: 'primary.contrastText',
                  px: 1, 
                  py: 0.5, 
                  borderRadius: 1,
                  fontSize: '0.75rem'
                }}>
                  {capability}
                </Box>
              ))}
            </Box>
          </Box>
        </CardContent>
      </Collapse>
      
      <CardActions>
        {active ? (
          <>
            <Button 
              size="small" 
              variant="outlined" 
              color="error" 
              startIcon={<Pause />}
              onClick={handleDeactivate}
            >
              Deactivate
            </Button>
            {!terminalVisible ? (
              <Button 
                size="small" 
                variant="outlined" 
                startIcon={<TerminalIcon />}
                onClick={handleShowTerminal}
              >
                Terminal
              </Button>
            ) : (
              <Button 
                size="small" 
                variant="outlined" 
                startIcon={<X />}
                onClick={handleHideTerminal}
              >
                Hide Terminal
              </Button>
            )}
          </>
        ) : (
          <Button 
            size="small" 
            variant="contained" 
            color="primary" 
            startIcon={<Play />}
            onClick={handleActivate}
          >
            Activate
          </Button>
        )}
        
        <Box sx={{ flexGrow: 1 }} />
        
        <IconButton size="small" onClick={() => onSettings && onSettings(agent.ID)}>
          <Settings size={18} />
        </IconButton>
        
        {!active && (
          <IconButton size="small" color="error" onClick={() => onRemove && onRemove(agent.ID)}>
            <X size={18} />
          </IconButton>
        )}
      </CardActions>
    </Card>
  );
};
```

### API Endpoints for Terminal Management

```go
// api/terminal_handlers.go
package api

import (
    "context"
    "encoding/json"
    "net/http"
    "time"
    
    "github.com/gorilla/mux"
    "github.com/gorilla/websocket"
)

// TerminalHandlers provides HTTP handlers for terminal operations
type TerminalHandlers struct {
    terminalManager runtime.TerminalManager
    upgrader        websocket.Upgrader
}

// NewTerminalHandlers creates new terminal handlers
func NewTerminalHandlers(terminalManager runtime.TerminalManager) *TerminalHandlers {
    return &TerminalHandlers{
        terminalManager: terminalManager,
        upgrader: websocket.Upgrader{
            ReadBufferSize:  1024,
            WriteBufferSize: 1024,
            CheckOrigin: func(r *http.Request) bool {
                return true // Allow all origins in development
            },
        },
    }
}

// RegisterRoutes registers terminal routes
func (h *TerminalHandlers) RegisterRoutes(router *mux.Router) {
    router.HandleFunc("/api/terminal/create", h.CreateTerminal).Methods("POST")
    router.HandleFunc("/api/terminal/{id}", h.GetTerminalInfo).Methods("GET")
    router.HandleFunc("/api/terminal/{id}/resize", h.ResizeTerminal).Methods("POST")
    router.HandleFunc("/api/terminal/{id}/write", h.WriteToTerminal).Methods("POST")
    router.HandleFunc("/api/terminal/{id}/read", h.ReadFromTerminal).Methods("GET")
    router.HandleFunc("/api/terminal/{id}/close", h.CloseTerminal).Methods("POST")
    router.HandleFunc("/api/terminal/{id}/ws", h.TerminalWebSocket)
    router.HandleFunc("/api/terminal/list/{sessionId}", h.ListTerminals).Methods("GET")
}

// CreateTerminal handles terminal creation requests
func (h *TerminalHandlers) CreateTerminal(w http.ResponseWriter, r *http.Request) {
    var request struct {
        AgentID   string            `json:"agent_id"`
        SessionID string            `json:"session_id"`
        Rows      int               `json:"rows"`
        Cols      int               `json:"cols"`
        Env       map[string]string `json:"env,omitempty"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    
    ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
    defer cancel()
    
    terminal, err := h.terminalManager.CreateTerminal(ctx, request.AgentID, request.SessionID, request.Rows, request.Cols)
    if err != nil {
        http.Error(w, "Failed to create terminal: "+err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(terminal)
}

// TerminalWebSocket handles WebSocket connections for terminal I/O
func (h *TerminalHandlers) TerminalWebSocket(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    terminalID := vars["id"]
    
    // Upgrade HTTP connection to WebSocket
    conn, err := h.upgrader.Upgrade(w, r, nil)
    if err != nil {
        http.Error(w, "Could not upgrade connection", http.StatusInternalServerError)
        return
    }
    defer conn.Close()
    
    // Subscribe to terminal output
    ctx, cancel := context.WithCancel(r.Context())
    defer cancel()
    
    outputChan, err := h.terminalManager.SubscribeToTerminalOutput(ctx, terminalID)
    if err != nil {
        conn.WriteMessage(websocket.TextMessage, []byte("Error: "+err.Error()))
        return
    }
    
    // Handle incoming messages (user input)
    go func() {
        for {
            _, message, err := conn.ReadMessage()
            if err != nil {
                break
            }
            
            if err := h.terminalManager.WriteToTerminal(ctx, terminalID, message); err != nil {
                conn.WriteMessage(websocket.TextMessage, []byte("Error: "+err.Error()))
            }
        }
        cancel() // Cancel context when connection is closed
    }()
    
    // Send terminal output to client
    for {
        select {
        case output, ok := <-outputChan:
            if !ok {
                return
            }
            if err := conn.WriteMessage(websocket.BinaryMessage, output); err != nil {
                return
            }
        case <-ctx.Done():
            return
        }
    }
}
```

## Implementation Plan

### Phase 1: Core Refactoring ✅

**Status:** COMPLETED (2025-07-02)
**Worklog:** `docs/worklogs/phase1_core_refactoring_worklog.md`

1. ✅ Create the unified agent data model (`UnifiedAgent`)
2. ✅ Implement the core service with storage operations
3. ✅ Migrate existing data to the new model
4. ✅ Update API endpoints to use the new core service

### Phase 2: Terminal System Implementation

1. Create the `TerminalManager` interface and implementation
2. Implement terminal backend providers (PTY, WASM, Docker)
3. Develop WebSocket-based terminal communication
4. Create frontend terminal component with XTerm.js

### Phase 3: Runtime Refactoring

1. Create the unified runtime interface
2. Implement plugin and WASM runtime adapters
3. Create the runtime manager
4. Update the inferencer to use the runtime manager
5. Integrate terminal management with agent runtimes

### Phase 4: Builder and Manager Refactoring

1. Refactor the agent builder to use the new interfaces
2. Implement the agent manager with advanced operations
3. Connect all components through the unified service

### Phase 5: API and UI Updates

1. Update API endpoints to use the unified service
2. Update UI components to work with the new model
3. Add new features enabled by the refactored architecture
4. Implement terminal UI improvements (themes, search, etc.)

## Error Handling and Integration with AI Error Inferencer

To ensure robust error handling and provide intelligent error analysis, the agent system will be integrated with the AI Error Inferencer system. This integration will allow for automatic detection, analysis, and resolution suggestions for errors that occur within the agent system.

### Error Inferencer Integration

```go
// agentify/error/error_inferencer.go
package error

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
)

// ErrorSource represents the source of an error in the agent system
type ErrorSource string

const (
    ErrorSourceAgentCore     ErrorSource = "agent_core"
    ErrorSourceAgentRuntime  ErrorSource = "agent_runtime"
    ErrorSourceAgentBuilder  ErrorSource = "agent_builder"
    ErrorSourceAgentManager  ErrorSource = "agent_manager"
    ErrorSourceAgentTerminal ErrorSource = "agent_terminal"
    ErrorSourceAgentCard     ErrorSource = "agent_card"
)

// ErrorContext contains contextual information about an error
type ErrorContext struct {
    Component    string                 `json:"component"`
    Action       string                 `json:"action,omitempty"`
    Parameters   map[string]interface{} `json:"parameters,omitempty"`
    StackTrace   string                 `json:"stack_trace,omitempty"`
    UserMessage  string                 `json:"user_message,omitempty"`
    SystemState  map[string]interface{} `json:"system_state,omitempty"`
}

// AgentError represents an error that occurred in the agent system
type AgentError struct {
    ID          string                 `json:"id"`
    Timestamp   time.Time              `json:"timestamp"`
    Source      ErrorSource            `json:"source"`
    AgentID     string                 `json:"agent_id,omitempty"`
    SessionID   string                 `json:"session_id,omitempty"`
    TerminalID  string                 `json:"terminal_id,omitempty"`
    Error       string                 `json:"error"`
    Context     ErrorContext           `json:"context,omitempty"`
    Analysis    map[string]interface{} `json:"analysis,omitempty"`
    Resolution  string                 `json:"resolution,omitempty"`
    IsResolved  bool                   `json:"is_resolved"`
    ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
    Severity    string                 `json:"severity"`
}

// ErrorInferencer provides AI-powered error analysis and resolution
type ErrorInferencer interface {
    // Analyze an error and provide insights
    AnalyzeError(ctx context.Context, err *AgentError) (*AgentError, error)
    
    // Get error history for an agent
    GetErrorHistory(ctx context.Context, agentID string) ([]*AgentError, error)
    
    // Mark an error as resolved
    ResolveError(ctx context.Context, errorID string, resolution string) error
    
    // Get similar errors to help with resolution
    GetSimilarErrors(ctx context.Context, errorID string) ([]*AgentError, error)
    
    // Get resolution suggestions for an error
    GetResolutionSuggestions(ctx context.Context, errorID string) ([]string, error)
}

// ErrorInferencerImpl implements the ErrorInferencer interface
type ErrorInferencerImpl struct {
    errorStorage ErrorStorage
    aiService    AIService
}

// ErrorStorage provides storage for agent errors
type ErrorStorage interface {
    StoreError(ctx context.Context, err *AgentError) error
    GetError(ctx context.Context, errorID string) (*AgentError, error)
    UpdateError(ctx context.Context, err *AgentError) error
    ListErrors(ctx context.Context, filter map[string]interface{}) ([]*AgentError, error)
    DeleteError(ctx context.Context, errorID string) error
}

// AIService provides AI capabilities for error analysis
type AIService interface {
    AnalyzeErrorWithAI(ctx context.Context, err *AgentError) (map[string]interface{}, error)
    GenerateResolutionSuggestions(ctx context.Context, err *AgentError) ([]string, error)
}

// NewErrorInferencer creates a new error inferencer
func NewErrorInferencer(errorStorage ErrorStorage, aiService AIService) *ErrorInferencerImpl {
    return &ErrorInferencerImpl{
        errorStorage: errorStorage,
        aiService:    aiService,
    }
}

// AnalyzeError analyzes an error and provides insights
func (e *ErrorInferencerImpl) AnalyzeError(ctx context.Context, err *AgentError) (*AgentError, error) {
    // Set default values
    if err.ID == "" {
        err.ID = generateErrorID()
    }
    if err.Timestamp.IsZero() {
        err.Timestamp = time.Now()
    }
    if err.Severity == "" {
        err.Severity = determineSeverity(err)
    }
    
    // Analyze the error with AI
    analysis, aiErr := e.aiService.AnalyzeErrorWithAI(ctx, err)
    if aiErr == nil {
        err.Analysis = analysis
    }
    
    // Store the error
    if storeErr := e.errorStorage.StoreError(ctx, err); storeErr != nil {
        return err, fmt.Errorf("failed to store error: %v", storeErr)
    }
    
    return err, nil
}
```

### Agent System Error Handling

Each component in the agent system will be updated to include comprehensive error handling and integration with the Error Inferencer:

```go
// agent/core/agent_core_service.go
package core

import (
    "context"
    "fmt"
    
    "github.com/guiperry/agentic-engine/agentify/error"
)

// AgentCoreServiceImpl implements the AgentCoreService interface
type AgentCoreServiceImpl struct {
    storage        AgentStorage
    errorInferencer error.ErrorInferencer
}

// NewAgentCoreService creates a new agent core service
func NewAgentCoreService(storage AgentStorage, errorInferencer error.ErrorInferencer) *AgentCoreServiceImpl {
    return &AgentCoreServiceImpl{
        storage:        storage,
        errorInferencer: errorInferencer,
    }
}

// CreateAgent creates a new agent
func (s *AgentCoreServiceImpl) CreateAgent(ctx context.Context, agent *UnifiedAgent) error {
    if err := s.storage.CreateAgent(ctx, agent); err != nil {
        // Log the error
        agentErr := &error.AgentError{
            Source:  error.ErrorSourceAgentCore,
            AgentID: agent.ID,
            Error:   err.Error(),
            Context: error.ErrorContext{
                Component: "AgentCoreService",
                Action:    "CreateAgent",
                Parameters: map[string]interface{}{
                    "agent_name": agent.Name,
                    "agent_type": agent.Type,
                },
            },
        }
        
        // Analyze the error with AI
        s.errorInferencer.AnalyzeError(ctx, agentErr)
        
        return fmt.Errorf("failed to create agent: %w", err)
    }
    
    // Trigger lifecycle hook
    s.OnAgentCreated(agent)
    
    return nil
}

// Additional methods with similar error handling...
```

## Benefits of Refactoring

1. **Simplified Architecture**
   - Clear separation of concerns
   - Reduced code duplication
   - More maintainable codebase

2. **Improved Developer Experience**
   - Consistent interfaces
   - Better dependency injection
   - Clearer documentation

3. **Enhanced Functionality**
   - Unified agent discovery
   - Consistent agent lifecycle management
   - Better versioning and deployment
   - Per-agent terminal management
   - Intelligent error handling and analysis

4. **Future Extensibility**
   - Easier to add new agent types
   - Simpler integration with new storage backends
   - More flexible deployment options
   - Expandable terminal capabilities

5. **Better User Experience**
   - Integrated terminals within agent cards
   - Expandable terminal views
   - Customizable terminal settings
   - Intelligent error reporting and resolution

## Avoiding New Redundancies

Throughout this refactoring, we must be vigilant to avoid introducing new redundancies. The following principles will guide our implementation:

1. **Single Source of Truth**
   - Each piece of functionality should exist in exactly one place
   - No duplicate implementations of the same logic across different components
   - Shared functionality should be extracted into common utilities

2. **Clear Component Boundaries**
   - Each component should have a well-defined responsibility
   - Components should interact through explicit interfaces
   - Avoid circular dependencies between components

3. **Consistent Data Models**
   - Use the `UnifiedAgent` model consistently throughout the system
   - Avoid creating derivative or parallel data structures
   - Convert legacy data structures at system boundaries

4. **Centralized Discovery**
   - Consolidate all agent discovery logic in one place
   - Implement a single registry for all agent types
   - Use adapters for different agent implementations (plugin, WASM)

5. **Unified Error Handling**
   - Use a single error handling and reporting mechanism
   - Avoid duplicate error tracking systems
   - Ensure all components report errors in a consistent format

6. **Careful Abstraction Layers**
   - Only add abstraction layers when they reduce complexity
   - Avoid "wrapper" classes that add no value
   - Ensure each layer has a clear, distinct purpose

7. **Code Review Focus**
   - During code reviews, specifically look for redundancies
   - Question any code that appears to duplicate existing functionality
   - Refactor immediately when redundancies are discovered

## Conclusion

The current agent system has evolved organically, leading to redundancies and complexity. The proposed refactoring consolidates the system into clear, focused components with well-defined interfaces, systematically eliminating the identified redundancies. This will improve maintainability, reduce bugs, and enable new features while preserving compatibility with existing code through careful interface design.

The addition of per-agent terminal management within agent cards provides a more intuitive and efficient user experience, allowing users to interact with each agent individually. The integration with the AI Error Inferencer system ensures that errors are not only caught but also analyzed intelligently, with potential resolutions suggested automatically.

By implementing this refactoring plan with a strict focus on eliminating redundancies, the Agentic Engine will have a more robust, scalable, and extensible agent system that better supports its mission of enabling autonomous AI agents through a no-code interface, with improved user interaction capabilities and error handling, all while maintaining a clean, non-redundant architecture.