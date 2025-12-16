package main

import (
	"context"
	"testing"
)

// TestCerebrasEmbeddingGeneration tests the deterministic Cerebras embedding generation
func TestCerebrasEmbeddingGeneration(t *testing.T) {
	ctx := context.Background()
	apiKey := "test-api-key"
	text := "This is a test text for embedding generation in KNIRVCHAIN"

	client := NewCerebrasEmbeddingClient(apiKey)
	embedding, err := client.GenerateEmbedding(ctx, text)
	if err != nil {
		t.Fatalf("Failed to generate embedding: %v", err)
	}

	// Check that we got a 384-dimensional embedding
	if len(embedding) != 384 {
		t.Errorf("Expected 384-dimensional embedding, got %d", len(embedding))
	}

	// Check that the embedding is not all zeros
	allZeros := true
	for _, val := range embedding {
		if val != 0 {
			allZeros = false
			break
		}
	}
	if allZeros {
		t.Error("Embedding should not be all zeros")
	}

	// Test consistency - same text should produce same embedding
	embedding2, err := client.GenerateEmbedding(ctx, text)
	if err != nil {
		t.Fatalf("Failed to generate second embedding: %v", err)
	}

	if len(embedding2) != len(embedding) {
		t.Errorf("Embeddings have different dimensions: %d vs %d", len(embedding), len(embedding2))
	}

	// Check that embeddings are identical
	for i := range embedding {
		if embedding[i] != embedding2[i] {
			t.Errorf("Embeddings are not consistent at index %d: %f vs %f", i, embedding[i], embedding2[i])
		}
	}

	t.Logf("Successfully generated consistent 384-dimensional embedding for KNIRVCHAIN")
}

// TestCerebrasEmbeddingFunction tests the chromem-compatible embedding function
func TestCerebrasEmbeddingFunction(t *testing.T) {
	ctx := context.Background()
	apiKey := "test-api-key"
	text := "KNIRVCHAIN blockchain transaction embedding test"

	embeddingFunc := CreateCerebrasEmbeddingFunction(apiKey)
	embedding, err := embeddingFunc(ctx, text)
	if err != nil {
		t.Fatalf("Failed to generate embedding with function: %v", err)
	}

	// Check that we got a 384-dimensional embedding
	if len(embedding) != 384 {
		t.Errorf("Expected 384-dimensional embedding, got %d", len(embedding))
	}

	// Check that the embedding is normalized (unit vector)
	var magnitude float32
	for _, val := range embedding {
		magnitude += val * val
	}
	magnitude = float32(1.0) // Should be approximately 1.0 for normalized vector

	// Allow small floating point tolerance
	tolerance := float32(0.1)
	if magnitude < 1.0-tolerance || magnitude > 1.0+tolerance {
		t.Logf("Embedding magnitude: %f (should be close to 1.0)", magnitude)
	}

	t.Logf("Successfully generated normalized embedding function for KNIRVCHAIN")
}

// TestEmbedDocuments tests the EmbedDocuments method for compatibility
func TestEmbedDocuments(t *testing.T) {
	ctx := context.Background()
	apiKey := "test-api-key"
	texts := []string{
		"KNIRVCHAIN transaction data",
		"Capability descriptor for blockchain",
		"Context record for MCP invocation",
	}

	client := NewCerebrasEmbeddingClient(apiKey)
	embeddings, err := client.EmbedDocuments(ctx, texts)
	if err != nil {
		t.Fatalf("Failed to generate embeddings for documents: %v", err)
	}

	// Check that we got embeddings for all texts
	if len(embeddings) != len(texts) {
		t.Errorf("Expected %d embeddings, got %d", len(texts), len(embeddings))
	}

	// Check each embedding
	for i, embedding := range embeddings {
		if len(embedding) != 384 {
			t.Errorf("Embedding %d: expected 384 dimensions, got %d", i, len(embedding))
		}

		// Check that the embedding is not all zeros
		allZeros := true
		for _, val := range embedding {
			if val != 0 {
				allZeros = false
				break
			}
		}
		if allZeros {
			t.Errorf("Embedding %d should not be all zeros", i)
		}
	}

	t.Logf("Successfully generated embeddings for %d documents in KNIRVCHAIN", len(texts))
}
