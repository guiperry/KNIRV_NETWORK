package dataengine

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewChromaDB(t *testing.T) {
	url := "http://localhost:8000"
	collection := "test-collection"

	chroma := NewChromaDB(url, collection)

	assert.NotNil(t, chroma)
	assert.Equal(t, url, chroma.url)
	assert.Equal(t, collection, chroma.collection)
	assert.NotNil(t, chroma.client)
	assert.False(t, chroma.connected)
}

func TestChromaDB_IsConnected(t *testing.T) {
	chroma := NewChromaDB("http://localhost:8000", "test")

	// Initially not connected
	assert.False(t, chroma.IsConnected())

	// Simulate connection (this would normally be done by Connect)
	chroma.connected = true
	assert.True(t, chroma.IsConnected())

	chroma.Close()
	assert.False(t, chroma.IsConnected())
}

func TestChromaDB_Close(t *testing.T) {
	chroma := NewChromaDB("http://localhost:8000", "test")

	// Simulate connection
	chroma.connected = true
	assert.True(t, chroma.IsConnected())

	chroma.Close()
	assert.False(t, chroma.IsConnected())
}

// Note: The following tests require a running ChromaDB instance
// They are designed to test the functionality when ChromaDB is available
// In a real test environment, these would be integration tests

func TestChromaDB_Connect_ConnectionRefused(t *testing.T) {
	chroma := NewChromaDB("http://nonexistent:8000", "test")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := chroma.Connect(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to ChromaDB")
	assert.False(t, chroma.IsConnected())
}

func TestChromaDB_AddEvent_NotConnected(t *testing.T) {
	chroma := NewChromaDB("http://localhost:8000", "test")

	event := Event{
		Type:      "test",
		Timestamp: time.Now(),
		Source:    "test",
		Data: map[string]interface{}{
			"key": "value",
		},
	}

	ctx := context.Background()
	err := chroma.AddEvent(ctx, event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected to ChromaDB")
}

func TestChromaDB_QueryEvents_NotConnected(t *testing.T) {
	chroma := NewChromaDB("http://localhost:8000", "test")

	ctx := context.Background()
	docs, err := chroma.QueryEvents(ctx, "test query", 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected to ChromaDB")
	assert.Nil(t, docs)
}

func TestChromaDB_GetEventsByType_NotConnected(t *testing.T) {
	chroma := NewChromaDB("http://localhost:8000", "test")

	ctx := context.Background()
	docs, err := chroma.GetEventsByType(ctx, "test", 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected to ChromaDB")
	assert.Nil(t, docs)
}

func TestChromaDB_GetRecentEvents_NotConnected(t *testing.T) {
	chroma := NewChromaDB("http://localhost:8000", "test")

	ctx := context.Background()
	docs, err := chroma.GetRecentEvents(ctx, 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected to ChromaDB")
	assert.Nil(t, docs)
}

// Mock HTTP server tests would go here for full integration testing
// These would require setting up a test HTTP server that mimics ChromaDB responses

func TestChromaDocument_Structure(t *testing.T) {
	doc := ChromaDocument{
		ID:        "test-doc-1",
		Embedding: []float64{0.1, 0.2, 0.3},
		Metadata: map[string]interface{}{
			"type":      "test",
			"timestamp": time.Now().Format(time.RFC3339),
		},
		Document:  "Test document content",
		Type:      "test",
		Timestamp: time.Now(),
	}

	assert.Equal(t, "test-doc-1", doc.ID)
	assert.Len(t, doc.Embedding, 3)
	assert.Equal(t, "Test document content", doc.Document)
	assert.Equal(t, "test", doc.Type)
	assert.NotZero(t, doc.Timestamp)

	// Test with nil metadata to ensure no panic
	docWithNilMetadata := ChromaDocument{
		ID:        "test-doc-2",
		Embedding: []float64{0.1, 0.2, 0.3},
		Metadata:  nil,
		Document:  "Test document content",
		Type:      "test",
		Timestamp: time.Now(),
	}

	assert.Equal(t, "test-doc-2", docWithNilMetadata.ID)
	assert.Len(t, docWithNilMetadata.Embedding, 3)
	assert.Equal(t, "Test document content", docWithNilMetadata.Document)
	assert.Equal(t, "test", docWithNilMetadata.Type)
	assert.NotZero(t, docWithNilMetadata.Timestamp)

	// Use the metadata field to avoid unused write warning
	_ = doc.Metadata
	_ = docWithNilMetadata.Metadata
}

func TestChromaQueryResult_Structure(t *testing.T) {
	result := ChromaQueryResult{
		IDs:        []string{"doc1", "doc2"},
		Embeddings: [][]float64{{0.1, 0.2}, {0.3, 0.4}},
		Metadatas: []map[string]interface{}{
			{"type": "test1"},
			{"type": "test2"},
		},
		Documents: []string{"doc1 content", "doc2 content"},
		Distances: []float64{0.1, 0.2},
	}

	assert.Len(t, result.IDs, 2)
	assert.Len(t, result.Embeddings, 2)
	assert.Len(t, result.Metadatas, 2)
	assert.Len(t, result.Documents, 2)
	assert.Len(t, result.Distances, 2)

	// Use the metadata field to avoid unused write warning
	_ = result.Metadatas
}