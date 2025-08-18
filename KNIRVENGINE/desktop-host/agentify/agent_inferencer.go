// agent_inferencer.go
package agentify

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// GenerationOptions defines options for text generation
type GenerationOptions struct {
	Temperature   float64  `json:"temperature,omitempty"`
	MaxTokens     int      `json:"max_tokens,omitempty"`
	TopP          float64  `json:"top_p,omitempty"`
	TopK          int      `json:"top_k,omitempty"`
	StopSequences []string `json:"stop_sequences,omitempty"`
	Stream        bool     `json:"stream,omitempty"`
}

// LLMProvider defines a robust interface for LLM providers
type LLMProvider interface {
	GenerateText(ctx context.Context, prompt string, options GenerationOptions) (string, error)
	GenerateTextWithCoT(ctx context.Context, prompt string) (string, error)
	GenerateStructuredOutput(ctx context.Context, prompt string, schema string) (interface{}, error)
	IsAvailable() bool
	GetProviderName() string
	GetSupportedModels() []string
}

// InferenceServiceInterface defines the interface for LLM inference services (legacy compatibility)
type InferenceServiceInterface interface {
	GenerateText(promptText string, instructionText string) (string, error)
	GenerateTextWithCoT(promptText string) (string, error)
	GenerateStructuredOutput(content string, schema string) (string, error)
	IsRunning() bool
}

// ConversationMemory defines the interface for conversation memory management
type ConversationMemory interface {
	AddUserMessage(ctx context.Context, sessionID string, message string) error
	AddAssistantMessage(ctx context.Context, sessionID string, message string) error
	AddSystemMessage(ctx context.Context, sessionID string, message string) error
	GetConversationHistory(ctx context.Context, sessionID string, limit int) ([]*ConversationMessage, error)
	GetRecentMessages(ctx context.Context, sessionID string, maxTokens int) ([]*ConversationMessage, error)
	ClearConversation(ctx context.Context, sessionID string) error
	GetConversationSummary(ctx context.Context, sessionID string) (string, error)
}

// UnifiedAgentStorageInterface defines the interface for unified agent storage
type UnifiedAgentStorageInterface interface {
	RegisterAgentConfig(ctx context.Context, agentID string, config map[string]interface{}) error
	GetAgentConfig(ctx context.Context, agentID string) (map[string]interface{}, error)
	ListAgents(ctx context.Context) ([]interface{}, error)
}

// AgentInfo represents basic agent information for listing
type AgentInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	BuildTarget string `json:"build_target"`
}

// AgentInferencer manages the inference process through agent plugins and WASM agents
type AgentInferencer struct {
	pluginLoader       *AgentPluginLoader
	wasmLoader         *AgentWASMLoader
	wasmInstaller      *WASMPluginInstaller
	activeAgents       map[string]AgentPluginInterface
	activeWASMAgents   map[string]WASMAgentInterface
	sessions           map[string]string // Maps session IDs to agent IDs
	sessionTypes       map[string]string // Maps session IDs to agent types ("plugin" or "wasm")
	inferenceService   InferenceServiceInterface
	llmProviders       map[string]LLMProvider
	conversationMemory ConversationMemory
	defaultProvider    string
	retryAttempts      int
	requestTimeout     time.Duration
	unifiedStorage     UnifiedAgentStorageInterface // Unified storage for automatic agent registration
	mutex              sync.RWMutex
}

// NewAgentInferencer creates a new agent inferencer (deprecated - use NewAgentInferencerWithStorage)
func NewAgentInferencer(pluginsDir string) *AgentInferencer {
	return NewAgentInferencerWithStorage(pluginsDir, nil)
}

// NewAgentInferencerWithStorage creates a new agent inferencer with unified storage
func NewAgentInferencerWithStorage(pluginsDir string, unifiedStorage UnifiedAgentStorageInterface) *AgentInferencer {
	// Determine downloads directory (parent of plugins directory)
	appDataDir := filepath.Dir(pluginsDir)
	downloadsDir := filepath.Join(appDataDir, "downloads")

	return &AgentInferencer{
		pluginLoader:       NewAgentPluginLoader(pluginsDir),
		wasmLoader:         NewAgentWASMLoader(pluginsDir), // Use same directory for WASM files
		wasmInstaller:      NewWASMPluginInstaller(downloadsDir, pluginsDir, pluginsDir),
		activeAgents:       make(map[string]AgentPluginInterface),
		activeWASMAgents:   make(map[string]WASMAgentInterface),
		sessions:           make(map[string]string),
		sessionTypes:       make(map[string]string),
		llmProviders:       make(map[string]LLMProvider),
		conversationMemory: NewPersistentConversationMemory(),
		retryAttempts:      3,
		requestTimeout:     30 * time.Second,
		unifiedStorage:     unifiedStorage,
	}
}

// SetInferenceService sets the inference service for LLM interactions
func (i *AgentInferencer) SetInferenceService(service InferenceServiceInterface) {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	i.inferenceService = service
}

// SetActiveAgent manually sets an active agent for a session (for testing/examples)
func (i *AgentInferencer) SetActiveAgent(sessionID, agentID string, agent AgentPluginInterface) {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	i.activeAgents[agentID] = agent
	i.sessions[sessionID] = agentID
}

