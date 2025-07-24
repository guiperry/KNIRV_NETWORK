package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/google/uuid"
)

// MetaFilesSubDirectory is the subdirectory within the output path for meta files
const MetaFilesSubDirectory = "data"

// AgentConfig represents the configuration for an agent
type AgentConfig struct {
	AgentID           string                 `json:"agent_id,omitempty"`
	AgentType         string                 `json:"agent_type"`
	Name              string                 `json:"name"`
	Model             string                 `json:"model,omitempty"`
	Instruction       string                 `json:"instruction,omitempty"`
	Description       string                 `json:"description,omitempty"`
	UseSearch         bool                   `json:"use_search,omitempty"`
	UseCodeExecution  bool                   `json:"use_code_execution,omitempty"`
	UseVertexSearch   bool                   `json:"use_vertex_search,omitempty"`
	VertexDatastoreID string                 `json:"vertex_datastore_id,omitempty"`
	CustomTools       []Tool                 `json:"custom_tools,omitempty"`
	SubAgents         []string               `json:"sub_agents,omitempty"`
	MaxIterations     int                    `json:"max_iterations,omitempty"`
	BuildTarget       string                 `json:"build_target,omitempty"` // "plugin" or "wasm"
	ExtraParams       map[string]interface{} `json:"extra_params,omitempty"`
}

// Tool represents a tool configuration for an agent
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Endpoint    string                 `json:"endpoint"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// TemplateData represents the data used for template processing
// Field names use camelCase to match template expectations
type TemplateData struct {
	AgentId                  string
	AgentName                string
	AgentDescription         string
	AgentVersion             string
	AgentType                string
	Model                    string
	Instruction              string
	UseSearch                bool
	UseCodeExecution         bool
	UseVertexSearch          bool
	VertexDatastoreID        string
	MaxIterations            int
	FactsUrl                 string
	PrivateFactsUrl          string
	AdaptiveRouterUrl        string
	Ttl                      int
	Signature                string
	IsolationLevel           string
	MemoryLimit              int
	CpuCores                 int
	TimeoutSec               int
	NetworkAccess            bool
	FileSystemAccess         bool
	ToolImports              string
	CustomTools              []Tool
	SubAgents                []string
	ExtraParams              map[string]interface{}
	PythonAgentServiceScript string
	PythonRequirements       string
}

// BuildStatus represents the status of an agent build operation
type BuildStatus struct {
	AgentID     string    `json:"agent_id"`
	Status      string    `json:"status"` // "pending", "building", "completed", "failed"
	Progress    int       `json:"progress"`
	Message     string    `json:"message"`
	Error       string    `json:"error,omitempty"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	LogOutput   []string  `json:"log_output"`
}

// TemplateValidationError represents a template validation error
type TemplateValidationError struct {
	TemplateName string `json:"template_name"`
	Error        string `json:"error"`
	Line         int    `json:"line,omitempty"`
	Column       int    `json:"column,omitempty"`
}

// AgentBuilder manages the creation and execution of agents
type AgentBuilder struct {
	registry      *AgentRegistry
	templatesPath string
	outputPath    string
	buildStatuses map[string]*BuildStatus
	statusMutex   sync.RWMutex // Protects buildStatuses map
}

// NewAgentBuilder creates a new agent builder (deprecated - use NewAgentBuilderWithStorage)
func NewAgentBuilder(dbPath, templatesPath, outputPath string) (*AgentBuilder, error) {
	// Create the agent registry
	registry, err := NewAgentRegistry(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent registry: %v", err)
	}

	return newAgentBuilderInternal(registry, templatesPath, outputPath)
}

// NewAgentBuilderWithStorage creates a new agent builder using an existing unified storage
func NewAgentBuilderWithStorage(storage *UnifiedAgentStorage, templatesPath, outputPath string) (*AgentBuilder, error) {
	// Create a registry adapter for the unified storage
	registry := &AgentRegistry{storage: storage}
	return newAgentBuilderInternal(registry, templatesPath, outputPath)
}

// newAgentBuilderInternal is the internal constructor used by both public constructors
func newAgentBuilderInternal(registry *AgentRegistry, templatesPath, outputPath string) (*AgentBuilder, error) {
	// Set default paths if not provided
	if templatesPath == "" {
		templatesPath = "agent/templates"
	}
	if outputPath == "" {
		outputPath = "plugins"
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %v", err)
	}

	return &AgentBuilder{
		registry:      registry,
		templatesPath: templatesPath,
		outputPath:    outputPath,
		buildStatuses: make(map[string]*BuildStatus),
		statusMutex:   sync.RWMutex{},
	}, nil
}

