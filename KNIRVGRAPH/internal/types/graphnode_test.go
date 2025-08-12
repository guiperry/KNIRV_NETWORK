package types

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestNewGraphNode(t *testing.T) {
	nodeID := "test-node-1"
	parents := []string{"parent-1", "parent-2"}
	data := GraphData{
		Transactions: []Transaction{},
		StateChanges: []StateChange{},
		Edges:        []Edge{},
		Payload: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		},
	}

	node := NewGraphNode(nodeID, parents, data)

	if node.ID != nodeID {
		t.Errorf("Expected node ID %s, got %s", nodeID, node.ID)
	}

	if len(node.Parents) != len(parents) {
		t.Errorf("Expected %d parents, got %d", len(parents), len(node.Parents))
	}

	for i, parent := range parents {
		if node.Parents[i] != parent {
			t.Errorf("Expected parent %s at index %d, got %s", parent, i, node.Parents[i])
		}
	}

	if node.Data.Payload["key1"] != "value1" {
		t.Errorf("Expected data key1 to be 'value1', got %v", node.Data.Payload["key1"])
	}

	if node.Data.Payload["key2"] != 42 {
		t.Errorf("Expected data key2 to be 42, got %v", node.Data.Payload["key2"])
	}

	if node.Weight != 1.0 {
		t.Errorf("Expected default weight 1.0, got %f", node.Weight)
	}

	if node.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

func TestGraphNodeValidation(t *testing.T) {
	tests := []struct {
		name      string
		nodeID    string
		parents   []string
		data      GraphData
		expectErr bool
	}{
		{
			name:    "Valid node",
			nodeID:  "valid-node",
			parents: []string{"parent-1"},
			data: GraphData{
				Payload: map[string]interface{}{"key": "value"},
			},
			expectErr: false,
		},
		{
			name:    "Empty node ID",
			nodeID:  "",
			parents: []string{"parent-1"},
			data: GraphData{
				Payload: map[string]interface{}{"key": "value"},
			},
			expectErr: true,
		},
		{
			name:    "No parents",
			nodeID:  "root-node",
			parents: []string{},
			data: GraphData{
				Payload: map[string]interface{}{"key": "value"},
			},
			expectErr: false, // Root nodes are allowed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := NewGraphNode(tt.nodeID, tt.parents, tt.data)

			// Basic validation - check if node ID is empty
			var err error
			if node.ID == "" {
				err = fmt.Errorf("node ID cannot be empty")
			}

			if tt.expectErr && err == nil {
				t.Error("Expected validation error, but got none")
			}

			if !tt.expectErr && err != nil {
				t.Errorf("Expected no validation error, but got: %v", err)
			}
		})
	}
}

func TestGraphNodeSerialization(t *testing.T) {
	original := NewGraphNode("test-node", []string{"parent-1"}, GraphData{
		Payload: map[string]interface{}{
			"string_key": "string_value",
			"int_key":    123,
			"float_key":  45.67,
			"bool_key":   true,
		},
	})
	original.Weight = 2.5

	// Test JSON serialization using existing Serialize method
	data, err := original.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize to JSON: %v", err)
	}

	// Test JSON deserialization
	var deserialized GraphNode
	err = json.Unmarshal(data, &deserialized)
	if err != nil {
		t.Fatalf("Failed to deserialize from JSON: %v", err)
	}

	// Verify all fields match
	if deserialized.ID != original.ID {
		t.Errorf("ID mismatch: expected %s, got %s", original.ID, deserialized.ID)
	}

	if len(deserialized.Parents) != len(original.Parents) {
		t.Errorf("Parents length mismatch: expected %d, got %d", len(original.Parents), len(deserialized.Parents))
	}

	if deserialized.Weight != original.Weight {
		t.Errorf("Weight mismatch: expected %f, got %f", original.Weight, deserialized.Weight)
	}

	// Check payload data fields with type-aware comparison
	for key, value := range original.Data.Payload {
		deserializedValue := deserialized.Data.Payload[key]

		// Handle JSON number unmarshaling (int becomes float64)
		if key == "int_key" {
			if originalInt, ok := value.(int); ok {
				if deserializedFloat, ok := deserializedValue.(float64); ok {
					if float64(originalInt) != deserializedFloat {
						t.Errorf("Data mismatch for key %s: expected %v, got %v", key, value, deserializedValue)
					}
					continue
				}
			}
		}

		if deserializedValue != value {
			t.Errorf("Data mismatch for key %s: expected %v, got %v", key, value, deserializedValue)
		}
	}
}

