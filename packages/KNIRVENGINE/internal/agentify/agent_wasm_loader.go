// agent_wasm_loader.go
package agentify

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// WASMAgentInterface defines the interface for WASM-based agents
type WASMAgentInterface interface {
	// Initialize the agent with configuration
	Initialize(config map[string]interface{}) error

	// Process an inference request and return a response
	ProcessInference(ctx context.Context, request *InferenceRequest) (*InferenceResponse, error)

	// Get the agent's capabilities
	GetCapabilities() *AgentCapabilities

	// Get the agent's schema (tools, resources, prompts)
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
	CallTool(ctx context.Context, toolName string, input map[string]interface{}) (string, error)

	// Lifecycle management
	Start() error
	Stop() error
}

// WASMAgent represents a WebAssembly-based agent
type WASMAgent struct {
	agentID         string
	wasmPath        string
	wasmBytes       []byte
	config          map[string]interface{}
	runtime         *WASMRuntime
	initialized     bool
	running         bool
	mutex           sync.RWMutex
	terminalManager *TerminalManager
}

// WASMAgentAdapter adapts a WASM agent to work with the TerminalManager
type WASMAgentAdapter struct {
	wasmAgent *WASMAgent
}

// Implement the minimal AgentPluginInterface methods needed for terminal management
func (w *WASMAgentAdapter) GetAgentID() string {
	return w.wasmAgent.agentID
}

func (w *WASMAgentAdapter) GetConfig() map[string]interface{} {
	return w.wasmAgent.config
}

func (w *WASMAgentAdapter) IsInitialized() bool {
	return w.wasmAgent.initialized
}

func (w *WASMAgentAdapter) IsRunning() bool {
	return w.wasmAgent.running
}

// Implement all required AgentPluginInterface methods
func (w *WASMAgentAdapter) Initialize(config map[string]interface{}) error {
	return w.wasmAgent.Initialize(config)
}

func (w *WASMAgentAdapter) Start() error {
	return w.wasmAgent.Start()
}

func (w *WASMAgentAdapter) Stop() error {
	return w.wasmAgent.Stop()
}

func (w *WASMAgentAdapter) ProcessInference(ctx context.Context, request *InferenceRequest) (*InferenceResponse, error) {
	return w.wasmAgent.ProcessInference(ctx, request)
}

func (w *WASMAgentAdapter) GetCapabilities() *AgentCapabilities {
	return w.wasmAgent.GetCapabilities()
}

func (w *WASMAgentAdapter) GetSchema() *AgentSchema {
	return w.wasmAgent.GetSchema()
}

func (w *WASMAgentAdapter) GetTEEInfo() map[string]interface{} {
	return map[string]interface{}{
		"type":    "wasm",
		"runtime": "wazero",
		"agentID": w.wasmAgent.agentID,
	}
}

func (w *WASMAgentAdapter) GetMemory(key string) (interface{}, error) {
	return w.wasmAgent.GetMemory(key)
}

func (w *WASMAgentAdapter) SetMemory(key string, value interface{}) error {
	return w.wasmAgent.SetMemory(key, value)
}

func (w *WASMAgentAdapter) CallTool(ctx context.Context, name string, params map[string]interface{}) (interface{}, error) {
	result, err := w.wasmAgent.CallTool(ctx, name, params)
	return result, err
}

// Terminal management methods
func (w *WASMAgentAdapter) CreateTerminal(rows, cols int) (string, error) {
	return w.wasmAgent.CreateTerminal(rows, cols)
}

func (w *WASMAgentAdapter) ResizeTerminal(terminalID string, rows, cols int) error {
	return w.wasmAgent.ResizeTerminal(terminalID, rows, cols)
}

func (w *WASMAgentAdapter) WriteToTerminal(terminalID string, data []byte) error {
	return w.wasmAgent.WriteToTerminal(terminalID, data)
}

func (w *WASMAgentAdapter) ReadFromTerminal(terminalID string) ([]byte, error) {
	return w.wasmAgent.ReadFromTerminal(terminalID)
}

func (w *WASMAgentAdapter) CloseTerminal(terminalID string) error {
	return w.wasmAgent.CloseTerminal(terminalID)
}

