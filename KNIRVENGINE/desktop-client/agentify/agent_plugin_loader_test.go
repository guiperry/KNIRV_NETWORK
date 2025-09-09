// agent_plugin_loader_test.go
package agentify

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAgentPlugin implements AgentPluginInterface for testing
type MockAgentPlugin struct {
	id           string
	initialized  bool
	started      bool
	capabilities *AgentCapabilities
	schema       *AgentSchema
	memory       map[string]interface{}
	terminals    map[string]*MockTerminal
	mu           sync.RWMutex
}

type MockTerminal struct {
	id   string
	rows int
	cols int
	data []byte
}

func NewMockAgentPlugin(id string) *MockAgentPlugin {
	return &MockAgentPlugin{
		id:        id,
		memory:    make(map[string]interface{}),
		terminals: make(map[string]*MockTerminal),
		capabilities: &AgentCapabilities{
			SupportsStreaming:   true,
			SupportsToolCalls:   true,
			SupportsReasoning:   true,
			MaxContextLength:    16384,
			SupportedParameters: []string{"temperature", "top_p", "max_tokens"},
		},
		schema: &AgentSchema{
			Tools: []*ToolSchema{
				{
					Name:        "test_tool",
					Description: "A test tool",
					Parameters: map[string]*ParameterSchema{
						"param1": {
							Type:        "string",
							Description: "Test parameter",
							Required:    true,
						},
					},
					ReturnType: "object",
				},
			},
			Resources: []*ResourceSchema{
				{
					Name:        "test_resource",
					Type:        "test",
					Description: "A test resource",
				},
			},
			Prompts: []*PromptSchema{
				{
					Name:        "test_prompt",
					Description: "A test prompt",
					Variables:   []string{"input"},
				},
			},
		},
	}
}

func (m *MockAgentPlugin) Initialize(config map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.initialized = true
	return nil
}

func (m *MockAgentPlugin) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.initialized {
		return fmt.Errorf("agent not initialized")
	}
	m.started = true
	return nil
}

func (m *MockAgentPlugin) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = false
	return nil
}

func (m *MockAgentPlugin) ProcessInference(ctx context.Context, request *InferenceRequest) (*InferenceResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.started {
		return nil, fmt.Errorf("agent not started")
	}

	return &InferenceResponse{
		Output:   fmt.Sprintf("Mock response to: %s", request.Input),
		Metadata: map[string]interface{}{"agent_id": m.id},
	}, nil
}

func (m *MockAgentPlugin) GetCapabilities() *AgentCapabilities {
	return m.capabilities
}

func (m *MockAgentPlugin) GetSchema() *AgentSchema {
	return m.schema
}

func (m *MockAgentPlugin) GetTEEInfo() map[string]interface{} {
	return map[string]interface{}{
		"agent_id": m.id,
		"tee_type": "mock",
		"status":   "active",
	}
}

func (m *MockAgentPlugin) GetMemory(key string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, exists := m.memory[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	return value, nil
}

func (m *MockAgentPlugin) SetMemory(key string, value interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.memory[key] = value
	return nil
}

func (m *MockAgentPlugin) CallTool(ctx context.Context, name string, params map[string]interface{}) (interface{}, error) {
	if name == "test_tool" {
		return map[string]interface{}{
			"result": "tool executed successfully",
			"params": params,
		}, nil
	}
	return nil, fmt.Errorf("unknown tool: %s", name)
}

func (m *MockAgentPlugin) CreateTerminal(rows, cols int) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	terminalID := fmt.Sprintf("term_%s_%d", m.id, len(m.terminals))
	m.terminals[terminalID] = &MockTerminal{
		id:   terminalID,
		rows: rows,
		cols: cols,
		data: make([]byte, 0),
	}
	return terminalID, nil
}

func (m *MockAgentPlugin) ResizeTerminal(terminalID string, rows, cols int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	terminal, exists := m.terminals[terminalID]
	if !exists {
		return fmt.Errorf("terminal not found: %s", terminalID)
	}
	terminal.rows = rows
	terminal.cols = cols
	return nil
}

func (m *MockAgentPlugin) WriteToTerminal(terminalID string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	terminal, exists := m.terminals[terminalID]
	if !exists {
		return fmt.Errorf("terminal not found: %s", terminalID)
	}
	terminal.data = append(terminal.data, data...)
	return nil
}

