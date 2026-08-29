package core

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AgentDiscoveryImpl implements the AgentDiscovery interface
type AgentDiscoveryImpl struct {
	pluginDir   string
	wasmDir     string
	templateDir string
}

// NewAgentDiscovery creates a new agent discovery implementation
func NewAgentDiscovery(pluginDir, wasmDir, templateDir string) *AgentDiscoveryImpl {
	return &AgentDiscoveryImpl{
		pluginDir:   pluginDir,
		wasmDir:     wasmDir,
		templateDir: templateDir,
	}
}

// DiscoverFromPlugins discovers agents from plugin files
func (d *AgentDiscoveryImpl) DiscoverFromPlugins(ctx context.Context, pluginDir string) ([]*UnifiedAgent, error) {
	if pluginDir == "" {
		pluginDir = d.pluginDir
	}

	var agents []*UnifiedAgent
	// Add a map to track unique agents by their file path to prevent duplicates
	uniqueAgents := make(map[string]*UnifiedAgent)

	// Walk through the plugin directory
	err := filepath.Walk(pluginDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip meta files to prevent confusion
		if strings.HasSuffix(path, ".meta") {
			return nil
		}

		// Handle plugin files (.so, .dll, .dylib)
		if strings.HasSuffix(path, ".so") || strings.HasSuffix(path, ".dll") || strings.HasSuffix(path, ".dylib") {
			agent, err := d.ExtractMetadataFromPlugin(ctx, path)
			if err != nil {
				// Log error but continue discovery
				return nil
			}

			// Use file path as unique key to prevent duplicates
			uniqueKey := path
			if _, exists := uniqueAgents[uniqueKey]; !exists {
				uniqueAgents[uniqueKey] = agent
			}
			return nil
		}

		// Handle ZIP files containing plugin agents
		if strings.HasSuffix(path, ".zip") {
			zipAgents, err := d.discoverFromZipFile(ctx, path, "plugin")
			if err != nil {
				// Log error but continue discovery
				return nil
			}

			// Add zip agents with unique keys
			for _, zipAgent := range zipAgents {
				uniqueKey := zipAgent.PluginPath
				if _, exists := uniqueAgents[uniqueKey]; !exists {
					uniqueAgents[uniqueKey] = zipAgent
				}
			}
			return nil
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to discover plugins: %v", err)
	}

	// Convert unique agents map to slice
	for _, agent := range uniqueAgents {
		agents = append(agents, agent)
	}

	return agents, nil
}

// DiscoverFromWASM discovers agents from WASM files
func (d *AgentDiscoveryImpl) DiscoverFromWASM(ctx context.Context, wasmDir string) ([]*UnifiedAgent, error) {
	if wasmDir == "" {
		wasmDir = d.wasmDir
	}

	var agents []*UnifiedAgent
	// Add a map to track unique agents by their file path to prevent duplicates
	uniqueAgents := make(map[string]*UnifiedAgent)

	// Walk through the WASM directory
	err := filepath.Walk(wasmDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip meta files to prevent confusion
		if strings.HasSuffix(path, ".meta") {
			return nil
		}

		// Handle WASM files
		if strings.HasSuffix(path, ".wasm") {
			agent, err := d.ExtractMetadataFromWASM(ctx, path)
			if err != nil {
				// Log error but continue discovery
				return nil
			}

			// Use file path as unique key to prevent duplicates
			uniqueKey := path
			if _, exists := uniqueAgents[uniqueKey]; !exists {
				uniqueAgents[uniqueKey] = agent
			}
			return nil
		}

		// Handle ZIP files containing WASM agents
		if strings.HasSuffix(path, ".zip") {
			zipAgents, err := d.discoverFromZipFile(ctx, path, "wasm")
			if err != nil {
				// Log error but continue discovery
				return nil
			}

			// Add zip agents with unique keys
			for _, zipAgent := range zipAgents {
				uniqueKey := zipAgent.PluginPath
				if _, exists := uniqueAgents[uniqueKey]; !exists {
					uniqueAgents[uniqueKey] = zipAgent
				}
			}
			return nil
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to discover WASM agents: %v", err)
	}

	// Convert unique agents map to slice
	for _, agent := range uniqueAgents {
		agents = append(agents, agent)
	}

	return agents, nil
}

// DiscoverFromTemplates discovers agents from template files
func (d *AgentDiscoveryImpl) DiscoverFromTemplates(ctx context.Context, templateDir string) ([]*UnifiedAgent, error) {
	if templateDir == "" {
		templateDir = d.templateDir
	}

	var agents []*UnifiedAgent

	// Walk through the template directory
	err := filepath.Walk(templateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and look for agent_prompt.json files
		if info.IsDir() || info.Name() != "agent_prompt.json" {
			return nil
		}

		// Extract metadata from template
		agent, err := d.extractMetadataFromTemplate(ctx, path)
		if err != nil {
			// Log error but continue discovery
			return nil
		}

		agents = append(agents, agent)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to discover template agents: %v", err)
	}

	return agents, nil
}

// ExtractMetadataFromPlugin extracts metadata from a plugin file
func (d *AgentDiscoveryImpl) ExtractMetadataFromPlugin(ctx context.Context, pluginPath string) (*UnifiedAgent, error) {
	// Extract agent information from filename
	filename := filepath.Base(pluginPath)
	agentInfo := d.parseAgentFilename(filename)

	// Get file info
	fileInfo, err := os.Stat(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %v", err)
	}

	agent := &UnifiedAgent{
		ID:           agentInfo.ID,
		Name:         agentInfo.Name,
		Type:         agentInfo.Type,
		Version:      agentInfo.Version,
		Description:  fmt.Sprintf("Plugin agent: %s", agentInfo.Name),
		CreatedAt:    fileInfo.ModTime(),
		UpdatedAt:    fileInfo.ModTime(),
		BuildTarget:  "plugin",
		PluginPath:   pluginPath,
		Status:       "inactive",
		Collection:   "discovered",
		Capabilities: []string{"inference", "tools"},
		TargetTypes:  []string{"general"},
		Config:       make(map[string]interface{}),
		DefaultTerminalConfig: &TerminalConfig{
			DefaultRows:    24,
			DefaultCols:    80,
			FontSize:       14,
			FontFamily:     "Menlo, Monaco, 'Courier New', monospace",
			Theme:          "dark",
			ScrollbackSize: 5000,
			AutoOpen:       false,
		},
	}

	return agent, nil
}

// ExtractMetadataFromWASM extracts metadata from a WASM file
func (d *AgentDiscoveryImpl) ExtractMetadataFromWASM(ctx context.Context, wasmPath string) (*UnifiedAgent, error) {
	// Extract agent information from filename
	filename := filepath.Base(wasmPath)
	agentInfo := d.parseAgentFilename(filename)

	// Get file info
	fileInfo, err := os.Stat(wasmPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %v", err)
	}

	agent := &UnifiedAgent{
		ID:           agentInfo.ID,
		Name:         agentInfo.Name,
		Type:         agentInfo.Type,
		Version:      agentInfo.Version,
		Description:  fmt.Sprintf("WASM agent: %s", agentInfo.Name),
		CreatedAt:    fileInfo.ModTime(),
		UpdatedAt:    fileInfo.ModTime(),
		BuildTarget:  "wasm",
		PluginPath:   wasmPath,
		Status:       "inactive",
		Collection:   "discovered",
		Capabilities: []string{"inference", "tools"},
		TargetTypes:  []string{"general"},
		Config:       make(map[string]interface{}),
		DefaultTerminalConfig: &TerminalConfig{
			DefaultRows:    24,
			DefaultCols:    80,
			FontSize:       14,
			FontFamily:     "Menlo, Monaco, 'Courier New', monospace",
			Theme:          "dark",
			ScrollbackSize: 5000,
			AutoOpen:       false,
		},
	}

	return agent, nil
}

// extractMetadataFromTemplate extracts metadata from a template file
func (d *AgentDiscoveryImpl) extractMetadataFromTemplate(ctx context.Context, templatePath string) (*UnifiedAgent, error) {
	// Use context for cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Continue processing
	}

	// Read the agent_prompt.json file
	data, err := ioutil.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %v", err)
	}

	// Parse the JSON
	var templateData map[string]interface{}
	if err := json.Unmarshal(data, &templateData); err != nil {
		return nil, fmt.Errorf("failed to parse template JSON: %v", err)
	}

	// Extract template directory name as agent name
	templateDir := filepath.Dir(templatePath)
	agentName := filepath.Base(templateDir)

	agent := &UnifiedAgent{
		ID:           uuid.New().String(),
		Name:         agentName,
		Type:         "template",
		Version:      "1.0.0",
		Description:  fmt.Sprintf("Template agent: %s", agentName),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		BuildTarget:  "template",
		PluginPath:   templatePath,
		Status:       "template",
		Collection:   "templates",
		Capabilities: []string{"template", "buildable"},
		TargetTypes:  []string{"general"},
		Config:       templateData,
		DefaultTerminalConfig: &TerminalConfig{
			DefaultRows:    24,
			DefaultCols:    80,
			FontSize:       14,
			FontFamily:     "Menlo, Monaco, 'Courier New', monospace",
			Theme:          "dark",
			ScrollbackSize: 5000,
			AutoOpen:       false,
		},
	}

	// Extract additional metadata from template data
	if name, ok := templateData["name"].(string); ok && name != "" {
		agent.Name = name
	}
	if description, ok := templateData["description"].(string); ok && description != "" {
		agent.Description = description
	}
	if version, ok := templateData["version"].(string); ok && version != "" {
		agent.Version = version
	}

	return agent, nil
}

