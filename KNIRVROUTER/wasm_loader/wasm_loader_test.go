package wasm_loader

import (
	"context"
	"testing"
	"time"
)

func TestNewWASMKNIRVChain(t *testing.T) {
	// Test creating a new WASM KNIRVCHAIN instance
	wasmChain, err := NewWASMKNIRVChain("test/path/knirvchain.wasm")
	if err != nil {
		t.Fatalf("Failed to create WASM KNIRVCHAIN: %v", err)
	}

	if wasmChain == nil {
		t.Fatal("WASM KNIRVCHAIN instance is nil")
	}

	// Note: wasmPath field is only available in the wasmloader build

	if wasmChain.initialized {
		t.Error("WASM KNIRVCHAIN should not be initialized on creation")
	}
}

func TestWASMKNIRVChainInitialize(t *testing.T) {
	wasmChain, err := NewWASMKNIRVChain("test/path/knirvchain.wasm")
	if err != nil {
		t.Fatalf("Failed to create WASM KNIRVCHAIN: %v", err)
	}

	// Test initialization
	err = wasmChain.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize WASM KNIRVCHAIN: %v", err)
	}

	if !wasmChain.IsInitialized() {
		t.Error("WASM KNIRVCHAIN should be initialized after Initialize() call")
	}

	// Test double initialization (should not error)
	err = wasmChain.Initialize()
	if err != nil {
		t.Fatalf("Double initialization should not fail: %v", err)
	}
}

