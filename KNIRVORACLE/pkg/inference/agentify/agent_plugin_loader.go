// agent_plugin_loader.go
package agentify

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"runtime"
	"sync"
)

// AgentPluginInterface defines the interface that all agent plugins must implement
type AgentPluginInterface interface {
	// Initialize the agent with configuration
	Initialize(config map[string]interface{}) error

	// Lifecycle management
	Start() error
	Stop() error

	// Process an inference request and return a response
	ProcessInference(ctx context.Context, request *InferenceRequest) (*InferenceResponse, error)

	// Get the agent's capabilities
	GetCapabilities() *AgentCapabilities

	// Get the agent's schema (tools, resources, prompts)
	GetSchema() *AgentSchema

	// Get information about the TEE
	GetTEEInfo() map[string]interface{}

	// Memory management methods
	GetMemory(key string) (interface{}, error)
	SetMemory(key string, value interface{}) error

	// Legacy memory management methods for backward compatibility
	StoreContext(contextID string, context map[string]interface{}) error
	GetContext(contextID string) (map[string]interface{}, error)
	TransferContext(contextID string, targetAgentID string) error
	StoreCredential(credentialID string, credential map[string]interface{}) error
	GetCredential(credentialID string) (map[string]interface{}, error)
	StoreRAGResult(queryHash string, result map[string]interface{}, ttl int64) error
	GetRAGResult(queryHash string) (map[string]interface{}, error)
	StoreCOTPlan(planID string, plan map[string]interface{}) error
	GetCOTPlan(planID string) (map[string]interface{}, error)
	StoreUserPreference(userID string, preference map[string]interface{}) error
	GetUserPreferences(userID string) (map[string]interface{}, error)
	GetUserPreference(userID string, key string) (interface{}, error)
}

// AgentPluginLoader handles loading and managing agent plugins
type AgentPluginLoader struct {
	pluginsDir    string
	loadedPlugins map[string]*loadedPlugin
	mutex         sync.RWMutex
}

type loadedPlugin struct {
	pluginPath string
	goPlugin   *plugin.Plugin
	instance   AgentPluginInterface
}

// NewAgentPluginLoader creates a new agent plugin loader
func NewAgentPluginLoader(pluginsDir string) *AgentPluginLoader {
	return &AgentPluginLoader{
		pluginsDir:    pluginsDir,
		loadedPlugins: make(map[string]*loadedPlugin),
	}
}

// LoadPlugin loads an agent plugin by ID and version
func (l *AgentPluginLoader) LoadPlugin(agentID string, version string, config ...map[string]interface{}) (AgentPluginInterface, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// Check if the plugin is already loaded
	pluginKey := fmt.Sprintf("%s_%s", agentID, version)
	if plugin, ok := l.loadedPlugins[pluginKey]; ok {
		return plugin.instance, nil
	}

	// Determine the plugin file extension based on the OS
	var extension string
	switch runtime.GOOS {
	case "windows":
		extension = ".dll"
	case "darwin":
		extension = ".dylib"
	default: // linux and others
		extension = ".so"
	}

	// Construct the plugin path
	pluginPath := filepath.Join(l.pluginsDir, fmt.Sprintf("agent_%s_%s%s", agentID, version, extension))

	// Check if the plugin file exists
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin file not found: %s", pluginPath)
	}

	// Load the plugin
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin: %v", err)
	}

	// Look up the Plugin symbol
	pluginSymbol, err := p.Lookup("Plugin")
	if err != nil {
		return nil, fmt.Errorf("failed to find Plugin symbol: %v", err)
	}

	// Assert that the symbol is of the AgentPluginInterface type
	instance, ok := pluginSymbol.(AgentPluginInterface)
	if !ok {
		return nil, fmt.Errorf("plugin does not implement AgentPluginInterface")
	}

	// Initialize the plugin with configuration
	pluginConfig := map[string]interface{}{
		"agentID": agentID,
		"version": version,
	}

	// If a custom config was provided, use it
	if len(config) > 0 && config[0] != nil {
		for k, v := range config[0] {
			pluginConfig[k] = v
		}
	}

	// Initialize the plugin
	if err := instance.Initialize(pluginConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize plugin: %v", err)
	}

	// Store the loaded plugin
	l.loadedPlugins[pluginKey] = &loadedPlugin{
		pluginPath: pluginPath,
		goPlugin:   p,
		instance:   instance,
	}

	return instance, nil
}

// UnloadPlugin unloads an agent plugin
func (l *AgentPluginLoader) UnloadPlugin(agentID string, version string) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// Check if the plugin is loaded
	pluginKey := fmt.Sprintf("%s_%s", agentID, version)
	plugin, ok := l.loadedPlugins[pluginKey]
	if !ok {
		return fmt.Errorf("plugin not loaded: %s", pluginKey)
	}

	// Stop the plugin
	if err := plugin.instance.Stop(); err != nil {
		return fmt.Errorf("failed to stop plugin: %v", err)
	}

	// Remove the plugin from the loaded plugins map
	delete(l.loadedPlugins, pluginKey)

	return nil
}

// ListLoadedPlugins returns a list of loaded plugins
func (l *AgentPluginLoader) ListLoadedPlugins() []string {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	plugins := make([]string, 0, len(l.loadedPlugins))
	for key := range l.loadedPlugins {
		plugins = append(plugins, key)
	}

	return plugins
}

// DiscoverPlugins scans the plugins directory and returns a list of available plugins
func (l *AgentPluginLoader) DiscoverPlugins() ([]string, error) {
	// Determine the plugin file extension based on the OS
	var extension string
	switch runtime.GOOS {
	case "windows":
		extension = ".dll"
	case "darwin":
		extension = ".dylib"
	default: // linux and others
		extension = ".so"
	}

	// Scan the plugins directory
	pattern := filepath.Join(l.pluginsDir, fmt.Sprintf("agent_*%s", extension))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to scan plugins directory: %v", err)
	}

	// Extract plugin IDs and versions from filenames
	plugins := make([]string, 0, len(matches))
	for _, match := range matches {
		filename := filepath.Base(match)
		// Remove the "agent_" prefix and the extension
		info := filename[6 : len(filename)-len(extension)]
		plugins = append(plugins, info)
	}

	return plugins, nil
}
