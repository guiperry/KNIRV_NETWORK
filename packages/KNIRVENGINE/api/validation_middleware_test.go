package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidationMiddleware_ValidateStruct(t *testing.T) {
	vm := NewValidationMiddleware()

	// Test valid agent create request
	validAgent := AgentCreateRequest{
		Name:        "Test Agent",
		Type:        "llm",
		Description: "A test agent",
		BuildTarget: "wasm",
		Config:      map[string]interface{}{"key": "value"},
		Capabilities: []string{"text processing"},
		TargetTypes:  []string{"api"},
	}

	err := vm.ValidateStruct(validAgent)
	if err != nil {
		t.Errorf("Expected valid agent to pass validation, got error: %v", err)
	}

	// Test invalid agent create request - missing required fields
	invalidAgent := AgentCreateRequest{
		Name: "", // Required field is empty
		Type: "invalid_type", // Invalid agent type
	}

	err = vm.ValidateStruct(invalidAgent)
	if err == nil {
		t.Error("Expected invalid agent to fail validation")
	}
}

func TestValidationMiddleware_CustomValidations(t *testing.T) {
	vm := NewValidationMiddleware()

	tests := []struct {
		name     string
		value    string
		tag      string
		expected bool
	}{
		{"valid agent type", "llm", "agent_type", true},
		{"invalid agent type", "invalid", "agent_type", false},
		{"valid build target", "wasm", "build_target", true},
		{"invalid build target", "invalid", "build_target", false},
		{"valid status", "running", "status", true},
		{"invalid status", "invalid", "status", false},
		{"valid capability", "text processing", "capability", true},
		{"invalid capability", "", "capability", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := vm.ValidateValue(tt.value, tt.tag)
			if tt.expected && err != nil {
				t.Errorf("Expected %s to be valid for tag %s, got error: %v", tt.value, tt.tag, err)
			}
			if !tt.expected && err == nil {
				t.Errorf("Expected %s to be invalid for tag %s, but validation passed", tt.value, tt.tag)
			}
		})
	}
}

func TestValidationMiddleware_ValidateRequestBody(t *testing.T) {
	vm := NewValidationMiddleware()

	// Test valid JSON request
	validAgent := AgentCreateRequest{
		Name:        "Test Agent",
		Type:        "llm",
		BuildTarget: "wasm",
		Config:      map[string]interface{}{"key": "value"},
	}

	jsonData, _ := json.Marshal(validAgent)
	req := httptest.NewRequest("POST", "/api/agents", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	var target AgentCreateRequest
	err := vm.ValidateRequestBody(req, &target)
	if err != nil {
		t.Errorf("Expected valid request to pass validation, got error: %v", err)
	}

	// Test invalid JSON
	invalidJSON := `{"name": "test", "type": "llm", "invalid_field": true}`
	req = httptest.NewRequest("POST", "/api/agents", strings.NewReader(invalidJSON))
	req.Header.Set("Content-Type", "application/json")

	var target2 AgentCreateRequest
	err = vm.ValidateRequestBody(req, &target2)
	if err == nil {
		t.Error("Expected invalid JSON with unknown fields to fail validation")
	}
}

func TestValidationMiddleware_HTTPMiddleware(t *testing.T) {
	vm := NewValidationMiddleware()
	middleware := vm.CreateValidationMiddleware()

	// Test handler that should be called
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	wrappedHandler := middleware(handler)

	// Test GET request (should pass through)
	req := httptest.NewRequest("GET", "/api/agents", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected GET request to pass through, got status %d", w.Code)
	}

	// Test POST request with valid content type
	req = httptest.NewRequest("POST", "/api/agents", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected POST request with valid content type to pass through, got status %d", w.Code)
	}

	// Test POST request with invalid content type
	req = httptest.NewRequest("POST", "/api/agents", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "text/plain")
	w = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected POST request with invalid content type to fail, got status %d", w.Code)
	}
}

