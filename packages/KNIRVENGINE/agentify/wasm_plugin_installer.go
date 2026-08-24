// wasm_plugin_installer.go
package agentify

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WASMPluginInfo represents information about a WASM plugin zip file
type WASMPluginInfo struct {
	ZipPath        string                 `json:"zip_path"`
	AgentID        string                 `json:"agent_id"`
	AgentName      string                 `json:"agent_name"`
	Version        string                 `json:"version"`
	Description    string                 `json:"description"`
	BuildTarget    string                 `json:"build_target"`
	CompiledAt     time.Time              `json:"compiled_at"`
	Files          []string               `json:"files"`
	IsInstalled    bool                   `json:"is_installed"`
	InstallPath    string                 `json:"install_path,omitempty"`
	Config         map[string]interface{} `json:"config,omitempty"`
	DeploymentInfo map[string]interface{} `json:"deployment_info,omitempty"`
}

// WASMPluginInstaller manages installation of WASM plugins from zip files
type WASMPluginInstaller struct {
	downloadsDir string
	pluginsDir   string
	wasmDir      string
}

// NewWASMPluginInstaller creates a new WASM plugin installer
func NewWASMPluginInstaller(downloadsDir, pluginsDir, wasmDir string) *WASMPluginInstaller {
	return &WASMPluginInstaller{
		downloadsDir: downloadsDir,
		pluginsDir:   pluginsDir,
		wasmDir:      wasmDir,
	}
}

// DiscoverZipPlugins scans the downloads directory for WASM plugin zip files
func (installer *WASMPluginInstaller) DiscoverZipPlugins() ([]*WASMPluginInfo, error) {
	// Ensure downloads directory exists
	if _, err := os.Stat(installer.downloadsDir); os.IsNotExist(err) {
		return []*WASMPluginInfo{}, nil
	}

	// Find all zip files in downloads directory
	pattern := filepath.Join(installer.downloadsDir, "*.zip")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to scan downloads directory: %v", err)
	}

	var plugins []*WASMPluginInfo
	for _, zipPath := range matches {
		pluginInfo, err := installer.analyzeZipPlugin(zipPath)
		if err != nil {
			// Log error but continue with other files
			fmt.Printf("Warning: Failed to analyze zip plugin %s: %v\n", zipPath, err)
			continue
		}
		plugins = append(plugins, pluginInfo)
	}

	return plugins, nil
}

// analyzeZipPlugin analyzes a zip file to extract plugin information
func (installer *WASMPluginInstaller) analyzeZipPlugin(zipPath string) (*WASMPluginInfo, error) {
	// Open the zip file
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip file: %v", err)
	}
	defer reader.Close()

	pluginInfo := &WASMPluginInfo{
		ZipPath: zipPath,
	}

	// Look for config.json and deployment_info.json
	for _, file := range reader.File {
		switch file.Name {
		case "config.json":
			if err := installer.extractAndParseConfig(file, pluginInfo); err != nil {
				return nil, fmt.Errorf("failed to parse config.json: %v", err)
			}
		case "deployment_info.json":
			if err := installer.extractAndParseDeploymentInfo(file, pluginInfo); err != nil {
				return nil, fmt.Errorf("failed to parse deployment_info.json: %v", err)
			}
		}
	}

	// Extract file list
	for _, file := range reader.File {
		pluginInfo.Files = append(pluginInfo.Files, file.Name)
	}

	// Check if plugin is already installed
	pluginInfo.IsInstalled = installer.isPluginInstalled(pluginInfo.AgentID, pluginInfo.Version)
	if pluginInfo.IsInstalled {
		pluginInfo.InstallPath = installer.getInstallPath(pluginInfo.AgentID, pluginInfo.Version)
	}

	return pluginInfo, nil
}

// extractAndParseConfig extracts and parses the config.json file from the zip
func (installer *WASMPluginInstaller) extractAndParseConfig(file *zip.File, pluginInfo *WASMPluginInfo) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	configData, err := io.ReadAll(rc)
	if err != nil {
		return err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(configData, &config); err != nil {
		return err
	}

	pluginInfo.Config = config

	// Extract key fields
	if agentID, ok := config["agent_id"].(string); ok {
		pluginInfo.AgentID = agentID
	}
	if agentName, ok := config["agent_name"].(string); ok {
		pluginInfo.AgentName = agentName
	}
	if version, ok := config["version"].(string); ok {
		pluginInfo.Version = version
	}
	if description, ok := config["description"].(string); ok {
		pluginInfo.Description = description
	}
	if buildTarget, ok := config["buildTarget"].(string); ok {
		pluginInfo.BuildTarget = buildTarget
	}

	return nil
}

// extractAndParseDeploymentInfo extracts and parses the deployment_info.json file from the zip
func (installer *WASMPluginInstaller) extractAndParseDeploymentInfo(file *zip.File, pluginInfo *WASMPluginInfo) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	deploymentData, err := io.ReadAll(rc)
	if err != nil {
		return err
	}

	var deploymentInfo map[string]interface{}
	if err := json.Unmarshal(deploymentData, &deploymentInfo); err != nil {
		return err
	}

	pluginInfo.DeploymentInfo = deploymentInfo

	// Extract compiled_at timestamp
	if compiledAtStr, ok := deploymentInfo["compiled_at"].(string); ok {
		if compiledAt, err := time.Parse(time.RFC3339, compiledAtStr); err == nil {
			pluginInfo.CompiledAt = compiledAt
		}
	}

	return nil
}

// isPluginInstalled checks if a plugin is already installed
func (installer *WASMPluginInstaller) isPluginInstalled(agentID, version string) bool {
	installPath := installer.getInstallPath(agentID, version)
	_, err := os.Stat(installPath)
	return err == nil
}

