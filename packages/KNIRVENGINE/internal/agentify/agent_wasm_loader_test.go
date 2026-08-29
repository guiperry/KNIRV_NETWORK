// agent_wasm_loader_test.go
package agentify

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockWASMAgent implements WASMAgentInterface for testing
type MockWASMAgent struct {
	id           string
	initialized  bool
	started      bool
	capabilities *AgentCapabilities
	schema       *AgentSchema
	memory       map[string]interface{}
	terminals    map[string]*MockWASMTerminal
	mu           sync.RWMutex
}

type MockWASMTerminal struct {
	id   string
	rows int
	cols int
	data []byte
}

func NewMockWASMAgent(id string) *MockWASMAgent {
	return &MockWASMAgent{
		id:        id,
		memory:    make(map[string]interface{}),
		terminals: make(map[string]*MockWASMTerminal),
		capabilities: &AgentCapabilities{
			SupportsStreaming:   true,
			SupportsToolCalls:   true,
			SupportsReasoning:   true,
			MaxContextLength:    8192,
			SupportedParameters: []string{"temperature", "max_tokens"},
		},
		schema: &AgentSchema{
			Tools: []*ToolSchema{
				{
					Name:        "wasm_tool",
					Description: "A WASM tool",
					Parameters: map[string]*ParameterSchema{
						"input": {
							Type:        "string",
							Description: "Input parameter",
							Required:    true,
						},
					},
					ReturnType: "string",
				},
			},
			Resources: []*ResourceSchema{
				{
					Name:        "wasm_resource",
					Type:        "wasm",
					Description: "A WASM resource",
				},
			},
		},
	}
}

func (m *MockWASMAgent) Initialize(config map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.initialized = true
	return nil
}

func (m *MockWASMAgent) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.initialized {
		return fmt.Errorf("agent not initialized")
	}
	m.started = true
	return nil
}

func (m *MockWASMAgent) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = false
	return nil
}

func (m *MockWASMAgent) ProcessInference(ctx context.Context, request *InferenceRequest) (*InferenceResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.started {
		return nil, fmt.Errorf("agent not started")
	}

	return &InferenceResponse{
		Output:   fmt.Sprintf("WASM response to: %s", request.Input),
		Metadata: map[string]interface{}{"agent_id": m.id, "type": "wasm"},
	}, nil
}

func (m *MockWASMAgent) GetCapabilities() *AgentCapabilities {
	return m.capabilities
}

func (m *MockWASMAgent) GetSchema() *AgentSchema {
	return m.schema
}

func (m *MockWASMAgent) GetMemory(key string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, exists := m.memory[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	return value, nil
}

func (m *MockWASMAgent) SetMemory(key string, value interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.memory[key] = value
	return nil
}

func (m *MockWASMAgent) CallTool(ctx context.Context, toolName string, input map[string]interface{}) (string, error) {
	if toolName == "wasm_tool" {
		return fmt.Sprintf("Tool result: %v", input), nil
	}
	return "", fmt.Errorf("unknown tool: %s", toolName)
}

func (m *MockWASMAgent) CreateTerminal(rows, cols int) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	terminalID := fmt.Sprintf("wasm_term_%s_%d", m.id, len(m.terminals))
	m.terminals[terminalID] = &MockWASMTerminal{
		id:   terminalID,
		rows: rows,
		cols: cols,
		data: make([]byte, 0),
	}
	return terminalID, nil
}

func (m *MockWASMAgent) ResizeTerminal(terminalID string, rows, cols int) error {
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

func (m *MockWASMAgent) WriteToTerminal(terminalID string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	terminal, exists := m.terminals[terminalID]
	if !exists {
		return fmt.Errorf("terminal not found: %s", terminalID)
	}
	terminal.data = append(terminal.data, data...)
	return nil
}

func (m *MockWASMAgent) ReadFromTerminal(terminalID string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	terminal, exists := m.terminals[terminalID]
	if !exists {
		return nil, fmt.Errorf("terminal not found: %s", terminalID)
	}
	return terminal.data, nil
}

func (m *MockWASMAgent) CloseTerminal(terminalID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.terminals[terminalID]; !exists {
		return fmt.Errorf("terminal not found: %s", terminalID)
	}
	delete(m.terminals, terminalID)
	return nil
}

// Test helper functions
func createTestWASMDir(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "test_wasm")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })
	return tempDir
}