// BuildAgent creates a new agent and builds the plugin
func (b *AgentBuilder) BuildAgent(config AgentConfig) (string, error) {
	// Generate an agent ID if not provided
	if config.AgentID == "" {
		config.AgentID = uuid.New().String()
	}

	// Initialize build status
	buildStatus := &BuildStatus{
		AgentID:   config.AgentID,
		Status:    "pending",
		Progress:  0,
		Message:   "Initializing build process",
		StartedAt: time.Now(),
		LogOutput: []string{},
	}
	b.statusMutex.Lock()
	b.buildStatuses[config.AgentID] = buildStatus
	b.statusMutex.Unlock()

	// Validate agent configuration
	b.updateBuildStatus(config.AgentID, "building", "Validating configuration", 10)
	validationErrors := b.ValidateAgentConfig(config)
	if len(validationErrors) > 0 {
		errorMsg := fmt.Sprintf("Configuration validation failed: %d errors", len(validationErrors))
		b.updateBuildStatus(config.AgentID, "failed", errorMsg, 0)
		return "", fmt.Errorf("configuration validation failed: %v", validationErrors)
	}

	// Validate templates
	b.updateBuildStatus(config.AgentID, "building", "Validating templates", 20)
	if err := b.validateAllTemplates(); err != nil {
		b.updateBuildStatus(config.AgentID, "failed", "Template validation failed", 0)
		return "", fmt.Errorf("template validation failed: %v", err)
	}

	// Pre-register the agent configuration before building
	// This ensures the agent exists in the registry when compilePlugin looks for it
	b.updateBuildStatus(config.AgentID, "building", "Pre-registering agent configuration", 25)
	preRegisterConfigMap := map[string]interface{}{
		"agent_id":            config.AgentID,
		"agent_type":          config.AgentType,
		"name":                config.Name,
		"model":               config.Model,
		"instruction":         config.Instruction,
		"description":         config.Description,
		"use_search":          config.UseSearch,
		"use_code_execution":  config.UseCodeExecution,
		"use_vertex_search":   config.UseVertexSearch,
		"vertex_datastore_id": config.VertexDatastoreID,
		"custom_tools":        config.CustomTools,
		"sub_agents":          config.SubAgents,
		"max_iterations":      config.MaxIterations,
		"build_target":        config.BuildTarget,
	}

	if err := b.registry.RegisterAgent(config.AgentID, preRegisterConfigMap); err != nil {
		b.updateBuildStatus(config.AgentID, "failed", "Failed to pre-register agent", 0)
		return "", fmt.Errorf("failed to pre-register agent: %v", err)
	}

	// Build the plugin from templates
	b.updateBuildStatus(config.AgentID, "building", "Building plugin from templates", 30)
	pluginPath, err := b.buildPluginFromTemplates(config)
	if err != nil {
		b.updateBuildStatus(config.AgentID, "failed", "Plugin build failed", 0)
		return "", fmt.Errorf("failed to build plugin: %v", err)
	}

	// Update the agent configuration with the plugin path
	b.updateBuildStatus(config.AgentID, "building", "Updating agent configuration", 80)
	configMap := map[string]interface{}{
		"agent_id":            config.AgentID,
		"agent_type":          config.AgentType,
		"name":                config.Name,
		"model":               config.Model,
		"instruction":         config.Instruction,
		"description":         config.Description,
		"use_search":          config.UseSearch,
		"use_code_execution":  config.UseCodeExecution,
		"use_vertex_search":   config.UseVertexSearch,
		"vertex_datastore_id": config.VertexDatastoreID,
		"custom_tools":        config.CustomTools,
		"sub_agents":          config.SubAgents,
		"max_iterations":      config.MaxIterations,
		"plugin_path":         pluginPath,
	}

	// Add extra parameters if provided
	if config.ExtraParams != nil {
		for k, v := range config.ExtraParams {
			configMap[k] = v
		}
	}

	// Store the agent configuration in the unified storage
	if err := b.registry.RegisterAgent(config.AgentID, configMap); err != nil {
		b.updateBuildStatus(config.AgentID, "failed", "Failed to register agent", 0)
		return "", fmt.Errorf("failed to register agent: %v", err)
	}

	// Build completed successfully
	b.updateBuildStatus(config.AgentID, "completed", "Agent build completed successfully", 100)
	b.addBuildLog(config.AgentID, fmt.Sprintf("Agent %s built successfully at %s", config.Name, pluginPath))

	return config.AgentID, nil
}

// GetAgent retrieves an agent configuration
func (b *AgentBuilder) GetAgent(agentID string) (AgentConfig, error) {
	// Get the agent configuration from the registry
	configMap, err := b.registry.GetAgent(agentID)
	if err != nil {
		return AgentConfig{}, fmt.Errorf("failed to get agent: %v", err)
	}

	// Convert map to AgentConfig with nil checks
	config := AgentConfig{}

	// Helper function to safely extract string values
	getString := func(key string) string {
		if val, ok := configMap[key]; ok && val != nil {
			if str, ok := val.(string); ok {
				return str
			}
		}
		return ""
	}

	// Helper function to safely extract bool values
	getBool := func(key string) bool {
		if val, ok := configMap[key]; ok && val != nil {
			if b, ok := val.(bool); ok {
				return b
			}
		}
		return false
	}

	config.AgentID = getString("agent_id")
	config.AgentType = getString("agent_type")
	config.Name = getString("name")
	config.Model = getString("model")
	config.Instruction = getString("instruction")
	config.Description = getString("description")
	config.UseSearch = getBool("use_search")
	config.UseCodeExecution = getBool("use_code_execution")
	config.UseVertexSearch = getBool("use_vertex_search")
	config.VertexDatastoreID = getString("vertex_datastore_id")

	// Convert max_iterations
	if maxIter, ok := configMap["max_iterations"].(float64); ok {
		config.MaxIterations = int(maxIter)
	}

	// Convert sub_agents
	if subAgents, ok := configMap["sub_agents"].([]interface{}); ok {
		config.SubAgents = make([]string, len(subAgents))
		for i, sa := range subAgents {
			config.SubAgents[i] = sa.(string)
		}
	}

	// Convert custom_tools
	if customTools, ok := configMap["custom_tools"].([]interface{}); ok {
		config.CustomTools = make([]Tool, len(customTools))
		for i, ct := range customTools {
			toolMap := ct.(map[string]interface{})
			config.CustomTools[i] = Tool{
				Name:        toolMap["name"].(string),
				Description: toolMap["description"].(string),
				Endpoint:    toolMap["endpoint"].(string),
				Parameters:  toolMap["parameters"].(map[string]interface{}),
			}
		}
	}

	return config, nil
}