func TestGraphNodeHash(t *testing.T) {
	data := GraphData{
		Payload: map[string]interface{}{"key": "value"},
	}

	node1 := NewGraphNode("test-node", []string{"parent-1"}, data)
	node2 := NewGraphNode("test-node", []string{"parent-1"}, data)
	node3 := NewGraphNode("different-node", []string{"parent-1"}, data)

	hash1 := node1.Hash
	hash2 := node2.Hash
	hash3 := node3.Hash

	if hash1 == hash2 {
		t.Error("Expected different nodes to have different hashes due to timestamp differences")
	}

	if hash1 == hash3 {
		t.Error("Expected different nodes to have different hashes")
	}

	if len(hash1) == 0 {
		t.Error("Expected non-empty hash")
	}
}

func TestGraphNodeMethods(t *testing.T) {
	original := NewGraphNode("test-node", []string{"parent-1", "parent-2"}, GraphData{
		Payload: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		},
	})
	original.Weight = 3.14

	// Test AddParent method
	original.AddParent("parent-3")
	if len(original.Parents) != 3 {
		t.Errorf("Expected 3 parents after adding, got %d", len(original.Parents))
	}

	// Test adding duplicate parent (should not add)
	original.AddParent("parent-1")
	if len(original.Parents) != 3 {
		t.Errorf("Expected 3 parents after adding duplicate, got %d", len(original.Parents))
	}

	// Test AddChild method
	original.AddChild("child-1")
	if len(original.Children) != 1 {
		t.Errorf("Expected 1 child after adding, got %d", len(original.Children))
	}

	// Test RemoveParent method
	original.RemoveParent("parent-2")
	if len(original.Parents) != 2 {
		t.Errorf("Expected 2 parents after removing, got %d", len(original.Parents))
	}

	// Test RemoveChild method
	original.RemoveChild("child-1")
	if len(original.Children) != 0 {
		t.Errorf("Expected 0 children after removing, got %d", len(original.Children))
	}
}

func TestGraphNodeTimestamp(t *testing.T) {
	before := time.Now()
	node := NewGraphNode("test-node", []string{}, GraphData{})
	after := time.Now()

	if node.Timestamp.Before(before) || node.Timestamp.After(after) {
		t.Error("Node timestamp should be set to current time during creation")
	}
}

func TestGraphNodeEdgeCases(t *testing.T) {
	t.Run("Large payload data", func(t *testing.T) {
		largePayload := make(map[string]interface{})
		for i := 0; i < 1000; i++ {
			largePayload[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d", i)
		}

		largeData := GraphData{
			Payload: largePayload,
		}

		node := NewGraphNode("large-node", []string{}, largeData)
		if len(node.Data.Payload) != 1000 {
			t.Errorf("Expected 1000 data entries, got %d", len(node.Data.Payload))
		}
	})

	t.Run("Many parents", func(t *testing.T) {
		parents := make([]string, 100)
		for i := 0; i < 100; i++ {
			parents[i] = fmt.Sprintf("parent-%d", i)
		}

		node := NewGraphNode("many-parents-node", parents, GraphData{})
		if len(node.Parents) != 100 {
			t.Errorf("Expected 100 parents, got %d", len(node.Parents))
		}
	})

	t.Run("Unicode data", func(t *testing.T) {
		unicodeData := GraphData{
			Payload: map[string]interface{}{
				"emoji":   "🚀🌟💫",
				"chinese": "你好世界",
				"arabic":  "مرحبا بالعالم",
			},
		}

		node := NewGraphNode("unicode-node", []string{}, unicodeData)

		// Test serialization with unicode
		data, err := node.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize unicode data: %v", err)
		}

		var deserialized GraphNode
		err = json.Unmarshal(data, &deserialized)
		if err != nil {
			t.Fatalf("Failed to deserialize unicode data: %v", err)
		}

		if deserialized.Data.Payload["emoji"] != "🚀🌟💫" {
			t.Error("Unicode emoji not preserved during serialization")
		}
	})
}
