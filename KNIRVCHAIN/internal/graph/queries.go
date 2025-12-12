package graph

import (
	"fmt"
	"sort"
	"time"

	"KNIRVCHAIN/internal/database"
	"KNIRVCHAIN/internal/types"
)

// GraphQueries provides high-level query operations on the knowledge graph
type GraphQueries struct {
	nodeStore    *NodeStore
	relManager   *RelationshipManager
	chromaDB     database.ChromemDBManager
}

// NewGraphQueries creates a new graph queries instance
func NewGraphQueries(nodeStore *NodeStore, relManager *RelationshipManager, chromaDB database.ChromemDBManager) *GraphQueries {
	return &GraphQueries{
		nodeStore:  nodeStore,
		relManager: relManager,
		chromaDB:   chromaDB,
	}
}

// GetNodeStore returns the node store
func (gq *GraphQueries) GetNodeStore() *NodeStore {
	return gq.nodeStore
}

// FindErrorNodesByType finds error nodes by error type
func (gq *GraphQueries) FindErrorNodesByType(errorType string, limit int) ([]*types.ErrorNode, error) {
	nodeIDs, err := gq.nodeStore.ListNodesByType("error_node")
	if err != nil {
		return nil, fmt.Errorf("failed to list error nodes: %w", err)
	}

	var errorNodes []*types.ErrorNode
	for _, id := range nodeIDs {
		node, err := gq.nodeStore.GetErrorNode(id)
		if err != nil {
			continue // Skip on error
		}

		if node.ErrorType == errorType {
			errorNodes = append(errorNodes, node)
			if limit > 0 && len(errorNodes) >= limit {
				break
			}
		}
	}

	return errorNodes, nil
}

// FindSkillNodesByMiner finds skill nodes by miner address
func (gq *GraphQueries) FindSkillNodesByMiner(minerAddress string, limit int) ([]*types.SkillNode, error) {
	nodeIDs, err := gq.nodeStore.ListNodesByType("skill_node")
	if err != nil {
		return nil, fmt.Errorf("failed to list skill nodes: %w", err)
	}

	var skillNodes []*types.SkillNode
	for _, id := range nodeIDs {
		node, err := gq.nodeStore.GetSkillNode(id)
		if err != nil {
			continue
		}

		if node.MinerAddress == minerAddress {
			skillNodes = append(skillNodes, node)
			if limit > 0 && len(skillNodes) >= limit {
				break
			}
		}
	}

	return skillNodes, nil
}

// GetCapabilityNode retrieves a single capability node by ID
func (gq *GraphQueries) GetCapabilityNode(id string) (*types.CapabilityNode, error) {
	return gq.nodeStore.GetCapabilityNode(id)
}

// FindCapabilityNodesByType finds capability nodes by capability type
func (gq *GraphQueries) FindCapabilityNodesByType(capabilityType types.CapabilityType, limit int) ([]*types.CapabilityNode, error) {
	nodeIDs, err := gq.nodeStore.ListNodesByType("capability_node")
	if err != nil {
		return nil, fmt.Errorf("failed to list capability nodes: %w", err)
	}

	var capabilityNodes []*types.CapabilityNode
	for _, id := range nodeIDs {
		node, err := gq.nodeStore.GetCapabilityNode(id)
		if err != nil {
			continue
		}

		if node.CapabilityType == capabilityType {
			capabilityNodes = append(capabilityNodes, node)
			if limit > 0 && len(capabilityNodes) >= limit {
				break
			}
		}
	}

	return capabilityNodes, nil
}

// FindCapabilityNodesByMinter finds capability nodes by minter address
func (gq *GraphQueries) FindCapabilityNodesByMinter(minterAddress string, limit int) ([]*types.CapabilityNode, error) {
	nodeIDs, err := gq.nodeStore.ListNodesByType("capability_node")
	if err != nil {
		return nil, fmt.Errorf("failed to list capability nodes: %w", err)
	}

	var capabilityNodes []*types.CapabilityNode
	for _, id := range nodeIDs {
		node, err := gq.nodeStore.GetCapabilityNode(id)
		if err != nil {
			continue
		}

		if node.MinterAddress == minterAddress {
			capabilityNodes = append(capabilityNodes, node)
			if limit > 0 && len(capabilityNodes) >= limit {
				break
			}
		}
	}

	return capabilityNodes, nil
}