// DeleteAgent deletes an agent
func (b *AgentBuilder) DeleteAgent(agentID string) error {
	return b.registry.DeleteAgent(agentID)
}

// ListAgents lists all agents
func (b *AgentBuilder) ListAgents() ([]string, error) {
	return b.registry.ListAgents()
}

// Close closes the agent builder
func (b *AgentBuilder) Close() error {
	return b.registry.Close()
}

// buildPluginFromTemplates builds an agent plugin from templates
func (b *AgentBuilder) buildPluginFromTemplates(config AgentConfig) (string, error) {
	// Create template data
	templateData := b.createTemplateData(config)

	// Create a temporary directory for building the plugin
	tempDir, err := os.MkdirTemp("", "agent_build_"+config.AgentID)
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Process all template files
	if err := b.processTemplateFiles(templateData, tempDir); err != nil {
		return "", fmt.Errorf("failed to process template files: %v", err)
	}

	// Build the plugin using the compilePluginWithConfig method
	pluginPath, err := b.compilePluginWithConfig(config, tempDir)
	if err != nil {
		return "", fmt.Errorf("failed to compile plugin: %v", err)
	}

	return pluginPath, nil
}

// createTemplateData creates template data from agent config
func (b *AgentBuilder) createTemplateData(config AgentConfig) map[string]interface{} {
	// Set default values
	agentVersion := "1.0"
	if version, ok := config.ExtraParams["version"].(string); ok && version != "" {
		agentVersion = version
	}

	// Set default TEE configuration
	isolationLevel := "container"
	memoryLimit := 512
	cpuCores := 1
	timeoutSec := 300
	networkAccess := true
	fileSystemAccess := true

	// Override with config values if provided
	if teeConfig, ok := config.ExtraParams["tee"].(map[string]interface{}); ok {
		if isolation, ok := teeConfig["isolationLevel"].(string); ok {
			isolationLevel = isolation
		}
		if memory, ok := teeConfig["memoryLimit"].(float64); ok {
			memoryLimit = int(memory)
		}
		if cores, ok := teeConfig["cpuCores"].(float64); ok {
			cpuCores = int(cores)
		}
		if timeout, ok := teeConfig["timeoutSec"].(float64); ok {
			timeoutSec = int(timeout)
		}
		if network, ok := teeConfig["networkAccess"].(bool); ok {
			networkAccess = network
		}
		if filesystem, ok := teeConfig["fileSystemAccess"].(bool); ok {
			fileSystemAccess = filesystem
		}
	}

	// Generate tool imports
	toolImports := b.generateToolImports(config.CustomTools)

	// Load Python script and requirements
	pythonScript, err := b.loadPythonAgentServiceScript()
	if err != nil {
		// Use a default empty script if loading fails
		pythonScript = ""
	}

	pythonRequirements := b.generatePythonRequirements()

	// Extract API keys from extra params
	apiKeys := map[string]string{
		"openai":   "",
		"claude":   "",
		"gemini":   "",
		"deepseek": "",
		"cerebras": "",
	}

	// Check if api_keys is nested in extra params
	if apiKeysMap, ok := config.ExtraParams["api_keys"].(map[string]interface{}); ok {
		if openai, ok := apiKeysMap["openai"].(string); ok {
			apiKeys["openai"] = openai
		}
		if claude, ok := apiKeysMap["claude"].(string); ok {
			apiKeys["claude"] = claude
		}
		if gemini, ok := apiKeysMap["gemini"].(string); ok {
			apiKeys["gemini"] = gemini
		}
		if deepseek, ok := apiKeysMap["deepseek"].(string); ok {
			apiKeys["deepseek"] = deepseek
		}
		if cerebras, ok := apiKeysMap["cerebras"].(string); ok {
			apiKeys["cerebras"] = cerebras
		}
	} else {
		// Fallback to direct keys in extra params for backward compatibility
		apiKeys["openai"] = getStringParam(config.ExtraParams, "openai_api_key", "")
		apiKeys["claude"] = getStringParam(config.ExtraParams, "claude_api_key", "")
		apiKeys["gemini"] = getStringParam(config.ExtraParams, "gemini_api_key", "")
		apiKeys["deepseek"] = getStringParam(config.ExtraParams, "deepseek_api_key", "")
		apiKeys["cerebras"] = getStringParam(config.ExtraParams, "cerebras_api_key", "")
	}

	// Set default build target
	buildTarget := config.BuildTarget
	if buildTarget == "" {
		buildTarget = "plugin"
	}

	return map[string]interface{}{
		"agentId":                  config.AgentID,
		"agentName":                config.Name,
		"agentDescription":         config.Description,
		"agentVersion":             agentVersion,
		"agentType":                config.AgentType,
		"model":                    config.Model,
		"instruction":              config.Instruction,
		"useSearch":                config.UseSearch,
		"useCodeExecution":         config.UseCodeExecution,
		"useVertexSearch":          config.UseVertexSearch,
		"vertexDatastoreId":        config.VertexDatastoreID,
		"maxIterations":            config.MaxIterations,
		"factsUrl":                 getStringParam(config.ExtraParams, "factsUrl", ""),
		"privateFactsUrl":          getStringParam(config.ExtraParams, "privateFactsUrl", ""),
		"adaptiveRouterUrl":        getStringParam(config.ExtraParams, "adaptiveRouterUrl", ""),
		"ttl":                      getIntParam(config.ExtraParams, "ttl", 3600),
		"signature":                getStringParam(config.ExtraParams, "signature", ""),
		"isolationLevel":           isolationLevel,
		"memoryLimit":              memoryLimit,
		"cpuCores":                 cpuCores,
		"timeoutSec":               timeoutSec,
		"networkAccess":            networkAccess,
		"fileSystemAccess":         fileSystemAccess,
		"toolImports":              toolImports,
		"customTools":              config.CustomTools,
		"subAgents":                config.SubAgents,
		"buildTarget":              buildTarget,
		"apiKeys":                  apiKeys,
		"extraParams":              config.ExtraParams,
		"pythonAgentServiceScript": pythonScript,
		"pythonRequirements":       pythonRequirements,
	}
}

