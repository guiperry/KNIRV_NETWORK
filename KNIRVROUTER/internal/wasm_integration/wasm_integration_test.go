package wasm_integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"KNIRVROUTER/internal/wasm_loader"

	"github.com/gorilla/mux"
)

func TestNewWASMIntegration(t *testing.T) {
	integration := NewWASMIntegration("test/assets")
	if integration == nil {
		t.Fatal("WASM integration instance is nil")
	}

	if integration.assetsDir != "test/assets" {
		t.Errorf("Expected assetsDir 'test/assets', got '%s'", integration.assetsDir)
	}

	if integration.initialized {
		t.Error("WASM integration should not be initialized on creation")
	}
}

func TestWASMIntegrationInitialize(t *testing.T) {
	integration := NewWASMIntegration("test/assets")

	err := integration.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize WASM integration: %v", err)
	}

	if !integration.initialized {
		t.Error("WASM integration should be initialized after Initialize() call")
	}

	if integration.wasmChain == nil {
		t.Error("WASM chain should not be nil after initialization")
	}
}

func TestWASMIntegrationHandleWASMStatus(t *testing.T) {
	integration := NewWASMIntegration("test/assets")
	err := integration.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize WASM integration: %v", err)
	}

	// Create a test HTTP request
	req, err := http.NewRequest("GET", "/wasm/status", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Call the handler
	integration.handleWASMStatus(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check the content type
	expectedContentType := "application/json"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("Expected content type '%s', got '%s'", expectedContentType, contentType)
	}

	// Parse the response
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	// Check response fields
	if initialized, ok := response["initialized"].(bool); !ok || !initialized {
		t.Error("Expected initialized to be true")
	}

	if engine, ok := response["engine"].(string); !ok || engine != "wasm" {
		t.Errorf("Expected engine to be 'wasm', got '%v'", engine)
	}

	if wasmInitialized, ok := response["wasm_initialized"].(bool); !ok || !wasmInitialized {
		t.Error("Expected wasm_initialized to be true")
	}

	if skillCount, ok := response["skill_count"].(float64); !ok || skillCount != 2 {
		t.Errorf("Expected skill_count to be 2, got %v", skillCount)
	}
}