// AgentFileInfo represents parsed agent file information
type AgentFileInfo struct {
	ID      string
	Name    string
	Type    string
	Version string
}

// parseAgentFilename parses agent information from filename
func (d *AgentDiscoveryImpl) parseAgentFilename(filename string) AgentFileInfo {
	// Remove extension
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Try to parse pattern: agent_<name>_<version>
	re := regexp.MustCompile(`^agent_(.+)_(.+)$`)
	matches := re.FindStringSubmatch(name)

	if len(matches) == 3 {
		agentName := matches[1]
		version := matches[2]
		// Create deterministic ID based on agent name and version
		deterministicID := fmt.Sprintf("%s_%s", agentName, version)
		return AgentFileInfo{
			ID:      deterministicID,
			Name:    agentName,
			Type:    "agent",
			Version: version,
		}
	}

	// Fallback to using the full filename as name
	// Create deterministic ID based on filename
	return AgentFileInfo{
		ID:      name,
		Name:    name,
		Type:    "agent",
		Version: "1.0.0",
	}
}

// DiscoverFromZip discovers agents from a zip archive
func (d *AgentDiscoveryImpl) DiscoverFromZip(ctx context.Context, zipPath string) ([]*UnifiedAgent, error) {
	// Determine the likely agent type based on zip contents
	return d.discoverFromZipFile(ctx, zipPath, "")
}