// Helper functions for extracting parameters
func getStringParam(params map[string]interface{}, key, defaultValue string) string {
	if value, ok := params[key].(string); ok {
		return value
	}
	return defaultValue
}

func getIntParam(params map[string]interface{}, key string, defaultValue int) int {
	if value, ok := params[key].(float64); ok {
		return int(value)
	}
	if value, ok := params[key].(int); ok {
		return value
	}
	return defaultValue
}

// generateToolImports generates import statements for custom tools
func (b *AgentBuilder) generateToolImports(tools []Tool) string {
	var imports []string

	// Add standard imports that are always needed
	imports = append(imports, "context")
	imports = append(imports, "encoding/json")
	imports = append(imports, "fmt")
	imports = append(imports, "strings")

	// Add tool-specific imports if needed
	for _, tool := range tools {
		// Add any specific imports based on tool type
		// This can be extended based on tool requirements
		_ = tool // Avoid unused variable warning for now
	}

	if len(imports) == 0 {
		return ""
	}

	// Format as Go import statements
	var formattedImports []string
	for _, imp := range imports {
		formattedImports = append(formattedImports, fmt.Sprintf(`"%s"`, imp))
	}

	return strings.Join(formattedImports, "\n\t")
}

// loadPythonAgentServiceScript loads the Python agent service script from template
func (b *AgentBuilder) loadPythonAgentServiceScript() (string, error) {
	scriptPath := filepath.Join(b.templatesPath, "agent_service.py.template")
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read Python script template: %v", err)
	}
	// Escape the content for Go string literal
	escaped := strings.ReplaceAll(string(content), "`", "` + \"`\" + `")
	escaped = strings.ReplaceAll(escaped, "\\", "\\\\")
	return escaped, nil
}

// generatePythonRequirements generates the Python requirements.txt content
func (b *AgentBuilder) generatePythonRequirements() string {
	requirements := []string{
		"flask>=2.0.0",
		"requests>=2.25.0",
		"typing-extensions>=4.0.0",
	}
	return strings.Join(requirements, "\n")
}

// processTemplateFiles processes all template files and generates the plugin source
func (b *AgentBuilder) processTemplateFiles(data map[string]interface{}, outputDir string) error {
	// Get list of template files
	templateFiles, err := filepath.Glob(filepath.Join(b.templatesPath, "*.template"))
	if err != nil {
		return fmt.Errorf("failed to find template files: %v", err)
	}

	if len(templateFiles) == 0 {
		return fmt.Errorf("no template files found in %s", b.templatesPath)
	}

	// Define essential templates that should always be processed
	essentialTemplates := map[string]bool{
		"main.go.template":                     true,
		"go.mod.template":                      true,
		"resources.go.template":                true,
		"tee.go.template":                      true,
		"llm_inference.go.template":            true,
		"subagent_manager.go.template":         true, // Sub-agent management for orchestration
		"agent_monitoring.go.template":         true, // Agent monitoring and logging
		"deterministic_embeddings.go.template": true, // Deterministic embeddings support
		"config.env.template":                  true, // Configuration template
		"test.go.template":                     true, // For testing
		"agent_prompt.json.template":           true, // Agent prompt generation framework
	}

	// Process each template file, but only essential ones for now
	for _, templateFile := range templateFiles {
		templateName := filepath.Base(templateFile)

		// Skip non-essential templates for basic agent functionality
		if !essentialTemplates[templateName] {
			continue
		}

		if err := b.processTemplateFile(templateFile, data, outputDir); err != nil {
			return fmt.Errorf("failed to process template %s: %v", templateFile, err)
		}
	}

	return nil
}

