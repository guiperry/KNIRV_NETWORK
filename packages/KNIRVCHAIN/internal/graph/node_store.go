package graph

import (
	"encoding/json"
	"fmt"
	"sync"

	"KNIRVCHAIN/internal/database"
	"KNIRVCHAIN/internal/errors"
	"KNIRVCHAIN/internal/types"
)

// NodeStore manages persistence of graph nodes
type NodeStore struct {
	db     database.LevelDBManager
	chroma database.ChromemDBManager
	mutex  sync.RWMutex
}

// NewNodeStore creates a new node store
func NewNodeStore(db database.LevelDBManager, chroma database.ChromemDBManager) *NodeStore {
	return &NodeStore{
		db:     db,
		chroma: chroma,
	}
}

// StoreErrorNode stores an ErrorNode
func (ns *NodeStore) StoreErrorNode(node *types.ErrorNode) error {
	ns.mutex.Lock()
	defer ns.mutex.Unlock()

	if node == nil {
		return errors.NewValidationError("node cannot be nil")
	}

	key := fmt.Sprintf("error_node:%s", node.ID)
	data, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("failed to marshal error node: %w", err)
	}

	if err := ns.db.Put([]byte(key), data); err != nil {
		return fmt.Errorf("failed to store error node: %w", err)
	}

	// Store in ChromaDB for semantic search
	if ns.chroma != nil {
		content := fmt.Sprintf("Error Type: %s, Signature: %s, Model: %s, Context: %v",
			node.ErrorType, node.ErrorSignature, node.ModelOrigin, node.Context)

		doc := database.Document{
			ID:      node.ID,
			Content: content,
			Metadata: map[string]interface{}{
				"type":           "error_node",
				"error_type":     node.ErrorType,
				"model_origin":   node.ModelOrigin,
				"failure_count":  node.FailureCount,
				"created_at":     node.CreatedAt,
				"status":         node.Status,
			},
		}

		collectionName := "knirvchain_nodes"
		if err := ns.chroma.AddDocuments(collectionName, []database.Document{doc}); err != nil {
			// Log warning but don't fail the operation
			fmt.Printf("Warning: failed to store error node in ChromaDB: %v\n", err)
		}
	}

	return nil
}

// GetErrorNode retrieves an ErrorNode by ID
func (ns *NodeStore) GetErrorNode(id string) (*types.ErrorNode, error) {
	ns.mutex.RLock()
	defer ns.mutex.RUnlock()

	key := fmt.Sprintf("error_node:%s", id)
	data, err := ns.db.Get([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("failed to get error node: %w", err)
	}

	var node types.ErrorNode
	if err := json.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("failed to unmarshal error node: %w", err)
	}

	return &node, nil
}

// StoreSkillNode stores a SkillNode
func (ns *NodeStore) StoreSkillNode(node *types.SkillNode) error {
	ns.mutex.Lock()
	defer ns.mutex.Unlock()

	if node == nil {
		return errors.NewValidationError("node cannot be nil")
	}

	key := fmt.Sprintf("skill_node:%s", node.ID)
	data, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("failed to marshal skill node: %w", err)
	}

	if err := ns.db.Put([]byte(key), data); err != nil {
		return fmt.Errorf("failed to store skill node: %w", err)
	}

	// Store in ChromaDB for semantic search
	if ns.chroma != nil {
		content := fmt.Sprintf("Skill: %s, Error Node: %s, Miner: %s, Performance: Accuracy %.2f, F1 %.2f",
			node.Name, node.ErrorNodeID, node.MinerAddress, node.Performance.Accuracy, node.Performance.F1Score)

		doc := database.Document{
			ID:      node.ID,
			Content: content,
			Metadata: map[string]interface{}{
				"type":           "skill_node",
				"name":           node.Name,
				"error_node_id":  node.ErrorNodeID,
				"miner_address":  node.MinerAddress,
				"nrn_reward":     node.NRNReward,
				"created_at":     node.CreatedAt,
				"accuracy":       node.Performance.Accuracy,
				"f1_score":       node.Performance.F1Score,
			},
		}

		collectionName := "knirvchain_nodes"
		if err := ns.chroma.AddDocuments(collectionName, []database.Document{doc}); err != nil {
			fmt.Printf("Warning: failed to store skill node in ChromaDB: %v\n", err)
		}
	}

	return nil
}

// GetSkillNode retrieves a SkillNode by ID
func (ns *NodeStore) GetSkillNode(id string) (*types.SkillNode, error) {
	ns.mutex.RLock()
	defer ns.mutex.RUnlock()

	key := fmt.Sprintf("skill_node:%s", id)
	data, err := ns.db.Get([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("failed to get skill node: %w", err)
	}

	var node types.SkillNode
	if err := json.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("failed to unmarshal skill node: %w", err)
	}

	return &node, nil
}