// FindPropertyNodesByOwner finds property nodes by current owner
func (gq *GraphQueries) FindPropertyNodesByOwner(ownerAddress string, limit int) ([]*types.PropertyNode, error) {
	nodeIDs, err := gq.nodeStore.ListNodesByType("property_node")
	if err != nil {
		return nil, fmt.Errorf("failed to list property nodes: %w", err)
	}

	var propertyNodes []*types.PropertyNode
	for _, id := range nodeIDs {
		node, err := gq.nodeStore.GetPropertyNode(id)
		if err != nil {
			continue
		}

		if node.Ownership.CurrentOwner == ownerAddress {
			propertyNodes = append(propertyNodes, node)
			if limit > 0 && len(propertyNodes) >= limit {
				break
			}
		}
	}

	return propertyNodes, nil
}

// GetSkillsForError finds all skill nodes that solve a specific error
func (gq *GraphQueries) GetSkillsForError(errorNodeID string) ([]*types.SkillNode, error) {
	relationships, err := gq.relManager.GetRelationshipsToNode("error_node", errorNodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get relationships to error node: %w", err)
	}

	var skillNodes []*types.SkillNode
	for _, rel := range relationships {
		if rel.RelationshipType == RelationshipTypeErrorToSkill && rel.FromNodeType == "skill_node" {
			node, err := gq.nodeStore.GetSkillNode(rel.FromNodeID)
			if err != nil {
				continue
			}
			skillNodes = append(skillNodes, node)
		}
	}

	return skillNodes, nil
}

// GetCapabilitiesForContext finds all capability nodes derived from a context
func (gq *GraphQueries) GetCapabilitiesForContext(contextNodeID string) ([]*types.CapabilityNode, error) {
	relationships, err := gq.relManager.GetRelationshipsToNode("context_node", contextNodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get relationships to context node: %w", err)
	}

	var capabilityNodes []*types.CapabilityNode
	for _, rel := range relationships {
		if rel.RelationshipType == RelationshipTypeContextToCapability && rel.FromNodeType == "capability_node" {
			node, err := gq.nodeStore.GetCapabilityNode(rel.FromNodeID)
			if err != nil {
				continue
			}
			capabilityNodes = append(capabilityNodes, node)
		}
	}

	return capabilityNodes, nil
}

// GetPropertiesForIdea finds all property nodes derived from an idea
func (gq *GraphQueries) GetPropertiesForIdea(ideaNodeID string) ([]*types.PropertyNode, error) {
	relationships, err := gq.relManager.GetRelationshipsToNode("idea_node", ideaNodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get relationships to idea node: %w", err)
	}

	var propertyNodes []*types.PropertyNode
	for _, rel := range relationships {
		if rel.RelationshipType == RelationshipTypeIdeaToProperty && rel.FromNodeType == "property_node" {
			node, err := gq.nodeStore.GetPropertyNode(rel.FromNodeID)
			if err != nil {
				continue
			}
			propertyNodes = append(propertyNodes, node)
		}
	}

	return propertyNodes, nil
}