// processTemplateFile processes a single template file
func (b *AgentBuilder) processTemplateFile(templatePath string, data map[string]interface{}, outputDir string) error {
	// Read template content
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template file: %v", err)
	}

	// Create template
	tmpl, err := template.New(filepath.Base(templatePath)).Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	// Determine output filename
	outputFilename := strings.TrimSuffix(filepath.Base(templatePath), ".template")
	outputPath := filepath.Join(outputDir, outputFilename)

	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outputFile.Close()

	// Execute template
	if err := tmpl.Execute(outputFile, data); err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	return nil
}


// compilePluginWithConfig compiles the generated source code into a plugin or WASM using provided config
func (b *AgentBuilder) compilePluginWithConfig(config AgentConfig, sourceDir string) (string, error) {
	// Determine build target (default to plugin for backward compatibility)
	buildTarget := config.BuildTarget
	if buildTarget == "" {
		buildTarget = "plugin"
	}

	// Debug logging

	var pluginFilename, outputPath string
	var cmd *exec.Cmd
	agentID := config.AgentID

	// Log compilation start
	b.addBuildLog(agentID, fmt.Sprintf("Starting %s compilation", buildTarget))
	b.updateBuildStatus(agentID, "building", fmt.Sprintf("Compiling %s", buildTarget), 50)

	// Find the main.go file in the source directory
	mainGoPath := filepath.Join(sourceDir, "main.go")
	if _, err := os.Stat(mainGoPath); os.IsNotExist(err) {
		b.addBuildLog(agentID, "ERROR: main.go not found in source directory")
		return "", fmt.Errorf("main.go not found in source directory")
	}

	// Initialize go module in the source directory
	b.updateBuildStatus(agentID, "building", "Initializing Go module", 60)
	if err := b.initGoModule(sourceDir, agentID); err != nil {
		b.addBuildLog(agentID, fmt.Sprintf("ERROR: Failed to initialize go module: %v", err))
		return "", fmt.Errorf("failed to initialize go module: %v", err)
	}

	// Configure build based on target
	switch buildTarget {
	case "wasm":
		pluginFilename = fmt.Sprintf("agent_%s_1.0.wasm", agentID)
		outputPath = filepath.Join(b.outputPath, pluginFilename)

		b.updateBuildStatus(agentID, "building", "Compiling WebAssembly", 70)
		b.addBuildLog(agentID, "Running WASM build command")

		cmd = exec.Command("go", "build", "-o", outputPath, ".")
		cmd.Dir = sourceDir
		cmd.Env = append(os.Environ(),
			"GOOS=wasip1",
			"GOARCH=wasm",
			"CGO_ENABLED=0", // Disable CGO for WASM builds
		)

	default: // "plugin"
		pluginFilename = fmt.Sprintf("agent_%s_1.0.so", agentID)
		outputPath = filepath.Join(b.outputPath, pluginFilename)

		b.updateBuildStatus(agentID, "building", "Compiling Go plugin", 70)
		b.addBuildLog(agentID, "Running plugin build command")

		cmd = exec.Command("go", "build", "-buildmode=plugin", "-o", outputPath, ".")
		cmd.Dir = sourceDir
		cmd.Env = append(os.Environ(), "CGO_ENABLED=1") // Enable CGO for plugin builds
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		errorMsg := fmt.Sprintf("Compilation failed: %v\nOutput: %s", err, string(output))
		b.addBuildLog(agentID, errorMsg)
		return "", fmt.Errorf("failed to compile %s: %v\nOutput: %s", buildTarget, err, string(output))
	}

	// Log successful compilation output
	if len(output) > 0 {
		b.addBuildLog(agentID, fmt.Sprintf("Compilation output: %s", string(output)))
	}

	// Verify the output was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		b.addBuildLog(agentID, fmt.Sprintf("ERROR: Output file was not created at %s", outputPath))
		return "", fmt.Errorf("output file was not created: %s", outputPath)
	}

	// Validate the compiled output (skip validation for WASM for now)
	if buildTarget != "wasm" {
		if err := b.validateCompiledPlugin(outputPath); err != nil {
			b.addBuildLog(agentID, fmt.Sprintf("ERROR: Plugin validation failed: %v", err))
			return "", fmt.Errorf("plugin validation failed: %v", err)
		}
	}

	// Create metadata file with creation time
	if err := b.createPluginMetadata(agentID, outputPath); err != nil {
		b.addBuildLog(agentID, fmt.Sprintf("WARNING: Failed to create metadata: %v", err))
		// Don't fail the build for metadata creation failure
	}

	b.addBuildLog(agentID, fmt.Sprintf("%s compiled successfully: %s", strings.Title(buildTarget), outputPath))
	return outputPath, nil
}

// ensureMetaDirectoryExists creates the meta files directory if it doesn't exist
func (b *AgentBuilder) ensureMetaDirectoryExists() error {
	metaDir := filepath.Join(b.outputPath, MetaFilesSubDirectory)
	return os.MkdirAll(metaDir, 0755)
}