// StoreContextNode stores a ContextNode
func (ns *NodeStore) StoreContextNode(node *types.ContextNode) error {
	ns.mutex.Lock()
	defer ns.mutex.Unlock()

	if node == nil {
		return errors.NewValidationError("node cannot be nil")
	}

	key := fmt.Sprintf("context_node:%s", node.ID)
	data, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("failed to marshal context node: %w", err)
	}

	if err := ns.db.Put([]byte(key), data); err != nil {
		return fmt.Errorf("failed to store context node: %w", err)
	}

	// Store in ChromaDB for semantic search
	if ns.chroma != nil {
		content := fmt.Sprintf("Context Type: %s, Pattern: %s, Frequency: %d, Resources: %v",
			node.ContextType, node.PatternSignature, node.UsageFrequency, node.RequiredResources)

		doc := database.Document{
			ID:      node.ID,
			Content: content,
			Metadata: map[string]interface{}{
				"type":             "context_node",
				"context_type":     node.ContextType,
				"usage_frequency":  node.UsageFrequency,
				"created_at":       node.CreatedAt,
			},
		}

		collectionName := "knirvchain_nodes"
		if err := ns.chroma.AddDocuments(collectionName, []database.Document{doc}); err != nil {
			fmt.Printf("Warning: failed to store context node in ChromaDB: %v\n", err)
		}
	}

	return nil
}

// GetContextNode retrieves a ContextNode by ID
func (ns *NodeStore) GetContextNode(id string) (*types.ContextNode, error) {
	ns.mutex.RLock()
	defer ns.mutex.RUnlock()

	key := fmt.Sprintf("context_node:%s", id)
	data, err := ns.db.Get([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("failed to get context node: %w", err)
	}

	var node types.ContextNode
	if err := json.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("failed to unmarshal context node: %w", err)
	}

	return &node, nil
}

// StoreCapabilityNode stores a CapabilityNode
func (ns *NodeStore) StoreCapabilityNode(node *types.CapabilityNode) error {
	ns.mutex.Lock()
	defer ns.mutex.Unlock()

	if node == nil {
		return errors.NewValidationError("node cannot be nil")
	}

	key := fmt.Sprintf("capability_node:%s", node.ID)
	data, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("failed to marshal capability node: %w", err)
	}

	if err := ns.db.Put([]byte(key), data); err != nil {
		return fmt.Errorf("failed to store capability node: %w", err)
	}

	// Store in ChromaDB for semantic search
	if ns.chroma != nil {
		content := fmt.Sprintf("Capability: %s, Context Node: %s, Minter: %s, Cost: %d NRN",
			node.CapabilityType, node.ContextNodeID, node.MinterAddress, node.NRNCost)

		doc := database.Document{
			ID:      node.ID,
			Content: content,
			Metadata: map[string]interface{}{
				"type":            "capability_node",
				"capability_type": node.CapabilityType,
				"context_node_id": node.ContextNodeID,
				"minter_address":  node.MinterAddress,
				"nrn_cost":        node.NRNCost,
				"created_at":      node.CreatedAt,
			},
		}

		collectionName := "knirvchain_nodes"
		if err := ns.chroma.AddDocuments(collectionName, []database.Document{doc}); err != nil {
			fmt.Printf("Warning: failed to store capability node in ChromaDB: %v\n", err)
		}
	}

	return nil
}

// GetCapabilityNode retrieves a CapabilityNode by ID
func (ns *NodeStore) GetCapabilityNode(id string) (*types.CapabilityNode, error) {
	ns.mutex.RLock()
	defer ns.mutex.RUnlock()

	key := fmt.Sprintf("capability_node:%s", id)
	data, err := ns.db.Get([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("failed to get capability node: %w", err)
	}

	var node types.CapabilityNode
	if err := json.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("failed to unmarshal capability node: %w", err)
	}

	return &node, nil
}

// StoreIdeaNode stores an IdeaNode
func (ns *NodeStore) StoreIdeaNode(node *types.IdeaNode) error {
	ns.mutex.Lock()
	defer ns.mutex.Unlock()

	if node == nil {
		return errors.NewValidationError("node cannot be nil")
	}

	key := fmt.Sprintf("idea_node:%s", node.ID)
	data, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("failed to marshal idea node: %w", err)
	}

	if err := ns.db.Put([]byte(key), data); err != nil {
		return fmt.Errorf("failed to store idea node: %w", err)
	}

	// Store in ChromaDB for semantic search
	if ns.chroma != nil {
		content := fmt.Sprintf("Idea Type: %s, Origin NIM: %s, Novelty Score: %.2f, Dependencies: %v",
			node.IdeaType, node.OriginNIM, node.Novelty.Score, node.Dependencies)

		doc := database.Document{
			ID:      node.ID,
			Content: content,
			Metadata: map[string]interface{}{
				"type":          "idea_node",
				"idea_type":     node.IdeaType,
				"origin_nim":    node.OriginNIM,
				"novelty_score": node.Novelty.Score,
				"created_at":    node.CreatedAt,
			},
		}

		collectionName := "knirvchain_nodes"
		if err := ns.chroma.AddDocuments(collectionName, []database.Document{doc}); err != nil {
			fmt.Printf("Warning: failed to store idea node in ChromaDB: %v\n", err)
		}
	}

	return nil
}

