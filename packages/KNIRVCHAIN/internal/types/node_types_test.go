package types

import (
	"testing"
	"time"
)

func TestNewErrorNode(t *testing.T) {
	context := map[string]interface{}{
		"model_version": "gpt-4",
		"task_type":     "classification",
	}

	node, err := NewErrorNode("test-error-1", "classification", "hash123", "gpt-4", context)
	if err != nil {
		t.Fatalf("Failed to create error node: %v", err)
	}

	if node.ID != "test-error-1" {
		t.Errorf("Expected ID 'test-error-1', got '%s'", node.ID)
	}

	if node.ErrorType != "classification" {
		t.Errorf("Expected error type 'classification', got '%s'", node.ErrorType)
	}

	if node.FailureCount != 1 {
		t.Errorf("Expected failure count 1, got %d", node.FailureCount)
	}

	if node.Status != NodeStatusOpen {
		t.Errorf("Expected status Open, got %s", node.Status)
	}
}

func TestErrorNodeIncrementFailureCount(t *testing.T) {
	node := &ErrorNode{
		ID:            "test-error-1",
		FailureCount:  1,
	}

	node.IncrementFailureCount()

	if node.FailureCount != 2 {
		t.Errorf("Expected failure count 2, got %d", node.FailureCount)
	}
}

func TestErrorNodeUpdateStatus(t *testing.T) {
	node := &ErrorNode{
		ID:     "test-error-1",
		Status: NodeStatusOpen,
	}

	err := node.UpdateStatus(NodeStatusResolved)
	if err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	if node.Status != NodeStatusResolved {
		t.Errorf("Expected status Resolved, got %s", node.Status)
	}

	// Test invalid status
	err = node.UpdateStatus("invalid")
	if err == nil {
		t.Error("Expected error for invalid status")
	}
}

func TestNewSkillNode(t *testing.T) {
	loraPointer, _ := NewLoRAAdapterPointer("adapter1", "ipfs123", "gpt-4", 8, 16.0, []string{"q_proj"})
	performance := SkillPerformance{
		Accuracy:       0.85,
		Precision:      0.82,
		Recall:         0.80,
		F1Score:        0.81,
		TestCasesRun:   100,
		TestCasesPass:  85,
		AvgResponseTime: 120,
	}

	node, err := NewSkillNode("skill1", "Error resolution skill", "error1", loraPointer, "proof123", performance, "miner1", 1000)
	if err != nil {
		t.Fatalf("Failed to create skill node: %v", err)
	}

	if node.ID != "skill1" {
		t.Errorf("Expected ID 'skill1', got '%s'", node.ID)
	}

	if node.MinerAddress != "miner1" {
		t.Errorf("Expected miner 'miner1', got '%s'", node.MinerAddress)
	}
}

func TestSkillPerformanceValidate(t *testing.T) {
	// Valid performance
	perf := &SkillPerformance{
		Accuracy:       0.85,
		Precision:      0.82,
		Recall:         0.80,
		F1Score:        0.81,
		TestCasesRun:   100,
		TestCasesPass:  85,
		AvgResponseTime: 120,
	}

	err := perf.Validate()
	if err != nil {
		t.Errorf("Expected valid performance, got error: %v", err)
	}

	// Invalid accuracy
	perf.Accuracy = 1.5
	err = perf.Validate()
	if err == nil {
		t.Error("Expected error for invalid accuracy")
	}
	perf.Accuracy = 0.85 // Reset

	// Invalid test cases
	perf.TestCasesPass = 150
	err = perf.Validate()
	if err == nil {
		t.Error("Expected error for invalid test case counts")
	}
}

func TestNewContextNode(t *testing.T) {
	metadata := map[string]interface{}{
		"frequency": 100,
		"pattern":   "tool_usage",
	}

	node, err := NewContextNode("context1", "tool_use", "hash123", []string{"tool1", "tool2"}, metadata)
	if err != nil {
		t.Fatalf("Failed to create context node: %v", err)
	}

	if node.ID != "context1" {
		t.Errorf("Expected ID 'context1', got '%s'", node.ID)
	}

	if node.UsageFrequency != 1 {
		t.Errorf("Expected usage frequency 1, got %d", node.UsageFrequency)
	}
}

func TestContextNodeIncrementUsageFrequency(t *testing.T) {
	node := &ContextNode{
		ID:             "context1",
		UsageFrequency: 1,
	}

	node.IncrementUsageFrequency()

	if node.UsageFrequency != 2 {
		t.Errorf("Expected usage frequency 2, got %d", node.UsageFrequency)
	}
}