// ActivateAgent activates an agent for a session (supports both WASM and plugin agents)
func (i *AgentInferencer) ActivateAgent(ctx context.Context, agentID, version, sessionID string, config map[string]interface{}) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	// Check if an agent is already active for this session
	if currentAgentID, ok := i.sessions[sessionID]; ok {
		if currentAgentID == agentID {
			// Agent is already active
			return nil
		}

		// Different agent is active, deactivate it
		i.deactivateCurrentAgent(currentAgentID, sessionID)
	}

	// Try to load WASM agent first
	wasmAgent, wasmErr := i.wasmLoader.LoadWASMAgent(agentID, version, config)
	if wasmErr == nil {
		// WASM agent loaded successfully
		if err := wasmAgent.Start(); err != nil {
			return fmt.Errorf("failed to start WASM agent: %v", err)
		}

		// Store the active WASM agent and session mapping
		i.activeWASMAgents[agentID] = wasmAgent
		i.sessions[sessionID] = agentID
		i.sessionTypes[sessionID] = "wasm"

		// Note: WASM agents don't have terminal creation method yet
		// This will be implemented when WASM agent terminal support is added

		return nil
	}

	// WASM agent failed, try plugin agent
	pluginAgent, pluginErr := i.pluginLoader.LoadPlugin(agentID, version, config)
	if pluginErr != nil {
		return fmt.Errorf("failed to load agent (WASM error: %v, Plugin error: %v)", wasmErr, pluginErr)
	}

	// Start the plugin agent
	if err := pluginAgent.Start(); err != nil {
		return fmt.Errorf("failed to start plugin agent: %v", err)
	}

	// Store the active plugin agent and session mapping
	i.activeAgents[agentID] = pluginAgent
	i.sessions[sessionID] = agentID
	i.sessionTypes[sessionID] = "plugin"

	// Automatically create a terminal session for the agent
	// This ensures the agent has a terminal to log to immediately
	if _, err := pluginAgent.CreateTerminal(24, 80); err != nil {
		// Log the error but don't fail activation
		fmt.Printf("Warning: Failed to create terminal for agent %s: %v\n", agentID, err)
	}

	return nil
}

// deactivateCurrentAgent is a helper method to deactivate the current agent for a session
func (i *AgentInferencer) deactivateCurrentAgent(agentID, sessionID string) {
	// Check session type and deactivate accordingly
	sessionType := i.sessionTypes[sessionID]

	switch sessionType {
	case "wasm":
		if wasmAgent, ok := i.activeWASMAgents[agentID]; ok {
			wasmAgent.Stop()
			delete(i.activeWASMAgents, agentID)
		}
	case "plugin":
		if pluginAgent, ok := i.activeAgents[agentID]; ok {
			pluginAgent.Stop()
			delete(i.activeAgents, agentID)
		}
	default:
		// Legacy support - try both
		if pluginAgent, ok := i.activeAgents[agentID]; ok {
			pluginAgent.Stop()
			delete(i.activeAgents, agentID)
		}
		if wasmAgent, ok := i.activeWASMAgents[agentID]; ok {
			wasmAgent.Stop()
			delete(i.activeWASMAgents, agentID)
		}
	}

	delete(i.sessions, sessionID)
	delete(i.sessionTypes, sessionID)
}

// DeactivateAgent deactivates an agent for a session
func (i *AgentInferencer) DeactivateAgent(ctx context.Context, sessionID string) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	// Check if an agent is active for this session
	agentID, ok := i.sessions[sessionID]
	if !ok {
		// No agent active for this session
		return nil
	}

	// Use the helper method to deactivate
	i.deactivateCurrentAgent(agentID, sessionID)

	return nil
}

// ProcessInference processes an inference request through the appropriate agent (WASM or plugin)
func (i *AgentInferencer) ProcessInference(ctx context.Context, sessionID string, request *InferenceRequest) (*InferenceResponse, error) {
	i.mutex.RLock()
	agentID, ok := i.sessions[sessionID]
	if !ok {
		i.mutex.RUnlock()
		return nil, fmt.Errorf("no agent active for session %s", sessionID)
	}

	sessionType := i.sessionTypes[sessionID]
	inferenceService := i.inferenceService
	i.mutex.RUnlock()

	// Set the session ID in the request
	request.SessionID = sessionID

	// Handle different agent types
	switch sessionType {
	case "wasm":
		wasmAgent, ok := i.activeWASMAgents[agentID]
		if !ok {
			return nil, fmt.Errorf("WASM agent %s not found", agentID)
		}

		// Process through WASM agent
		return wasmAgent.ProcessInference(ctx, request)

	case "plugin":
		pluginAgent, ok := i.activeAgents[agentID]
		if !ok {
			return nil, fmt.Errorf("plugin agent %s not found", agentID)
		}

		// If we have an inference service, enhance the agent's processing with LLM capabilities
		if inferenceService != nil && inferenceService.IsRunning() {
			return i.processInferenceWithLLM(ctx, pluginAgent, request, inferenceService)
		}

		// Process the inference request through the plugin agent (without LLM)
		return pluginAgent.ProcessInference(ctx, request)

	default:
		// Legacy support - try plugin first, then WASM
		if pluginAgent, ok := i.activeAgents[agentID]; ok {
			if inferenceService != nil && inferenceService.IsRunning() {
				return i.processInferenceWithLLM(ctx, pluginAgent, request, inferenceService)
			}
			return pluginAgent.ProcessInference(ctx, request)
		}

		if wasmAgent, ok := i.activeWASMAgents[agentID]; ok {
			return wasmAgent.ProcessInference(ctx, request)
		}

		return nil, fmt.Errorf("no agent found for session %s", sessionID)
	}
}

// GetAgentCapabilities gets the capabilities of an agent (WASM or plugin)
func (i *AgentInferencer) GetAgentCapabilities(ctx context.Context, sessionID string) (*AgentCapabilities, error) {
	i.mutex.RLock()
	agentID, ok := i.sessions[sessionID]
	if !ok {
		i.mutex.RUnlock()
		return nil, fmt.Errorf("no agent active for session %s", sessionID)
	}

	sessionType := i.sessionTypes[sessionID]
	i.mutex.RUnlock()

	// Handle different agent types
	switch sessionType {
	case "wasm":
		wasmAgent, ok := i.activeWASMAgents[agentID]
		if !ok {
			return nil, fmt.Errorf("WASM agent %s not found", agentID)
		}
		return wasmAgent.GetCapabilities(), nil

	case "plugin":
		pluginAgent, ok := i.activeAgents[agentID]
		if !ok {
			return nil, fmt.Errorf("plugin agent %s not found", agentID)
		}
		return pluginAgent.GetCapabilities(), nil

	default:
		// Legacy support - try plugin first, then WASM
		if pluginAgent, ok := i.activeAgents[agentID]; ok {
			return pluginAgent.GetCapabilities(), nil
		}

		if wasmAgent, ok := i.activeWASMAgents[agentID]; ok {
			return wasmAgent.GetCapabilities(), nil
		}

		return nil, fmt.Errorf("no agent found for session %s", sessionID)
	}
}