// GetTopPerformingSkills returns skill nodes sorted by performance
func (gq *GraphQueries) GetTopPerformingSkills(limit int) ([]*SkillPerformanceResult, error) {
	nodeIDs, err := gq.nodeStore.ListNodesByType("skill_node")
	if err != nil {
		return nil, fmt.Errorf("failed to list skill nodes: %w", err)
	}

	var results []*SkillPerformanceResult
	for _, id := range nodeIDs {
		node, err := gq.nodeStore.GetSkillNode(id)
		if err != nil {
			continue
		}

		results = append(results, &SkillPerformanceResult{
			SkillNode: *node,
			Score:     calculatePerformanceScore(node.Performance),
		})
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// SkillPerformanceResult represents a skill with its performance score
type SkillPerformanceResult struct {
	SkillNode types.SkillNode `json:"skill_node"`
	Score     float64         `json:"score"`
}

// calculatePerformanceScore calculates a composite performance score
func calculatePerformanceScore(perf types.SkillPerformance) float64 {
	// Weighted score: 40% accuracy, 30% F1, 20% precision, 10% recall
	score := (perf.Accuracy * 0.4) +
		(perf.F1Score * 0.3) +
		(perf.Precision * 0.2) +
		(perf.Recall * 0.1)

	// Boost score based on test coverage
	testCoverage := float64(perf.TestCasesPass) / float64(perf.TestCasesRun)
	score *= (0.5 + testCoverage*0.5) // Range: 0.5 to 1.0

	return score
}

// GetMostValuableProperties returns property nodes sorted by value
func (gq *GraphQueries) GetMostValuableProperties(limit int) ([]*PropertyValueResult, error) {
	nodeIDs, err := gq.nodeStore.ListNodesByType("property_node")
	if err != nil {
		return nil, fmt.Errorf("failed to list property nodes: %w", err)
	}

	var results []*PropertyValueResult
	for _, id := range nodeIDs {
		node, err := gq.nodeStore.GetPropertyNode(id)
		if err != nil {
			continue
		}

		value := calculatePropertyValue(node)
		results = append(results, &PropertyValueResult{
			PropertyNode: *node,
			Value:        value,
		})
	}

	// Sort by value descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Value > results[j].Value
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// PropertyValueResult represents a property with its calculated value
type PropertyValueResult struct {
	PropertyNode types.PropertyNode `json:"property_node"`
	Value        float64            `json:"value"`
}

// calculatePropertyValue calculates the value of a property based on various factors
func calculatePropertyValue(node *types.PropertyNode) float64 {
	baseValue := 100.0 // Base value

	// Factor in novelty score from the idea
	if node.IdeaNodeID != "" {
		// This would need to be looked up, for now use a placeholder
		metadataBonus := 0.0
		if node.NFTPointer.MetadataURI != "" {
			metadataBonus = 1.0
		}
		noveltyMultiplier := 1.0 + metadataBonus // Simple heuristic
		baseValue *= noveltyMultiplier
	}

	// Factor in IP type
	switch node.IPType {
	case types.IPTypePatent:
		baseValue *= 3.0
	case types.IPTypeCopyright:
		baseValue *= 2.0
	case types.IPTypeTradeSecret:
		baseValue *= 2.5
	}

	// Factor in ownership history (more transfers = more valuable)
	transferCount := len(node.Ownership.History)
	baseValue *= (1.0 + float64(transferCount)*0.1)

	return baseValue
}

// SemanticSearch performs semantic search across all nodes
func (gq *GraphQueries) SemanticSearch(query string, limit int) ([]*SemanticSearchResult, error) {
	if gq.chromaDB == nil {
		return nil, fmt.Errorf("ChromaDB not available for semantic search")
	}

	collectionName := "knirvchain_nodes"
	searchResults, err := gq.chromaDB.Search(collectionName, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to perform semantic search: %w", err)
	}

	var results []*SemanticSearchResult
	for _, result := range searchResults {
		// Get the full node data based on the ID and metadata
		nodeType, ok := result.Metadata["type"].(string)
		if !ok {
			continue
		}

		fullNode, err := gq.getNodeByTypeAndID(nodeType, result.ID)
		if err != nil {
			continue
		}

		results = append(results, &SemanticSearchResult{
			NodeType: nodeType,
			NodeID:   result.ID,
			Node:     fullNode,
			Score:    result.Score,
			Content:  result.Content,
		})
	}

	return results, nil
}

// SemanticSearchResult represents a result from semantic search
type SemanticSearchResult struct {
	NodeType string      `json:"node_type"`
	NodeID   string      `json:"node_id"`
	Node     interface{} `json:"node"`
	Score    float32     `json:"score"`
	Content  string      `json:"content"`
}

// getNodeByTypeAndID retrieves a node by type and ID
func (gq *GraphQueries) getNodeByTypeAndID(nodeType, id string) (interface{}, error) {
	switch nodeType {
	case "error_node":
		return gq.nodeStore.GetErrorNode(id)
	case "skill_node":
		return gq.nodeStore.GetSkillNode(id)
	case "context_node":
		return gq.nodeStore.GetContextNode(id)
	case "capability_node":
		return gq.nodeStore.GetCapabilityNode(id)
	case "idea_node":
		return gq.nodeStore.GetIdeaNode(id)
	case "property_node":
		return gq.nodeStore.GetPropertyNode(id)
	default:
		return nil, fmt.Errorf("unknown node type: %s", nodeType)
	}
}

// GetGraphStatistics returns overall statistics about the graph
func (gq *GraphQueries) GetGraphStatistics() (*GraphStatistics, error) {
	stats := &GraphStatistics{}

	// Count nodes by type
	errorCount, _ := gq.nodeStore.GetNodeCount("error_node")
	skillCount, _ := gq.nodeStore.GetNodeCount("skill_node")
	contextCount, _ := gq.nodeStore.GetNodeCount("context_node")
	capabilityCount, _ := gq.nodeStore.GetNodeCount("capability_node")
	ideaCount, _ := gq.nodeStore.GetNodeCount("idea_node")
	propertyCount, _ := gq.nodeStore.GetNodeCount("property_node")

	stats.NodeCounts = map[string]int{
		"error_node":      errorCount,
		"skill_node":      skillCount,
		"context_node":    contextCount,
		"capability_node": capabilityCount,
		"idea_node":       ideaCount,
		"property_node":   propertyCount,
	}

	// Count relationships by type
	errorToSkillCount, _ := gq.relManager.GetRelationshipCount(RelationshipTypeErrorToSkill)
	contextToCapabilityCount, _ := gq.relManager.GetRelationshipCount(RelationshipTypeContextToCapability)
	ideaToPropertyCount, _ := gq.relManager.GetRelationshipCount(RelationshipTypeIdeaToProperty)
	dependencyCount, _ := gq.relManager.GetRelationshipCount(RelationshipTypeDependency)

	stats.RelationshipCounts = map[string]int{
		"error_to_skill":       errorToSkillCount,
		"context_to_capability": contextToCapabilityCount,
		"idea_to_property":     ideaToPropertyCount,
		"dependency":           dependencyCount,
	}

	// Calculate total nodes and relationships
	stats.TotalNodes = errorCount + skillCount + contextCount + capabilityCount + ideaCount + propertyCount
	stats.TotalRelationships = errorToSkillCount + contextToCapabilityCount + ideaToPropertyCount + dependencyCount

	return stats, nil
}

// GraphStatistics represents statistics about the knowledge graph
type GraphStatistics struct {
	NodeCounts         map[string]int `json:"node_counts"`
	RelationshipCounts map[string]int `json:"relationship_counts"`
	TotalNodes         int            `json:"total_nodes"`
	TotalRelationships int            `json:"total_relationships"`
}

// GetRecentNodes returns nodes created within a time window
func (gq *GraphQueries) GetRecentNodes(hours int, limit int) ([]*RecentNodeResult, error) {
	cutoffTime := time.Now().Add(-time.Duration(hours) * time.Hour).Unix()

	var results []*RecentNodeResult

	// Check each node type
	nodeTypes := []string{"error_node", "skill_node", "context_node", "capability_node", "idea_node", "property_node"}

	for _, nodeType := range nodeTypes {
		nodeIDs, err := gq.nodeStore.ListNodesByType(nodeType)
		if err != nil {
			continue
		}

		for _, id := range nodeIDs {
			node, err := gq.getNodeByTypeAndID(nodeType, id)
			if err != nil {
				continue
			}

			// Extract creation time (this is a simplified approach)
			var createdAt int64
			switch n := node.(type) {
			case *types.ErrorNode:
				createdAt = n.CreatedAt
			case *types.SkillNode:
				createdAt = n.CreatedAt
			case *types.ContextNode:
				createdAt = n.CreatedAt
			case *types.CapabilityNode:
				createdAt = n.CreatedAt
			case *types.IdeaNode:
				createdAt = n.CreatedAt
			case *types.PropertyNode:
				createdAt = n.CreatedAt
			}

			if createdAt >= cutoffTime {
				results = append(results, &RecentNodeResult{
					NodeType: nodeType,
					NodeID:   id,
					Node:     node,
					CreatedAt: createdAt,
				})
			}
		}
	}

	// Sort by creation time descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt > results[j].CreatedAt
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// RecentNodeResult represents a recently created node
type RecentNodeResult struct {
	NodeType  string      `json:"node_type"`
	NodeID    string      `json:"node_id"`
	Node      interface{} `json:"node"`
	CreatedAt int64       `json:"created_at"`
}