func TestNewCapabilityNode(t *testing.T) {
	mcpPointer, _ := NewMCPServerPointer("server1", "ws://server.com", "2024-11-05", []string{"tools"}, "none", "ipfs123")
	accessControl := AccessControlPolicy{
		AllowedAddresses: []string{},
		RequiredNRN:      0,
		RateLimit:        100,
	}

	node, err := NewCapabilityNode("cap1", "context1", mcpPointer, CapabilityTypeTool, accessControl, "minter1", 50)
	if err != nil {
		t.Fatalf("Failed to create capability node: %v", err)
	}

	if node.ID != "cap1" {
		t.Errorf("Expected ID 'cap1', got '%s'", node.ID)
	}

	if node.NRNCost != 50 {
		t.Errorf("Expected NRN cost 50, got %d", node.NRNCost)
	}
}

func TestNewIdeaNode(t *testing.T) {
	novelty := NoveltyScore{
		Score:         0.8,
		SimilarIdeas:  2,
		Uniqueness:    0.7,
		AssessmentBy:  "assessor1",
		AssessedAt:    time.Now().Unix(),
	}

	node, err := NewIdeaNode("idea1", "insight", "hash123", "nim1", novelty, []string{"dep1"})
	if err != nil {
		t.Fatalf("Failed to create idea node: %v", err)
	}

	if node.ID != "idea1" {
		t.Errorf("Expected ID 'idea1', got '%s'", node.ID)
	}

	if node.Novelty.Score != 0.8 {
		t.Errorf("Expected novelty score 0.8, got %.2f", node.Novelty.Score)
	}
}

func TestNoveltyScoreValidate(t *testing.T) {
	novelty := &NoveltyScore{
		Score:         0.8,
		SimilarIdeas:  2,
		Uniqueness:    0.7,
		AssessmentBy:  "assessor1",
		AssessedAt:    time.Now().Unix(),
	}

	err := novelty.Validate()
	if err != nil {
		t.Errorf("Expected valid novelty score, got error: %v", err)
	}

	// Invalid score
	novelty.Score = 1.5
	err = novelty.Validate()
	if err == nil {
		t.Error("Expected error for invalid score")
	}
}

func TestNewPropertyNode(t *testing.T) {
	nftPointer, _ := NewInferenceNFTPointer("token1", "0x123", "metadata.json", "license")
	royalties := RoyaltyStructure{
		OriginNIMShare: 50,
		NetworkShare:   10,
		DependencyShares: map[string]uint8{
			"dep1": 20,
			"dep2": 20,
		},
	}

	node, err := NewPropertyNode("prop1", "idea1", nftPointer, IPTypePatent, royalties, "maker1")
	if err != nil {
		t.Fatalf("Failed to create property node: %v", err)
	}

	if node.ID != "prop1" {
		t.Errorf("Expected ID 'prop1', got '%s'", node.ID)
	}

	if node.Ownership.CurrentOwner != "maker1" {
		t.Errorf("Expected owner 'maker1', got '%s'", node.Ownership.CurrentOwner)
	}
}

func TestPropertyNodeTransferOwnership(t *testing.T) {
	royalties := RoyaltyStructure{
		OriginNIMShare: 50,
		NetworkShare:   10,
	}

	node := &PropertyNode{
		ID: "prop1",
		Ownership: OwnershipRecord{
			CurrentOwner: "owner1",
			History:      []OwnershipEntry{},
		},
		Royalties: royalties,
	}

	err := node.TransferOwnership("owner2", 1000)
	if err != nil {
		t.Fatalf("Failed to transfer ownership: %v", err)
	}

	if node.Ownership.CurrentOwner != "owner2" {
		t.Errorf("Expected owner 'owner2', got '%s'", node.Ownership.CurrentOwner)
	}

	if len(node.Ownership.History) != 1 {
		t.Errorf("Expected 1 ownership entry, got %d", len(node.Ownership.History))
	}
}

func TestOwnershipRecordAddEntry(t *testing.T) {
	record := &OwnershipRecord{
		CurrentOwner: "owner1",
		History:      []OwnershipEntry{},
	}

	record.AddEntry("owner2", 1000)

	if record.CurrentOwner != "owner2" {
		t.Errorf("Expected current owner 'owner2', got '%s'", record.CurrentOwner)
	}

	if len(record.History) != 1 {
		t.Errorf("Expected 1 history entry, got %d", len(record.History))
	}

	if record.History[0].PriceNRN != 1000 {
		t.Errorf("Expected price 1000, got %d", record.History[0].PriceNRN)
	}
}