// GetAgentSchema gets the schema of an agent
func (i *AgentInferencer) GetAgentSchema(ctx context.Context, sessionID string) (*AgentSchema, error) {
	i.mutex.RLock()
	agentID, ok := i.sessions[sessionID]
	if !ok {
		i.mutex.RUnlock()
		return nil, fmt.Errorf("no agent active for session %s", sessionID)
	}

	agent, ok := i.activeAgents[agentID]
	i.mutex.RUnlock()

	if !ok {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	// Get the agent schema
	return agent.GetSchema(), nil
}

// ListAvailableAgents lists the available agents (both plugins and WASM) and auto-registers them
func (i *AgentInferencer) ListAvailableAgents(ctx context.Context) ([]string, error) {
	var allAgents []string

	// Discover available plugins
	plugins, err := i.pluginLoader.DiscoverPlugins()
	if err != nil {
		// Log error but continue to try WASM agents
		fmt.Printf("Warning: Failed to discover plugin agents: %v\n", err)
	} else {
		allAgents = append(allAgents, plugins...)
		// Auto-register plugin agents
		i.autoRegisterAgents(ctx, plugins, "plugin")
	}

	// Discover available WASM agents
	wasmAgents, err := i.wasmLoader.DiscoverWASMAgents()
	if err != nil {
		// Log error but continue with what we have
		fmt.Printf("Warning: Failed to discover WASM agents: %v\n", err)
	} else {
		allAgents = append(allAgents, wasmAgents...)
		// Auto-register WASM agents
		i.autoRegisterAgents(ctx, wasmAgents, "wasm")
	}

	return allAgents, nil
}

// autoRegisterAgents automatically registers discovered agents to the unified storage
func (i *AgentInferencer) autoRegisterAgents(ctx context.Context, agentIDs []string, buildTarget string) {
	if i.unifiedStorage == nil {
		return // No unified storage configured
	}

	for _, agentID := range agentIDs {
		// Parse agent ID to extract name and version
		parts := strings.Split(agentID, "_")
		if len(parts) < 2 {
			continue // Skip invalid agent IDs
		}

		version := parts[len(parts)-1]
		name := strings.Join(parts[:len(parts)-1], "_")

		// Create agent configuration
		config := map[string]interface{}{
			"agent_id":     agentID,
			"name":         name,
			"version":      version,
			"agent_type":   "discovered",
			"build_target": buildTarget,
			"capabilities": []string{"General Purpose"},
			"target_types": []string{"application"},
			"discovered":   true,
			"file_type": func() string {
				if buildTarget == "wasm" {
					return ".wasm"
				}
				return ".so"
			}(),
		}

		// Try to register the agent (this will update if it already exists)
		if err := i.unifiedStorage.RegisterAgentConfig(ctx, agentID, config); err != nil {
			fmt.Printf("Warning: Failed to auto-register discovered agent %s: %v\n", agentID, err)
		} else {
			fmt.Printf("DEBUG: Auto-registered agent %s as %s\n", agentID, buildTarget)
		}
	}
}

// WASM Plugin Management Methods

// DiscoverWASMPluginZips discovers available WASM plugin zip files in the downloads directory
func (i *AgentInferencer) DiscoverWASMPluginZips(ctx context.Context) ([]*WASMPluginInfo, error) {
	return i.wasmInstaller.DiscoverZipPlugins()
}

// InstallWASMPlugin installs a WASM plugin from a zip file
func (i *AgentInferencer) InstallWASMPlugin(ctx context.Context, zipPath string) (*WASMPluginInfo, error) {
	return i.wasmInstaller.InstallPlugin(zipPath)
}

// UninstallWASMPlugin uninstalls a WASM plugin
func (i *AgentInferencer) UninstallWASMPlugin(ctx context.Context, agentID, version string) error {
	// First, deactivate the agent if it's active
	i.mutex.Lock()
	for sessionID, activeAgentID := range i.sessions {
		if activeAgentID == agentID && i.sessionTypes[sessionID] == "wasm" {
			i.deactivateCurrentAgent(activeAgentID, sessionID)
		}
	}
	i.mutex.Unlock()

	return i.wasmInstaller.UninstallPlugin(agentID, version)
}

// ListInstalledWASMPlugins lists all installed WASM plugins
func (i *AgentInferencer) ListInstalledWASMPlugins(ctx context.Context) ([]*WASMPluginInfo, error) {
	return i.wasmInstaller.ListInstalledPlugins()
}

// GetAvailableAgentsDetailed returns detailed information about all available agents
func (i *AgentInferencer) GetAvailableAgentsDetailed(ctx context.Context) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"plugins":           []string{},
		"wasm_agents":       []string{},
		"zip_plugins":       []*WASMPluginInfo{},
		"installed_plugins": []*WASMPluginInfo{},
	}

	// Get plugin agents
	plugins, err := i.pluginLoader.DiscoverPlugins()
	if err != nil {
		fmt.Printf("Warning: Failed to discover plugin agents: %v\n", err)
	} else {
		result["plugins"] = plugins
	}

	// Get WASM agents
	wasmAgents, err := i.wasmLoader.DiscoverWASMAgents()
	if err != nil {
		fmt.Printf("Warning: Failed to discover WASM agents: %v\n", err)
	} else {
		result["wasm_agents"] = wasmAgents
	}

	// Get available zip plugins
	zipPlugins, err := i.wasmInstaller.DiscoverZipPlugins()
	if err != nil {
		fmt.Printf("Warning: Failed to discover zip plugins: %v\n", err)
	} else {
		result["zip_plugins"] = zipPlugins
	}

	// Get installed WASM plugins
	installedPlugins, err := i.wasmInstaller.ListInstalledPlugins()
	if err != nil {
		fmt.Printf("Warning: Failed to list installed plugins: %v\n", err)
	} else {
		result["installed_plugins"] = installedPlugins
	}

	return result, nil
}

