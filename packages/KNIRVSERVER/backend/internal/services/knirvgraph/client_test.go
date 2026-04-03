package knirvgraph

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:7090")

	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:7090", client.baseURL)
	assert.NotNil(t, client.client)
}

func TestClient_Health_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.Health(context.Background())

	require.NoError(t, err)
}

func TestClient_Health_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.Health(context.Background())

	assert.Error(t, err)
}

func TestClient_Health_ConnectionRefused(t *testing.T) {
	client := NewClient("http://localhost:19999")
	err := client.Health(context.Background())

	assert.Error(t, err)
}

func TestClient_Health_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		baseURL: server.URL,
		client: &http.Client{
			Timeout: 100 * time.Millisecond,
		},
	}
	err := client.Health(context.Background())

	assert.Error(t, err)
}

func TestClient_CommitNode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/commit", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true, "commit_hash": "abc123"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	node := GraphNode{
		NodeID:    "test_node_1",
		Type:      "error_node",
		Data:      map[string]interface{}{"task_id": "task_123"},
		Timestamp: time.Now(),
	}

	err := client.CommitNode(context.Background(), node, "Test commit", "test_user")

	require.NoError(t, err)
}

func TestClient_CommitNode_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"success": false, "error": "invalid request"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	node := GraphNode{
		NodeID:    "test_node_1",
		Type:      "error_node",
		Data:      map[string]interface{}{},
		Timestamp: time.Now(),
	}

	err := client.CommitNode(context.Background(), node, "Test commit", "test_user")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

func TestClient_CommitNode_ConnectionError(t *testing.T) {
	client := NewClient("http://localhost:19999")
	node := GraphNode{
		NodeID:    "test_node_1",
		Type:      "error_node",
		Data:      map[string]interface{}{},
		Timestamp: time.Now(),
	}

	err := client.CommitNode(context.Background(), node, "Test commit", "test_user")

	assert.Error(t, err)
}
