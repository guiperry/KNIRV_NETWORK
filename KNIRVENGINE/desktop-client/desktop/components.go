package desktop

import (
	"log"
	"sync"
)

// Note: SecureBridge is defined in secure_bridge.go

// TargetSystemManager manages target systems for agent deployment
type TargetSystemManager struct {
	targetSystems map[string]*TargetSystem
	mutex         sync.RWMutex
}

// TargetSystem represents a target system for agent deployment
type TargetSystem struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Status       string                 `json:"status"`
	Capabilities []string               `json:"capabilities"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// NewTargetSystemManager creates a new target system manager
func NewTargetSystemManager() *TargetSystemManager {
	return &TargetSystemManager{
		targetSystems: make(map[string]*TargetSystem),
	}
}

// RegisterTargetSystem registers a new target system
func (tsm *TargetSystemManager) RegisterTargetSystem(system *TargetSystem) {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()

	tsm.targetSystems[system.ID] = system
	log.Printf("Target system registered: %s (%s)", system.Name, system.ID)
}

// GetTargetSystem retrieves a target system by ID
func (tsm *TargetSystemManager) GetTargetSystem(id string) (*TargetSystem, bool) {
	tsm.mutex.RLock()
	defer tsm.mutex.RUnlock()

	system, exists := tsm.targetSystems[id]
	return system, exists
}

// AgentPluginManager manages agent plugins
type AgentPluginManager struct {
	plugins map[string]*AgentPlugin
	mutex   sync.RWMutex
}

// AgentPlugin represents an agent plugin
type AgentPlugin struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	Type         string                 `json:"type"`
	Capabilities []string               `json:"capabilities"`
	Config       map[string]interface{} `json:"config"`
	Loaded       bool                   `json:"loaded"`
}

// NewAgentPluginManager creates a new agent plugin manager
func NewAgentPluginManager() *AgentPluginManager {
	return &AgentPluginManager{
		plugins: make(map[string]*AgentPlugin),
	}
}

// LoadPlugin loads an agent plugin
func (apm *AgentPluginManager) LoadPlugin(plugin *AgentPlugin) error {
	apm.mutex.Lock()
	defer apm.mutex.Unlock()

	plugin.Loaded = true
	apm.plugins[plugin.ID] = plugin
	log.Printf("Agent plugin loaded: %s v%s", plugin.Name, plugin.Version)

	return nil
}

// GetPlugin retrieves a plugin by ID
func (apm *AgentPluginManager) GetPlugin(id string) (*AgentPlugin, bool) {
	apm.mutex.RLock()
	defer apm.mutex.RUnlock()

	plugin, exists := apm.plugins[id]
	return plugin, exists
}

// Note: TEEManager is defined as DesktopTEEManager in tee_manager.go