// Legacy memory management methods for backward compatibility
func (w *WASMAgentAdapter) StoreContext(contextID string, context map[string]interface{}) error {
	// For WASM agents, we'll store this in the agent's memory
	return w.wasmAgent.SetMemory("context_"+contextID, context)
}

func (w *WASMAgentAdapter) GetContext(contextID string) (map[string]interface{}, error) {
	result, err := w.wasmAgent.GetMemory("context_" + contextID)
	if err != nil {
		return nil, err
	}
	if context, ok := result.(map[string]interface{}); ok {
		return context, nil
	}
	return nil, fmt.Errorf("context not found or invalid type")
}

func (w *WASMAgentAdapter) TransferContext(contextID string, targetAgentID string) error {
	// For WASM agents, this is a no-op since context transfer is complex
	return fmt.Errorf("context transfer not implemented for WASM agents")
}

func (w *WASMAgentAdapter) StoreCredential(credentialID string, credential map[string]interface{}) error {
	return w.wasmAgent.SetMemory("credential_"+credentialID, credential)
}

func (w *WASMAgentAdapter) GetCredential(credentialID string) (map[string]interface{}, error) {
	result, err := w.wasmAgent.GetMemory("credential_" + credentialID)
	if err != nil {
		return nil, err
	}
	if credential, ok := result.(map[string]interface{}); ok {
		return credential, nil
	}
	return nil, fmt.Errorf("credential not found or invalid type")
}

func (w *WASMAgentAdapter) StoreRAGResult(queryHash string, result map[string]interface{}, ttl int64) error {
	return w.wasmAgent.SetMemory("rag_"+queryHash, result)
}

func (w *WASMAgentAdapter) GetRAGResult(queryHash string) (map[string]interface{}, error) {
	result, err := w.wasmAgent.GetMemory("rag_" + queryHash)
	if err != nil {
		return nil, err
	}
	if ragResult, ok := result.(map[string]interface{}); ok {
		return ragResult, nil
	}
	return nil, fmt.Errorf("RAG result not found or invalid type")
}

func (w *WASMAgentAdapter) StoreCOTPlan(planID string, plan map[string]interface{}) error {
	return w.wasmAgent.SetMemory("cot_"+planID, plan)
}

func (w *WASMAgentAdapter) GetCOTPlan(planID string) (map[string]interface{}, error) {
	result, err := w.wasmAgent.GetMemory("cot_" + planID)
	if err != nil {
		return nil, err
	}
	if plan, ok := result.(map[string]interface{}); ok {
		return plan, nil
	}
	return nil, fmt.Errorf("COT plan not found or invalid type")
}

func (w *WASMAgentAdapter) StoreUserPreference(userID string, preference map[string]interface{}) error {
	return w.wasmAgent.SetMemory("user_pref_"+userID, preference)
}

func (w *WASMAgentAdapter) GetUserPreferences(userID string) (map[string]interface{}, error) {
	result, err := w.wasmAgent.GetMemory("user_pref_" + userID)
	if err != nil {
		return nil, err
	}
	if preferences, ok := result.(map[string]interface{}); ok {
		return preferences, nil
	}
	return nil, fmt.Errorf("user preferences not found or invalid type")
}

func (w *WASMAgentAdapter) GetUserPreference(userID string, key string) (interface{}, error) {
	preferences, err := w.GetUserPreferences(userID)
	if err != nil {
		return nil, err
	}
	if value, ok := preferences[key]; ok {
		return value, nil
	}
	return nil, fmt.Errorf("user preference key not found: %s", key)
}

// WASMRuntime manages the WebAssembly runtime environment using wazero
type WASMRuntime struct {
	ctx         context.Context
	runtime     wazero.Runtime
	module      api.Module
	initialized bool
	exports     map[string]api.Function
}

// AgentWASMLoader manages loading and execution of WASM-based agents
type AgentWASMLoader struct {
	wasmDir      string
	loadedAgents map[string]*WASMAgent
	mutex        sync.RWMutex
}

// NewAgentWASMLoader creates a new WASM agent loader
func NewAgentWASMLoader(wasmDir string) *AgentWASMLoader {
	return &AgentWASMLoader{
		wasmDir:      wasmDir,
		loadedAgents: make(map[string]*WASMAgent),
	}
}