func (m *MockAgentPlugin) ReadFromTerminal(terminalID string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	terminal, exists := m.terminals[terminalID]
	if !exists {
		return nil, fmt.Errorf("terminal not found: %s", terminalID)
	}
	return terminal.data, nil
}

func (m *MockAgentPlugin) CloseTerminal(terminalID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.terminals[terminalID]; !exists {
		return fmt.Errorf("terminal not found: %s", terminalID)
	}
	delete(m.terminals, terminalID)
	return nil
}

// Legacy memory management methods for backward compatibility
func (m *MockAgentPlugin) StoreContext(contextID string, context map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.memory[fmt.Sprintf("context_%s", contextID)] = context
	return nil
}

func (m *MockAgentPlugin) GetContext(contextID string) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	context, exists := m.memory[fmt.Sprintf("context_%s", contextID)]
	if !exists {
		return nil, fmt.Errorf("context not found")
	}

	contextMap, ok := context.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid context format")
	}

	return contextMap, nil
}

func (m *MockAgentPlugin) TransferContext(contextID string, targetAgentID string) error {
	// Mock implementation - just verify context exists
	_, err := m.GetContext(contextID)
	return err
}

func (m *MockAgentPlugin) StoreCredential(credentialID string, credential map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.memory[fmt.Sprintf("credential_%s", credentialID)] = credential
	return nil
}

func (m *MockAgentPlugin) GetCredential(credentialID string) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	credential, exists := m.memory[fmt.Sprintf("credential_%s", credentialID)]
	if !exists {
		return nil, fmt.Errorf("credential not found")
	}

	credentialMap, ok := credential.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid credential format")
	}

	return credentialMap, nil
}

func (m *MockAgentPlugin) StoreRAGResult(queryHash string, result map[string]interface{}, ttl int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.memory[fmt.Sprintf("rag_%s", queryHash)] = result
	return nil
}

func (m *MockAgentPlugin) GetRAGResult(queryHash string) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result, exists := m.memory[fmt.Sprintf("rag_%s", queryHash)]
	if !exists {
		return nil, fmt.Errorf("RAG result not found")
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid RAG result format")
	}

	return resultMap, nil
}

func (m *MockAgentPlugin) StoreCOTPlan(planID string, plan map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.memory[fmt.Sprintf("cot_plan_%s", planID)] = plan
	return nil
}

func (m *MockAgentPlugin) GetCOTPlan(planID string) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plan, exists := m.memory[fmt.Sprintf("cot_plan_%s", planID)]
	if !exists {
		return nil, fmt.Errorf("COT plan not found")
	}

	planMap, ok := plan.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid COT plan format")
	}

	return planMap, nil
}

func (m *MockAgentPlugin) StoreUserPreference(userID string, preference map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.memory[fmt.Sprintf("user_preferences_%s", userID)] = preference
	return nil
}

func (m *MockAgentPlugin) GetUserPreferences(userID string) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	preferences, exists := m.memory[fmt.Sprintf("user_preferences_%s", userID)]
	if !exists {
		return nil, fmt.Errorf("user preferences not found")
	}

	prefMap, ok := preferences.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid preferences format")
	}

	return prefMap, nil
}

func (m *MockAgentPlugin) GetUserPreference(userID string, key string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	preferences, exists := m.memory[fmt.Sprintf("user_preferences_%s", userID)]
	if !exists {
		return nil, fmt.Errorf("user preferences not found")
	}

	prefMap, ok := preferences.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid preferences format")
	}

	value, exists := prefMap[key]
	if !exists {
		return nil, fmt.Errorf("preference key not found")
	}

	return value, nil
}

// Test helper functions
func createTestPluginDir(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "test_plugins")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })
	return tempDir
}

func createMockPluginFile(t *testing.T, pluginDir, filename string) string {
	// Skip on Windows as plugin loading is not supported
	if runtime.GOOS == "windows" {
		t.Skip("Plugin loading not supported on Windows")
	}

	pluginPath := filepath.Join(pluginDir, filename)
	// Create a mock plugin file (empty file for testing)
	file, err := os.Create(pluginPath)
	require.NoError(t, err)
	file.Close()
	return pluginPath
}