func createMockWASMFile(t *testing.T, wasmDir, filename string) string {
	wasmPath := filepath.Join(wasmDir, filename)
	// Create a mock WASM file (empty file for testing)
	file, err := os.Create(wasmPath)
	require.NoError(t, err)
	file.Close()
	return wasmPath
}

// TestAgentWASMLoader_NewAgentWASMLoader tests the constructor
func TestAgentWASMLoader_NewAgentWASMLoader(t *testing.T) {
	wasmDir := createTestWASMDir(t)

	loader := NewAgentWASMLoader(wasmDir)

	assert.NotNil(t, loader)
	// Test functionality since fields are private
	agents, err := loader.DiscoverWASMAgents()
	assert.NoError(t, err)
	assert.NotNil(t, agents)
}

// TestAgentWASMLoader_LoadWASMAgent tests WASM agent loading
func TestAgentWASMLoader_LoadWASMAgent(t *testing.T) {
	wasmDir := createTestWASMDir(t)
	loader := NewAgentWASMLoader(wasmDir)

	// Create a mock WASM file
	createMockWASMFile(t, wasmDir, "agent_test_1.0.0.wasm")

	// This will fail because it's not a real WASM file, but we test the error handling
	agent, err := loader.LoadWASMAgent("test", "1.0.0", map[string]interface{}{
		"memory_limit": 1024,
	})

	assert.Error(t, err) // Expected to fail with mock file
	assert.Nil(t, agent)
}

// TestAgentWASMLoader_LoadWASMAgent_NonexistentFile tests loading nonexistent WASM file
func TestAgentWASMLoader_LoadWASMAgent_NonexistentFile(t *testing.T) {
	wasmDir := createTestWASMDir(t)
	loader := NewAgentWASMLoader(wasmDir)

	agent, err := loader.LoadWASMAgent("nonexistent", "1.0.0")

	assert.Error(t, err)
	assert.Nil(t, agent)
}

// TestAgentWASMLoader_LoadWASMAgent_InvalidConfig tests loading with invalid config
func TestAgentWASMLoader_LoadWASMAgent_InvalidConfig(t *testing.T) {
	wasmDir := createTestWASMDir(t)
	loader := NewAgentWASMLoader(wasmDir)

	// Test with empty agentID
	agent, err := loader.LoadWASMAgent("", "1.0.0")
	assert.Error(t, err)
	assert.Nil(t, agent)

	// Test with empty version
	agent, err = loader.LoadWASMAgent("test", "")
	assert.Error(t, err)
	assert.Nil(t, agent)
}

// TestAgentWASMLoader_DiscoverWASMAgents tests discovering WASM agents
func TestAgentWASMLoader_DiscoverWASMAgents(t *testing.T) {
	wasmDir := createTestWASMDir(t)
	loader := NewAgentWASMLoader(wasmDir)

	// Initially should be empty
	agents, err := loader.DiscoverWASMAgents()
	assert.NoError(t, err)
	assert.Empty(t, agents)
}

// TestAgentWASMLoader_UnloadWASMAgent tests agent unloading
func TestAgentWASMLoader_UnloadWASMAgent(t *testing.T) {
	wasmDir := createTestWASMDir(t)
	loader := NewAgentWASMLoader(wasmDir)

	// Test unloading non-existent agent
	err := loader.UnloadWASMAgent("nonexistent", "1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestAgentWASMLoader_ConcurrentAccess tests thread safety
func TestAgentWASMLoader_ConcurrentAccess(t *testing.T) {
	wasmDir := createTestWASMDir(t)
	loader := NewAgentWASMLoader(wasmDir)

	var wg sync.WaitGroup
	numGoroutines := 10

	// Test concurrent access to DiscoverWASMAgents
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			agents, err := loader.DiscoverWASMAgents()
			assert.NoError(t, err)
			assert.NotNil(t, agents)
		}()
	}

	wg.Wait()
}

// TestAgentWASMLoader_MemoryManagement tests memory management
func TestAgentWASMLoader_MemoryManagement(t *testing.T) {
	wasmDir := createTestWASMDir(t)
	loader := NewAgentWASMLoader(wasmDir)

	// Test with various memory configurations
	configs := []map[string]interface{}{
		{"memory_limit": 512},
		{"memory_limit": 1024},
		{"memory_limit": 2048},
	}

	for i, config := range configs {
		agentID := fmt.Sprintf("test_agent_%d", i)
		createMockWASMFile(t, wasmDir, fmt.Sprintf("agent_%s_1.0.0.wasm", agentID))

		// This will fail because it's not a real WASM file
		agent, err := loader.LoadWASMAgent(agentID, "1.0.0", config)
		assert.Error(t, err)
		assert.Nil(t, agent)
	}
}