// LoadWASMAgent loads a WASM agent by ID and version
func (l *AgentWASMLoader) LoadWASMAgent(agentID string, version string, config ...map[string]interface{}) (WASMAgentInterface, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// Check if the agent is already loaded
	agentKey := fmt.Sprintf("%s_%s", agentID, version)
	if agent, ok := l.loadedAgents[agentKey]; ok {
		return agent, nil
	}

	// Try to find WASM file in multiple locations
	var wasmPath string
	var wasmBytes []byte
	var err error
	var possibleVersions []string

	// Handle different version formats (1.0, 1.0.0, etc.)
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
	// Declare pluginDir at function scope
	var pluginDir string

	for _, ver := range possibleVersions {
		versionSuffix := ver
		if versionSuffix != "" {
			versionSuffix = "_" + versionSuffix
		}

		// First, try the new extracted plugin directory structure
		pluginDir = filepath.Join(l.wasmDir, fmt.Sprintf("%s%s", agentID, versionSuffix))
		if _, err := os.Stat(pluginDir); err == nil {
			// Look for WASM file in the plugin directory
			entries, err := os.ReadDir(pluginDir)
			if err == nil {
				for _, entry := range entries {
					if strings.HasSuffix(entry.Name(), ".wasm") {
						wasmPath = filepath.Join(pluginDir, entry.Name())
						break
					}
				}
			}
		}

		// If not found in plugin directory, try the old single-file format
		if wasmPath == "" {
			possiblePath := filepath.Join(l.wasmDir, fmt.Sprintf("agent_%s%s.wasm", agentID, versionSuffix))
			if _, err := os.Stat(possiblePath); err == nil {
				wasmPath = possiblePath
				break
			}
		} else {
			break
		}
	}

	// Check if the WASM file exists
	if wasmPath == "" || os.IsNotExist(err) {
		return nil, fmt.Errorf("WASM file not found: %s", filepath.Join(l.wasmDir, fmt.Sprintf("agent_%s_%s.wasm", agentID, version)))
	}

	// Load the WASM file
	wasmBytes, err = os.ReadFile(wasmPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read WASM file: %v", err)
	}

	// Create the WASM agent
	agent := &WASMAgent{
		agentID:   agentID,
		wasmPath:  wasmPath,
		wasmBytes: wasmBytes,
		runtime:   &WASMRuntime{},
	}

	// Initialize the agent with configuration
	agentConfig := map[string]interface{}{
		"agentID": agentID,
		"version": version,
	}

	// Try to load configuration from config.json if it exists in plugin directory
	configPath := filepath.Join(pluginDir, "config.json")
	if _, statErr := os.Stat(configPath); statErr == nil {
		configData, readErr := os.ReadFile(configPath)
		if readErr == nil {
			var pluginConfig map[string]interface{}
			if json.Unmarshal(configData, &pluginConfig) == nil {
				// Merge plugin config into agent config
				for k, v := range pluginConfig {
					agentConfig[k] = v
				}
			}
		}
	}

	// If a custom config was provided, use it (overrides plugin config)
	if len(config) > 0 && config[0] != nil {
		for k, v := range config[0] {
			agentConfig[k] = v
		}
	}

	// Initialize the agent
	if err := agent.Initialize(agentConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize WASM agent: %v", err)
	}

	// Store the loaded agent
	l.loadedAgents[agentKey] = agent

	return agent, nil
}

// UnloadWASMAgent unloads a WASM agent
func (l *AgentWASMLoader) UnloadWASMAgent(agentID string, version string) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	agentKey := fmt.Sprintf("%s_%s", agentID, version)
	if agent, ok := l.loadedAgents[agentKey]; ok {
		if err := agent.Stop(); err != nil {
			return fmt.Errorf("failed to stop WASM agent: %v", err)
		}
		delete(l.loadedAgents, agentKey)
	}

	return nil
}

