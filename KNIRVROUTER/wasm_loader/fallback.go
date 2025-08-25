//go:build !wasmloader
// +build !wasmloader

// Fallback implementation when WASM is not enabled
package wasm_loader

import (
	"context"
	"fmt"
	"log"
)

// WASMKNIRVChain fallback implementation
type WASMKNIRVChain struct {
	initialized bool
}

// SkillInvocationRequest represents a skill invocation request
type SkillInvocationRequest struct {
	InvocationID string `json:"invocation_id"`
	AgentID      string `json:"agent_id"`
	NRNToken     string `json:"nrn_token"`
	SkillURI     string `json:"skill_uri"`
	Priority     string `json:"priority"`
	Timestamp    int64  `json:"timestamp"`
}

// SkillInvocationResponse represents a skill invocation response
type SkillInvocationResponse struct {
	InvocationID     string `json:"invocation_id"`
	Status           string `json:"status"`
	ErrorMessage     string `json:"error_message"`
	ExecutionTime    int64  `json:"execution_time"`
	MemoryUsed       int64  `json:"memory_used"`
	ConsensusReached bool   `json:"consensus_reached"`
	SkillData        string `json:"skill_data"`
}

// NewWASMKNIRVChain creates a fallback instance
func NewWASMKNIRVChain(wasmPath string) (*WASMKNIRVChain, error) {
	log.Printf("⚠️  WASM support not enabled, using fallback implementation")
	return &WASMKNIRVChain{initialized: false}, nil
}

// Initialize initializes the fallback
func (w *WASMKNIRVChain) Initialize() error {
	log.Printf("🔧 Initializing fallback KNIRVCHAIN (WASM disabled)...")
	w.initialized = true
	log.Printf("✅ Fallback KNIRVCHAIN initialized")
	return nil
}

// InvokeSkill provides a fallback skill invocation
func (w *WASMKNIRVChain) InvokeSkill(ctx context.Context, request *SkillInvocationRequest) (*SkillInvocationResponse, error) {
	if !w.initialized {
		return nil, fmt.Errorf("fallback KNIRVCHAIN not initialized")
	}

	log.Printf("🎯 Fallback skill invocation: %s (agent: %s)", request.InvocationID, request.AgentID)

	// Return a mock successful response
	return &SkillInvocationResponse{
		InvocationID:     request.InvocationID,
		Status:           "SUCCESS",
		ErrorMessage:     "",
		ExecutionTime:    25,
		MemoryUsed:       512,
		ConsensusReached: true,
		SkillData:        `{"skill_name":"fallback-skill","version":1,"note":"WASM not enabled"}`,
	}, nil
}

// GetSkillCount returns a fallback skill count
func (w *WASMKNIRVChain) GetSkillCount() (int, error) {
	if !w.initialized {
		return 0, fmt.Errorf("fallback KNIRVCHAIN not initialized")
	}
	return 2, nil // Mock count
}

// IsInitialized checks if initialized
func (w *WASMKNIRVChain) IsInitialized() bool {
	return w.initialized
}

// GetVersion returns fallback version
func (w *WASMKNIRVChain) GetVersion() (string, error) {
	return "1.0.0-fallback", nil
}

// GetBuildInfo returns fallback build info
func (w *WASMKNIRVChain) GetBuildInfo() (string, error) {
	return "KNIRVCHAIN Fallback - WASM support not enabled", nil
}

// Shutdown shuts down the fallback
func (w *WASMKNIRVChain) Shutdown() error {
	log.Printf("🛑 Shutting down fallback KNIRVCHAIN...")
	w.initialized = false
	log.Printf("✅ Fallback KNIRVCHAIN shutdown complete")
	return nil
}

// LoadWASMKNIRVChain loads the fallback implementation
func LoadWASMKNIRVChain(assetsDir string) (*WASMKNIRVChain, error) {
	return NewWASMKNIRVChain("")
}