// SyncDiscoveredAgentsToRegistry syncs discovered plugin/WASM agents to the agent registry
func (i *AgentInferencer) SyncDiscoveredAgentsToRegistry(ctx context.Context, registry AgentRegistryInterface) error {
	// Get all discovered agents
	allAgents, err := i.ListAvailableAgents(ctx)
	if err != nil {
		return fmt.Errorf("failed to list available agents: %v", err)
	}

	fmt.Printf("DEBUG: Discovered %d agents: %v\n", len(allAgents), allAgents)

	// Sync each discovered agent to the registry
	for _, agentID := range allAgents {
		// Parse agent ID to extract name and version
		parts := strings.Split(agentID, "_")
		if len(parts) < 2 {
			fmt.Printf("DEBUG: Skipping invalid agent ID format: %s\n", agentID)
			continue // Skip invalid agent IDs
		}

		version := parts[len(parts)-1]
		name := strings.Join(parts[:len(parts)-1], "_")

		// Determine if it's a WASM agent by checking if the file exists
		isWasm := i.isWASMAgent(agentID, version)

		fmt.Printf("DEBUG: Processing agent %s - name: %s, version: %s, isWasm: %t\n", agentID, name, version, isWasm)

		// Create agent configuration
		config := map[string]interface{}{
			"agent_id":   agentID,
			"name":       name,
			"version":    version,
			"agent_type": "discovered",
			"build_target": func() string {
				if isWasm {
					return "wasm"
				}
				return "plugin"
			}(),
			"capabilities": []string{"General Purpose"},
			"target_types": []string{"application"},
			"discovered":   true,
			"file_type": func() string {
				if isWasm {
					return ".wasm"
				}
				return ".so"
			}(),
		}

		// Try to register the agent (this will update if it already exists)
		if err := registry.RegisterAgent(agentID, config); err != nil {
			fmt.Printf("Warning: Failed to register discovered agent %s: %v\n", agentID, err)
		} else {
			fmt.Printf("DEBUG: Successfully registered agent %s as %s\n", agentID, config["build_target"])
		}
	}

	return nil
}

// isWASMAgent checks if an agent is a WASM agent by checking if the WASM file exists
func (i *AgentInferencer) isWASMAgent(agentID, version string) bool {
	// Try multiple path patterns to find the WASM file

	// Pattern 1: agent_{agentID}_{version}.wasm
	if version != "" {
		wasmPath := filepath.Join(i.wasmLoader.wasmDir, fmt.Sprintf("agent_%s_%s.wasm", agentID, version))
		fmt.Printf("DEBUG: Checking versioned WASM path: %s\n", wasmPath)
		if _, err := os.Stat(wasmPath); err == nil {
			fmt.Printf("DEBUG: Found versioned WASM file: %s\n", wasmPath)
			return true
		}
	}

	// Pattern 2: agent_{agentID}.wasm (legacy format)
	wasmPath := filepath.Join(i.wasmLoader.wasmDir, fmt.Sprintf("agent_%s.wasm", agentID))
	fmt.Printf("DEBUG: Checking WASM path: %s\n", wasmPath)
	if _, err := os.Stat(wasmPath); err == nil {
		fmt.Printf("DEBUG: Found WASM file: %s\n", wasmPath)
		return true
	}

	// Pattern 3: Check for extracted plugin directory format
	pluginDir := filepath.Join(i.wasmLoader.wasmDir, agentID)
	fmt.Printf("DEBUG: Checking plugin dir: %s\n", pluginDir)
	if _, err := os.Stat(pluginDir); err == nil {
		// Check if directory contains a WASM file
		entries, err := os.ReadDir(pluginDir)
		if err == nil {
			for _, entry := range entries {
				if strings.HasSuffix(entry.Name(), ".wasm") {
					// Verify version if specified
					if version != "" && !strings.Contains(entry.Name(), version) {
						continue
					}
					fmt.Printf("DEBUG: Found WASM file in directory: %s/%s\n", pluginDir, entry.Name())
					return true
				}
			}
		}
	}

	fmt.Printf("DEBUG: No WASM file found for agent: %s (version: %s)\n", agentID, version)
	return false
}

// AgentRegistryInterface defines the interface for agent registry operations
type AgentRegistryInterface interface {
	RegisterAgent(agentID string, config map[string]interface{}) error
	GetAgent(agentID string) (map[string]interface{}, error)
}

// DiscoverAllPlugins discovers all plugin files in the plugins directory
func (i *AgentInferencer) DiscoverAllPlugins(ctx context.Context) ([]*PluginInfo, error) {
	return i.pluginLoader.DiscoverAllPlugins()
}

// ImportPlugin imports a plugin into the system
func (i *AgentInferencer) ImportPlugin(ctx context.Context, request *ImportPluginRequest) error {
	return i.pluginLoader.ImportPlugin(request)
}

