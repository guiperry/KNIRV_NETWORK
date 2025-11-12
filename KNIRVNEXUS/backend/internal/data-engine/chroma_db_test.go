package dataengine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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

	// Manually set connected (for testing)
	chroma.mutex.Lock()
	chroma.connected = true
	chroma.mutex.Unlock()

	assert.True(t, chroma.IsConnected())
}

func TestChromaDB_Close(t *testing.T) {
	chroma := NewChromaDB("http://localhost:8000", "test")

	// Manually set connected
	chroma.mutex.Lock()
	chroma.connected = true
	chroma.mutex.Unlock()

	chroma.Close()

	assert.False(t, chroma.connected)
}

func TestChromaDB_Connect_ConnectionRefused(t *testing.T) {
	chroma := NewChromaDB("http://nonexistent:8000", "test")

	ctx := context.Background()
	err := chroma.Connect(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to ChromaDB")
	assert.False(t, chroma.connected)
}

func TestChromaDB_AddEvent_NotConnected(t *testing.T) {
	chroma := NewChromaDB("http://localhost:8000", "test")

	event := Event{
		Type:      EventType("test"),
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"key": "value"},
	}

	ctx := context.Background()
	err := chroma.AddEvent(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected to ChromaDB")
}

func TestChromaDB_QueryEvents_NotConnected(t *testing.T) {
	chroma := NewChromaDB("http://localhost:8000", "test")

	ctx := context.Background()
	_, err := chroma.QueryEvents(ctx, "test query", 10)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected to ChromaDB")
}

func TestChromaDB_GetEventsByType_NotConnected(t *testing.T) {
	chroma := NewChromaDB("http://localhost:8000", "test")

	ctx := context.Background()
	_, err := chroma.GetEventsByType(ctx, EventType("test"), 10)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected to ChromaDB")
}

func TestChromaDB_GetRecentEvents_NotConnected(t *testing.T) {
	chroma := NewChromaDB("http://localhost:8000", "test")

	ctx := context.Background()
	_, err := chroma.GetRecentEvents(ctx, 10)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected to ChromaDB")
}

func TestChromaDocument_Structure(t *testing.T) {
	doc := ChromaDocument{
		ID:        "test-id",
		Embedding: []float64{0.1, 0.2, 0.3},
		Metadata:  map[string]interface{}{"key": "value"},
		Document:  "test document",
		Type:      "test-type",
		Timestamp: time.Now(),
	}

	assert.Equal(t, "test-id", doc.ID)
	assert.Equal(t, []float64{0.1, 0.2, 0.3}, doc.Embedding)
	assert.Equal(t, map[string]interface{}{"key": "value"}, doc.Metadata)
	assert.Equal(t, "test document", doc.Document)
	assert.Equal(t, "test-type", doc.Type)
	assert.NotZero(t, doc.Timestamp)
}

func TestChromaQueryResult_Structure(t *testing.T) {
	result := ChromaQueryResult{
		IDs:        []string{"id1", "id2"},
		Embeddings: [][]float64{{0.1, 0.2}, {0.3, 0.4}},
		Metadatas:  []map[string]interface{}{{"key1": "value1"}, {"key2": "value2"}},
		Documents:  []string{"doc1", "doc2"},
		Distances:  []float64{0.1, 0.2},
	}

	assert.Equal(t, []string{"id1", "id2"}, result.IDs)
	assert.Equal(t, [][]float64{{0.1, 0.2}, {0.3, 0.4}}, result.Embeddings)
	assert.Equal(t, []map[string]interface{}{{"key1": "value1"}, {"key2": "value2"}}, result.Metadatas)
	assert.Equal(t, []string{"doc1", "doc2"}, result.Documents)
	assert.Equal(t, []float64{0.1, 0.2}, result.Distances)
}

// Mock HTTP server for testing ChromaDB methods that require HTTP calls
func createMockChromaServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
		case "/api/v1/collections":
			switch r.Method {
			case "GET":
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"collections": []}`))
			case "POST":
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"name": "test-collection"}`))
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestChromaDB_Connect_Success(t *testing.T) {
	server := createMockChromaServer()
	defer server.Close()

	chroma := NewChromaDB(server.URL, "test-collection")

	ctx := context.Background()
	err := chroma.Connect(ctx)

	assert.NoError(t, err)
	assert.True(t, chroma.connected)
}

func TestChromaDB_createCollection(t *testing.T) {
	server := createMockChromaServer()
	defer server.Close()

	chroma := NewChromaDB(server.URL, "test-collection")

	ctx := context.Background()
	err := chroma.createCollection(ctx)

	assert.NoError(t, err)
}

func TestChromaDB_listCollections(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/api/v1/collections" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"collections": [{"name": "collection1"}, {"name": "collection2"}]}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	chroma := NewChromaDB(server.URL, "test-collection")

	ctx := context.Background()
	collections, err := chroma.listCollections(ctx)

	assert.NoError(t, err)
	assert.Len(t, collections, 2)
	assert.Contains(t, collections, "collection1")
	assert.Contains(t, collections, "collection2")
}

func TestChromaDB_addDocuments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/add") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "success"}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	chroma := NewChromaDB(server.URL, "test-collection")
	chroma.connected = true // Manually set connected for testing

	docs := []ChromaDocument{
		{
			ID:       "doc1",
			Document: "test document 1",
			Metadata: map[string]interface{}{"key": "value1"},
		},
		{
			ID:       "doc2",
			Document: "test document 2",
			Metadata: map[string]interface{}{"key": "value2"},
		},
	}

	ctx := context.Background()
	err := chroma.addDocuments(ctx, docs)

	assert.NoError(t, err)
}