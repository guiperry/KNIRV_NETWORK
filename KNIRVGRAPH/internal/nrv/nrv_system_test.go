package nrv

import (
	"fmt"
	"testing"
)

func TestNewNRVSystem(t *testing.T) {
	peerID := "test-peer-123"
	nrvSystem := NewNRVSystem(peerID, nil)

	if nrvSystem == nil {
		t.Fatal("Expected non-nil NRV system")
	}

	// Test that the system was created with default config
	if nrvSystem.config == nil {
		t.Error("Expected config to be initialized")
	}

	if nrvSystem.config.MaxVectors != 10000 {
		t.Errorf("Expected default MaxVectors 10000, got %d", nrvSystem.config.MaxVectors)
	}
}

func TestNRVSystemCreateVector(t *testing.T) {
	nrvSystem := NewNRVSystem("test-peer", nil)

	targetHash := "test-hash-123"
	coordinates := []float64{1.0, 2.0, 3.0}
	metadata := map[string]interface{}{
		"test": "data",
		"type": "test-vector",
	}

	vector, err := nrvSystem.CreateVector(targetHash, coordinates, metadata)
	if err != nil {
		t.Fatalf("Failed to create vector: %v", err)
	}

	if vector == nil {
		t.Fatal("Expected non-nil vector")
	}

	if vector.TargetHash != targetHash {
		t.Errorf("Expected target hash %s, got %s", targetHash, vector.TargetHash)
	}

	if len(vector.Coordinates) != len(coordinates) {
		t.Errorf("Expected %d coordinates, got %d", len(coordinates), len(vector.Coordinates))
	}

	if vector.SourcePeer != "test-peer" {
		t.Errorf("Expected source peer 'test-peer', got %s", vector.SourcePeer)
	}

	if vector.Confidence != 1.0 {
		t.Errorf("Expected initial confidence 1.0, got %f", vector.Confidence)
	}

	if len(vector.Signatures) != 1 {
		t.Errorf("Expected 1 signature, got %d", len(vector.Signatures))
	}
}

func TestNRVSystemResolveTarget(t *testing.T) {
	nrvSystem := NewNRVSystem("test-peer", nil)

	// Create a vector first
	targetHash := "test-hash-456"
	coordinates := []float64{4.0, 5.0, 6.0}
	metadata := map[string]interface{}{"type": "test"}

	vector, err := nrvSystem.CreateVector(targetHash, coordinates, metadata)
	if err != nil {
		t.Fatalf("Failed to create vector: %v", err)
	}

	// Resolve the target
	vectors, err := nrvSystem.ResolveTarget(targetHash)
	if err != nil {
		t.Fatalf("Failed to resolve target: %v", err)
	}

	if len(vectors) != 1 {
		t.Errorf("Expected 1 vector, got %d", len(vectors))
	}

	if vectors[0].ID != vector.ID {
		t.Errorf("Expected vector ID %s, got %s", vector.ID, vectors[0].ID)
	}
}

func TestNRVSystemCreateErrorNode(t *testing.T) {
	nrvSystem := NewNRVSystem("test-peer", nil)

	errorType := "network_error"
	description := "Connection timeout"
	context := map[string]interface{}{
		"host": "example.com",
		"port": 8080,
	}
	severity := 5

	errorNode, err := nrvSystem.CreateErrorNode(errorType, description, context, severity)
	if err != nil {
		t.Fatalf("Failed to create error node: %v", err)
	}

	if errorNode == nil {
		t.Fatal("Expected non-nil error node")
	}

	if errorNode.ErrorType != errorType {
		t.Errorf("Expected error type %s, got %s", errorType, errorNode.ErrorType)
	}

	if errorNode.Description != description {
		t.Errorf("Expected description %s, got %s", description, errorNode.Description)
	}

	if errorNode.Severity != severity {
		t.Errorf("Expected severity %d, got %d", severity, errorNode.Severity)
	}
}