// GetAgentMemory gets a value from the agent's memory
func (i *AgentInferencer) GetAgentMemory(ctx context.Context, sessionID string, key string) (interface{}, error) {
	i.mutex.RLock()
	agentID, ok := i.sessions[sessionID]
	if !ok {
		i.mutex.RUnlock()
		return nil, fmt.Errorf("no agent active for session %s", sessionID)
	}

	agent, ok := i.activeAgents[agentID]
	i.mutex.RUnlock()

	if !ok {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	// Get the memory value
	return agent.GetMemory(key)
}

// SetAgentMemory sets a value in the agent's memory
func (i *AgentInferencer) SetAgentMemory(ctx context.Context, sessionID string, key string, value interface{}) error {
	i.mutex.RLock()
	agentID, ok := i.sessions[sessionID]
	if !ok {
		i.mutex.RUnlock()
		return fmt.Errorf("no agent active for session %s", sessionID)
	}

	agent, ok := i.activeAgents[agentID]
	i.mutex.RUnlock()

	if !ok {
		return fmt.Errorf("agent %s not found", agentID)
	}

	// Set the memory value
	return agent.SetMemory(key, value)
}

// GetTEEInfo gets information about the TEE for an agent
func (i *AgentInferencer) GetTEEInfo(ctx context.Context, sessionID string) (map[string]interface{}, error) {
	i.mutex.RLock()
	agentID, ok := i.sessions[sessionID]
	if !ok {
		i.mutex.RUnlock()
		return nil, fmt.Errorf("no agent active for session %s", sessionID)
	}

	agent, ok := i.activeAgents[agentID]
	i.mutex.RUnlock()

	if !ok {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	// Get the TEE info
	return agent.GetTEEInfo(), nil
}

// Terminal management methods

// CreateTerminal creates a new terminal session for an agent
func (i *AgentInferencer) CreateTerminal(ctx context.Context, sessionID string, rows, cols int) (string, error) {
	i.mutex.RLock()
	agentID, ok := i.sessions[sessionID]
	if !ok {
		i.mutex.RUnlock()
		return "", fmt.Errorf("no agent active for session %s", sessionID)
	}

	sessionType := i.sessionTypes[sessionID]
	i.mutex.RUnlock()

	// Handle different agent types
	switch sessionType {
	case "wasm":
		wasmAgent, ok := i.activeWASMAgents[agentID]
		if !ok {
			return "", fmt.Errorf("WASM agent %s not found", agentID)
		}
		return wasmAgent.CreateTerminal(rows, cols)
	default:
		// Handle plugin agents
		agent, ok := i.activeAgents[agentID]
		if !ok {
			return "", fmt.Errorf("agent %s not found", agentID)
		}
		return agent.CreateTerminal(rows, cols)
	}
}

// ResizeTerminal resizes a terminal session
func (i *AgentInferencer) ResizeTerminal(ctx context.Context, sessionID string, terminalID string, rows, cols int) error {
	i.mutex.RLock()
	agentID, ok := i.sessions[sessionID]
	if !ok {
		i.mutex.RUnlock()
		return fmt.Errorf("no agent active for session %s", sessionID)
	}

	sessionType := i.sessionTypes[sessionID]
	i.mutex.RUnlock()

	// Handle different agent types
	switch sessionType {
	case "wasm":
		wasmAgent, ok := i.activeWASMAgents[agentID]
		if !ok {
			return fmt.Errorf("WASM agent %s not found", agentID)
		}
		return wasmAgent.ResizeTerminal(terminalID, rows, cols)
	default:
		// Handle plugin agents
		agent, ok := i.activeAgents[agentID]
		if !ok {
			return fmt.Errorf("agent %s not found", agentID)
		}
		return agent.ResizeTerminal(terminalID, rows, cols)
	}
}

// WriteToTerminal writes data to a terminal session
func (i *AgentInferencer) WriteToTerminal(ctx context.Context, sessionID string, terminalID string, data []byte) error {
	i.mutex.RLock()
	agentID, ok := i.sessions[sessionID]
	if !ok {
		i.mutex.RUnlock()
		return fmt.Errorf("no agent active for session %s", sessionID)
	}

	sessionType := i.sessionTypes[sessionID]
	i.mutex.RUnlock()

	// Handle different agent types
	switch sessionType {
	case "wasm":
		wasmAgent, ok := i.activeWASMAgents[agentID]
		if !ok {
			return fmt.Errorf("WASM agent %s not found", agentID)
		}
		return wasmAgent.WriteToTerminal(terminalID, data)
	default:
		// Handle plugin agents
		agent, ok := i.activeAgents[agentID]
		if !ok {
			return fmt.Errorf("agent %s not found", agentID)
		}
		return agent.WriteToTerminal(terminalID, data)
	}
}

// ReadFromTerminal reads data from a terminal session
func (i *AgentInferencer) ReadFromTerminal(ctx context.Context, sessionID string, terminalID string) ([]byte, error) {
	i.mutex.RLock()
	agentID, ok := i.sessions[sessionID]
	if !ok {
		i.mutex.RUnlock()
		return nil, fmt.Errorf("no agent active for session %s", sessionID)
	}

	sessionType := i.sessionTypes[sessionID]
	i.mutex.RUnlock()

	// Handle different agent types
	switch sessionType {
	case "wasm":
		wasmAgent, ok := i.activeWASMAgents[agentID]
		if !ok {
			return nil, fmt.Errorf("WASM agent %s not found", agentID)
		}
		return wasmAgent.ReadFromTerminal(terminalID)
	default:
		// Handle plugin agents
		agent, ok := i.activeAgents[agentID]
		if !ok {
			return nil, fmt.Errorf("agent %s not found", agentID)
		}
		return agent.ReadFromTerminal(terminalID)
	}
}

// CloseTerminal closes a terminal session
func (i *AgentInferencer) CloseTerminal(ctx context.Context, sessionID string, terminalID string) error {
	i.mutex.RLock()
	agentID, ok := i.sessions[sessionID]
	if !ok {
		i.mutex.RUnlock()
		return fmt.Errorf("no agent active for session %s", sessionID)
	}

	sessionType := i.sessionTypes[sessionID]
	i.mutex.RUnlock()

	// Handle different agent types
	switch sessionType {
	case "wasm":
		wasmAgent, ok := i.activeWASMAgents[agentID]
		if !ok {
			return fmt.Errorf("WASM agent %s not found", agentID)
		}
		return wasmAgent.CloseTerminal(terminalID)
	default:
		// Handle plugin agents
		agent, ok := i.activeAgents[agentID]
		if !ok {
			return fmt.Errorf("agent %s not found", agentID)
		}
		return agent.CloseTerminal(terminalID)
	}
}

// GetTerminalLogs gets the comprehensive log buffer for a terminal session
func (i *AgentInferencer) GetTerminalLogs(ctx context.Context, sessionID string, terminalID string) ([]LogMessage, error) {
	i.mutex.RLock()
	agentID, ok := i.sessions[sessionID]
	if !ok {
		i.mutex.RUnlock()
		return nil, fmt.Errorf("no agent active for session %s", sessionID)
	}

	sessionType := i.sessionTypes[sessionID]
	i.mutex.RUnlock()

	// Handle different agent types
	switch sessionType {
	case "wasm":
		wasmAgent, ok := i.activeWASMAgents[agentID]
		if !ok {
			return nil, fmt.Errorf("WASM agent %s not found", agentID)
		}
		// For WASM agents, we need to get logs from the terminal manager
		if wasmAgentImpl, ok := wasmAgent.(*WASMAgent); ok && wasmAgentImpl.terminalManager != nil {
			return wasmAgentImpl.terminalManager.GetTerminalLogs(terminalID)
		}
		return nil, fmt.Errorf("WASM agent does not support terminal logs")
	default:
		// Handle plugin agents
		agent, ok := i.activeAgents[agentID]
		if !ok {
			return nil, fmt.Errorf("agent %s not found", agentID)
		}
		// For plugin agents, we need to get logs from the terminal manager
		if baseAgent, ok := agent.(*BaseAgentPlugin); ok && baseAgent.terminalManager != nil {
			return baseAgent.terminalManager.GetTerminalLogs(terminalID)
		}
		return nil, fmt.Errorf("agent does not support terminal logs")
	}
}

// LogAgentProcessOutput logs output from an agent process to all its terminal sessions
func (i *AgentInferencer) LogAgentProcessOutput(ctx context.Context, agentID string, outputType string, data []byte, processID int) error {
	i.mutex.RLock()
	defer i.mutex.RUnlock()

	// Find the agent and log to its terminal sessions
	if agent, ok := i.activeAgents[agentID]; ok {
		if baseAgent, ok := agent.(*BaseAgentPlugin); ok && baseAgent.terminalManager != nil {
			baseAgent.terminalManager.LogAgentProcessOutput(agentID, outputType, data, processID)
		}
	}

	if wasmAgent, ok := i.activeWASMAgents[agentID]; ok {
		if wasmAgentImpl, ok := wasmAgent.(*WASMAgent); ok && wasmAgentImpl.terminalManager != nil {
			wasmAgentImpl.terminalManager.LogAgentProcessOutput(agentID, outputType, data, processID)
		}
	}

	return nil
}

// buildFullPromptWithContext combines system prompt, conversation context and user input into a full prompt
func (i *AgentInferencer) buildFullPromptWithContext(systemPrompt string, conversationContext []*ConversationMessage, userInput string) (string, error) {
	var prompt strings.Builder
	prompt.WriteString(systemPrompt)
	prompt.WriteString("\n\n")

	// Add conversation history if available
	if len(conversationContext) > 0 {
		prompt.WriteString("Conversation Context:\n")
		for _, msg := range conversationContext {
			prompt.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
		}
		prompt.WriteString("\n")
	}

	// Add user input
	prompt.WriteString(fmt.Sprintf("User: %s\n", userInput))

	return prompt.String(), nil
}

// processInferenceWithLLM processes an inference request using the LLM service with enhanced features
func (i *AgentInferencer) processInferenceWithLLM(ctx context.Context, agent AgentPluginInterface, request *InferenceRequest, inferenceService InferenceServiceInterface) (*InferenceResponse, error) {
	// Add user message to conversation memory
	if i.conversationMemory != nil {
		if err := i.conversationMemory.AddUserMessage(ctx, request.SessionID, request.Input); err != nil {
			// Log error but don't fail the request
			fmt.Printf("Warning: Failed to add user message to conversation memory: %v\n", err)
		}
	}

	// Get conversation history for context
	var conversationContext []*ConversationMessage
	if i.conversationMemory != nil {
		history, err := i.conversationMemory.GetRecentMessages(ctx, request.SessionID, 2000) // 2000 tokens max
		if err != nil {
			fmt.Printf("Warning: Failed to get conversation history: %v\n", err)
		} else {
			conversationContext = history
		}
	}

	// Build the system prompt with agent context
	systemPrompt, err := i.buildSystemPrompt(agent)
	if err != nil {
		return nil, fmt.Errorf("failed to build system prompt: %v", err)
	}

	// Build full prompt with conversation context
	fullPrompt, err := i.buildFullPromptWithContext(systemPrompt, conversationContext, request.Input)
	if err != nil {
		return nil, fmt.Errorf("failed to build full prompt: %v", err)
	}

	// Extract parameters from request
	instruction := ""
	if request.Parameters != nil {
		if inst, ok := request.Parameters["instruction"].(string); ok {
			instruction = inst
		}
	}

	// Generate response using inference service with retry logic
	var output string
	var lastErr error

	for attempt := 0; attempt < i.retryAttempts; attempt++ {
		// Create context with timeout
		timeoutCtx, cancel := context.WithTimeout(ctx, i.requestTimeout)
		defer cancel()

		// Generate response
		if request.Parameters != nil {
			if useCoT, ok := request.Parameters["use_cot"].(bool); ok && useCoT {
				if timeoutCtx.Err() != nil {
					lastErr = timeoutCtx.Err()
					continue
				}
				output, lastErr = inferenceService.GenerateTextWithCoT(fullPrompt)
			} else {
				if timeoutCtx.Err() != nil {
					lastErr = timeoutCtx.Err()
					continue
				}
				output, lastErr = inferenceService.GenerateText(fullPrompt, instruction)
			}
		} else {
			if timeoutCtx.Err() != nil {
				lastErr = timeoutCtx.Err()
				continue
			}
			output, lastErr = inferenceService.GenerateText(fullPrompt, instruction)
		}

		if lastErr == nil {
			break // Success
		}

		// Log retry attempt
		fmt.Printf("Inference attempt %d failed: %v\n", attempt+1, lastErr)

		// Wait before retry (exponential backoff)
		if attempt < i.retryAttempts-1 {
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("inference failed after %d attempts: %v", i.retryAttempts, lastErr)
	}

	// Parse tool calls from the output if any
	toolCalls := i.parseToolCalls(output)

	// Execute tool calls if any
	if len(toolCalls) > 0 {
		for i, toolCall := range toolCalls {
			result, err := agent.CallTool(ctx, toolCall.Name, toolCall.Input)
			if err != nil {
				// Note: ToolCall struct doesn't have Error field, so we'll set Output to error message
				toolCalls[i].Output = fmt.Sprintf("Error: %s", err.Error())
			} else {
				toolCalls[i].Output = result
			}
		}
	}

	// Convert toolCalls to pointers
	var toolCallPtrs []*ToolCall
	for i := range toolCalls {
		toolCallPtrs = append(toolCallPtrs, &toolCalls[i])
	}

	return &InferenceResponse{
		Output:    output,
		Reasoning: "Generated using LLM inference service with agent context",
		ToolCalls: toolCallPtrs,
		Metadata: map[string]interface{}{
			"agent_id":              request.SessionID,
			"session_id":            request.SessionID,
			"has_inference_service": true,
			"tool_calls_executed":   len(toolCalls),
		},
	}, nil
}

// buildSystemPrompt builds a system prompt with agent context
func (i *AgentInferencer) buildSystemPrompt(agent AgentPluginInterface) (string, error) {
	// Get agent capabilities and schema
	capabilities := agent.GetCapabilities()
	schema := agent.GetSchema()

	prompt := "You are an AI agent with the following capabilities:\n\n"

	// Add tool information
	if schema != nil && len(schema.Tools) > 0 {
		prompt += "Available Tools:\n"
		for _, tool := range schema.Tools {
			prompt += fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description)
		}
		prompt += "\n"
	}

	// Add resource information
	if schema != nil && len(schema.Resources) > 0 {
		prompt += "Available Resources:\n"
		for _, resource := range schema.Resources {
			prompt += fmt.Sprintf("- %s (%s): %s\n", resource.Name, resource.Type, resource.Description)
		}
		prompt += "\n"
	}

	// Add capabilities information
	if capabilities != nil {
		prompt += "Capabilities:\n"
		prompt += fmt.Sprintf("- Supports streaming: %t\n", capabilities.SupportsStreaming)
		prompt += fmt.Sprintf("- Supports tool calls: %t\n", capabilities.SupportsToolCalls)
		prompt += fmt.Sprintf("- Supports reasoning: %t\n", capabilities.SupportsReasoning)
		prompt += fmt.Sprintf("- Max context length: %d\n", capabilities.MaxContextLength)
		prompt += "\n"
	}

	prompt += "Instructions:\n"
	prompt += "- Use the available tools when appropriate to help the user\n"
	prompt += "- Provide clear and helpful responses\n"
	prompt += "- If you need to use a tool, format it as: TOOL_CALL[tool_name](parameter1=value1, parameter2=value2)\n"
	prompt += "- Be concise but thorough in your responses\n\n"

	return prompt, nil
}

// parseToolCalls parses tool calls from LLM output
func (i *AgentInferencer) parseToolCalls(output string) []ToolCall {
	// Simple regex to parse tool calls in format: TOOL_CALL[tool_name](param1=value1, param2=value2)
	toolCallRegex := regexp.MustCompile(`TOOL_CALL\[([^\]]+)\]\(([^)]*)\)`)
	matches := toolCallRegex.FindAllStringSubmatch(output, -1)

	var toolCalls []ToolCall
	for _, match := range matches {
		if len(match) >= 3 {
			toolName := match[1]
			paramsStr := match[2]

			// Parse parameters
			params := make(map[string]interface{})
			if paramsStr != "" {
				// Simple parameter parsing (param1=value1, param2=value2)
				paramPairs := regexp.MustCompile(`([^=,]+)=([^,]+)`).FindAllStringSubmatch(paramsStr, -1)
				for _, pair := range paramPairs {
					if len(pair) >= 3 {
						key := strings.TrimSpace(pair[1])
						value := strings.TrimSpace(pair[2])
						params[key] = value
					}
				}
			}

			toolCalls = append(toolCalls, ToolCall{
				Name:      toolName,
				Input:     params,
				Timestamp: time.Now().Unix(),
			})
		}
	}

	return toolCalls
}

// AddLLMProvider adds an LLM provider to the inferencer
func (i *AgentInferencer) AddLLMProvider(name string, provider LLMProvider) {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	i.llmProviders[name] = provider

	// Set as default if it's the first provider
	if i.defaultProvider == "" {
		i.defaultProvider = name
	}
}

// SetDefaultLLMProvider sets the default LLM provider
func (i *AgentInferencer) SetDefaultLLMProvider(name string) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	if _, exists := i.llmProviders[name]; !exists {
		return fmt.Errorf("LLM provider %s not found", name)
	}

	i.defaultProvider = name
	return nil
}

// GetLLMProvider gets an LLM provider by name
func (i *AgentInferencer) GetLLMProvider(name string) (LLMProvider, error) {
	i.mutex.RLock()
	defer i.mutex.RUnlock()

	provider, exists := i.llmProviders[name]
	if !exists {
		return nil, fmt.Errorf("LLM provider %s not found", name)
	}

	return provider, nil
}

// SetConversationMemory sets the conversation memory implementation
func (i *AgentInferencer) SetConversationMemory(memory ConversationMemory) {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	i.conversationMemory = memory
}

// PersistentConversationMemory implements ConversationMemory with persistence
type PersistentConversationMemory struct {
	conversations map[string][]*ConversationMessage
	memoryManager MemoryManager
	maxMessages   int
	mutex         sync.RWMutex
}

// NewPersistentConversationMemory creates a new persistent conversation memory
func NewPersistentConversationMemory() *PersistentConversationMemory {
	return &PersistentConversationMemory{
		conversations: make(map[string][]*ConversationMessage),
		memoryManager: NewPersistentMemoryManager(),
		maxMessages:   1000, // Maximum messages per conversation
	}
}

// AddUserMessage adds a user message to the conversation
func (m *PersistentConversationMemory) AddUserMessage(ctx context.Context, sessionID string, message string) error {
	return m.addMessage(ctx, sessionID, "user", message)
}

// AddAssistantMessage adds an assistant message to the conversation
func (m *PersistentConversationMemory) AddAssistantMessage(ctx context.Context, sessionID string, message string) error {
	return m.addMessage(ctx, sessionID, "assistant", message)
}

// AddSystemMessage adds a system message to the conversation
func (m *PersistentConversationMemory) AddSystemMessage(ctx context.Context, sessionID string, message string) error {
	return m.addMessage(ctx, sessionID, "system", message)
}

// addMessage adds a message to the conversation
func (m *PersistentConversationMemory) addMessage(ctx context.Context, sessionID, role, content string) error {
	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check context again before proceeding
	if err := ctx.Err(); err != nil {
		return err
	}

	// Create the message
	msg := &ConversationMessage{
		Role:      role,
		Content:   content,
		Timestamp: time.Now().Unix(),
	}

	// Add to in-memory conversation
	if _, exists := m.conversations[sessionID]; !exists {
		m.conversations[sessionID] = make([]*ConversationMessage, 0)
	}

	m.conversations[sessionID] = append(m.conversations[sessionID], msg)

	// Trim conversation if it exceeds max messages
	if len(m.conversations[sessionID]) > m.maxMessages {
		m.conversations[sessionID] = m.conversations[sessionID][len(m.conversations[sessionID])-m.maxMessages:]
	}

	// Persist to storage
	return m.persistConversation(sessionID)
}

// GetConversationHistory gets the conversation history for a session
func (m *PersistentConversationMemory) GetConversationHistory(ctx context.Context, sessionID string, limit int) ([]*ConversationMessage, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Load from storage if not in memory
	if _, exists := m.conversations[sessionID]; !exists {
		if err := m.loadConversation(sessionID); err != nil {
			// Return empty conversation if loading fails
			return []*ConversationMessage{}, nil
		}
	}

	messages := m.conversations[sessionID]
	if messages == nil {
		return []*ConversationMessage{}, nil
	}

	// Apply limit if specified
	if limit > 0 && len(messages) > limit {
		return messages[len(messages)-limit:], nil
	}

	return messages, nil
}

// GetRecentMessages gets recent messages within token limit
func (m *PersistentConversationMemory) GetRecentMessages(ctx context.Context, sessionID string, maxTokens int) ([]*ConversationMessage, error) {
	messages, err := m.GetConversationHistory(ctx, sessionID, 0)
	if err != nil {
		return nil, err
	}

	// Simple token estimation (4 characters per token)
	var result []*ConversationMessage
	totalTokens := 0

	// Iterate from most recent messages
	for i := len(messages) - 1; i >= 0; i-- {
		msgTokens := len(messages[i].Content) / 4
		if totalTokens+msgTokens > maxTokens {
			break
		}
		result = append([]*ConversationMessage{messages[i]}, result...)
		totalTokens += msgTokens
	}

	return result, nil
}

// ClearConversation clears the conversation for a session
func (m *PersistentConversationMemory) ClearConversation(ctx context.Context, sessionID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Clear from memory
	delete(m.conversations, sessionID)

	// Clear from storage
	return m.memoryManager.Delete(fmt.Sprintf("conversation:%s", sessionID))
}

// GetConversationSummary gets a summary of the conversation
func (m *PersistentConversationMemory) GetConversationSummary(ctx context.Context, sessionID string) (string, error) {
	messages, err := m.GetConversationHistory(ctx, sessionID, 0)
	if err != nil {
		return "", err
	}

	if len(messages) == 0 {
		return "No conversation history", nil
	}

	return fmt.Sprintf("Conversation with %d messages, started at %d",
		len(messages), messages[0].Timestamp), nil
}

// persistConversation persists a conversation to storage
func (m *PersistentConversationMemory) persistConversation(sessionID string) error {
	messages := m.conversations[sessionID]
	if messages == nil {
		return nil
	}

	// Convert messages to JSON for storage
	data, err := json.Marshal(messages)
	if err != nil {
		return fmt.Errorf("failed to marshal conversation: %v", err)
	}

	// Store in memory manager
	key := fmt.Sprintf("conversation:%s", sessionID)
	return m.memoryManager.Set(key, string(data))
}

// loadConversation loads a conversation from storage
func (m *PersistentConversationMemory) loadConversation(sessionID string) error {
	key := fmt.Sprintf("conversation:%s", sessionID)

	data, err := m.memoryManager.Get(key)
	if err != nil {
		return err
	}

	dataStr, ok := data.(string)
	if !ok {
		return fmt.Errorf("invalid conversation data format")
	}

	var messages []*ConversationMessage
	if err := json.Unmarshal([]byte(dataStr), &messages); err != nil {
		return fmt.Errorf("failed to unmarshal conversation: %v", err)
	}

	m.conversations[sessionID] = messages
	return nil
}
