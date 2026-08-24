// agent_http_api_test.go
package agentify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAgentInferencer implements the inferencer interface for testing
type MockAgentInferencer struct {
	agents   map[string]*MockAgentPlugin
	sessions map[string]string // sessionID -> agentID
}

func NewMockAgentInferencer() *MockAgentInferencer {
	return &MockAgentInferencer{
		agents:   make(map[string]*MockAgentPlugin),
		sessions: make(map[string]string),
	}
}

func (m *MockAgentInferencer) AddAgent(id string, agent *MockAgentPlugin) {
	m.agents[id] = agent
}

// ListAvailableAgents returns the IDs of registered agents
func (m *MockAgentInferencer) ListAvailableAgents(ctx context.Context) ([]string, error) {
	ids := make([]string, 0, len(m.agents))
	for id := range m.agents {
		ids = append(ids, id)
	}
	return ids, nil
}

// ActivateAgent activates the given agent for the provided session
func (m *MockAgentInferencer) ActivateAgent(ctx context.Context, agentID, version, sessionID string, config map[string]interface{}) error {
	if _, exists := m.agents[agentID]; !exists {
		return fmt.Errorf("agent not found: %s", agentID)
	}
	if sessionID == "" {
		sessionID = "default"
	}
	m.sessions[sessionID] = agentID
	return nil
}

func (m *MockAgentInferencer) DeactivateAgent(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		sessionID = "default"
	}
	delete(m.sessions, sessionID)
	return nil
}

func (m *MockAgentInferencer) ProcessInference(ctx context.Context, sessionID string, request *InferenceRequest) (*InferenceResponse, error) {
	// Determine session to use
	sid := sessionID
	if sid == "" {
		sid = request.SessionID
	}
	if sid == "" {
		sid = "default"
	}

	agentID, ok := m.sessions[sid]
	if !ok {
		// fallback to first agent if available
		for id := range m.agents {
			agentID = id
			break
		}
		if agentID == "" {
			return nil, fmt.Errorf("no active agent")
		}
	}

	agent, ok := m.agents[agentID]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	return agent.ProcessInference(ctx, request)
}

// The following methods are minimal implementations to satisfy the interface
func (m *MockAgentInferencer) GetAgentSchema(ctx context.Context, sessionID string) (*AgentSchema, error) {
	return &AgentSchema{}, nil
}

func (m *MockAgentInferencer) GetAgentCapabilities(ctx context.Context, sessionID string) (*AgentCapabilities, error) {
	return &AgentCapabilities{}, nil
}

func (m *MockAgentInferencer) GetTEEInfo(ctx context.Context, sessionID string) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (m *MockAgentInferencer) GetAgentMemory(ctx context.Context, sessionID, key string) (interface{}, error) {
	return nil, nil
}

func (m *MockAgentInferencer) SetAgentMemory(ctx context.Context, sessionID, key string, value interface{}) error {
	return nil
}

func (m *MockAgentInferencer) CreateTerminal(ctx context.Context, sessionID string, rows, cols int) (string, error) {
	return "terminal-1", nil
}

func (m *MockAgentInferencer) ResizeTerminal(ctx context.Context, sessionID, terminalID string, rows, cols int) error {
	return nil
}

func (m *MockAgentInferencer) WriteToTerminal(ctx context.Context, sessionID, terminalID string, data []byte) error {
	return nil
}

func (m *MockAgentInferencer) ReadFromTerminal(ctx context.Context, sessionID, terminalID string) ([]byte, error) {
	return []byte{}, nil
}

func (m *MockAgentInferencer) CloseTerminal(ctx context.Context, sessionID, terminalID string) error {
	return nil
}

// WASM plugin management methods
func (m *MockAgentInferencer) DiscoverWASMPluginZips(ctx context.Context) ([]*WASMPluginInfo, error) {
	return []*WASMPluginInfo{}, nil
}

func (m *MockAgentInferencer) InstallWASMPlugin(ctx context.Context, zipPath string) (*WASMPluginInfo, error) {
	return &WASMPluginInfo{}, nil
}

func (m *MockAgentInferencer) UninstallWASMPlugin(ctx context.Context, agentID, version string) error {
	return nil
}

func (m *MockAgentInferencer) ListInstalledWASMPlugins(ctx context.Context) ([]*WASMPluginInfo, error) {
	return []*WASMPluginInfo{}, nil
}

