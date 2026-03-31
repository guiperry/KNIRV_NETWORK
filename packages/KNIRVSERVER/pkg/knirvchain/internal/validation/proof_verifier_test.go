package validation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"KNIRVCHAIN/internal/types"
)

func TestProofVerifier_ValidateSkill(t *testing.T) {
	pv := NewProofVerifier("http://localhost:8080")

	errorNode := &types.ErrorNode{
		ID:             "test_error_001",
		ErrorType:      "RuntimeError",
		ErrorSignature: "sig_001",
		ModelOrigin:    "gpt-4",
		Context: map[string]interface{}{
			"language":  "Go",
			"framework": "Gin",
		},
		Status: types.NodeStatusOpen,
	}

	loraAdapter := &types.LoRAAdapterPointer{
		AdapterID: "adapter_001",
		Rank:      8,
		Alpha:     0.5,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := pv.ValidateSkill(ctx, errorNode, loraAdapter)
	if err != nil {
		t.Fatalf("ValidateSkill failed: %v", err)
	}

	if !response.IsValid {
		t.Error("Expected validation to be valid")
	}

	if response.Performance.Accuracy <= 0 || response.Performance.Accuracy > 1.0 {
		t.Errorf("Invalid accuracy: %v", response.Performance.Accuracy)
	}

	if response.ValidationProof == "" {
		t.Error("Expected validation proof to be set")
	}
}

func TestProofVerifier_GenerateTestCases(t *testing.T) {
	pv := NewProofVerifier("http://localhost:8080")

	errorNode := &types.ErrorNode{
		ID:             "test_error_002",
		ErrorType:      "TypeError",
		ErrorSignature: "sig_002",
		ModelOrigin:    "claude-3",
		Context:        map[string]interface{}{},
	}

	testCases, err := pv.generateTestCases(errorNode)
	if err != nil {
		t.Fatalf("generateTestCases failed: %v", err)
	}

	if len(testCases) == 0 {
		t.Error("Expected test cases to be generated")
	}

	for _, tc := range testCases {
		if tc.ID == "" {
			t.Error("Test case ID should not be empty")
		}
		if tc.Input == "" {
			t.Error("Test case input should not be empty")
		}
	}
}

func TestProofVerifier_VerifyValidationResponse(t *testing.T) {
	pv := NewProofVerifier("http://localhost:8080")

	request := &ValidationRequest{
		ErrorNode: &types.ErrorNode{
			ID: "test_error_003",
		},
		LoRAAdapter: &types.LoRAAdapterPointer{
			AdapterID: "adapter_002",
		},
		TestCases: []TestCase{
			{ID: "test_1"},
			{ID: "test_2"},
		},
		RequestID:   "req_001",
		RequestedAt: time.Now().Unix(),
	}

	response := &ValidationResponse{
		RequestID: "req_001",
		IsValid:   true,
		Performance: types.SkillPerformance{
			Accuracy:      0.9,
			Precision:     0.85,
			Recall:        0.88,
			F1Score:       0.86,
			TestCasesRun:  2,
			TestCasesPass: 2,
		},
		TestResults: []TestResult{
			{TestCaseID: "test_1", Passed: true},
			{TestCaseID: "test_2", Passed: true},
		},
		ValidatedAt:     time.Now().Unix(),
		ValidationProof: "",
	}

	proofData := fmt.Sprintf("proof:%s:%t:%d", request.RequestID, response.IsValid, response.ValidatedAt)
	proofHash := sha256.Sum256([]byte(proofData))
	response.ValidationProof = hex.EncodeToString(proofHash[:])

	err := pv.verifyValidationResponse(request, response)
	if err != nil {
		t.Fatalf("verifyValidationResponse failed: %v", err)
	}
}

func TestProofVerifier_VerifyValidationResponse_RequestIDMismatch(t *testing.T) {
	pv := NewProofVerifier("http://localhost:8080")

	request := &ValidationRequest{
		RequestID: "req_001",
		TestCases: []TestCase{{ID: "test_1"}},
	}

	response := &ValidationResponse{
		RequestID:   "req_002",
		IsValid:     true,
		Performance: types.SkillPerformance{TestCasesRun: 1, TestCasesPass: 1},
		TestResults: []TestResult{{TestCaseID: "test_1", Passed: true}},
	}

	err := pv.verifyValidationResponse(request, response)
	if err == nil {
		t.Error("Expected error for request ID mismatch")
	}
}

func TestProofVerifier_VerifyValidationResponse_InvalidPerformance(t *testing.T) {
	pv := NewProofVerifier("http://localhost:8080")

	request := &ValidationRequest{
		RequestID: "req_001",
		TestCases: []TestCase{{ID: "test_1"}},
	}

	response := &ValidationResponse{
		RequestID:   "req_001",
		IsValid:     true,
		Performance: types.SkillPerformance{TestCasesRun: 0},
		TestResults: []TestResult{},
	}

	err := pv.verifyValidationResponse(request, response)
	if err == nil {
		t.Error("Expected error for invalid performance")
	}
}

func TestProofVerifier_VerifyProof(t *testing.T) {
	pv := NewProofVerifier("http://localhost:8080")

	errorNodeID := "test_error_004"
	adapterID := "adapter_003"

	requestID := pv.generateRequestID(errorNodeID, adapterID)
	validatedAt := time.Now().Unix()
	proofData := fmt.Sprintf("proof:%s:%t:%d", requestID, true, validatedAt)
	proofHash := sha256.Sum256([]byte(proofData))
	validProof := hex.EncodeToString(proofHash[:])

	performance := types.SkillPerformance{
		Accuracy: 0.9,
	}

	err := pv.VerifyProof(errorNodeID, adapterID, validProof, performance)
	if err != nil {
		t.Fatalf("VerifyProof failed for valid proof: %v", err)
	}
}

func TestProofVerifier_VerifyProof_InvalidProof(t *testing.T) {
	pv := NewProofVerifier("http://localhost:8080")

	errorNodeID := "test_error_005"
	adapterID := "adapter_004"
	invalidProof := "invalid_proof_hash_12345"

	performance := types.SkillPerformance{
		Accuracy: 0.9,
	}

	err := pv.VerifyProof(errorNodeID, adapterID, invalidProof, performance)
	if err == nil {
		t.Error("Expected error for invalid proof")
	}
}

func TestProofVerifier_ValidateCapabilityProposal(t *testing.T) {
	pv := NewProofVerifier("http://localhost:8080")

	capabilityNode := &types.CapabilityNode{
		MCPPointer: &types.MCPServerPointer{
			ServerID:    "server_001",
			EndpointURI: "http://localhost:9000",
		},
	}

	result, err := pv.ValidateCapabilityProposal(capabilityNode)
	if err != nil {
		t.Fatalf("ValidateCapabilityProposal failed: %v", err)
	}

	if !result.IsValid {
		t.Error("Expected capability proposal to be valid")
	}

	if result.ComplianceScore <= 0 || result.ComplianceScore > 1.0 {
		t.Errorf("Invalid compliance score: %v", result.ComplianceScore)
	}
}

func TestProofVerifier_GenerateRequestID(t *testing.T) {
	pv := NewProofVerifier("http://localhost:8080")

	requestID1 := pv.generateRequestID("error_001", "adapter_001")
	requestID2 := pv.generateRequestID("error_001", "adapter_001")

	if requestID1 == "" {
		t.Error("Expected request ID to be generated")
	}

	if requestID1 != requestID2 {
		t.Log("Note: Different timestamps generate different IDs (expected)")
	}
}
