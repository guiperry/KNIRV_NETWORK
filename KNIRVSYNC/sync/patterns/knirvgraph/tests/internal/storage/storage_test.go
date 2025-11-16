package storage

import (
	"blockchain-app/internal/types"
	"os"
	"testing"
	"time"
)

func TestMemoryStorage(t *testing.T) {
	storage, err := NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create memory storage: %v", err)
	}

	// Test storing and retrieving a node
	node := createTestNode("test-node-1")
	nodeData, err := node.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize node: %v", err)
	}

	err = storage.PutNode("test-node-1", nodeData)
	if err != nil {
		t.Fatalf("Failed to store node: %v", err)
	}

	retrievedData, err := storage.GetNode("test-node-1")
	if err != nil {
		t.Fatalf("Failed to retrieve node: %v", err)
	}

	if len(retrievedData) == 0 {
		t.Error("Expected non-empty node data")
	}

	// Test storing and retrieving an edge
	edge := createTestEdge("node1", "node2")
	edgeData, err := edge.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize edge: %v", err)
	}

	err = storage.PutEdge(edge.ID, edgeData)
	if err != nil {
		t.Fatalf("Failed to store edge: %v", err)
	}

	retrievedEdgeData, err := storage.GetEdge(edge.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve edge: %v", err)
	}

	if len(retrievedEdgeData) == 0 {
		t.Error("Expected non-empty edge data")
	}

	// Test getting all nodes with prefix
	node2 := createTestNode("test-node-2")
	node2Data, _ := node2.Serialize()
	storage.PutNode("test-node-2", node2Data)

	nodes, err := storage.GetAllNodesWithPrefix("test-")
	if err != nil {
		t.Fatalf("Failed to get all nodes: %v", err)
	}

	if len(nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(nodes))
	}

	// Test parents and children
	err = storage.PutParents("test-node-1", []string{"parent-1", "parent-2"})
	if err != nil {
		t.Fatalf("Failed to store parents: %v", err)
	}

	parents, err := storage.GetParents("test-node-1")
	if err != nil {
		t.Fatalf("Failed to get parents: %v", err)
	}

	if len(parents) != 2 {
		t.Errorf("Expected 2 parents, got %d", len(parents))
	}

	// Test heads
	err = storage.UpdateHeads([]string{"head-1", "head-2"})
	if err != nil {
		t.Fatalf("Failed to update heads: %v", err)
	}

	heads, err := storage.GetHeads()
	if err != nil {
		t.Fatalf("Failed to get heads: %v", err)
	}

	if len(heads) != 2 {
		t.Errorf("Expected 2 heads, got %d", len(heads))
	}
}

func TestBluntDBStorage(t *testing.T) {
	// Create temporary directory for test database
	tempDir := "/tmp/knirvgraph_test_" + time.Now().Format("20060102150405")
	defer os.RemoveAll(tempDir)

	storage, err := NewBluntDBStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create BluntDB storage: %v", err)
	}
	defer storage.Close()

	// Test storing and retrieving a node
	node := createTestNode("test-node-1")
	nodeData, err := node.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize node: %v", err)
	}

	err = storage.PutNode("test-node-1", nodeData)
	if err != nil {
		t.Fatalf("Failed to store node: %v", err)
	}

	retrievedData, err := storage.GetNode("test-node-1")
	if err != nil {
		t.Fatalf("Failed to retrieve node: %v", err)
	}

	if len(retrievedData) == 0 {
		t.Error("Expected non-empty node data")
	}

	// Test persistence by closing and reopening
	storage.Close()

	storage, err = NewBluntDBStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to reopen BluntDB storage: %v", err)
	}
	defer storage.Close()

	retrievedData, err = storage.GetNode("test-node-1")
	if err != nil {
		t.Fatalf("Failed to retrieve node after reopen: %v", err)
	}

	if len(retrievedData) == 0 {
		t.Error("Expected persisted node data to be non-empty")
	}
}

func TestLevelDBStorage(t *testing.T) {
	// Create temporary directory for test database
	tempDir := "/tmp/knirvgraph_leveldb_test_" + time.Now().Format("20060102150405")
	defer os.RemoveAll(tempDir)

	storage, err := NewLevelDBStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create LevelDB storage: %v", err)
	}
	defer storage.Close()

	// Test basic key-value operations
	testKey := []byte("test-key")
	testValue := []byte("test-value")

	err = storage.Put(testKey, testValue)
	if err != nil {
		t.Fatalf("Failed to put data: %v", err)
	}

	retrievedValue, err := storage.Get(testKey)
	if err != nil {
		t.Fatalf("Failed to get data: %v", err)
	}

	if string(retrievedValue) != string(testValue) {
		t.Errorf("Expected value %s, got %s", string(testValue), string(retrievedValue))
	}

	// Test Has operation
	exists, err := storage.Has(testKey)
	if err != nil {
		t.Fatalf("Failed to check key existence: %v", err)
	}

	if !exists {
		t.Error("Expected key to exist")
	}

	// Test Delete operation
	err = storage.Delete(testKey)
	if err != nil {
		t.Fatalf("Failed to delete key: %v", err)
	}

	exists, err = storage.Has(testKey)
	if err != nil {
		t.Fatalf("Failed to check key existence after delete: %v", err)
	}

	if exists {
		t.Error("Expected key to not exist after deletion")
	}
}

func TestStorageInterface(t *testing.T) {
	// Test memory storage
	memStorage, err := NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create memory storage: %v", err)
	}

	t.Run("MemoryStorage", func(t *testing.T) {
		testGraphStorageOperations(t, memStorage)
	})

	// Test BluntDB storage if available
	tempDir := "/tmp/knirvgraph_interface_test_" + time.Now().Format("20060102150405")
	defer os.RemoveAll(tempDir)

	if bluntStorage, err := NewBluntDBStorage(tempDir + "/blunt"); err == nil {
		defer bluntStorage.Close()
		t.Run("BluntDBStorage", func(t *testing.T) {
			testGraphStorageOperations(t, bluntStorage)
		})
	}
}

func testGraphStorageOperations(t *testing.T, storage GraphStorage) {
	// Test node operations
	node := createTestNode("interface-test-node")
	nodeData, err := node.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize node: %v", err)
	}

	err = storage.PutNode("interface-test-node", nodeData)
	if err != nil {
		t.Fatalf("Failed to store node: %v", err)
	}

	retrievedData, err := storage.GetNode("interface-test-node")
	if err != nil {
		t.Fatalf("Failed to retrieve node: %v", err)
	}

	if len(retrievedData) == 0 {
		t.Error("Expected non-empty node data")
	}

	// Test edge operations
	edge := createTestEdge("from-node", "to-node")
	edgeData, err := edge.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize edge: %v", err)
	}

	err = storage.PutEdge(edge.ID, edgeData)
	if err != nil {
		t.Fatalf("Failed to store edge: %v", err)
	}

	retrievedEdgeData, err := storage.GetEdge(edge.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve edge: %v", err)
	}

	if len(retrievedEdgeData) == 0 {
		t.Error("Expected non-empty edge data")
	}
}

// Helper functions
func createTestNode(id string) *types.GraphNode {
	return types.NewGraphNode(id, []string{}, types.GraphData{
		Payload: map[string]interface{}{
			"test":      "data",
			"timestamp": time.Now().Unix(),
		},
	})
}

func createTestEdge(from, to string) *types.Edge {
	return types.NewEdge(from, to, types.TransactionEdge, 1.0)
}