// createPluginMetadata creates a metadata file for the plugin with creation time and other info
func (b *AgentBuilder) createPluginMetadata(agentID, pluginPath string) error {
	// Get plugin file info
	pluginInfo, err := os.Stat(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to stat plugin file: %v", err)
	}

	// Get agent config for additional metadata
	config, err := b.GetAgent(agentID)
	if err != nil {
		return fmt.Errorf("failed to get agent config: %v", err)
	}

	// Create metadata structure with all configuration settings
	metadata := map[string]interface{}{
		"agent_id":        agentID,
		"agent_name":      config.Name,
		"version":         "1.0", // Default version
		"created_at":      time.Now().Format(time.RFC3339),
		"file_size":       pluginInfo.Size(),
		"file_mode":       pluginInfo.Mode().String(),
		"builder_version": "1.0",
		"build_timestamp": time.Now().Unix(),
		"status":          "idle", // Ensure new agents start with idle status
	}

	// Add all configuration settings from ExtraParams
	if config.ExtraParams != nil {
		if capabilities, ok := config.ExtraParams["capabilities"]; ok {
			metadata["capabilities"] = capabilities
		}
		if targetTypes, ok := config.ExtraParams["target_types"]; ok {
			metadata["target_types"] = targetTypes
		}
		if apiKeys, ok := config.ExtraParams["api_keys"]; ok {
			metadata["api_keys"] = apiKeys
		}
		if version, ok := config.ExtraParams["version"].(string); ok && version != "" {
			metadata["version"] = version
		}
	}

	// Ensure the meta directory exists
	if err := b.ensureMetaDirectoryExists(); err != nil {
		return fmt.Errorf("failed to create meta directory: %v", err)
	}

	// Create metadata file path in the new location
	metaDir := filepath.Join(b.outputPath, MetaFilesSubDirectory)
	metadataPath := filepath.Join(metaDir, filepath.Base(pluginPath)+".meta")

	// Marshal metadata to JSON
	metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %v", err)
	}

	// Write metadata file
	if err := os.WriteFile(metadataPath, metadataJSON, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %v", err)
	}

	return nil
}

// GetPluginMetadata reads metadata for a plugin
func (b *AgentBuilder) GetPluginMetadata(agentID string) (map[string]interface{}, error) {
	pluginPath, err := b.GetPluginPath(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin path: %v", err)
	}

	// Try new location first (in /plugins/data subdirectory)
	metaDir := filepath.Join(b.outputPath, MetaFilesSubDirectory)
	newMetadataPath := filepath.Join(metaDir, filepath.Base(pluginPath)+".meta")

	metadataData, err := os.ReadFile(newMetadataPath)
	if err != nil {
		// Fallback to old location for backward compatibility
		oldMetadataPath := pluginPath + ".meta"
		metadataData, err = os.ReadFile(oldMetadataPath)
		if err != nil {
			// Return basic metadata from file info if metadata file doesn't exist
			if info, err := os.Stat(pluginPath); err == nil {
				return map[string]interface{}{
					"agent_id":   agentID,
					"created_at": info.ModTime().Format(time.RFC3339),
					"file_size":  info.Size(),
					"version":    "1.0",
					"source":     "file_stat",
				}, nil
			}
			return nil, fmt.Errorf("failed to read metadata file and plugin file: %v", err)
		}
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(metadataData, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %v", err)
	}

	metadata["source"] = "metadata_file"
	return metadata, nil
}

// initGoModule initializes a go module in the source directory
func (b *AgentBuilder) initGoModule(sourceDir, agentID string) error {
	// Check if go.mod already exists
	goModPath := filepath.Join(sourceDir, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		// go.mod already exists, just run go mod tidy
		cmd := exec.Command("go", "mod", "tidy")
		cmd.Dir = sourceDir
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to run go mod tidy: %v\nOutput: %s", err, string(output))
		}
		return nil
	}

	// Initialize new module
	moduleName := fmt.Sprintf("agent_%s", agentID)
	cmd := exec.Command("go", "mod", "init", moduleName)
	cmd.Dir = sourceDir

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to initialize go module: %v\nOutput: %s", err, string(output))
	}

	// Run go mod tidy to download dependencies
	cmd = exec.Command("go", "mod", "tidy")
	cmd.Dir = sourceDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to download dependencies: %v\nOutput: %s", err, string(output))
	}

	return nil
}

// RebuildAgent rebuilds an existing agent plugin
func (b *AgentBuilder) RebuildAgent(agentID string) error {
	// Get the agent configuration
	config, err := b.GetAgent(agentID)
	if err != nil {
		return fmt.Errorf("failed to get agent configuration: %v", err)
	}

	// Build the plugin from templates
	pluginPath, err := b.buildPluginFromTemplates(config)
	if err != nil {
		return fmt.Errorf("failed to rebuild plugin: %v", err)
	}

	// Update the plugin path in the registry
	configMap := map[string]interface{}{
		"agent_id":            config.AgentID,
		"agent_type":          config.AgentType,
		"name":                config.Name,
		"model":               config.Model,
		"instruction":         config.Instruction,
		"description":         config.Description,
		"use_search":          config.UseSearch,
		"use_code_execution":  config.UseCodeExecution,
		"use_vertex_search":   config.UseVertexSearch,
		"vertex_datastore_id": config.VertexDatastoreID,
		"custom_tools":        config.CustomTools,
		"sub_agents":          config.SubAgents,
		"max_iterations":      config.MaxIterations,
		"plugin_path":         pluginPath,
	}

	// Add extra parameters if provided
	if config.ExtraParams != nil {
		for k, v := range config.ExtraParams {
			configMap[k] = v
		}
	}

	// Update the agent configuration in the registry
	if err := b.registry.RegisterAgent(config.AgentID, configMap); err != nil {
		return fmt.Errorf("failed to update agent registry: %v", err)
	}

	return nil
}