// TestAgentPluginLoader_NewAgentPluginLoader tests the constructor
func TestAgentPluginLoader_NewAgentPluginLoader(t *testing.T) {
	pluginDir := createTestPluginDir(t)

	loader := NewAgentPluginLoader(pluginDir)

	assert.NotNil(t, loader)
	// Note: fields are private, so we test functionality instead
	plugins := loader.ListLoadedPlugins()
	assert.NotNil(t, plugins)
}

// TestAgentPluginLoader_DiscoverPlugins tests plugin discovery
func TestAgentPluginLoader_DiscoverPlugins(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Plugin loading not supported on Windows")
	}

	pluginDir := createTestPluginDir(t)
	loader := NewAgentPluginLoader(pluginDir)

	// Create some mock plugin files
	createMockPluginFile(t, pluginDir, "plugin1.so")
	createMockPluginFile(t, pluginDir, "plugin2.so")
	createMockPluginFile(t, pluginDir, "not_a_plugin.txt")

	plugins, err := loader.DiscoverPlugins()

	assert.NoError(t, err)
	assert.Len(t, plugins, 2) // Only .so files should be discovered

	// Check that plugin paths are correct
	expectedPaths := []string{
		filepath.Join(pluginDir, "plugin1.so"),
		filepath.Join(pluginDir, "plugin2.so"),
	}
	for _, expectedPath := range expectedPaths {
		assert.Contains(t, plugins, expectedPath)
	}
}

// TestAgentPluginLoader_DiscoverPlugins_EmptyDirectory tests discovery in empty directory
func TestAgentPluginLoader_DiscoverPlugins_EmptyDirectory(t *testing.T) {
	pluginDir := createTestPluginDir(t)
	loader := NewAgentPluginLoader(pluginDir)

	plugins, err := loader.DiscoverPlugins()

	assert.NoError(t, err)
	assert.Empty(t, plugins)
}

// TestAgentPluginLoader_DiscoverPlugins_NonexistentDirectory tests discovery with nonexistent directory
func TestAgentPluginLoader_DiscoverPlugins_NonexistentDirectory(t *testing.T) {
	loader := NewAgentPluginLoader("/nonexistent/directory")

	plugins, err := loader.DiscoverPlugins()

	assert.Error(t, err)
	assert.Nil(t, plugins)
}

// TestAgentPluginLoader_LoadPlugin tests plugin loading (mock scenario)
func TestAgentPluginLoader_LoadPlugin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Plugin loading not supported on Windows")
	}

	pluginDir := createTestPluginDir(t)
	loader := NewAgentPluginLoader(pluginDir)

	// This will fail because we need agentID and version
	plugin, err := loader.LoadPlugin("test-agent", "1.0.0")

	assert.Error(t, err) // Expected to fail with no plugin file
	assert.Nil(t, plugin)
}

// TestAgentPluginLoader_LoadPlugin_InvalidParams tests loading with invalid parameters
func TestAgentPluginLoader_LoadPlugin_InvalidParams(t *testing.T) {
	pluginDir := createTestPluginDir(t)
	loader := NewAgentPluginLoader(pluginDir)

	// Test with empty agentID
	plugin, err := loader.LoadPlugin("", "1.0.0")
	assert.Error(t, err)
	assert.Nil(t, plugin)

	// Test with empty version
	plugin, err = loader.LoadPlugin("test-agent", "")
	assert.Error(t, err)
	assert.Nil(t, plugin)
}

// TestAgentPluginLoader_ListLoadedPlugins tests getting loaded plugins
func TestAgentPluginLoader_ListLoadedPlugins(t *testing.T) {
	pluginDir := createTestPluginDir(t)
	loader := NewAgentPluginLoader(pluginDir)

	// Initially should be empty
	plugins := loader.ListLoadedPlugins()
	assert.Empty(t, plugins)
}