func TestWASMIntegrationHandleWASMVersion(t *testing.T) {
	integration := NewWASMIntegration("test/assets")
	err := integration.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize WASM integration: %v", err)
	}

	// Create a test HTTP request
	req, err := http.NewRequest("GET", "/wasm/version", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Call the handler
	integration.handleWASMVersion(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Parse the response
	var response map[string]string
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	// Check response fields
	if version, ok := response["version"]; !ok || version == "" {
		t.Error("Expected version to be non-empty")
	}

	if buildInfo, ok := response["build_info"]; !ok || buildInfo == "" {
		t.Error("Expected build_info to be non-empty")
	}

	if engine, ok := response["engine"]; !ok || engine != "wasm" {
		t.Errorf("Expected engine to be 'wasm', got '%s'", engine)
	}
}

func TestWASMIntegrationHandleWASMSkillCount(t *testing.T) {
	integration := NewWASMIntegration("test/assets")
	err := integration.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize WASM integration: %v", err)
	}

	// Create a test HTTP request
	req, err := http.NewRequest("GET", "/wasm/skills/count", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Call the handler
	integration.handleWASMSkillCount(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Parse the response
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	// Check response fields
	if skillCount, ok := response["skill_count"].(float64); !ok || skillCount != 2 {
		t.Errorf("Expected skill_count to be 2, got %v", skillCount)
	}

	if engine, ok := response["engine"]; !ok || engine != "wasm" {
		t.Errorf("Expected engine to be 'wasm', got '%v'", engine)
	}
}

func TestWASMIntegrationHandleWASMInvoke(t *testing.T) {
	integration := NewWASMIntegration("test/assets")
	err := integration.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize WASM integration: %v", err)
	}

	// Create a test skill invocation request
	request := wasm_loader.SkillInvocationRequest{
		InvocationID: "test-invoke-001",
		AgentID:      "test-agent-123",
		NRNToken:     "test-nrn-token-abcdef123456789012345678901234567890",
		SkillURI:     "knirv://skill/javascript-type-checker-v1",
		Priority:     "high",
		Timestamp:    time.Now().Unix(),
	}

	// Marshal the request to JSON
	requestJSON, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// Create a test HTTP request
	req, err := http.NewRequest("POST", "/wasm/invoke", bytes.NewBuffer(requestJSON))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Call the handler
	integration.handleWASMInvoke(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check response headers
	if invocationID := rr.Header().Get("X-KNIRV-Invocation-ID"); invocationID != request.InvocationID {
		t.Errorf("Expected X-KNIRV-Invocation-ID '%s', got '%s'", request.InvocationID, invocationID)
	}

	if engine := rr.Header().Get("X-KNIRV-Engine"); engine != "wasm" {
		t.Errorf("Expected X-KNIRV-Engine 'wasm', got '%s'", engine)
	}

	// Parse the response
	var response wasm_loader.SkillInvocationResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	// Check response fields
	if response.InvocationID != request.InvocationID {
		t.Errorf("Expected invocation ID '%s', got '%s'", request.InvocationID, response.InvocationID)
	}

	if response.Status != "SUCCESS" {
		t.Errorf("Expected status 'SUCCESS', got '%s'", response.Status)
	}

	if response.ExecutionTime <= 0 {
		t.Error("Execution time should be greater than 0")
	}
}

func TestWASMIntegrationRegisterRoutes(t *testing.T) {
	integration := NewWASMIntegration("test/assets")
	err := integration.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize WASM integration: %v", err)
	}

	// Create a new router
	router := mux.NewRouter()

	// Register routes
	integration.RegisterRoutes(router)

	// Test that routes are registered by making requests
	testCases := []struct {
		method string
		path   string
	}{
		{"GET", "/wasm/status"},
		{"GET", "/wasm/version"},
		{"GET", "/wasm/skills/count"},
		{"POST", "/wasm/invoke"},
	}

	for _, tc := range testCases {
		var req *http.Request
		var err error

		if tc.method == "POST" && tc.path == "/wasm/invoke" {
			// Create a valid request body for the invoke endpoint
			testRequest := map[string]interface{}{
				"invocation_id": "test-route-001",
				"agent_id":      "test-agent-route",
				"nrn_token":     "test-token-123456789012345678901234567890",
				"skill_uri":     "knirv://skill/test-v1",
			}
			requestJSON, _ := json.Marshal(testRequest)
			req, err = http.NewRequest(tc.method, tc.path, bytes.NewBuffer(requestJSON))
			if err == nil {
				req.Header.Set("Content-Type", "application/json")
			}
		} else {
			req, err = http.NewRequest(tc.method, tc.path, nil)
		}

		if err != nil {
			t.Fatalf("Failed to create request for %s %s: %v", tc.method, tc.path, err)
		}

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// We expect either 200 OK or 400 Bad Request (for invalid requests)
		// but not 404 Not Found, which would indicate the route isn't registered
		if rr.Code == http.StatusNotFound {
			t.Errorf("Route %s %s not found - routes may not be registered correctly", tc.method, tc.path)
		}
	}
}

func TestWASMIntegrationShutdown(t *testing.T) {
	integration := NewWASMIntegration("test/assets")
	err := integration.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize WASM integration: %v", err)
	}

	// Test shutdown
	err = integration.Shutdown()
	if err != nil {
		t.Fatalf("Failed to shutdown WASM integration: %v", err)
	}

	if integration.initialized {
		t.Error("WASM integration should not be initialized after shutdown")
	}

	// Test double shutdown (should not error)
	err = integration.Shutdown()
	if err != nil {
		t.Fatalf("Double shutdown should not fail: %v", err)
	}
}

func TestGetAssetsPath(t *testing.T) {
	assetsPath := GetAssetsPath()
	// The function returns filepath.Join(".", "assets") which is "assets" on Unix
	if assetsPath != "assets" && assetsPath != "./assets" {
		t.Errorf("Expected assets path 'assets' or './assets', got '%s'", assetsPath)
	}
}
