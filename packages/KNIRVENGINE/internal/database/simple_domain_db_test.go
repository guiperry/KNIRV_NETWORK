package database

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSimpleDomainDB(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "simple_domain_db_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")

	// Create database
	db, err := NewSimpleDomainDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Test collection creation with embeddings
	collection, err := db.GetOrCreateCollection("test_collection")
	if err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}

	if collection == nil {
		t.Error("Collection should not be nil")
	}

	// Test agent repository
	agentRepo := NewSimpleAgentRepository(collection)

	// Create test agent
	agent := &SimpleAgent{
		Name:         "Test Agent",
		Collection:   "test",
		ImageURL:     "https://example.com/image.png",
		Status:       "active",
		Capabilities: []string{"chat", "search"},
		TokenID:      "token123",
		ContractAddr: "0x123",
		OwnerID:      1,
		CreatedAt:    time.Now(),
	}

	// Test creating agent
	ctx := context.Background()
	err = agentRepo.CreateAgent(ctx, agent)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	if agent.ID == "" {
		t.Error("Agent ID should be generated")
	}

	// Test retrieving agent
	retrievedAgent, err := agentRepo.GetAgentByID(ctx, agent.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve agent: %v", err)
	}

	if retrievedAgent.Name != agent.Name {
		t.Errorf("Expected agent name %s, got %s", agent.Name, retrievedAgent.Name)
	}

	if retrievedAgent.OwnerID != agent.OwnerID {
		t.Errorf("Expected owner ID %d, got %d", agent.OwnerID, retrievedAgent.OwnerID)
	}

	// Test getting agents by owner
	agents, err := agentRepo.GetAgentsByOwner(ctx, 1)
	if err != nil {
		t.Fatalf("Failed to get agents by owner: %v", err)
	}

	if len(agents) != 1 {
		t.Errorf("Expected 1 agent, got %d", len(agents))
	}

	if agents[0].ID != agent.ID {
		t.Errorf("Expected agent ID %s, got %s", agent.ID, agents[0].ID)
	}
}

func TestCerebrasEmbeddingGeneration(t *testing.T) {
	// Test the deterministic embedding generation function

	text := "This is a test text for embedding generation"

	embedding, err := generateCerebrasEmbeddingProper(text)
	if err != nil {
		t.Fatalf("Failed to generate embedding: %v", err)
	}

	if len(embedding) == 0 {
		t.Error("Embedding should not be empty")
	}

	// Log the actual dimension for debugging
	t.Logf("Embedding dimension: %d", len(embedding))

	// Test that embeddings are deterministic and consistent
	t.Log("Deterministic embedding generation works!")

	// Test that embeddings are not zero vectors
	allZero := true
	for _, val := range embedding {
		if val != 0 {
			allZero = false
			break
		}
	}

	if allZero {
		t.Error("Embeddings should not be zero vectors")
	}

	// Test deterministic behavior - same input should produce same output
	embedding2, err := generateCerebrasEmbeddingProper(text)
	if err != nil {
		t.Fatalf("Failed to generate second embedding: %v", err)
	}

	if len(embedding) != len(embedding2) {
		t.Errorf("Embeddings should have same length: %d vs %d", len(embedding), len(embedding2))
	}

	// Check that embeddings are identical
	for i := range embedding {
		if embedding[i] != embedding2[i] {
			t.Errorf("Embeddings should be deterministic - difference at index %d: %f vs %f", i, embedding[i], embedding2[i])
			break
		}
	}
}

func TestCreateCerebrasEmbeddingFunc(t *testing.T) {
	// Test the deterministic embedding function creation

	embeddingFunc, err := createCerebrasEmbeddingFunc()
	if err != nil {
		t.Fatalf("Failed to create embedding function: %v", err)
	}

	if embeddingFunc == nil {
		t.Error("Embedding function should not be nil")
	}

	// The function creation itself is what we're testing
	t.Log("Deterministic embedding function created successfully")
}
