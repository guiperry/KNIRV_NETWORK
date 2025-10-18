package dvemanager

import (
	"time"

	"backend-server/internal/models"
)

// AddNode adds a node to the tracker
func (nt *NodeTracker) AddNode(node *models.DVENode) {
	nt.mu.Lock()
	defer nt.mu.Unlock()
	nt.nodes[node.ID] = node
}

// UpdateNode updates a node in the tracker
func (nt *NodeTracker) UpdateNode(node *models.DVENode) {
	nt.mu.Lock()
	defer nt.mu.Unlock()
	nt.nodes[node.ID] = node
}

// RemoveNode removes a node from the tracker
func (nt *NodeTracker) RemoveNode(nodeID string) {
	nt.mu.Lock()
	defer nt.mu.Unlock()
	delete(nt.nodes, nodeID)
}

// GetNode retrieves a specific node
func (nt *NodeTracker) GetNode(nodeID string) (*models.DVENode, bool) {
	nt.mu.RLock()
	defer nt.mu.RUnlock()
	node, exists := nt.nodes[nodeID]
	return node, exists
}

// GetAllNodes returns all nodes
func (nt *NodeTracker) GetAllNodes() []*models.DVENode {
	nt.mu.RLock()
	defer nt.mu.RUnlock()

	nodes := make([]*models.DVENode, 0, len(nt.nodes))
	for _, node := range nt.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// GetActiveNodes returns only active (online) nodes
func (nt *NodeTracker) GetActiveNodes() []*models.DVENode {
	nt.mu.RLock()
	defer nt.mu.RUnlock()

	var activeNodes []*models.DVENode
	for _, node := range nt.nodes {
		if node.Status == "online" {
			activeNodes = append(activeNodes, node)
		}
	}
	return activeNodes
}

// GetTotalNodes returns the total number of nodes
func (nt *NodeTracker) GetTotalNodes() int {
	nt.mu.RLock()
	defer nt.mu.RUnlock()
	return len(nt.nodes)
}

// UpdateHeartbeat updates the last heartbeat time for a node
func (nt *NodeTracker) UpdateHeartbeat(nodeID string) {
	nt.mu.Lock()
	defer nt.mu.Unlock()

	if node, exists := nt.nodes[nodeID]; exists {
		node.LastHeartbeat = time.Now()
		node.Status = "online"
		nt.nodes[nodeID] = node
	}
}

// GetNodesByTEEType returns nodes filtered by TEE type
func (nt *NodeTracker) GetNodesByTEEType(teeType string) []*models.DVENode {
	nt.mu.RLock()
	defer nt.mu.RUnlock()

	var filteredNodes []*models.DVENode
	for _, node := range nt.nodes {
		if node.TEEType == teeType && node.Status == "online" {
			filteredNodes = append(filteredNodes, node)
		}
	}
	return filteredNodes
}

// GetNodesByLocation returns nodes filtered by location
func (nt *NodeTracker) GetNodesByLocation(location string) []*models.DVENode {
	nt.mu.RLock()
	defer nt.mu.RUnlock()

	var filteredNodes []*models.DVENode
	for _, node := range nt.nodes {
		if node.Location == location && node.Status == "online" {
			filteredNodes = append(filteredNodes, node)
		}
	}
	return filteredNodes
}

// GetTopReputationNodes returns nodes sorted by reputation score
func (nt *NodeTracker) GetTopReputationNodes(limit int) []*models.DVENode {
	nt.mu.RLock()
	defer nt.mu.RUnlock()

	activeNodes := make([]*models.DVENode, 0)
	for _, node := range nt.nodes {
		if node.Status == "online" {
			activeNodes = append(activeNodes, node)
		}
	}

	// Sort by reputation score (descending)
	for i := 0; i < len(activeNodes)-1; i++ {
		for j := i + 1; j < len(activeNodes); j++ {
			if activeNodes[i].ReputationScore < activeNodes[j].ReputationScore {
				activeNodes[i], activeNodes[j] = activeNodes[j], activeNodes[i]
			}
		}
	}

	if limit > 0 && limit < len(activeNodes) {
		return activeNodes[:limit]
	}
	return activeNodes
}

// GetNodeStats returns statistics about tracked nodes
func (nt *NodeTracker) GetNodeStats() map[string]int {
	nt.mu.RLock()
	defer nt.mu.RUnlock()

	stats := map[string]int{
		"total":       0,
		"online":      0,
		"offline":     0,
		"maintenance": 0,
		"error":       0,
	}

	for _, node := range nt.nodes {
		stats["total"]++
		stats[node.Status]++
	}

	return stats
}

// GetTEETypeDistribution returns distribution of TEE types
func (nt *NodeTracker) GetTEETypeDistribution() map[string]int {
	nt.mu.RLock()
	defer nt.mu.RUnlock()

	distribution := make(map[string]int)
	for _, node := range nt.nodes {
		if node.Status == "online" {
			distribution[node.TEEType]++
		}
	}

	return distribution
}

// GetLocationDistribution returns distribution of node locations
func (nt *NodeTracker) GetLocationDistribution() map[string]int {
	nt.mu.RLock()
	defer nt.mu.RUnlock()

	distribution := make(map[string]int)
	for _, node := range nt.nodes {
		if node.Status == "online" {
			distribution[node.Location]++
		}
	}

	return distribution
}

// GetAverageReputation calculates average reputation score of active nodes
func (nt *NodeTracker) GetAverageReputation() float64 {
	nt.mu.RLock()
	defer nt.mu.RUnlock()

	var totalReputation int
	var activeCount int

	for _, node := range nt.nodes {
		if node.Status == "online" {
			totalReputation += node.ReputationScore
			activeCount++
		}
	}

	if activeCount == 0 {
		return 0.0
	}

	return float64(totalReputation) / float64(activeCount)
}

// GetTotalStake calculates total stake amount across all active nodes
func (nt *NodeTracker) GetTotalStake() int64 {
	nt.mu.RLock()
	defer nt.mu.RUnlock()

	var totalStake int64
	for _, node := range nt.nodes {
		if node.Status == "online" {
			totalStake += node.StakeAmount
		}
	}

	return totalStake
}

// IsNodeHealthy checks if a node is considered healthy
func (nt *NodeTracker) IsNodeHealthy(nodeID string) bool {
	nt.mu.RLock()
	defer nt.mu.RUnlock()

	node, exists := nt.nodes[nodeID]
	if !exists {
		return false
	}

	// Node is healthy if it's online and has sent heartbeat recently
	return node.Status == "online" && time.Since(node.LastHeartbeat) < 2*time.Minute
}

// GetUnhealthyNodes returns nodes that are considered unhealthy
func (nt *NodeTracker) GetUnhealthyNodes() []*models.DVENode {
	nt.mu.RLock()
	defer nt.mu.RUnlock()

	var unhealthyNodes []*models.DVENode
	for _, node := range nt.nodes {
		if !nt.isNodeHealthyInternal(node) {
			unhealthyNodes = append(unhealthyNodes, node)
		}
	}

	return unhealthyNodes
}

// isNodeHealthyInternal is an internal helper that doesn't acquire locks
func (nt *NodeTracker) isNodeHealthyInternal(node *models.DVENode) bool {
	return node.Status == "online" && time.Since(node.LastHeartbeat) < 2*time.Minute
}