// GetRegistry returns the agent registry
func (b *AgentBuilder) GetRegistry() *AgentRegistry {
	return b.registry
}

// GetPluginPath returns the path to the compiled plugin for an agent
func (b *AgentBuilder) GetPluginPath(agentID string) (string, error) {
	configMap, err := b.registry.GetAgent(agentID)
	if err != nil {
		return "", fmt.Errorf("failed to get agent: %v", err)
	}

	if pluginPath, ok := configMap["plugin_path"].(string); ok {
		return pluginPath, nil
	}

	return "", fmt.Errorf("plugin path not found for agent %s", agentID)
}

// DiscoverTemplates scans the templates directory and returns available templates
func (b *AgentBuilder) DiscoverTemplates() ([]string, error) {
	templateFiles, err := filepath.Glob(filepath.Join(b.templatesPath, "*.template"))
	if err != nil {
		return nil, fmt.Errorf("failed to discover templates: %v", err)
	}

	var templates []string
	for _, templateFile := range templateFiles {
		templateName := filepath.Base(templateFile)
		templates = append(templates, templateName)
	}

	return templates, nil
}

// ValidateTemplate validates a template file for syntax and structure
func (b *AgentBuilder) ValidateTemplate(templatePath string) error {
	// Check if template file exists
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return fmt.Errorf("template file does not exist: %s", templatePath)
	}

	// Read template content
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template file: %v", err)
	}

	// Parse template to check for syntax errors
	templateName := filepath.Base(templatePath)
	_, err = template.New(templateName).Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("template syntax error: %v", err)
	}

	// Additional validation based on template type
	if err := b.validateTemplateContent(templateName, string(templateContent)); err != nil {
		return fmt.Errorf("template content validation failed: %v", err)
	}

	return nil
}

// validateTemplateContent performs content-specific validation
func (b *AgentBuilder) validateTemplateContent(templateName, content string) error {
	switch templateName {
	case "main.go.template":
		return b.validateGoTemplate(content)
	case "go.mod.template":
		return b.validateGoModTemplate(content)
	case "config.env.template":
		return b.validateConfigTemplate(content)
	default:
		// Basic validation for other templates
		return b.validateBasicTemplate(content)
	}
}

// validateGoTemplate validates Go template content
func (b *AgentBuilder) validateGoTemplate(content string) error {
	// Check for required Go package declaration
	if !strings.Contains(content, "package main") {
		return fmt.Errorf("go template must contain 'package main' declaration")
	}

	// Check for required AgentPlugin struct
	if !strings.Contains(content, "type AgentPlugin struct") {
		return fmt.Errorf("go template must contain 'AgentPlugin' struct definition")
	}

	// Check for required interface methods
	requiredMethods := []string{
		"func (p *AgentPlugin) Initialize",
		"func (p *AgentPlugin) ProcessInference",
		"func (p *AgentPlugin) CallTool",
		"func (p *AgentPlugin) Terminate",
	}

	for _, method := range requiredMethods {
		if !strings.Contains(content, method) {
			return fmt.Errorf("go template missing required method: %s", method)
		}
	}

	return nil
}

// validateGoModTemplate validates go.mod template content
func (b *AgentBuilder) validateGoModTemplate(content string) error {
	// Check for module declaration
	if !strings.Contains(content, "module ") {
		return fmt.Errorf("go.mod template must contain module declaration")
	}

	// Check for Go version
	if !strings.Contains(content, "go ") {
		return fmt.Errorf("go.mod template must specify Go version")
	}

	return nil
}

// validateConfigTemplate validates configuration template content
func (b *AgentBuilder) validateConfigTemplate(content string) error {
	// Basic validation for environment variables format
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for KEY=VALUE format
		if !strings.Contains(line, "=") {
			return fmt.Errorf("invalid config format at line %d: %s", i+1, line)
		}
	}

	return nil
}

// validateBasicTemplate performs basic template validation
func (b *AgentBuilder) validateBasicTemplate(content string) error {
	// Check for template variables that might be malformed
	// Look for unmatched braces
	openBraces := strings.Count(content, "{{")
	closeBraces := strings.Count(content, "}}")

	if openBraces != closeBraces {
		return fmt.Errorf("unmatched template braces: %d open, %d close", openBraces, closeBraces)
	}

	return nil
}

// GetBuildStatus returns the build status for an agent
func (b *AgentBuilder) GetBuildStatus(agentID string) (*BuildStatus, error) {
	b.statusMutex.RLock()
	defer b.statusMutex.RUnlock()
	
	if status, exists := b.buildStatuses[agentID]; exists {
		return status, nil
	}
	return nil, fmt.Errorf("build status not found for agent %s", agentID)
}