// DiscoverWASMAgents scans the WASM directory and returns a list of available agents
func (l *AgentWASMLoader) DiscoverWASMAgents() ([]string, error) {
	var agents []string
	agentSet := make(map[string]bool) // To avoid duplicates

	// First, scan for old-style single WASM files
	pattern := filepath.Join(l.wasmDir, "agent_*.wasm")
	matches, err := filepath.Glob(pattern)
	if err == nil {
		for _, match := range matches {
			filename := filepath.Base(match)
			// Extract agent ID and version from filename
			// Expected format: agent_{agentID}_{version}.wasm
			if strings.HasPrefix(filename, "agent_") && strings.HasSuffix(filename, ".wasm") {
				// Remove "agent_" prefix and ".wasm" suffix
				nameWithoutExt := strings.TrimSuffix(strings.TrimPrefix(filename, "agent_"), ".wasm")
				if !agentSet[nameWithoutExt] {
					agents = append(agents, nameWithoutExt)
					agentSet[nameWithoutExt] = true
				}
			}
		}
	}

	// Second, scan for extracted plugin directories
	entries, err := os.ReadDir(l.wasmDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			// Check if directory contains a WASM file
			pluginDir := filepath.Join(l.wasmDir, entry.Name())
			pluginEntries, err := os.ReadDir(pluginDir)
			if err != nil {
				continue
			}

			hasWASM := false
			for _, pluginEntry := range pluginEntries {
				if strings.HasSuffix(pluginEntry.Name(), ".wasm") {
					hasWASM = true
					break
				}
			}

			if hasWASM {
				// Directory name format: {agentID}_{version}
				agentKey := entry.Name()
				if !agentSet[agentKey] {
					agents = append(agents, agentKey)
					agentSet[agentKey] = true
				}
			}
		}
	}

	return agents, nil
}

// WASM Agent Implementation

// Initialize initializes the WASM agent with configuration
func (w *WASMAgent) Initialize(config map[string]interface{}) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.config = config

	// Initialize the terminal manager
	w.terminalManager = NewTerminalManager()

	// Initialize the WASM runtime
	if err := w.initializeWASMRuntime(); err != nil {
		return fmt.Errorf("failed to initialize WASM runtime: %v", err)
	}

	// Try to call the WASM agent's initialize function
	// If it doesn't exist, that's okay - the main function should have initialized it
	_, err := w.callWASMFunction("agentInitialize", config)
	if err != nil {
		// Log the error but don't fail - WASM agent might be self-initializing
		fmt.Printf("WASM agent initialize function not found or failed: %v\n", err)
	}

	w.initialized = true
	return nil
}

// ProcessInference processes an inference request through the WASM agent
func (w *WASMAgent) ProcessInference(ctx context.Context, request *InferenceRequest) (*InferenceResponse, error) {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	if !w.initialized {
		return nil, fmt.Errorf("WASM agent not initialized")
	}

	// Try to call the WASM agent's process inference function
	result, err := w.callWASMFunction("agentProcessInference", request)
	if err != nil {
		// If the function doesn't exist, return a default response
		fmt.Printf("WASM agent process inference function not found: %v\n", err)
		return &InferenceResponse{
			Output:    "WASM agent response (fallback)",
			Reasoning: "WASM agent processed request using fallback method",
			Metadata: map[string]interface{}{
				"agent_type": "wasm",
				"fallback":   true,
			},
		}, nil
	}

	// Parse the result as an InferenceResponse
	var response InferenceResponse
	if resultStr, ok := result.(string); ok {
		if err := json.Unmarshal([]byte(resultStr), &response); err != nil {
			return nil, fmt.Errorf("failed to parse WASM agent response: %v", err)
		}
	} else {
		return nil, fmt.Errorf("unexpected response type from WASM agent")
	}

	return &response, nil
}

// GetCapabilities returns the agent's capabilities
func (w *WASMAgent) GetCapabilities() *AgentCapabilities {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	if !w.initialized {
		return &AgentCapabilities{}
	}

	// Call the WASM agent's get capabilities function
	result, err := w.callWASMFunction("agentGetCapabilities", nil)
	if err != nil {
		return &AgentCapabilities{}
	}

	// Parse the result as AgentCapabilities
	var capabilities AgentCapabilities
	if resultStr, ok := result.(string); ok {
		json.Unmarshal([]byte(resultStr), &capabilities)
	}

	return &capabilities
}

// GetSchema returns the agent's schema
func (w *WASMAgent) GetSchema() *AgentSchema {
	// For now, return a basic schema
	// This would be implemented by calling the WASM agent
	return &AgentSchema{
		Tools:     []*ToolSchema{},
		Resources: []*ResourceSchema{},
		Prompts:   []*PromptSchema{},
	}
}

// Memory management methods
func (w *WASMAgent) GetMemory(key string) (interface{}, error) {
	// This would call the WASM agent's memory functions
	return nil, fmt.Errorf("memory management not yet implemented for WASM agents")
}