func TestValidationMiddleware_FormatValidationErrors(t *testing.T) {
	vm := NewValidationMiddleware()

	// Create an invalid struct to generate validation errors
	invalidAgent := AgentCreateRequest{
		Name: "", // Required field is empty
		Type: "invalid_type", // Invalid agent type
	}

	err := vm.ValidateStruct(invalidAgent)
	if err == nil {
		t.Fatal("Expected validation to fail")
	}

	errors := vm.FormatValidationErrors(err)
	if len(errors) == 0 {
		t.Error("Expected validation errors to be formatted")
	}

	// Check that errors have required fields
	for _, validationError := range errors {
		if validationError.Field == "" {
			t.Error("Expected validation error to have field name")
		}
		if validationError.Message == "" {
			t.Error("Expected validation error to have message")
		}
	}
}

func TestValidationMiddleware_ValidateAndRespond(t *testing.T) {
	vm := NewValidationMiddleware()

	// Test valid request
	validAgent := AgentCreateRequest{
		Name:        "Test Agent",
		Type:        "llm",
		BuildTarget: "wasm",
		Config:      map[string]interface{}{"key": "value"},
	}

	jsonData, _ := json.Marshal(validAgent)
	req := httptest.NewRequest("POST", "/api/agents", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	var target AgentCreateRequest
	success := vm.ValidateAndRespond(w, req, &target)
	if !success {
		t.Error("Expected valid request to succeed")
	}

	// Test invalid request
	invalidAgent := AgentCreateRequest{
		Name: "", // Required field is empty
		Type: "invalid_type", // Invalid agent type
	}

	jsonData, _ = json.Marshal(invalidAgent)
	req = httptest.NewRequest("POST", "/api/agents", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	var target2 AgentCreateRequest
	success = vm.ValidateAndRespond(w, req, &target2)
	if success {
		t.Error("Expected invalid request to fail")
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid request, got %d", w.Code)
	}

	// Check response format
	var response ValidationResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Errorf("Failed to decode validation response: %v", err)
	}

	if response.Success {
		t.Error("Expected response success to be false")
	}

	if len(response.Details) == 0 {
		t.Error("Expected validation details in response")
	}
}

func TestValidationModels_UserValidation(t *testing.T) {
	vm := NewValidationMiddleware()

	// Test valid user create request
	validUser := UserCreateRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
		Role:     "user",
	}

	err := vm.ValidateStruct(validUser)
	if err != nil {
		t.Errorf("Expected valid user to pass validation, got error: %v", err)
	}

	// Test invalid user create request
	invalidUser := UserCreateRequest{
		Username: "ab", // Too short
		Email:    "invalid-email", // Invalid email format
		Password: "123", // Too short
		Role:     "invalid_role", // Invalid role
	}

	err = vm.ValidateStruct(invalidUser)
	if err == nil {
		t.Error("Expected invalid user to fail validation")
	}
}

func TestValidationModels_WorkflowValidation(t *testing.T) {
	vm := NewValidationMiddleware()

	// Test valid workflow create request
	validWorkflow := WorkflowCreateRequest{
		AgentID:      "550e8400-e29b-41d4-a716-446655440000",
		TargetID:     "550e8400-e29b-41d4-a716-446655440001",
		CapabilityID: "550e8400-e29b-41d4-a716-446655440002",
		Priority:     5,
		Timeout:      300,
	}

	err := vm.ValidateStruct(validWorkflow)
	if err != nil {
		t.Errorf("Expected valid workflow to pass validation, got error: %v", err)
	}

	// Test invalid workflow create request
	invalidWorkflow := WorkflowCreateRequest{
		AgentID:      "invalid-uuid",
		TargetID:     "",
		CapabilityID: "invalid-uuid",
		Priority:     15, // Too high
		Timeout:      5000, // Too high
	}

	err = vm.ValidateStruct(invalidWorkflow)
	if err == nil {
		t.Error("Expected invalid workflow to fail validation")
	}
}