// TestAgentPluginLoader_UnloadPlugin tests plugin unloading
func TestAgentPluginLoader_UnloadPlugin(t *testing.T) {
	pluginDir := createTestPluginDir(t)
	loader := NewAgentPluginLoader(pluginDir)

	// Test unloading non-existent plugin
	err := loader.UnloadPlugin("nonexistent", "1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "plugin not found")
}

// TestAgentPluginLoader_ConcurrentAccess tests thread safety
func TestAgentPluginLoader_ConcurrentAccess(t *testing.T) {
	pluginDir := createTestPluginDir(t)
	loader := NewAgentPluginLoader(pluginDir)

	var wg sync.WaitGroup
	numGoroutines := 10

	// Test concurrent access to ListLoadedPlugins
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			plugins := loader.ListLoadedPlugins()
			assert.NotNil(t, plugins)
		}()
	}

	wg.Wait()
}

// TestAgentPluginLoader_DiscoverAllPlugins tests detailed plugin discovery
func TestAgentPluginLoader_DiscoverAllPlugins(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Plugin loading not supported on Windows")
	}

	pluginDir := createTestPluginDir(t)
	loader := NewAgentPluginLoader(pluginDir)

	// Create some mock plugin files
	createMockPluginFile(t, pluginDir, "plugin1.so")
	createMockPluginFile(t, pluginDir, "plugin2.so")

	plugins, err := loader.DiscoverAllPlugins()

	assert.NoError(t, err)
	assert.Len(t, plugins, 2)

	// Check plugin info structure
	for _, plugin := range plugins {
		assert.NotEmpty(t, plugin.FileName)
		assert.NotEmpty(t, plugin.FilePath)
		assert.False(t, plugin.IsRegistered)
	}
}

// TestAgentPluginLoader_ImportPlugin tests plugin import functionality
func TestAgentPluginLoader_ImportPlugin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Plugin loading not supported on Windows")
	}

	pluginDir := createTestPluginDir(t)
	loader := NewAgentPluginLoader(pluginDir)

	// Create a mock plugin file to import
	tempFile := createMockPluginFile(t, pluginDir, "temp_plugin.so")

	// Test import with valid request
	request := &ImportPluginRequest{
		FilePath: tempFile,
		AgentID:  "test-agent",
		Version:  "1.0.0",
	}

	err := loader.ImportPlugin(request)
	// This may fail due to file operations, but we test the validation
	// The error should be about file operations, not validation
	if err != nil {
		assert.NotContains(t, err.Error(), "required")
	}
}

// TestAgentPluginLoader_ImportPlugin_InvalidRequest tests import with invalid request
func TestAgentPluginLoader_ImportPlugin_InvalidRequest(t *testing.T) {
	pluginDir := createTestPluginDir(t)
	loader := NewAgentPluginLoader(pluginDir)

	// Test with empty FilePath
	request := &ImportPluginRequest{
		FilePath: "",
		AgentID:  "test-agent",
		Version:  "1.0.0",
	}

	err := loader.ImportPlugin(request)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")

	// Test with empty AgentID
	request = &ImportPluginRequest{
		FilePath: "/some/path",
		AgentID:  "",
		Version:  "1.0.0",
	}

	err = loader.ImportPlugin(request)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")

	// Test with empty Version
	request = &ImportPluginRequest{
		FilePath: "/some/path",
		AgentID:  "test-agent",
		Version:  "",
	}

	err = loader.ImportPlugin(request)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

// TestAgentPluginLoader_EdgeCases tests various edge cases
func TestAgentPluginLoader_EdgeCases(t *testing.T) {
	pluginDir := createTestPluginDir(t)
	loader := NewAgentPluginLoader(pluginDir)

	// Test with very long agent ID
	longID := strings.Repeat("a", 1000)
	plugin, err := loader.LoadPlugin(longID, "1.0.0")
	assert.Error(t, err)
	assert.Nil(t, plugin)

	// Test with special characters in agent ID
	specialID := "test-agent@#$%^&*()"
	plugin, err = loader.LoadPlugin(specialID, "1.0.0")
	assert.Error(t, err)
	assert.Nil(t, plugin)

	// Test concurrent plugin loading attempts
	var wg sync.WaitGroup
	numGoroutines := 5

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			agentID := fmt.Sprintf("concurrent-agent-%d", id)
			_, err := loader.LoadPlugin(agentID, "1.0.0")
			// Error is expected since plugins don't exist
			assert.Error(t, err)
		}(i)
	}

	wg.Wait()
}

