// agent_plugin_loader.go
package agentify

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"runtime"
	"strings"
	"sync"
	"time"
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

	// Tool management methods
	CallTool(ctx context.Context, name string, params map[string]interface{}) (interface{}, error)

	// Terminal management methods
	CreateTerminal(rows, cols int) (string, error)
	ResizeTerminal(terminalID string, rows, cols int) error
	WriteToTerminal(terminalID string, data []byte) error
	ReadFromTerminal(terminalID string) ([]byte, error)
	CloseTerminal(terminalID string) error

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

	// Handle different version formats (1.0, 1.0.0, etc.)
	var possibleVersions []string
	possibleVersions = append(possibleVersions, version)
	if !strings.Contains(version, ".") {
		possibleVersions = append(possibleVersions, version+".0")
	}
	if strings.Count(version, ".") == 1 {
		possibleVersions = append(possibleVersions, version+".0")
	}
	// Also try without version suffix
	possibleVersions = append(possibleVersions, "")

	// Try each possible version format
	var pluginPath string
	for _, ver := range possibleVersions {
		versionSuffix := ver
		if versionSuffix != "" {
			versionSuffix = "_" + versionSuffix
		}

		// Construct the plugin path
		possiblePath := filepath.Join(l.pluginsDir, fmt.Sprintf("agent_%s%s%s", agentID, versionSuffix, extension))

		// Check if the plugin file exists
		if _, err := os.Stat(possiblePath); err == nil {
			pluginPath = possiblePath
			break
		}
	}

	// Check if any plugin file was found
	if pluginPath == "" {
		return nil, fmt.Errorf("plugin file not found: %s", filepath.Join(l.pluginsDir, fmt.Sprintf("agent_%s_%s%s", agentID, version, extension)))
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

// PluginInfo represents metadata about a discovered plugin
type PluginInfo struct {
	FilePath     string                 `json:"filePath"`
	FileName     string                 `json:"fileName"`
	AgentID      string                 `json:"agentId,omitempty"`
	Version      string                 `json:"version,omitempty"`
	IsRegistered bool                   `json:"isRegistered"`
	Size         int64                  `json:"size"`
	ModTime      time.Time              `json:"modTime"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Error        string                 `json:"error,omitempty"`
}

// DiscoverAllPlugins scans the plugins directory and returns detailed information about all plugin files
func (l *AgentPluginLoader) DiscoverAllPlugins() ([]*PluginInfo, error) {
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

	// Scan the plugins directory for all plugin files
	pattern := filepath.Join(l.pluginsDir, fmt.Sprintf("*%s", extension))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to scan plugins directory: %v", err)
	}

	plugins := make([]*PluginInfo, 0, len(matches))

	for _, match := range matches {
		info, err := l.analyzePluginFile(match, extension)
		if err != nil {
			// Still include the file but with error information
			stat, _ := os.Stat(match)
			info = &PluginInfo{
				FilePath:     match,
				FileName:     filepath.Base(match),
				IsRegistered: false,
				Size:         stat.Size(),
				ModTime:      stat.ModTime(),
				Error:        err.Error(),
			}
		}
		plugins = append(plugins, info)
	}

	return plugins, nil
}

// analyzePluginFile analyzes a plugin file and extracts metadata
func (l *AgentPluginLoader) analyzePluginFile(filePath, extension string) (*PluginInfo, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %v", err)
	}

	filename := filepath.Base(filePath)
	info := &PluginInfo{
		FilePath: filePath,
		FileName: filename,
		Size:     stat.Size(),
		ModTime:  stat.ModTime(),
		Metadata: make(map[string]interface{}),
	}

	// Check if this follows the registered plugin naming convention
	if strings.HasPrefix(filename, "agent_") && strings.HasSuffix(filename, extension) {
		// Extract agent ID and version from filename
		nameWithoutPrefix := filename[6:] // Remove "agent_" prefix
		nameWithoutExt := nameWithoutPrefix[:len(nameWithoutPrefix)-len(extension)]

		// Split by last underscore to separate ID and version
		parts := strings.Split(nameWithoutExt, "_")
		if len(parts) >= 2 {
			info.Version = parts[len(parts)-1]
			info.AgentID = strings.Join(parts[:len(parts)-1], "_")
			info.IsRegistered = true
		}
	}

	// Try to extract metadata without fully loading the plugin
	metadata, err := l.extractPluginMetadata(filePath)
	if err != nil {
		info.Error = fmt.Sprintf("metadata extraction failed: %v", err)
	} else {
		info.Metadata = metadata
	}

	return info, nil
}

// extractPluginMetadata attempts to extract metadata from a plugin without fully loading it
func (l *AgentPluginLoader) extractPluginMetadata(filePath string) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})

	// For now, we'll avoid loading plugins to extract metadata due to runtime issues
	// Instead, we'll just mark basic file information and indicate that full validation
	// would require loading the plugin
	metadata["hasPluginInterface"] = false
	metadata["validationStatus"] = "pending"
	metadata["note"] = "Plugin validation requires loading - will be validated during import"

	// Check if the file appears to be a valid shared library
	if strings.HasSuffix(filePath, ".so") || strings.HasSuffix(filePath, ".dll") || strings.HasSuffix(filePath, ".dylib") {
		metadata["fileType"] = "shared_library"
		metadata["potentialPlugin"] = true
	} else {
		metadata["fileType"] = "unknown"
		metadata["potentialPlugin"] = false
		metadata["error"] = "File does not appear to be a shared library"
	}

	return metadata, nil
}

// ImportPluginRequest represents a request to import a plugin
type ImportPluginRequest struct {
	FilePath string `json:"filePath"`
	AgentID  string `json:"agentId"`
	Version  string `json:"version"`
}

// ImportPlugin imports a plugin by copying/renaming it to follow the naming convention
func (l *AgentPluginLoader) ImportPlugin(request *ImportPluginRequest) error {
	// Validate the request
	if request.FilePath == "" || request.AgentID == "" || request.Version == "" {
		return fmt.Errorf("filePath, agentId, and version are required")
	}

	// Check if the source file exists
	if _, err := os.Stat(request.FilePath); os.IsNotExist(err) {
		return fmt.Errorf("source plugin file not found: %s", request.FilePath)
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

	// Construct the target filename
	targetFilename := fmt.Sprintf("agent_%s_%s%s", request.AgentID, request.Version, extension)
	targetPath := filepath.Join(l.pluginsDir, targetFilename)

	// Check if target already exists
	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("plugin with ID '%s' and version '%s' already exists", request.AgentID, request.Version)
	}

	// If the source is already in the correct location and name, no need to copy
	if request.FilePath == targetPath {
		return nil
	}

	// Copy the file to the target location
	sourceFile, err := os.Open(request.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer sourceFile.Close()

	targetFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file: %v", err)
	}
	defer targetFile.Close()

	// Copy the file contents
	_, err = targetFile.ReadFrom(sourceFile)
	if err != nil {
		// Clean up the partially created file
		os.Remove(targetPath)
		return fmt.Errorf("failed to copy plugin file: %v", err)
	}

	// Note: We skip plugin validation during import to avoid runtime crashes
	// The plugin will be validated when it's actually loaded and used

	return nil
}
