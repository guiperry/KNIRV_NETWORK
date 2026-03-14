package graph

import (
	"encoding/json"
	"fmt"
	"sync"

	"KNIRVCHAIN/internal/database"
	"KNIRVCHAIN/internal/errors"
)

// RelationshipType defines the type of relationship between nodes
type RelationshipType string

const (
	RelationshipTypeErrorToSkill     RelationshipType = "error_to_skill"
	RelationshipTypeContextToCapability RelationshipType = "context_to_capability"
	RelationshipTypeIdeaToProperty    RelationshipType = "idea_to_property"
	RelationshipTypeDependency        RelationshipType = "dependency"
	RelationshipTypeReference         RelationshipType = "reference"
	RelationshipTypeDerivedFrom       RelationshipType = "derived_from"
)

// Relationship represents a relationship between two nodes
type Relationship struct {
	ID             string           `json:"id"`
	FromNodeID     string           `json:"from_node_id"`
	FromNodeType   string           `json:"from_node_type"`
	ToNodeID       string           `json:"to_node_id"`
	ToNodeType     string           `json:"to_node_type"`
	RelationshipType RelationshipType `json:"relationship_type"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt      int64            `json:"created_at"`
}

// RelationshipManager manages node relationships
type RelationshipManager struct {
	db    database.LevelDBManager
	mutex sync.RWMutex
}

// NewRelationshipManager creates a new relationship manager
func NewRelationshipManager(db database.LevelDBManager) *RelationshipManager {
	return &RelationshipManager{
		db: db,
	}
}

// CreateRelationship creates a relationship between two nodes
func (rm *RelationshipManager) CreateRelationship(fromNodeType, fromNodeID, toNodeType, toNodeID string, relType RelationshipType, metadata map[string]interface{}) (*Relationship, error) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if fromNodeID == "" || toNodeID == "" {
		return nil, errors.NewValidationError("node IDs cannot be empty")
	}

	if fromNodeType == "" || toNodeType == "" {
		return nil, errors.NewValidationError("node types cannot be empty")
	}

	// Generate relationship ID
	id := fmt.Sprintf("%s:%s:%s:%s", fromNodeType, fromNodeID, relType, toNodeID)

	relationship := &Relationship{
		ID:               id,
		FromNodeID:       fromNodeID,
		FromNodeType:     fromNodeType,
		ToNodeID:         toNodeID,
		ToNodeType:       toNodeType,
		RelationshipType: relType,
		Metadata:         metadata,
		CreatedAt:        0, // Will be set by caller if needed
	}

	// Store relationship
	key := fmt.Sprintf("relationship:%s", id)
	data, err := json.Marshal(relationship)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal relationship: %w", err)
	}

	if err := rm.db.Put([]byte(key), data); err != nil {
		return nil, fmt.Errorf("failed to store relationship: %w", err)
	}

	// Store reverse relationship for bidirectional traversal
	reverseID := fmt.Sprintf("%s:%s:%s:%s", toNodeType, toNodeID, getReverseRelationshipType(relType), fromNodeID)
	reverseRelationship := &Relationship{
		ID:               reverseID,
		FromNodeID:       toNodeID,
		FromNodeType:     toNodeType,
		ToNodeID:         fromNodeID,
		ToNodeType:       fromNodeType,
		RelationshipType: getReverseRelationshipType(relType),
		Metadata:         metadata,
		CreatedAt:        relationship.CreatedAt,
	}

	reverseKey := fmt.Sprintf("relationship:%s", reverseID)
	reverseData, err := json.Marshal(reverseRelationship)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal reverse relationship: %w", err)
	}

	if err := rm.db.Put([]byte(reverseKey), reverseData); err != nil {
		return nil, fmt.Errorf("failed to store reverse relationship: %w", err)
	}

	return relationship, nil
}

// getReverseRelationshipType returns the reverse relationship type
func getReverseRelationshipType(relType RelationshipType) RelationshipType {
	switch relType {
	case RelationshipTypeErrorToSkill:
		return RelationshipTypeErrorToSkill // Bidirectional
	case RelationshipTypeContextToCapability:
		return RelationshipTypeContextToCapability // Bidirectional
	case RelationshipTypeIdeaToProperty:
		return RelationshipTypeIdeaToProperty // Bidirectional
	case RelationshipTypeDependency:
		return RelationshipTypeReference // Reverse of dependency
	case RelationshipTypeReference:
		return RelationshipTypeDependency // Reverse of reference
	case RelationshipTypeDerivedFrom:
		return RelationshipTypeDerivedFrom // Bidirectional for derivation
	default:
		return relType
	}
}

// GetRelationship retrieves a relationship by ID
func (rm *RelationshipManager) GetRelationship(id string) (*Relationship, error) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	key := fmt.Sprintf("relationship:%s", id)
	data, err := rm.db.Get([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("failed to get relationship: %w", err)
	}

	var relationship Relationship
	if err := json.Unmarshal(data, &relationship); err != nil {
		return nil, fmt.Errorf("failed to unmarshal relationship: %w", err)
	}

	return &relationship, nil
}

// GetRelationshipsFromNode returns all relationships from a specific node
func (rm *RelationshipManager) GetRelationshipsFromNode(nodeType, nodeID string) ([]*Relationship, error) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	var relationships []*Relationship
	prefix := []byte(fmt.Sprintf("relationship:%s:%s:", nodeType, nodeID))

	iter := rm.db.NewIterator(prefix)
	defer iter.Release()

	for iter.Valid() {
		if iter.Error() != nil {
			return nil, fmt.Errorf("iterator error: %w", iter.Error())
		}

		var relationship Relationship
		if err := json.Unmarshal(iter.Value(), &relationship); err != nil {
			return nil, fmt.Errorf("failed to unmarshal relationship: %w", err)
		}

		relationships = append(relationships, &relationship)
		iter.Next()
	}

	return relationships, nil
}

// GetRelationshipsToNode returns all relationships to a specific node
func (rm *RelationshipManager) GetRelationshipsToNode(nodeType, nodeID string) ([]*Relationship, error) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	var relationships []*Relationship
	// For reverse relationships, we need to search for relationships where ToNode matches
	prefix := []byte("relationship:")

	iter := rm.db.NewIterator(prefix)
	defer iter.Release()

	for iter.Valid() {
		if iter.Error() != nil {
			return nil, fmt.Errorf("iterator error: %w", iter.Error())
		}

		var relationship Relationship
		if err := json.Unmarshal(iter.Value(), &relationship); err != nil {
			return nil, fmt.Errorf("failed to unmarshal relationship: %w", err)
		}

		if relationship.ToNodeType == nodeType && relationship.ToNodeID == nodeID {
			relationships = append(relationships, &relationship)
		}

		iter.Next()
	}

	return relationships, nil
}

// GetRelationshipsByType returns all relationships of a specific type
func (rm *RelationshipManager) GetRelationshipsByType(relType RelationshipType) ([]*Relationship, error) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	var relationships []*Relationship
	prefix := []byte("relationship:")

	iter := rm.db.NewIterator(prefix)
	defer iter.Release()

	for iter.Valid() {
		if iter.Error() != nil {
			return nil, fmt.Errorf("iterator error: %w", iter.Error())
		}

		var relationship Relationship
		if err := json.Unmarshal(iter.Value(), &relationship); err != nil {
			return nil, fmt.Errorf("failed to unmarshal relationship: %w", err)
		}

		if relationship.RelationshipType == relType {
			relationships = append(relationships, &relationship)
		}

		iter.Next()
	}

	return relationships, nil
}

// DeleteRelationship deletes a relationship by ID
func (rm *RelationshipManager) DeleteRelationship(id string) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	key := fmt.Sprintf("relationship:%s", id)

	// Get the relationship first to delete the reverse relationship too
	data, err := rm.db.Get([]byte(key))
	if err != nil {
		return fmt.Errorf("failed to get relationship for deletion: %w", err)
	}

	var relationship Relationship
	if err := json.Unmarshal(data, &relationship); err != nil {
		return fmt.Errorf("failed to unmarshal relationship for deletion: %w", err)
	}

	// Delete main relationship
	if err := rm.db.Delete([]byte(key)); err != nil {
		return fmt.Errorf("failed to delete relationship: %w", err)
	}

	// Delete reverse relationship
	reverseID := fmt.Sprintf("%s:%s:%s:%s", relationship.ToNodeType, relationship.ToNodeID, getReverseRelationshipType(relationship.RelationshipType), relationship.FromNodeID)
	reverseKey := fmt.Sprintf("relationship:%s", reverseID)

	if err := rm.db.Delete([]byte(reverseKey)); err != nil {
		// Log warning but don't fail - reverse relationship might not exist
		fmt.Printf("Warning: failed to delete reverse relationship %s: %v\n", reverseID, err)
	}

	return nil
}

// TraverseFromNode performs a breadth-first traversal from a starting node
func (rm *RelationshipManager) TraverseFromNode(nodeType, nodeID string, maxDepth int, relTypes []RelationshipType) ([]*TraversalResult, error) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	visited := make(map[string]bool)
	queue := []*TraversalNode{{NodeType: nodeType, NodeID: nodeID, Depth: 0, Path: []string{fmt.Sprintf("%s:%s", nodeType, nodeID)}}}
	var results []*TraversalResult

	relTypeSet := make(map[RelationshipType]bool)
	for _, rt := range relTypes {
		relTypeSet[rt] = true
	}

	for len(queue) > 0 && len(results) < 1000 { // Limit results to prevent memory issues
		current := queue[0]
		queue = queue[1:]

		nodeKey := fmt.Sprintf("%s:%s", current.NodeType, current.NodeID)
		if visited[nodeKey] {
			continue
		}
		visited[nodeKey] = true

		// Add current node to results
		results = append(results, &TraversalResult{
			NodeType:   current.NodeType,
			NodeID:     current.NodeID,
			Depth:      current.Depth,
			Path:       append([]string{}, current.Path...),
		})

		// Stop if we've reached max depth
		if current.Depth >= maxDepth {
			continue
		}

		// Get relationships from current node
		relationships, err := rm.GetRelationshipsFromNode(current.NodeType, current.NodeID)
		if err != nil {
			continue // Skip on error
		}

		for _, rel := range relationships {
			// Filter by relationship types if specified
			if len(relTypes) > 0 && !relTypeSet[rel.RelationshipType] {
				continue
			}

			nextNodeKey := fmt.Sprintf("%s:%s", rel.ToNodeType, rel.ToNodeID)
			if !visited[nextNodeKey] {
				newPath := append(append([]string{}, current.Path...), nextNodeKey)
				queue = append(queue, &TraversalNode{
					NodeType: rel.ToNodeType,
					NodeID:   rel.ToNodeID,
					Depth:    current.Depth + 1,
					Path:     newPath,
				})
			}
		}
	}

	return results, nil
}

// TraversalNode represents a node in the traversal queue
type TraversalNode struct {
	NodeType string
	NodeID   string
	Depth    int
	Path     []string
}

// TraversalResult represents a result from graph traversal
type TraversalResult struct {
	NodeType string   `json:"node_type"`
	NodeID   string   `json:"node_id"`
	Depth    int      `json:"depth"`
	Path     []string `json:"path"`
}

// GetShortestPath finds the shortest path between two nodes
func (rm *RelationshipManager) GetShortestPath(fromType, fromID, toType, toID string, maxDepth int) ([]*TraversalResult, error) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	visited := make(map[string]bool)
	queue := []*TraversalNode{{NodeType: fromType, NodeID: fromID, Depth: 0, Path: []string{fmt.Sprintf("%s:%s", fromType, fromID)}}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		nodeKey := fmt.Sprintf("%s:%s", current.NodeType, current.NodeID)
		if visited[nodeKey] {
			continue
		}
		visited[nodeKey] = true

		// Check if we found the target
		if current.NodeType == toType && current.NodeID == toID {
			var results []*TraversalResult
			for _, pathNode := range current.Path {
				// Parse node type and ID from path
				parts := splitNodeKey(pathNode)
				if len(parts) == 2 {
					results = append(results, &TraversalResult{
						NodeType: parts[0],
						NodeID:   parts[1],
						Depth:    len(results), // Depth in path
						Path:     current.Path,
					})
				}
			}
			return results, nil
		}

		// Stop if we've reached max depth
		if current.Depth >= maxDepth {
			continue
		}

		// Get relationships from current node
		relationships, err := rm.GetRelationshipsFromNode(current.NodeType, current.NodeID)
		if err != nil {
			continue
		}

		for _, rel := range relationships {
			nextNodeKey := fmt.Sprintf("%s:%s", rel.ToNodeType, rel.ToNodeID)
			if !visited[nextNodeKey] {
				newPath := append(append([]string{}, current.Path...), nextNodeKey)
				queue = append(queue, &TraversalNode{
					NodeType: rel.ToNodeType,
					NodeID:   rel.ToNodeID,
					Depth:    current.Depth + 1,
					Path:     newPath,
				})
			}
		}
	}

	return nil, fmt.Errorf("no path found between %s:%s and %s:%s within max depth %d", fromType, fromID, toType, toID, maxDepth)
}

// splitNodeKey splits a node key into type and ID
func splitNodeKey(key string) []string {
	for i, char := range key {
		if char == ':' && i > 0 {
			return []string{key[:i], key[i+1:]}
		}
	}
	return []string{key}
}

// GetRelationshipCount returns the count of relationships by type
func (rm *RelationshipManager) GetRelationshipCount(relType RelationshipType) (int, error) {
	relationships, err := rm.GetRelationshipsByType(relType)
	if err != nil {
		return 0, err
	}
	return len(relationships), nil
}

// Close closes the relationship manager
func (rm *RelationshipManager) Close() error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if rm.db != nil {
		return rm.db.Close()
	}
	return nil
}