// GetIdeaNode retrieves an IdeaNode by ID
func (ns *NodeStore) GetIdeaNode(id string) (*types.IdeaNode, error) {
	ns.mutex.RLock()
	defer ns.mutex.RUnlock()

	key := fmt.Sprintf("idea_node:%s", id)
	data, err := ns.db.Get([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("failed to get idea node: %w", err)
	}

	var node types.IdeaNode
	if err := json.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("failed to unmarshal idea node: %w", err)
	}

	return &node, nil
}

// StorePropertyNode stores a PropertyNode
func (ns *NodeStore) StorePropertyNode(node *types.PropertyNode) error {
	ns.mutex.Lock()
	defer ns.mutex.Unlock()

	if node == nil {
		return errors.NewValidationError("node cannot be nil")
	}

	key := fmt.Sprintf("property_node:%s", node.ID)
	data, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("failed to marshal property node: %w", err)
	}

	if err := ns.db.Put([]byte(key), data); err != nil {
		return fmt.Errorf("failed to store property node: %w", err)
	}

	// Store in ChromaDB for semantic search
	if ns.chroma != nil {
		content := fmt.Sprintf("Property: %s, Idea Node: %s, Maker: %s, IP Type: %s",
			node.IPType, node.IdeaNodeID, node.MakerAddress, node.IPType)

		doc := database.Document{
			ID:      node.ID,
			Content: content,
			Metadata: map[string]interface{}{
				"type":           "property_node",
				"idea_node_id":   node.IdeaNodeID,
				"maker_address":  node.MakerAddress,
				"ip_type":        node.IPType,
				"created_at":     node.CreatedAt,
			},
		}

		collectionName := "knirvchain_nodes"
		if err := ns.chroma.AddDocuments(collectionName, []database.Document{doc}); err != nil {
			fmt.Printf("Warning: failed to store property node in ChromaDB: %v\n", err)
		}
	}

	return nil
}

// GetPropertyNode retrieves a PropertyNode by ID
func (ns *NodeStore) GetPropertyNode(id string) (*types.PropertyNode, error) {
	ns.mutex.RLock()
	defer ns.mutex.RUnlock()

	key := fmt.Sprintf("property_node:%s", id)
	data, err := ns.db.Get([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("failed to get property node: %w", err)
	}

	var node types.PropertyNode
	if err := json.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("failed to unmarshal property node: %w", err)
	}

	return &node, nil
}

// DeleteNode deletes a node by ID and type
func (ns *NodeStore) DeleteNode(nodeType, id string) error {
	ns.mutex.Lock()
	defer ns.mutex.Unlock()

	key := fmt.Sprintf("%s:%s", nodeType, id)

	// Delete from LevelDB
	if err := ns.db.Delete([]byte(key)); err != nil {
		return fmt.Errorf("failed to delete node from LevelDB: %w", err)
	}

	// Delete from ChromaDB if available
	if ns.chroma != nil {
		collectionName := "knirvchain_nodes"
		if err := ns.chroma.DeleteDocuments(collectionName, []string{id}); err != nil {
			fmt.Printf("Warning: failed to delete node from ChromaDB: %v\n", err)
		}
	}

	return nil
}

// ListNodesByType returns all nodes of a specific type
func (ns *NodeStore) ListNodesByType(nodeType string) ([]string, error) {
	ns.mutex.RLock()
	defer ns.mutex.RUnlock()

	var nodes []string
	prefix := []byte(fmt.Sprintf("%s:", nodeType))

	iter := ns.db.NewIterator(prefix)
	defer iter.Release()

	for iter.Valid() {
		if iter.Error() != nil {
			return nil, fmt.Errorf("iterator error: %w", iter.Error())
		}

		key := string(iter.Key())
		// Extract ID from key (format: "type:id")
		if len(key) > len(nodeType)+1 {
			id := key[len(nodeType)+1:]
			nodes = append(nodes, id)
		}

		iter.Next()
	}

	return nodes, nil
}

// GetNodeCount returns the count of nodes by type
func (ns *NodeStore) GetNodeCount(nodeType string) (int, error) {
	nodes, err := ns.ListNodesByType(nodeType)
	if err != nil {
		return 0, err
	}
	return len(nodes), nil
}

// Close closes the node store
func (ns *NodeStore) Close() error {
	ns.mutex.Lock()
	defer ns.mutex.Unlock()

	if ns.db != nil {
		if err := ns.db.Close(); err != nil {
			return fmt.Errorf("failed to close LevelDB: %w", err)
		}
	}

	if ns.chroma != nil {
		if err := ns.chroma.Stop(); err != nil {
			return fmt.Errorf("failed to stop ChromaDB: %w", err)
		}
	}

	return nil
}