// updateBuildStatus updates the build status for an agent
func (b *AgentBuilder) updateBuildStatus(agentID, status, message string, progress int) {
	b.statusMutex.Lock()
	defer b.statusMutex.Unlock()
	
	if buildStatus, exists := b.buildStatuses[agentID]; exists {
		buildStatus.Status = status
		buildStatus.Message = message
		buildStatus.Progress = progress
		if status == "completed" || status == "failed" {
			buildStatus.CompletedAt = time.Now()
		}
		// Log the status update
		fmt.Printf("Agent %s build status: %s (%d%%) - %s\n", agentID, status, progress, message)
	}
}

// addBuildLog adds a log entry to the build status
func (b *AgentBuilder) addBuildLog(agentID, logEntry string) {
	b.statusMutex.Lock()
	defer b.statusMutex.Unlock()
	
	if buildStatus, exists := b.buildStatuses[agentID]; exists {
		buildStatus.LogOutput = append(buildStatus.LogOutput, logEntry)
		fmt.Printf("Agent %s build log: %s\n", agentID, logEntry)
	}
}

// ValidateAgentConfig validates an agent configuration before building
func (b *AgentBuilder) ValidateAgentConfig(config AgentConfig) []TemplateValidationError {
	var errors []TemplateValidationError

	// Validate required fields
	if config.Name == "" {
		errors = append(errors, TemplateValidationError{
			TemplateName: "config",
			Error:        "agent name is required",
		})
	}

	if config.AgentType == "" {
		errors = append(errors, TemplateValidationError{
			TemplateName: "config",
			Error:        "agent type is required",
		})
	}

	// Validate agent name format (alphanumeric and underscores only)
	if config.Name != "" {
		validName := true
		for _, char := range config.Name {
			if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
				(char >= '0' && char <= '9') || char == '_') {
				validName = false
				break
			}
		}
		if !validName {
			errors = append(errors, TemplateValidationError{
				TemplateName: "config",
				Error:        "agent name must contain only alphanumeric characters and underscores",
			})
		}
	}

	// Validate model if specified
	if config.Model != "" {
		validModels := []string{
			"llama-4-scout-17b-16e-instruct",
			"gpt-4",
			"claude-3-sonnet",
			"gemini-1.5-flash-latest",
			"deepseek-chat",
			"gemini-1.5-pro-latest",
		}

		modelValid := false
		for _, validModel := range validModels {
			if config.Model == validModel {
				modelValid = true
				break
			}
		}

		if !modelValid {
			errors = append(errors, TemplateValidationError{
				TemplateName: "config",
				Error:        fmt.Sprintf("unsupported model: %s", config.Model),
			})
		}
	}

	// Validate custom tools if provided
	for i, tool := range config.CustomTools {
		if tool.Name == "" {
			errors = append(errors, TemplateValidationError{
				TemplateName: "config",
				Error:        fmt.Sprintf("custom tool %d: name is required", i),
			})
		}
		if tool.Endpoint == "" {
			errors = append(errors, TemplateValidationError{
				TemplateName: "config",
				Error:        fmt.Sprintf("custom tool %d: endpoint is required", i),
			})
		}
	}

	return errors
}

// validateAllTemplates validates all templates in the templates directory
func (b *AgentBuilder) validateAllTemplates() error {
	templates, err := b.DiscoverTemplates()
	if err != nil {
		return fmt.Errorf("failed to discover templates: %v", err)
	}

	var validationErrors []string
	for _, templateName := range templates {
		templatePath := filepath.Join(b.templatesPath, templateName)
		if err := b.ValidateTemplate(templatePath); err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("%s: %v", templateName, err))
		}
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf("template validation errors: %s", strings.Join(validationErrors, "; "))
	}

	return nil
}

// validateCompiledPlugin validates a compiled plugin file
func (b *AgentBuilder) validateCompiledPlugin(pluginPath string) error {
	// Check if plugin file exists and is readable
	fileInfo, err := os.Stat(pluginPath)
	if err != nil {
		return fmt.Errorf("plugin file not accessible: %v", err)
	}

	// Check file size (should be > 0)
	if fileInfo.Size() == 0 {
		return fmt.Errorf("plugin file is empty")
	}

	// Check file permissions
	if fileInfo.Mode().Perm()&0444 == 0 {
		return fmt.Errorf("plugin file is not readable")
	}

	// Try to load the plugin to verify it's valid (basic check)
	// Note: This is a basic validation - in production you might want more thorough checks
	file, err := os.Open(pluginPath)
	if err != nil {
		return fmt.Errorf("cannot open plugin file: %v", err)
	}
	defer file.Close()

	// Read first few bytes to check if it's a valid binary
	header := make([]byte, 4)
	_, err = file.Read(header)
	if err != nil {
		return fmt.Errorf("cannot read plugin header: %v", err)
	}

	// Check for ELF magic number (Linux shared library)
	if len(header) >= 4 && header[0] == 0x7f && header[1] == 'E' && header[2] == 'L' && header[3] == 'F' {
		return nil // Valid ELF file
	}

	return fmt.Errorf("plugin file does not appear to be a valid shared library")
}