func TestWASMKNIRVChainInvokeSkill(t *testing.T) {
	wasmChain, err := NewWASMKNIRVChain("test/path/knirvchain.wasm")
	if err != nil {
		t.Fatalf("Failed to create WASM KNIRVCHAIN: %v", err)
	}

	err = wasmChain.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize WASM KNIRVCHAIN: %v", err)
	}

	// Test skill invocation
	request := &SkillInvocationRequest{
		InvocationID: "test-invocation-001",
		AgentID:      "test-agent-123",
		NRNToken:     "test-nrn-token-abcdef123456789012345678901234567890",
		SkillURI:     "knirv://skill/javascript-type-checker-v1",
		Priority:     "high",
		Timestamp:    time.Now().Unix(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := wasmChain.InvokeSkill(ctx, request)
	if err != nil {
		t.Fatalf("Failed to invoke skill: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	if response.InvocationID != request.InvocationID {
		t.Errorf("Expected invocation ID '%s', got '%s'", request.InvocationID, response.InvocationID)
	}

	if response.Status != "SUCCESS" {
		t.Errorf("Expected status 'SUCCESS', got '%s'", response.Status)
	}

	if response.ExecutionTime <= 0 {
		t.Error("Execution time should be greater than 0")
	}

	if response.SkillData == "" {
		t.Error("Skill data should not be empty")
	}
}

func TestWASMKNIRVChainInvokeSkillNotInitialized(t *testing.T) {
	wasmChain, err := NewWASMKNIRVChain("test/path/knirvchain.wasm")
	if err != nil {
		t.Fatalf("Failed to create WASM KNIRVCHAIN: %v", err)
	}

	// Don't initialize, try to invoke skill
	request := &SkillInvocationRequest{
		InvocationID: "test-invocation-002",
		AgentID:      "test-agent-456",
		NRNToken:     "test-nrn-token-abcdef123456789012345678901234567890",
		SkillURI:     "knirv://skill/syntax-error-fixer-v2",
		Priority:     "normal",
		Timestamp:    time.Now().Unix(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = wasmChain.InvokeSkill(ctx, request)
	if err == nil {
		t.Error("Expected error when invoking skill on uninitialized WASM chain")
	}

	// Error message varies between implementations
	if err.Error() != "WASM KNIRVCHAIN not initialized" && err.Error() != "fallback KNIRVCHAIN not initialized" {
		t.Errorf("Expected initialization error, got '%s'", err.Error())
	}
}

func TestWASMKNIRVChainGetSkillCount(t *testing.T) {
	wasmChain, err := NewWASMKNIRVChain("test/path/knirvchain.wasm")
	if err != nil {
		t.Fatalf("Failed to create WASM KNIRVCHAIN: %v", err)
	}

	err = wasmChain.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize WASM KNIRVCHAIN: %v", err)
	}

	count, err := wasmChain.GetSkillCount()
	if err != nil {
		t.Fatalf("Failed to get skill count: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected skill count 2, got %d", count)
	}
}

func TestWASMKNIRVChainGetVersion(t *testing.T) {
	wasmChain, err := NewWASMKNIRVChain("test/path/knirvchain.wasm")
	if err != nil {
		t.Fatalf("Failed to create WASM KNIRVCHAIN: %v", err)
	}

	version, err := wasmChain.GetVersion()
	if err != nil {
		t.Fatalf("Failed to get version: %v", err)
	}

	// Version varies between implementations
	if version != "1.0.0-wasm-mock" && version != "1.0.0-fallback" {
		t.Errorf("Expected valid version, got '%s'", version)
	}
}

func TestWASMKNIRVChainGetBuildInfo(t *testing.T) {
	wasmChain, err := NewWASMKNIRVChain("test/path/knirvchain.wasm")
	if err != nil {
		t.Fatalf("Failed to create WASM KNIRVCHAIN: %v", err)
	}

	buildInfo, err := wasmChain.GetBuildInfo()
	if err != nil {
		t.Fatalf("Failed to get build info: %v", err)
	}

	// Build info varies between implementations
	if buildInfo != "KNIRVCHAIN WASM Mock - Revolutionary Embedded Skill Execution Engine" &&
		buildInfo != "KNIRVCHAIN Fallback - WASM support not enabled" {
		t.Errorf("Expected valid build info, got '%s'", buildInfo)
	}
}

func TestWASMKNIRVChainShutdown(t *testing.T) {
	wasmChain, err := NewWASMKNIRVChain("test/path/knirvchain.wasm")
	if err != nil {
		t.Fatalf("Failed to create WASM KNIRVCHAIN: %v", err)
	}

	err = wasmChain.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize WASM KNIRVCHAIN: %v", err)
	}

	// Test shutdown
	err = wasmChain.Shutdown()
	if err != nil {
		t.Fatalf("Failed to shutdown WASM KNIRVCHAIN: %v", err)
	}

	if wasmChain.IsInitialized() {
		t.Error("WASM KNIRVCHAIN should not be initialized after shutdown")
	}

	// Test double shutdown (should not error)
	err = wasmChain.Shutdown()
	if err != nil {
		t.Fatalf("Double shutdown should not fail: %v", err)
	}
}

func TestLoadWASMKNIRVChain(t *testing.T) {
	// Test loading WASM KNIRVCHAIN from assets directory
	wasmChain, err := LoadWASMKNIRVChain("test/assets")
	if err != nil {
		t.Fatalf("Failed to load WASM KNIRVCHAIN: %v", err)
	}

	if wasmChain == nil {
		t.Fatal("Loaded WASM KNIRVCHAIN instance is nil")
	}

	// Note: wasmPath field is only available in the wasmloader build
}

func TestSkillInvocationRequestValidation(t *testing.T) {
	wasmChain, err := NewWASMKNIRVChain("test/path/knirvchain.wasm")
	if err != nil {
		t.Fatalf("Failed to create WASM KNIRVCHAIN: %v", err)
	}

	err = wasmChain.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize WASM KNIRVCHAIN: %v", err)
	}

	// Test with short NRN token (should still work in mock)
	request := &SkillInvocationRequest{
		InvocationID: "test-invocation-003",
		AgentID:      "test-agent-789",
		NRNToken:     "short-token", // Less than 32 characters
		SkillURI:     "knirv://skill/test-skill-v1",
		Priority:     "low",
		Timestamp:    time.Now().Unix(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := wasmChain.InvokeSkill(ctx, request)
	if err != nil {
		t.Fatalf("Failed to invoke skill with short token: %v", err)
	}

	// In mock implementation, it should still succeed
	if response.Status != "SUCCESS" {
		t.Errorf("Expected status 'SUCCESS' even with short token, got '%s'", response.Status)
	}
}
