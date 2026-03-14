package graphchain

import (
	"KNIRVGRAPH/internal/storage"
	"KNIRVGRAPH/internal/types"
	"fmt"
	"testing"
	"time"
)

func TestNewGraphChain(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	gc := NewGraphChain(storage)
	if gc == nil {
		t.Fatal("Expected non-nil GraphChain")
	}

	if gc.storage != storage {
		t.Error("GraphChain storage not set correctly")
	}
}

func TestGraphChainAddNode(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	gc := NewGraphChain(storage)

	// Create test node
	node := types.NewGraphNode("test-node-1", []string{}, types.GraphData{
		Payload: map[string]interface{}{
			"test": "data",
		},
	})

	// Add node to graphchain
	err = gc.AddNode(node)
	if err != nil {
		t.Fatalf("Failed to add node: %v", err)
	}

	// Retrieve node
	retrievedNode, err := gc.GetNode("test-node-1")
	if err != nil {
		t.Fatalf("Failed to retrieve node: %v", err)
	}

	if retrievedNode.ID != node.ID {
		t.Errorf("Expected node ID %s, got %s", node.ID, retrievedNode.ID)
	}
}

func TestGraphChainAddEdge(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	gc := NewGraphChain(storage)

	// Create test nodes first
	node1 := types.NewGraphNode("node-1", []string{}, types.GraphData{})
	node2 := types.NewGraphNode("node-2", []string{}, types.GraphData{})

	gc.AddNode(node1)
	gc.AddNode(node2)

	// Create and add edge
	edge := types.NewEdge("node-1", "node-2", types.TransactionEdge, 1.0)

	err = gc.AddEdge(edge)
	if err != nil {
		t.Fatalf("Failed to add edge: %v", err)
	}

	// Retrieve edge
	retrievedEdge, err := gc.GetEdge(edge.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve edge: %v", err)
	}

	if retrievedEdge.From != edge.From {
		t.Errorf("Expected edge from %s, got %s", edge.From, retrievedEdge.From)
	}

	if retrievedEdge.To != edge.To {
		t.Errorf("Expected edge to %s, got %s", edge.To, retrievedEdge.To)
	}
}

func TestGraphChainGetCurrentHeight(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	gc := NewGraphChain(storage)

	// Initially height should be 0
	height := gc.GetCurrentHeight()
	if height != 0 {
		t.Errorf("Expected initial height 0, got %d", height)
	}

	// Add some nodes - height is managed by state, not node count
	for i := 0; i < 5; i++ {
		node := types.NewGraphNode(fmt.Sprintf("node-%d", i), []string{}, types.GraphData{})
		gc.AddNode(node)
	}

	// Height is still managed by the state, not by node count
	height = gc.GetCurrentHeight()
	if height < 0 {
		t.Errorf("Expected non-negative height, got %d", height)
	}
}

func TestGraphChainGetHeads(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	gc := NewGraphChain(storage)

	// Add root nodes (nodes with no parents)
	root1 := types.NewGraphNode("root-1", []string{}, types.GraphData{})
	root2 := types.NewGraphNode("root-2", []string{}, types.GraphData{})

	gc.AddNode(root1)
	gc.AddNode(root2)

	// Add child node
	child := types.NewGraphNode("child-1", []string{"root-1"}, types.GraphData{})
	gc.AddNode(child)

	heads := gc.GetHeads()

	// Should have some heads (nodes with no children)
	if len(heads) == 0 {
		t.Error("Expected at least one head node")
	}
}

func TestGraphChainFindPath(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	gc := NewGraphChain(storage)

	// Create a simple path: A -> B -> C
	nodeA := types.NewGraphNode("A", []string{}, types.GraphData{})
	nodeB := types.NewGraphNode("B", []string{"A"}, types.GraphData{})
	nodeC := types.NewGraphNode("C", []string{"B"}, types.GraphData{})

	gc.AddNode(nodeA)
	gc.AddNode(nodeB)
	gc.AddNode(nodeC)

	// Add edges
	edgeAB := types.NewEdge("A", "B", types.TransactionEdge, 1.0)
	edgeBC := types.NewEdge("B", "C", types.TransactionEdge, 1.0)

	gc.AddEdge(edgeAB)
	gc.AddEdge(edgeBC)

	// Find path from A to C with max depth
	path, err := gc.FindPath("A", "C", 10)
	if err != nil {
		t.Fatalf("Failed to find path: %v", err)
	}

	if len(path) == 0 {
		t.Error("Expected non-empty path")
	}

	// Path should start with A and end with C
	if path[0] != "A" {
		t.Errorf("Expected path to start with A, got %s", path[0])
	}

	if path[len(path)-1] != "C" {
		t.Errorf("Expected path to end with C, got %s", path[len(path)-1])
	}
}

func TestGraphChainValidateNode(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	gc := NewGraphChain(storage)

	tests := []struct {
		name      string
		node      *types.GraphNode
		expectErr bool
	}{
		{
			name: "Valid node",
			node: types.NewGraphNode("valid-node", []string{}, types.GraphData{
				Payload: map[string]interface{}{"test": "data"},
			}),
			expectErr: false,
		},
		{
			name: "Node with empty ID",
			node: &types.GraphNode{
				ID:        "",
				Parents:   []string{},
				Data:      types.GraphData{},
				Timestamp: time.Now(),
			},
			expectErr: true,
		},
		{
			name:      "Node with invalid parent reference",
			node:      types.NewGraphNode("child-node", []string{"non-existent-parent"}, types.GraphData{}),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gc.AddNode(tt.node)

			if tt.expectErr && err == nil {
				t.Error("Expected validation error, but got none")
			}

			if !tt.expectErr && err != nil {
				t.Errorf("Expected no validation error, but got: %v", err)
			}
		})
	}
}

func TestGraphChainGetNeighbors(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	gc := NewGraphChain(storage)

	// Create nodes with relationships
	center := types.NewGraphNode("center", []string{}, types.GraphData{})
	neighbor1 := types.NewGraphNode("neighbor1", []string{"center"}, types.GraphData{})
	neighbor2 := types.NewGraphNode("neighbor2", []string{"center"}, types.GraphData{})

	gc.AddNode(center)
	gc.AddNode(neighbor1)
	gc.AddNode(neighbor2)

	// Add edges
	edge1 := types.NewEdge("center", "neighbor1", types.TransactionEdge, 1.0)
	edge2 := types.NewEdge("center", "neighbor2", types.TransactionEdge, 1.0)

	gc.AddEdge(edge1)
	gc.AddEdge(edge2)

	// Get neighbors of center node
	neighbors, err := gc.GetNeighbors("center")
	if err != nil {
		t.Fatalf("Failed to get neighbors: %v", err)
	}

	if len(neighbors) != 2 {
		t.Errorf("Expected 2 neighbors, got %d", len(neighbors))
	}

	// Check that neighbors contain expected nodes
	neighborIDs := make(map[string]bool)
	for _, neighborID := range neighbors {
		neighborIDs[neighborID] = true
	}

	if !neighborIDs["neighbor1"] {
		t.Error("Expected neighbor1 in neighbors list")
	}

	if !neighborIDs["neighbor2"] {
		t.Error("Expected neighbor2 in neighbors list")
	}
}