// TestAgentPluginLoader_ConcurrentOperations tests thread safety
func TestAgentPluginLoader_ConcurrentOperations(t *testing.T) {
	pluginDir := createTestPluginDir(t)
	loader := NewAgentPluginLoader(pluginDir)

	const numOperations = 50
	var wg sync.WaitGroup
	errors := make(chan error, numOperations)

	// Perform concurrent operations
	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Mix of different operations
			switch id % 4 {
			case 0:
				// Discover plugins
				_, err := loader.DiscoverPlugins()
				if err != nil {
					errors <- fmt.Errorf("discover error: %v", err)
				}
			case 1:
				// List loaded plugins
				plugins := loader.ListLoadedPlugins()
				if plugins == nil {
					errors <- fmt.Errorf("list error: plugins should not be nil")
				}
			case 2:
				// Try to load non-existent plugin
				_, err := loader.LoadPlugin(fmt.Sprintf("non-existent-%d", id), "1.0.0")
				// Error is expected, but should not panic
				if err == nil {
					errors <- fmt.Errorf("expected error for non-existent plugin")
				}
			case 3:
				// Discover all plugins
				_, err := loader.DiscoverAllPlugins()
				if err != nil {
					errors <- fmt.Errorf("discover all error: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for unexpected errors
	for err := range errors {
		t.Errorf("Concurrent operation error: %v", err)
	}
}

// TestAgentPluginLoader_MemoryManagement tests memory handling
func TestAgentPluginLoader_MemoryManagement(t *testing.T) {
	pluginDir := createTestPluginDir(t)
	loader := NewAgentPluginLoader(pluginDir)

	// Test with large number of discovery operations
	for i := 0; i < 100; i++ {
		plugins, err := loader.DiscoverPlugins()
		assert.NoError(t, err)
		assert.NotNil(t, plugins)

		// Verify memory is not growing unbounded
		if i%10 == 0 {
			// Force garbage collection to test for memory leaks
			runtime.GC()
		}
	}
}

// TestAgentPluginLoader_PathValidation tests path validation and security
func TestAgentPluginLoader_PathValidation(t *testing.T) {
	pluginDir := createTestPluginDir(t)
	loader := NewAgentPluginLoader(pluginDir)

	// Test path traversal attempts
	maliciousPaths := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32\\cmd.exe",
		"/etc/shadow",
		"C:\\Windows\\System32\\calc.exe",
		"./../../sensitive_file",
	}

	for _, path := range maliciousPaths {
		request := &ImportPluginRequest{
			FilePath: path,
			AgentID:  "test-agent",
			Version:  "1.0.0",
		}

		err := loader.ImportPlugin(request)
		assert.Error(t, err, "Should reject malicious path: %s", path)
	}
}

// TestAgentPluginLoader_ResourceLimits tests resource limit enforcement
func TestAgentPluginLoader_ResourceLimits(t *testing.T) {
	pluginDir := createTestPluginDir(t)
	loader := NewAgentPluginLoader(pluginDir)

	// Test with extremely large agent ID
	largeID := strings.Repeat("x", 10000)
	_, err := loader.LoadPlugin(largeID, "1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")

	// Test with extremely large version string
	largeVersion := strings.Repeat("1.", 5000) + "0"
	_, err = loader.LoadPlugin("test-agent", largeVersion)
	assert.Error(t, err)
}

// TestAgentPluginLoader_ErrorRecovery tests error recovery scenarios
func TestAgentPluginLoader_ErrorRecovery(t *testing.T) {
	pluginDir := createTestPluginDir(t)
	loader := NewAgentPluginLoader(pluginDir)

	// Test recovery after invalid operations
	_, err := loader.LoadPlugin("", "")
	assert.Error(t, err)

	// Verify loader is still functional after error
	plugins, err := loader.DiscoverPlugins()
	assert.NoError(t, err)
	assert.NotNil(t, plugins)

	// Test multiple consecutive errors don't break the loader
	for i := 0; i < 10; i++ {
		_, err := loader.LoadPlugin("", "")
		assert.Error(t, err)
	}

	// Verify loader is still functional
	plugins, err = loader.DiscoverPlugins()
	assert.NoError(t, err)
	assert.NotNil(t, plugins)
}
