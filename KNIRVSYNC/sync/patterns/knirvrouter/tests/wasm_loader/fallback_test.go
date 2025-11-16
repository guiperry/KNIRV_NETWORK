//go:build !wasmloader
// +build !wasmloader

package wasm_loader

import (
	"context"
	"testing"
	"time"
)

func TestFallbackNewWASMKNIRVChain(t *testing.T) {
	// Test creating a new fallback WASM KNIRVCHAIN instance
	wasmChain, err := NewWASMKNIRVChain("test/path/knirvchain.wasm")
	if err != nil {
		t.Fatalf("Failed to create fallback WASM KNIRVCHAIN: %v", err)
	}

	if wasmChain == nil {
		t.Fatal("Fallback WASM KNIRVCHAIN instance is nil")
	}

	if wasmChain.initialized {
		t.Error("Fallback WASM KNIRVCHAIN should not be initialized on creation")
	}
}

func TestFallbackWASMKNIRVChainInitialize(t *testing.T) {
	wasmChain, err := NewWASMKNIRVChain("test/path/knirvchain.wasm")
	if err != nil {
		t.Fatalf("Failed to create fallback WASM KNIRVCHAIN: %v", err)
	}

	// Test initialization
	err = wasmChain.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize fallback WASM KNIRVCHAIN: %v", err)
	}

	if !wasmChain.IsInitialized() {
		t.Error("Fallback WASM KNIRVCHAIN should be initialized after Initialize() call")
	}
}

func TestFallbackWASMKNIRVChainInvokeSkill(t *testing.T) {
	wasmChain, err := NewWASMKNIRVChain("test/path/knirvchain.wasm")
	if err != nil {
		t.Fatalf("Failed to create fallback WASM KNIRVCHAIN: %v", err)
	}

	err = wasmChain.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize fallback WASM KNIRVCHAIN: %v", err)
	}

	// Test skill invocation
	request := &SkillInvocationRequest{
		InvocationID: "fallback-test-001",
		AgentID:      "fallback-agent-123",
		NRNToken:     "fallback-nrn-token-abcdef123456789012345678901234567890",
		SkillURI:     "knirv://skill/fallback-skill-v1",
		Priority:     "normal",
		Timestamp:    time.Now().Unix(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := wasmChain.InvokeSkill(ctx, request)
	if err != nil {
		t.Fatalf("Failed to invoke skill in fallback: %v", err)
	}

	if response == nil {
		t.Fatal("Fallback response is nil")
	}

	if response.InvocationID != request.InvocationID {
		t.Errorf("Expected invocation ID '%s', got '%s'", request.InvocationID, response.InvocationID)
	}

	if response.Status != "SUCCESS" {
		t.Errorf("Expected status 'SUCCESS', got '%s'", response.Status)
	}

	// Check that it's clearly a fallback response
	if response.SkillData != `{"skill_name":"fallback-skill","version":1,"note":"WASM not enabled"}` {
		t.Errorf("Expected fallback skill data, got '%s'", response.SkillData)
	}
}

func TestFallbackWASMKNIRVChainGetSkillCount(t *testing.T) {
	wasmChain, err := NewWASMKNIRVChain("test/path/knirvchain.wasm")
	if err != nil {
		t.Fatalf("Failed to create fallback WASM KNIRVCHAIN: %v", err)
	}

	err = wasmChain.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize fallback WASM KNIRVCHAIN: %v", err)
	}

	count, err := wasmChain.GetSkillCount()
	if err != nil {
		t.Fatalf("Failed to get skill count in fallback: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected fallback skill count 2, got %d", count)
	}
}

func TestFallbackWASMKNIRVChainGetVersion(t *testing.T) {
	wasmChain, err := NewWASMKNIRVChain("test/path/knirvchain.wasm")
	if err != nil {
		t.Fatalf("Failed to create fallback WASM KNIRVCHAIN: %v", err)
	}

	version, err := wasmChain.GetVersion()
	if err != nil {
		t.Fatalf("Failed to get version in fallback: %v", err)
	}

	expectedVersion := "1.0.0-fallback"
	if version != expectedVersion {
		t.Errorf("Expected fallback version '%s', got '%s'", expectedVersion, version)
	}
}

func TestFallbackWASMKNIRVChainGetBuildInfo(t *testing.T) {
	wasmChain, err := NewWASMKNIRVChain("test/path/knirvchain.wasm")
	if err != nil {
		t.Fatalf("Failed to create fallback WASM KNIRVCHAIN: %v", err)
	}

	buildInfo, err := wasmChain.GetBuildInfo()
	if err != nil {
		t.Fatalf("Failed to get build info in fallback: %v", err)
	}

	expectedBuildInfo := "KNIRVCHAIN Fallback - WASM support not enabled"
	if buildInfo != expectedBuildInfo {
		t.Errorf("Expected fallback build info '%s', got '%s'", expectedBuildInfo, buildInfo)
	}
}

func TestFallbackWASMKNIRVChainShutdown(t *testing.T) {
	wasmChain, err := NewWASMKNIRVChain("test/path/knirvchain.wasm")
	if err != nil {
		t.Fatalf("Failed to create fallback WASM KNIRVCHAIN: %v", err)
	}

	err = wasmChain.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize fallback WASM KNIRVCHAIN: %v", err)
	}

	// Test shutdown
	err = wasmChain.Shutdown()
	if err != nil {
		t.Fatalf("Failed to shutdown fallback WASM KNIRVCHAIN: %v", err)
	}

	if wasmChain.IsInitialized() {
		t.Error("Fallback WASM KNIRVCHAIN should not be initialized after shutdown")
	}
}

func TestFallbackLoadWASMKNIRVChain(t *testing.T) {
	// Test loading fallback WASM KNIRVCHAIN
	wasmChain, err := LoadWASMKNIRVChain("test/assets")
	if err != nil {
		t.Fatalf("Failed to load fallback WASM KNIRVCHAIN: %v", err)
	}

	if wasmChain == nil {
		t.Fatal("Loaded fallback WASM KNIRVCHAIN instance is nil")
	}
}