func (w *WASMAgent) SetMemory(key string, value interface{}) error {
	// This would call the WASM agent's memory functions
	return fmt.Errorf("memory management not yet implemented for WASM agents")
}

// Terminal management methods
func (w *WASMAgent) CreateTerminal(rows, cols int) (string, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.terminalManager == nil {
		return "", fmt.Errorf("terminal manager not initialized")
	}

	// Create a real terminal session using the terminal manager
	adapter := &WASMAgentAdapter{wasmAgent: w}
	session, err := w.terminalManager.NewTerminalSession(adapter, rows, cols)
	if err != nil {
		return "", fmt.Errorf("failed to create terminal session: %v", err)
	}

	// Also call the WASM function to notify the agent (optional)
	_, wasmErr := w.callWASMFunction("agentCreateTerminal", map[string]interface{}{
		"rows": rows,
		"cols": cols,
	})
	if wasmErr != nil {
		// Log the error but don't fail - the real terminal is what matters
		fmt.Printf("WASM agent terminal notification failed: %v\n", wasmErr)
	}

	return session.ID, nil
}

func (w *WASMAgent) ResizeTerminal(terminalID string, rows, cols int) error {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	if w.terminalManager == nil {
		return fmt.Errorf("terminal manager not initialized")
	}

	return w.terminalManager.ResizeTerminal(terminalID, rows, cols)
}

func (w *WASMAgent) WriteToTerminal(terminalID string, data []byte) error {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	if w.terminalManager == nil {
		return fmt.Errorf("terminal manager not initialized")
	}

	return w.terminalManager.WriteToTerminal(terminalID, data)
}

func (w *WASMAgent) ReadFromTerminal(terminalID string) ([]byte, error) {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	if w.terminalManager == nil {
		return nil, fmt.Errorf("terminal manager not initialized")
	}

	return w.terminalManager.ReadFromTerminal(terminalID)
}

func (w *WASMAgent) CloseTerminal(terminalID string) error {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	if w.terminalManager == nil {
		return fmt.Errorf("terminal manager not initialized")
	}

	return w.terminalManager.CloseTerminalSession(terminalID)
}

// Tool execution
func (w *WASMAgent) CallTool(ctx context.Context, toolName string, input map[string]interface{}) (string, error) {
	result, err := w.callWASMFunction("agentCallTool", map[string]interface{}{
		"toolName": toolName,
		"input":    input,
	})
	if err != nil {
		return "", err
	}

	if resultStr, ok := result.(string); ok {
		return resultStr, nil
	}

	return "", fmt.Errorf("unexpected response type from WASM agent")
}

// Lifecycle management
func (w *WASMAgent) Start() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if !w.initialized {
		return fmt.Errorf("WASM agent not initialized")
	}

	// Try to call the start function, but don't fail if it doesn't exist
	_, err := w.callWASMFunction("agentStart", nil)
	if err != nil {
		// Log the error but don't fail - WASM agent might be self-starting
		fmt.Printf("WASM agent start function not found or failed: %v\n", err)
	}

	w.running = true
	return nil
}

func (w *WASMAgent) Stop() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if !w.running {
		return nil
	}

	_, err := w.callWASMFunction("agentStop", nil)
	if err != nil {
		return err
	}

	w.running = false
	return nil
}

// Helper methods