// discoverFromZipFile discovers agents from a zip file
func (d *AgentDiscoveryImpl) discoverFromZipFile(ctx context.Context, zipPath string, agentType string) ([]*UnifiedAgent, error) {
	// Use context for cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Continue processing
	}
	var agents []*UnifiedAgent

	// Open the zip file
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip file %s: %v", zipPath, err)
	}
	defer reader.Close()

	// Look for config.json to get agent metadata
	var configData map[string]interface{}
	var hasConfig bool

	for _, file := range reader.File {
		if file.Name == "config.json" || strings.HasSuffix(file.Name, "/config.json") {
			rc, err := file.Open()
			if err != nil {
				continue
			}
			defer rc.Close()

			configBytes, err := io.ReadAll(rc)
			if err != nil {
				continue
			}

			if err := json.Unmarshal(configBytes, &configData); err != nil {
				continue
			}
			hasConfig = true
			break
		}
	}

	// If no config found, try to extract from filename
	if !hasConfig {
		fileInfo := d.parseAgentFilename(filepath.Base(zipPath))
		configData = map[string]interface{}{
			"agentId":   fileInfo.ID,
			"agentName": fileInfo.Name,
			"version":   fileInfo.Version,
		}
	}

	// Create agent based on type and config
	agent := &UnifiedAgent{
		ID:           getStringFromConfig(configData, "agentId", uuid.New().String()),
		Name:         getStringFromConfig(configData, "agentName", "Unknown Agent"),
		Description:  getStringFromConfig(configData, "description", "Agent loaded from zip file"),
		Type:         "agent",
		BuildTarget:  agentType,
		Version:      getStringFromConfig(configData, "version", "1.0.0"),
		Status:       "inactive",
		PluginPath:   zipPath,
		Collection:   "default",
		Tags:         []string{"zip", agentType},
		Capabilities: []string{},
		TargetTypes:  []string{},
		Config:       configData,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		DefaultTerminalConfig: &TerminalConfig{
			DefaultRows:    24,
			DefaultCols:    80,
			FontSize:       14,
			FontFamily:     "monospace",
			Theme:          "dark",
			ScrollbackSize: 1000,
		},
	}

	agents = append(agents, agent)
	return agents, nil
}

// getStringFromConfig safely gets a string value from config map
func getStringFromConfig(config map[string]interface{}, key string, defaultValue string) string {
	if val, ok := config[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}