// getInstallPath returns the installation path for a plugin
func (installer *WASMPluginInstaller) getInstallPath(agentID, version string) string {
	return filepath.Join(installer.wasmDir, fmt.Sprintf("%s_%s", agentID, version))
}

// InstallPlugin installs a WASM plugin from a zip file
func (installer *WASMPluginInstaller) InstallPlugin(zipPath string) (*WASMPluginInfo, error) {
	// First analyze the plugin
	pluginInfo, err := installer.analyzeZipPlugin(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze plugin: %v", err)
	}

	// Check if already installed
	if pluginInfo.IsInstalled {
		return pluginInfo, fmt.Errorf("plugin %s version %s is already installed", pluginInfo.AgentID, pluginInfo.Version)
	}

	// Create installation directory
	installPath := installer.getInstallPath(pluginInfo.AgentID, pluginInfo.Version)
	if err := os.MkdirAll(installPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create install directory: %v", err)
	}

	// Extract the zip file to the installation directory
	if err := installer.extractZipToDirectory(zipPath, installPath); err != nil {
		// Clean up on failure
		os.RemoveAll(installPath)
		return nil, fmt.Errorf("failed to extract plugin: %v", err)
	}

	// Update plugin info
	pluginInfo.IsInstalled = true
	pluginInfo.InstallPath = installPath

	return pluginInfo, nil
}

// extractZipToDirectory extracts a zip file to the specified directory
func (installer *WASMPluginInstaller) extractZipToDirectory(zipPath, destDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		// Create the file path
		filePath := filepath.Join(destDir, file.Name)

		// Ensure the directory exists
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return err
		}

		// Skip directories
		if file.FileInfo().IsDir() {
			continue
		}

		// Extract the file
		if err := installer.extractFile(file, filePath); err != nil {
			return fmt.Errorf("failed to extract file %s: %v", file.Name, err)
		}
	}

	return nil
}

// extractFile extracts a single file from the zip archive
func (installer *WASMPluginInstaller) extractFile(file *zip.File, destPath string) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	outFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, rc)
	return err
}

// UninstallPlugin removes an installed WASM plugin
func (installer *WASMPluginInstaller) UninstallPlugin(agentID, version string) error {
	installPath := installer.getInstallPath(agentID, version)
	
	if _, err := os.Stat(installPath); os.IsNotExist(err) {
		return fmt.Errorf("plugin %s version %s is not installed", agentID, version)
	}

	return os.RemoveAll(installPath)
}

// ListInstalledPlugins returns a list of installed WASM plugins
func (installer *WASMPluginInstaller) ListInstalledPlugins() ([]*WASMPluginInfo, error) {
	// Ensure WASM directory exists
	if _, err := os.Stat(installer.wasmDir); os.IsNotExist(err) {
		return []*WASMPluginInfo{}, nil
	}

	// Find all plugin directories
	entries, err := os.ReadDir(installer.wasmDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read WASM directory: %v", err)
	}

	var plugins []*WASMPluginInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Parse agent ID and version from directory name
		parts := strings.Split(entry.Name(), "_")
		if len(parts) < 2 {
			continue
		}

		agentID := strings.Join(parts[:len(parts)-1], "_")
		version := parts[len(parts)-1]

		pluginPath := filepath.Join(installer.wasmDir, entry.Name())
		pluginInfo, err := installer.analyzeInstalledPlugin(pluginPath, agentID, version)
		if err != nil {
			// Log error but continue
			fmt.Printf("Warning: Failed to analyze installed plugin %s: %v\n", pluginPath, err)
			continue
		}

		plugins = append(plugins, pluginInfo)
	}

	return plugins, nil
}

// analyzeInstalledPlugin analyzes an installed plugin directory
func (installer *WASMPluginInstaller) analyzeInstalledPlugin(pluginPath, agentID, version string) (*WASMPluginInfo, error) {
	pluginInfo := &WASMPluginInfo{
		AgentID:     agentID,
		Version:     version,
		IsInstalled: true,
		InstallPath: pluginPath,
	}

	// Read config.json if it exists
	configPath := filepath.Join(pluginPath, "config.json")
	if _, err := os.Stat(configPath); err == nil {
		configData, err := os.ReadFile(configPath)
		if err == nil {
			var config map[string]interface{}
			if json.Unmarshal(configData, &config) == nil {
				pluginInfo.Config = config
				
				// Extract fields from config
				if agentName, ok := config["agent_name"].(string); ok {
					pluginInfo.AgentName = agentName
				}
				if description, ok := config["description"].(string); ok {
					pluginInfo.Description = description
				}
			}
		}
	}

	// Read deployment_info.json if it exists
	deploymentPath := filepath.Join(pluginPath, "deployment_info.json")
	if _, err := os.Stat(deploymentPath); err == nil {
		deploymentData, err := os.ReadFile(deploymentPath)
		if err == nil {
			var deploymentInfo map[string]interface{}
			if json.Unmarshal(deploymentData, &deploymentInfo) == nil {
				pluginInfo.DeploymentInfo = deploymentInfo
				
				// Extract compiled_at timestamp
				if compiledAtStr, ok := deploymentInfo["compiled_at"].(string); ok {
					if compiledAt, err := time.Parse(time.RFC3339, compiledAtStr); err == nil {
						pluginInfo.CompiledAt = compiledAt
					}
				}
			}
		}
	}

	// List files in the plugin directory
	entries, err := os.ReadDir(pluginPath)
	if err == nil {
		for _, entry := range entries {
			pluginInfo.Files = append(pluginInfo.Files, entry.Name())
		}
	}

	return pluginInfo, nil
}