// initializeWASMRuntime initializes the WebAssembly runtime using wazero with Go support
func (w *WASMAgent) initializeWASMRuntime() error {
	// Create a new wazero runtime
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)

	// Instantiate WASI, which implements host functions needed for WASI programs
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	// For Go WASM modules, we need to provide the "gojs" module
	// This implements the Go JavaScript bridge functions
	if err := w.instantiateGoJSModule(ctx, r); err != nil {
		return fmt.Errorf("failed to instantiate gojs module: %v", err)
	}

	// Compile and instantiate the WASM module
	compiledModule, err := r.CompileModule(ctx, w.wasmBytes)
	if err != nil {
		return fmt.Errorf("failed to compile WASM module: %v", err)
	}

	// Create module config with proper memory settings for Go WASM
	moduleConfig := wazero.NewModuleConfig().
		WithName("main").
		WithStdout(os.Stdout).
		WithStderr(os.Stderr)

	module, err := r.InstantiateModule(ctx, compiledModule, moduleConfig)
	if err != nil {
		return fmt.Errorf("failed to instantiate WASM module: %v", err)
	}

	// Store runtime components
	w.runtime.ctx = ctx
	w.runtime.runtime = r
	w.runtime.module = module
	w.runtime.exports = make(map[string]api.Function)

	// Cache commonly used exported functions
	functionNames := []string{
		"agentInitialize", "agentProcessInference", "agentGetCapabilities",
		"agentCreateTerminal", "agentStart", "agentStop",
		// Go WASM standard functions
		"_start", "run", "resume",
	}

	for _, name := range functionNames {
		if fn := module.ExportedFunction(name); fn != nil {
			w.runtime.exports[name] = fn
		}
	}

	// For Go WASM modules, we need to call the _start function to initialize
	if startFn := module.ExportedFunction("_start"); startFn != nil {
		_, err := startFn.Call(ctx)
		if err != nil {
			// Log but don't fail - some Go WASM modules might not have _start
			fmt.Printf("Warning: Failed to call _start function: %v\n", err)
		}
	}

	w.runtime.initialized = true
	return nil
}

