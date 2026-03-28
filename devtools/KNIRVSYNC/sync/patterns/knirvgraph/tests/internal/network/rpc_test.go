package network

import (
	"blockchain-app/internal/graphchain"
	"blockchain-app/internal/nrv"
	"blockchain-app/internal/storage"
	"blockchain-app/internal/types"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewRPCServer(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	gc := graphchain.NewGraphChain(storage)
	logger, _ := zap.NewDevelopment()

	server := NewRPCServer(gc, logger, 8080)
	if server == nil {
		t.Fatal("Expected non-nil RPC server")
	}

	if server.port != 8080 {
		t.Errorf("Expected port 8080, got %d", server.port)
	}
}

func TestNewRPCServerWithNRV(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	gc := graphchain.NewGraphChain(storage)
	nrvSystem := nrv.NewNRVSystem("test-peer", nil)
	logger, _ := zap.NewDevelopment()

	server := NewRPCServerWithNRV(gc, nrvSystem, logger, 8080)
	if server == nil {
		t.Fatal("Expected non-nil RPC server with NRV")
	}

	if server.nrvSystem == nil {
		t.Error("Expected NRV system to be set")
	}
}

func TestRPCServerHealthEndpoint(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	gc := graphchain.NewGraphChain(storage)
	logger, _ := zap.NewDevelopment()
	server := NewRPCServer(gc, logger, 8080)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}
}

func TestRPCServerGetHeightEndpoint(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	gc := graphchain.NewGraphChain(storage)
	logger, _ := zap.NewDevelopment()
	server := NewRPCServer(gc, logger, 8080)

	req := httptest.NewRequest("GET", "/height", nil)
	w := httptest.NewRecorder()

	server.handleGetHeight(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if _, exists := response["height"]; !exists {
		t.Error("Expected 'height' field in response")
	}
}

func TestRPCServerGetNodeEndpoint(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	gc := graphchain.NewGraphChain(storage)
	logger, _ := zap.NewDevelopment()
	server := NewRPCServer(gc, logger, 8080)

	// Add a test node first
	testNode := types.NewGraphNode("test-node", []string{}, types.GraphData{
		Payload: map[string]interface{}{"test": "data"},
	})
	gc.AddNode(testNode)

	req := httptest.NewRequest("GET", "/node/test-node", nil)
	w := httptest.NewRecorder()

	server.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response types.GraphNode
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.ID != "test-node" {
		t.Errorf("Expected node ID 'test-node', got %s", response.ID)
	}
}

func TestRPCServerCreateNodeEndpoint(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	gc := graphchain.NewGraphChain(storage)
	logger, _ := zap.NewDevelopment()
	server := NewRPCServer(gc, logger, 8080)

	// Create test node data
	nodeData := types.GraphNode{
		ID:      "new-test-node",
		Parents: []string{},
		Data: types.GraphData{
			Payload: map[string]interface{}{"test": "data"},
		},
		Timestamp: time.Now(),
		Weight:    1.0,
	}

	jsonData, err := json.Marshal(nodeData)
	if err != nil {
		t.Fatalf("Failed to marshal node data: %v", err)
	}

	req := httptest.NewRequest("POST", "/node", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	// Verify node was created
	node, err := gc.GetNode("new-test-node")
	if err != nil {
		t.Fatalf("Failed to retrieve created node: %v", err)
	}

	if node.ID != "new-test-node" {
		t.Errorf("Expected node ID 'new-test-node', got %s", node.ID)
	}
}

func TestRPCServerGetHeadsEndpoint(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	gc := graphchain.NewGraphChain(storage)
	logger, _ := zap.NewDevelopment()
	server := NewRPCServer(gc, logger, 8080)

	// Add some test nodes
	node1 := types.NewGraphNode("head-1", []string{}, types.GraphData{})
	node2 := types.NewGraphNode("head-2", []string{}, types.GraphData{})
	gc.AddNode(node1)
	gc.AddNode(node2)

	req := httptest.NewRequest("GET", "/graph/heads", nil)
	w := httptest.NewRecorder()

	server.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string][]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	heads, ok := response["heads"]
	if !ok {
		t.Error("Expected 'heads' field in response")
	}

	if len(heads) == 0 {
		t.Error("Expected at least one head node")
	}
}

func TestRPCServerFindPathEndpoint(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	gc := graphchain.NewGraphChain(storage)
	logger, _ := zap.NewDevelopment()
	server := NewRPCServer(gc, logger, 8080)

	// Create a simple path: A -> B
	nodeA := types.NewGraphNode("A", []string{}, types.GraphData{})
	nodeB := types.NewGraphNode("B", []string{"A"}, types.GraphData{})
	gc.AddNode(nodeA)
	gc.AddNode(nodeB)

	req := httptest.NewRequest("GET", "/graph/path/A/B", nil)
	w := httptest.NewRecorder()

	server.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string][]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	path, ok := response["path"]
	if !ok {
		t.Error("Expected 'path' field in response")
	}

	if len(path) == 0 {
		t.Error("Expected non-empty path")
	}
}

func TestRPCServerErrorHandling(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	gc := graphchain.NewGraphChain(storage)
	logger, _ := zap.NewDevelopment()
	server := NewRPCServer(gc, logger, 8080)

	// Test getting non-existent node
	req := httptest.NewRequest("GET", "/node/non-existent", nil)
	w := httptest.NewRecorder()

	server.handleGetNode(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	// Test invalid JSON for node creation
	req = httptest.NewRequest("POST", "/node", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	server.handleCreateNode(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestRPCServerCORS(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	gc := graphchain.NewGraphChain(storage)
	logger, _ := zap.NewDevelopment()
	server := NewRPCServer(gc, logger, 8080)

	req := httptest.NewRequest("OPTIONS", "/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	server.enableCORS(http.HandlerFunc(server.handleHealth)).ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Expected CORS headers to be set")
	}
}