func TestNRVSystemCreateSkillNode(t *testing.T) {
	nrvSystem := NewNRVSystem("test-peer", nil)

	skillType := "network_repair"
	capabilities := []string{"connection_timeout", "dns_resolution", "general"}
	requirements := map[string]interface{}{
		"min_memory": 512,
		"network":    true,
	}

	skillNode, err := nrvSystem.CreateSkillNode(skillType, capabilities, requirements)
	if err != nil {
		t.Fatalf("Failed to create skill node: %v", err)
	}

	if skillNode == nil {
		t.Fatal("Expected non-nil skill node")
	}

	if skillNode.SkillType != skillType {
		t.Errorf("Expected skill type %s, got %s", skillType, skillNode.SkillType)
	}

	if len(skillNode.Capabilities) != len(capabilities) {
		t.Errorf("Expected %d capabilities, got %d", len(capabilities), len(skillNode.Capabilities))
	}

	if skillNode.Performance == nil {
		t.Error("Expected performance metrics to be initialized")
	}

	if skillNode.Validation == nil {
		t.Error("Expected validation status to be initialized")
	}

	if skillNode.Performance.SuccessRate != 0.0 {
		t.Errorf("Expected initial success rate 0.0, got %f", skillNode.Performance.SuccessRate)
	}
}

func TestNRVSystemGetSkillsForErrorType(t *testing.T) {
	nrvSystem := NewNRVSystem("test-peer", nil)

	// Create skills with different capabilities
	skill1, err := nrvSystem.CreateSkillNode("network_repair", []string{"connection_timeout", "dns_resolution"}, nil)
	if err != nil {
		t.Fatalf("Failed to create skill1: %v", err)
	}

	skill2, err := nrvSystem.CreateSkillNode("general_repair", []string{"general", "system_error"}, nil)
	if err != nil {
		t.Fatalf("Failed to create skill2: %v", err)
	}

	_, err = nrvSystem.CreateSkillNode("database_repair", []string{"database_error", "query_timeout"}, nil)
	if err != nil {
		t.Fatalf("Failed to create skill3: %v", err)
	}

	// Test finding skills for connection_timeout
	skills, err := nrvSystem.GetSkillsForErrorType("connection_timeout")
	if err != nil {
		t.Fatalf("Failed to get skills for error type: %v", err)
	}

	if len(skills) != 1 {
		t.Errorf("Expected 1 skill for connection_timeout, got %d", len(skills))
	}

	if skills[0].ID != skill1.ID {
		t.Errorf("Expected skill1 ID %s, got %s", skill1.ID, skills[0].ID)
	}

	// Test finding skills for general error type
	generalSkills, err := nrvSystem.GetSkillsForErrorType("unknown_error")
	if err != nil {
		t.Fatalf("Failed to get skills for general error: %v", err)
	}

	// Should find skill2 because it has "general" capability
	foundGeneral := false
	for _, skill := range generalSkills {
		if skill.ID == skill2.ID {
			foundGeneral = true
			break
		}
	}

	if !foundGeneral {
		t.Error("Expected to find general skill for unknown error type")
	}
}

func TestNRVSystemGetAllVectors(t *testing.T) {
	nrvSystem := NewNRVSystem("test-peer", nil)

	// Initially should have no vectors
	vectors := nrvSystem.GetAllVectors()
	if len(vectors) != 0 {
		t.Errorf("Expected 0 vectors initially, got %d", len(vectors))
	}

	// Create some vectors
	vector1, err := nrvSystem.CreateVector("hash1", []float64{1.0, 2.0}, map[string]interface{}{"type": "test1"})
	if err != nil {
		t.Fatalf("Failed to create vector1: %v", err)
	}

	vector2, err := nrvSystem.CreateVector("hash2", []float64{3.0, 4.0}, map[string]interface{}{"type": "test2"})
	if err != nil {
		t.Fatalf("Failed to create vector2: %v", err)
	}

	// Should now have 2 vectors
	vectors = nrvSystem.GetAllVectors()
	if len(vectors) != 2 {
		t.Errorf("Expected 2 vectors, got %d", len(vectors))
	}

	// Verify vectors are present
	vectorIDs := make(map[string]bool)
	for _, vector := range vectors {
		vectorIDs[vector.ID] = true
	}

	if !vectorIDs[vector1.ID] {
		t.Error("Expected vector1 to be in all vectors list")
	}

	if !vectorIDs[vector2.ID] {
		t.Error("Expected vector2 to be in all vectors list")
	}
}