func (m *MockAgentInferencer) GetAvailableAgentsDetailed(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (m *MockAgentInferencer) GetTerminalOutputChannel(ctx context.Context, sessionID, terminalID string) (<-chan []byte, error) {
	// Create a simple channel that will be closed immediately since this is a mock
	ch := make(chan []byte)
	close(ch)
	return ch, nil
}

// Test helper functions
func createTestHTTPAPI(t *testing.T) (*AgentHTTPAPI, *MockAgentInferencer) {
	tempDir, err := os.MkdirTemp("", "test_http_api")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	mockInferencer := NewMockAgentInferencer()

	// Add a test agent
	testAgent := NewMockAgentPlugin("test-agent")
	err = testAgent.Initialize(map[string]interface{}{"test": "config"})
	require.NoError(t, err)
	err = testAgent.Start()
	require.NoError(t, err)

	mockInferencer.AddAgent("test-agent", testAgent)

	api := NewAgentHTTPAPI(mockInferencer)
	return api, mockInferencer
}

// TestAgentHTTPAPI_NewAgentHTTPAPI tests the constructor
func TestAgentHTTPAPI_NewAgentHTTPAPI(t *testing.T) {
	api, _ := createTestHTTPAPI(t)

	assert.NotNil(t, api)
}

// TestAgentHTTPAPI_ListAgents tests the list agents endpoint
func TestAgentHTTPAPI_ListAgents(t *testing.T) {
	api, _ := createTestHTTPAPI(t)

	mux := http.NewServeMux()
	api.RegisterHandlers(mux)

	req := httptest.NewRequest("GET", "/v1/agents", nil)
	req.Header.Set("Authorization", "Bearer test-api-key")

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	assert.NoError(t, err)

	agents, ok := response["agents"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, agents, 1)
	assert.Equal(t, "test-agent", agents[0])
}

// TestAgentHTTPAPI_ListAgents_Unauthorized tests unauthorized access
func TestAgentHTTPAPI_ListAgents_Unauthorized(t *testing.T) {
	api, _ := createTestHTTPAPI(t)

	mux := http.NewServeMux()
	api.RegisterHandlers(mux)

	req := httptest.NewRequest("GET", "/v1/agents", nil)
	// No authorization header

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}

// TestAgentHTTPAPI_ActivateAgent tests the activate agent endpoint
func TestAgentHTTPAPI_ActivateAgent(t *testing.T) {
	api, _ := createTestHTTPAPI(t)

	mux := http.NewServeMux()
	api.RegisterHandlers(mux)

	requestBody := map[string]interface{}{
		"agent_id": "test-agent",
	}

	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/v1/agents/activate", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer test-api-key")
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "success", response["status"])
}

// TestAgentHTTPAPI_ActivateAgent_InvalidAgent tests activating invalid agent
func TestAgentHTTPAPI_ActivateAgent_InvalidAgent(t *testing.T) {
	api, _ := createTestHTTPAPI(t)

	mux := http.NewServeMux()
	api.RegisterHandlers(mux)

	requestBody := map[string]interface{}{
		"agent_id": "nonexistent-agent",
	}

	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/v1/agents/activate", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer test-api-key")
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

// TestAgentHTTPAPI_ActivateAgent_InvalidJSON tests activating with invalid JSON
func TestAgentHTTPAPI_ActivateAgent_InvalidJSON(t *testing.T) {
	api, _ := createTestHTTPAPI(t)

	mux := http.NewServeMux()
	api.RegisterHandlers(mux)

	req := httptest.NewRequest("POST", "/v1/agents/activate", strings.NewReader("invalid json"))
	req.Header.Set("Authorization", "Bearer test-api-key")
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

// TestAgentHTTPAPI_ProcessInference tests the inference endpoint
func TestAgentHTTPAPI_ProcessInference(t *testing.T) {
	api, _ := createTestHTTPAPI(t)

	mux := http.NewServeMux()
	api.RegisterHandlers(mux)

	requestBody := map[string]interface{}{
		"input": "test input message",
		"parameters": map[string]interface{}{
			"temperature": 0.7,
		},
	}

	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/v1/inference", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer test-api-key")
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Contains(t, response["output"], "test input message")
	assert.NotNil(t, response["metadata"])
}

// TestAgentHTTPAPI_ProcessInference_NoActiveAgent tests inference with no active agent
func TestAgentHTTPAPI_ProcessInference_NoActiveAgent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_http_api")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create inferencer with no agents
	mockInferencer := NewMockAgentInferencer()
	api := NewAgentHTTPAPI(mockInferencer)

	mux := http.NewServeMux()
	api.RegisterHandlers(mux)

	requestBody := map[string]interface{}{
		"input": "test input message",
	}

	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/v1/inference", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer test-api-key")
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

// TestAgentHTTPAPI_SetAuthProvider tests setting custom auth provider
func TestAgentHTTPAPI_SetAuthProvider(t *testing.T) {
	api, _ := createTestHTTPAPI(t)

	// Create a custom auth provider
	customAuthProvider := NewAPIKeyAuthProvider()
	customAuthProvider.AddAPIKey("custom-key", "custom-user")

	api.SetAuthProvider(customAuthProvider)

	mux := http.NewServeMux()
	api.RegisterHandlers(mux)

	// Test with custom key
	req := httptest.NewRequest("GET", "/v1/agents", nil)
	req.Header.Set("Authorization", "Bearer custom-key")

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	// Test with old key (should fail)
	req = httptest.NewRequest("GET", "/v1/agents", nil)
	req.Header.Set("Authorization", "Bearer test-api-key")

	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}

// TestAgentHTTPAPI_ConcurrentRequests tests handling concurrent requests
func TestAgentHTTPAPI_ConcurrentRequests(t *testing.T) {
	api, _ := createTestHTTPAPI(t)

	mux := http.NewServeMux()
	api.RegisterHandlers(mux)

	server := httptest.NewServer(mux)
	defer server.Close()

	// Make multiple concurrent requests
	numRequests := 10
	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req, err := http.NewRequest("GET", server.URL+"/v1/agents", nil)
			if err != nil {
				results <- 0
				return
			}

			req.Header.Set("Authorization", "Bearer test-api-key")

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				results <- 0
				return
			}
			defer resp.Body.Close()

			results <- resp.StatusCode
		}()
	}

	// Collect results
	successCount := 0
	for i := 0; i < numRequests; i++ {
		statusCode := <-results
		if statusCode == http.StatusOK {
			successCount++
		}
	}

	assert.Equal(t, numRequests, successCount)
}