// TestAgentWASMLoader_EdgeCases tests various edge cases
func TestAgentWASMLoader_EdgeCases(t *testing.T) {
	wasmDir := createTestWASMDir(t)
	loader := NewAgentWASMLoader(wasmDir)

	// Test with very long agent ID
	longAgentID := string(make([]byte, 1000))
	agent, err := loader.LoadWASMAgent(longAgentID, "1.0.0")
	assert.Error(t, err)
	assert.Nil(t, agent)

	// Test with special characters in agent ID
	specialAgentID := "test@#$%^&*()"
	agent, err = loader.LoadWASMAgent(specialAgentID, "1.0.0")
	assert.Error(t, err)
	assert.Nil(t, agent)

	// Test concurrent loading attempts
	var wg sync.WaitGroup
	numGoroutines := 5

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			agentID := fmt.Sprintf("concurrent_agent_%d", id)
			_, err := loader.LoadWASMAgent(agentID, "1.0.0")
			// Error is expected since files don't exist
			assert.Error(t, err)
		}(i)
	}

	wg.Wait()
}

// TestAgentWASMLoader_ConfigValidation tests configuration validation
func TestAgentWASMLoader_ConfigValidation(t *testing.T) {
	wasmDir := createTestWASMDir(t)
	loader := NewAgentWASMLoader(wasmDir)

	agentID := "test_agent"
	createMockWASMFile(t, wasmDir, fmt.Sprintf("agent_%s_1.0.0.wasm", agentID))

	// Test with invalid memory limit
	invalidConfigs := []map[string]interface{}{
		{"memory_limit": -1},
		{"memory_limit": "invalid"},
		{"memory_limit": 0},
	}

	for _, config := range invalidConfigs {
		agent, err := loader.LoadWASMAgent(agentID, "1.0.0", config)
		// Should fail due to invalid config or file format
		assert.Error(t, err)
		assert.Nil(t, agent)
	}
}

// TestMockWASMAgent_Functionality tests the mock WASM agent functionality
func TestMockWASMAgent_Functionality(t *testing.T) {
	agent := NewMockWASMAgent("test-wasm-agent")

	// Test initialization
	err := agent.Initialize(map[string]interface{}{"test": "config"})
	assert.NoError(t, err)

	// Test start
	err = agent.Start()
	assert.NoError(t, err)

	// Test capabilities
	capabilities := agent.GetCapabilities()
	assert.NotNil(t, capabilities)
	assert.True(t, capabilities.SupportsStreaming)
	assert.True(t, capabilities.SupportsToolCalls)
	assert.Equal(t, 8192, capabilities.MaxContextLength)

	// Test schema
	schema := agent.GetSchema()
	assert.NotNil(t, schema)
	assert.Len(t, schema.Tools, 1)
	assert.Equal(t, "wasm_tool", schema.Tools[0].Name)

	// Test inference
	ctx := context.Background()
	request := &InferenceRequest{
		Input: "test input",
	}

	response, err := agent.ProcessInference(ctx, request)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Contains(t, response.Output, "test input")
	assert.Equal(t, "test-wasm-agent", response.Metadata["agent_id"])

	// Test memory operations
	err = agent.SetMemory("test_key", "test_value")
	assert.NoError(t, err)

	value, err := agent.GetMemory("test_key")
	assert.NoError(t, err)
	assert.Equal(t, "test_value", value)

	// Test tool calls
	result, err := agent.CallTool(ctx, "wasm_tool", map[string]interface{}{"param": "value"})
	assert.NoError(t, err)
	assert.Contains(t, result, "Tool result")

	// Test terminal operations
	terminalID, err := agent.CreateTerminal(24, 80)
	assert.NoError(t, err)
	assert.NotEmpty(t, terminalID)

	err = agent.WriteToTerminal(terminalID, []byte("test data"))
	assert.NoError(t, err)

	data, err := agent.ReadFromTerminal(terminalID)
	assert.NoError(t, err)
	assert.Equal(t, []byte("test data"), data)

	err = agent.CloseTerminal(terminalID)
	assert.NoError(t, err)

	// Test stop
	err = agent.Stop()
	assert.NoError(t, err)
}