// instantiateGoJSModule creates the "gojs" module required by Go WASM modules
func (w *WASMAgent) instantiateGoJSModule(ctx context.Context, r wazero.Runtime) error {
	// Create the gojs module builder
	gojs := r.NewHostModuleBuilder("gojs")

	// Add essential Go JavaScript bridge functions
	// These functions use stack pointer (sp) parameter as per Go WASM convention
	// All Go WASM runtime functions receive a stack pointer and access memory directly

	// func wasmExit(code int32)
	gojs.NewFunctionBuilder().
		WithName("runtime.wasmExit").
		WithParameterNames("sp").
		WithFunc(func(ctx context.Context, m api.Module, sp int32) {
			// Read exit code from memory at sp + 8
			// For now, just log that the program is exiting
			fmt.Printf("Go WASM program exiting (mock implementation)\n")
		}).
		Export("runtime.wasmExit")

	// func wasmWrite(fd uintptr, p unsafe.Pointer, n int32)
	gojs.NewFunctionBuilder().
		WithName("runtime.wasmWrite").
		WithParameterNames("sp").
		WithFunc(func(ctx context.Context, m api.Module, sp int32) {
			// Read fd, p, n from memory starting at sp + 8
			// For now, just acknowledge the write operation
		}).
		Export("runtime.wasmWrite")

	// func resetMemoryDataView()
	gojs.NewFunctionBuilder().
		WithName("runtime.resetMemoryDataView").
		WithParameterNames("sp").
		WithFunc(func(ctx context.Context, m api.Module, sp int32) {
			// Handle memory reset - for now just acknowledge
		}).
		Export("runtime.resetMemoryDataView")

	// func nanotime1() int64
	gojs.NewFunctionBuilder().
		WithName("runtime.nanotime1").
		WithParameterNames("sp").
		WithFunc(func(ctx context.Context, m api.Module, sp int32) {
			// Write current nanosecond timestamp to memory at sp + 8
			// For now, just acknowledge the call
		}).
		Export("runtime.nanotime1")

	// func walltime() (sec int64, nsec int32)
	gojs.NewFunctionBuilder().
		WithName("runtime.walltime").
		WithParameterNames("sp").
		WithFunc(func(ctx context.Context, m api.Module, sp int32) {
			// Write sec and nsec to memory starting at sp + 8
			// For now, just acknowledge the call
		}).
		Export("runtime.walltime")

	// func scheduleTimeoutEvent(delay int64) int32
	gojs.NewFunctionBuilder().
		WithName("runtime.scheduleTimeoutEvent").
		WithParameterNames("sp").
		WithFunc(func(ctx context.Context, m api.Module, sp int32) {
			// Read delay from memory at sp + 8, write timeout ID to sp + 16
			// For now, just acknowledge the call
		}).
		Export("runtime.scheduleTimeoutEvent")

	// func clearTimeoutEvent(id int32)
	gojs.NewFunctionBuilder().
		WithName("runtime.clearTimeoutEvent").
		WithParameterNames("sp").
		WithFunc(func(ctx context.Context, m api.Module, sp int32) {
			// Read timeout ID from memory at sp + 8
			// For now, just acknowledge the call
		}).
		Export("runtime.clearTimeoutEvent")

	// func getRandomData(r []byte)
	gojs.NewFunctionBuilder().
		WithName("runtime.getRandomData").
		WithParameterNames("sp").
		WithFunc(func(ctx context.Context, m api.Module, sp int32) {
			// Read buffer pointer and length from memory, fill with random data
			// For now, just acknowledge the call
		}).
		Export("runtime.getRandomData")

	// syscall/js functions for JavaScript interop
	gojs.NewFunctionBuilder().
		WithName("syscall/js.finalizeRef").
		WithParameterNames("sp").
		WithFunc(func(ctx context.Context, m api.Module, sp int32) {
			// Finalize JavaScript reference - for now just acknowledge
		}).
		Export("syscall/js.finalizeRef")

	gojs.NewFunctionBuilder().
		WithName("syscall/js.stringVal").
		WithParameterNames("sp").
		WithFunc(func(ctx context.Context, m api.Module, sp int32) {
			// Create JavaScript string value - for now just acknowledge
		}).
		Export("syscall/js.stringVal")

	gojs.NewFunctionBuilder().
		WithName("syscall/js.valueGet").
		WithParameterNames("sp").
		WithFunc(func(ctx context.Context, m api.Module, sp int32) {
			// Get JavaScript object property - for now just acknowledge
		}).
		Export("syscall/js.valueGet")

	gojs.NewFunctionBuilder().
		WithName("syscall/js.valueSet").
		WithParameterNames("sp").
		WithFunc(func(ctx context.Context, m api.Module, sp int32) {
			// Set JavaScript object property - for now just acknowledge
		}).
		Export("syscall/js.valueSet")

	gojs.NewFunctionBuilder().
		WithName("syscall/js.valueCall").
		WithParameterNames("sp").
		WithFunc(func(ctx context.Context, m api.Module, sp int32) {
			// Call JavaScript function - for now just acknowledge
		}).
		Export("syscall/js.valueCall")

	gojs.NewFunctionBuilder().
		WithName("syscall/js.valueNew").
		WithParameterNames("sp").
		WithFunc(func(ctx context.Context, m api.Module, sp int32) {
			// Create new JavaScript object - for now just acknowledge
		}).
		Export("syscall/js.valueNew")

	gojs.NewFunctionBuilder().
		WithName("syscall/js.valueLength").
		WithParameterNames("sp").
		WithFunc(func(ctx context.Context, m api.Module, sp int32) {
			// Get JavaScript array/string length - for now just acknowledge
		}).
		Export("syscall/js.valueLength")

	gojs.NewFunctionBuilder().
		WithName("syscall/js.valuePrepareString").
		WithParameterNames("sp").
		WithFunc(func(ctx context.Context, m api.Module, sp int32) {
			// Prepare string for JavaScript - for now just acknowledge
		}).
		Export("syscall/js.valuePrepareString")

	gojs.NewFunctionBuilder().
		WithName("syscall/js.valueLoadString").
		WithParameterNames("sp").
		WithFunc(func(ctx context.Context, m api.Module, sp int32) {
			// Load string from JavaScript - for now just acknowledge
		}).
		Export("syscall/js.valueLoadString")

	gojs.NewFunctionBuilder().
		WithName("syscall/js.valueIndex").
		WithParameterNames("sp").
		WithFunc(func(ctx context.Context, m api.Module, sp int32) {
			// Get JavaScript array index - for now just acknowledge
		}).
		Export("syscall/js.valueIndex")

	gojs.NewFunctionBuilder().
		WithName("syscall/js.valueSetIndex").
		WithParameterNames("sp").
		WithFunc(func(ctx context.Context, m api.Module, sp int32) {
			// Set JavaScript array index - for now just acknowledge
		}).
		Export("syscall/js.valueSetIndex")

	gojs.NewFunctionBuilder().
		WithName("syscall/js.valueInstanceOf").
		WithParameterNames("sp").
		WithFunc(func(ctx context.Context, m api.Module, sp int32) {
			// Check JavaScript instanceof - for now just acknowledge
		}).
		Export("syscall/js.valueInstanceOf")

	gojs.NewFunctionBuilder().
		WithName("syscall/js.copyBytesToGo").
		WithParameterNames("sp").
		WithFunc(func(ctx context.Context, m api.Module, sp int32) {
			// Copy bytes from JavaScript to Go - for now just acknowledge
		}).
		Export("syscall/js.copyBytesToGo")

	gojs.NewFunctionBuilder().
		WithName("syscall/js.copyBytesToJS").
		WithParameterNames("sp").
		WithFunc(func(ctx context.Context, m api.Module, sp int32) {
			// Copy bytes from Go to JavaScript - for now just acknowledge
		}).
		Export("syscall/js.copyBytesToJS")

	// Instantiate the gojs module
	_, err := gojs.Instantiate(ctx)
	if err != nil {
		return fmt.Errorf("failed to instantiate gojs module: %v", err)
	}

	return nil
}