func TestNRVSystemGetAllErrorNodes(t *testing.T) {
	nrvSystem := NewNRVSystem("test-peer", nil)

	// Initially should have no error nodes
	errorNodes := nrvSystem.GetAllErrorNodes()
	if len(errorNodes) != 0 {
		t.Errorf("Expected 0 error nodes initially, got %d", len(errorNodes))
	}

	// Create some error nodes
	error1, err := nrvSystem.CreateErrorNode("network_error", "Connection failed", map[string]interface{}{"host": "example.com"}, 5)
	if err != nil {
		t.Fatalf("Failed to create error1: %v", err)
	}

	error2, err := nrvSystem.CreateErrorNode("database_error", "Query timeout", map[string]interface{}{"query": "SELECT *"}, 3)
	if err != nil {
		t.Fatalf("Failed to create error2: %v", err)
	}

	// Should now have 2 error nodes
	errorNodes = nrvSystem.GetAllErrorNodes()
	if len(errorNodes) != 2 {
		t.Errorf("Expected 2 error nodes, got %d", len(errorNodes))
	}

	// Verify error nodes are present
	errorIDs := make(map[string]bool)
	for _, errorNode := range errorNodes {
		errorIDs[errorNode.ID] = true
	}

	if !errorIDs[error1.ID] {
		t.Error("Expected error1 to be in all error nodes list")
	}

	if !errorIDs[error2.ID] {
		t.Error("Expected error2 to be in all error nodes list")
	}
}

func TestNRVSystemGetAllSkillNodes(t *testing.T) {
	nrvSystem := NewNRVSystem("test-peer", nil)

	// Initially should have no skill nodes
	skillNodes := nrvSystem.GetAllSkillNodes()
	if len(skillNodes) != 0 {
		t.Errorf("Expected 0 skill nodes initially, got %d", len(skillNodes))
	}

	// Create some skill nodes
	skill1, err := nrvSystem.CreateSkillNode("network_repair", []string{"connection_timeout"}, nil)
	if err != nil {
		t.Fatalf("Failed to create skill1: %v", err)
	}

	skill2, err := nrvSystem.CreateSkillNode("database_repair", []string{"query_timeout"}, nil)
	if err != nil {
		t.Fatalf("Failed to create skill2: %v", err)
	}

	// Should now have 2 skill nodes
	skillNodes = nrvSystem.GetAllSkillNodes()
	if len(skillNodes) != 2 {
		t.Errorf("Expected 2 skill nodes, got %d", len(skillNodes))
	}

	// Verify skill nodes are present
	skillIDs := make(map[string]bool)
	for _, skillNode := range skillNodes {
		skillIDs[skillNode.ID] = true
	}

	if !skillIDs[skill1.ID] {
		t.Error("Expected skill1 to be in all skill nodes list")
	}

	if !skillIDs[skill2.ID] {
		t.Error("Expected skill2 to be in all skill nodes list")
	}
}

func TestNRVSystemConcurrency(t *testing.T) {
	nrvSystem := NewNRVSystem("concurrent-test", nil)

	// Test concurrent vector creations
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			targetHash := fmt.Sprintf("concurrent-hash-%d", id)
			coordinates := []float64{float64(id), float64(id * 2)}
			metadata := map[string]interface{}{
				"concurrent": true,
				"id":         id,
			}

			_, err := nrvSystem.CreateVector(targetHash, coordinates, metadata)
			if err != nil {
				t.Errorf("Failed to create vector %d: %v", id, err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all vectors were added
	allVectors := nrvSystem.GetAllVectors()
	if len(allVectors) != 10 {
		t.Errorf("Expected 10 vectors after concurrent additions, got %d", len(allVectors))
	}
}