// callWASMFunction calls a function in the WASM agent
func (w *WASMAgent) callWASMFunction(functionName string, _ interface{}) (interface{}, error) {
	if !w.runtime.initialized {
		return nil, fmt.Errorf("WASM runtime not initialized")
	}

	// For Go WASM modules, we need to handle the runtime differently
	// Go WASM modules typically export _start, run, and resume functions
	// and communicate through memory rather than direct function calls

	// Check if this is a Go WASM module by looking for standard Go exports
	if w.runtime.module.ExportedFunction("run") != nil || w.runtime.module.ExportedFunction("resume") != nil {
		return w.handleGoWASMFunction(functionName)
	}

	// Fallback to direct function calls for non-Go WASM modules
	fn, exists := w.runtime.exports[functionName]
	if !exists {
		return nil, fmt.Errorf("function %s not found in WASM module", functionName)
	}

	// Mock implementation for direct function calls
	switch functionName {
	case "agentInitialize":
		_, err := fn.Call(w.runtime.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to call %s: %v", functionName, err)
		}
		return map[string]interface{}{"success": true}, nil

	case "agentProcessInference":
		return `{"output": "WASM agent response", "reasoning": "Processed through WebAssembly runtime"}`, nil

	case "agentGetCapabilities":
		return `{"supportsStreaming": true, "supportsToolCalls": true, "maxContextLength": 4096}`, nil

	case "agentCreateTerminal":
		return map[string]interface{}{"terminalId": fmt.Sprintf("wasm_terminal_%d", time.Now().UnixNano())}, nil

	case "agentStart", "agentStop":
		_, err := fn.Call(w.runtime.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to call %s: %v", functionName, err)
		}
		return map[string]interface{}{"success": true}, nil

	default:
		return nil, fmt.Errorf("unknown WASM function: %s", functionName)
	}
}

// handleGoWASMFunction handles function calls for Go WASM modules
func (w *WASMAgent) handleGoWASMFunction(functionName string) (interface{}, error) {
	// For Go WASM modules, we provide mock responses since the actual
	// communication would require a full JavaScript bridge implementation

	switch functionName {
	case "agentInitialize":
		// Go WASM module is already initialized via _start
		return map[string]interface{}{"success": true, "type": "go-wasm"}, nil

	case "agentProcessInference":
		return map[string]interface{}{
			"output":    "Go WASM agent response",
			"reasoning": "Processed through Go WebAssembly runtime",
			"metadata": map[string]interface{}{
				"agent_type": "go-wasm",
				"runtime":    "wazero-gojs",
			},
		}, nil

	case "agentGetCapabilities":
		return map[string]interface{}{
			"supportsStreaming": true,
			"supportsToolCalls": true,
			"maxContextLength":  4096,
			"runtime":           "go-wasm",
		}, nil

	case "agentCreateTerminal":
		return map[string]interface{}{
			"terminalId": fmt.Sprintf("go_wasm_terminal_%d", time.Now().UnixNano()),
			"type":       "go-wasm",
		}, nil

	case "agentStart":
		// For Go WASM, we can try to call the run function if it exists
		if runFn := w.runtime.module.ExportedFunction("run"); runFn != nil {
			// Note: In a real implementation, this would be more complex
			// as Go WASM modules expect specific memory layout and event loop
			fmt.Printf("Go WASM agent started (mock implementation)\n")
		}
		return map[string]interface{}{"success": true, "type": "go-wasm"}, nil

	case "agentStop":
		return map[string]interface{}{"success": true, "type": "go-wasm"}, nil

	default:
		return nil, fmt.Errorf("unknown Go WASM function: %s", functionName)
	}
}
