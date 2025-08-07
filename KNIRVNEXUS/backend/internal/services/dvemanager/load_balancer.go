package dvemanager

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/knirv/nexus-backend/internal/models"
)

// SelectNode selects the optimal node for a given task
func (lb *LoadBalancer) SelectNode(task *models.ValidationTask, availableNodes []*models.DVENode) (*models.DVENode, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if len(availableNodes) == 0 {
		return nil, fmt.Errorf("no available nodes")
	}

	// Filter nodes by task requirements
	eligibleNodes := lb.filterEligibleNodes(task, availableNodes)
	if len(eligibleNodes) == 0 {
		return nil, fmt.Errorf("no eligible nodes for task requirements")
	}

	// Select node based on algorithm
	switch lb.algorithm {
	case "round_robin":
		return lb.selectRoundRobin(eligibleNodes), nil
	case "reputation_based":
		return lb.selectByReputation(eligibleNodes), nil
	case "resource_based":
		return lb.selectByResources(eligibleNodes), nil
	case "geographic":
		return lb.selectByGeography(task, eligibleNodes), nil
	case "hybrid":
		return lb.selectHybrid(task, eligibleNodes), nil
	default:
		return lb.selectByReputation(eligibleNodes), nil
	}
}

// filterEligibleNodes filters nodes based on task requirements
func (lb *LoadBalancer) filterEligibleNodes(task *models.ValidationTask, nodes []*models.DVENode) []*models.DVENode {
	var eligible []*models.DVENode

	for _, node := range nodes {
		if lb.isNodeEligible(task, node) {
			eligible = append(eligible, node)
		}
	}

	return eligible
}

// isNodeEligible checks if a node meets task requirements
func (lb *LoadBalancer) isNodeEligible(task *models.ValidationTask, node *models.DVENode) bool {
	// Check node status
	if node.Status != "online" {
		return false
	}

	// Check TEE type requirement
	if task.RequiredTEEType != "" && node.TEEType != task.RequiredTEEType {
		return false
	}

	// Check minimum reputation for high-priority tasks
	if task.Priority >= 8 && node.ReputationScore < 80 {
		return false
	}

	// Check if node has required capabilities
	if len(task.Parameters) > 0 {
		if requiredCaps, ok := task.Parameters["required_capabilities"].([]string); ok {
			nodeCapMap := make(map[string]bool)
			for _, cap := range node.Capabilities {
				nodeCapMap[cap] = true
			}
			for _, reqCap := range requiredCaps {
				if !nodeCapMap[reqCap] {
					return false
				}
			}
		}
	}

	// Check resource availability (simplified)
	if node.CPUUsage > 90 || node.MemoryUsage > 90 {
		return false
	}

	return true
}

// selectRoundRobin selects nodes in round-robin fashion
func (lb *LoadBalancer) selectRoundRobin(nodes []*models.DVENode) *models.DVENode {
	// Simple round-robin based on current time
	index := int(time.Now().Unix()) % len(nodes)
	return nodes[index]
}

// selectByReputation selects the node with highest reputation
func (lb *LoadBalancer) selectByReputation(nodes []*models.DVENode) *models.DVENode {
	if len(nodes) == 0 {
		return nil
	}

	// Sort by reputation score (descending)
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ReputationScore > nodes[j].ReputationScore
	})

	// Add some randomness among top nodes to avoid always selecting the same node
	topCount := int(math.Min(3, float64(len(nodes))))
	selectedIndex := rand.Intn(topCount)
	
	return nodes[selectedIndex]
}

// selectByResources selects the node with best resource availability
func (lb *LoadBalancer) selectByResources(nodes []*models.DVENode) *models.DVENode {
	if len(nodes) == 0 {
		return nil
	}

	// Calculate resource score for each node
	type nodeScore struct {
		node  *models.DVENode
		score float64
	}

	var scores []nodeScore
	for _, node := range nodes {
		// Resource score based on available CPU and memory
		cpuScore := (100 - node.CPUUsage) / 100
		memoryScore := (100 - node.MemoryUsage) / 100
		
		// Combine scores with weights
		resourceScore := (cpuScore * 0.6) + (memoryScore * 0.4)
		
		scores = append(scores, nodeScore{
			node:  node,
			score: resourceScore,
		})
	}

	// Sort by resource score (descending)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	return scores[0].node
}

// selectByGeography selects the node closest to the task origin
func (lb *LoadBalancer) selectByGeography(task *models.ValidationTask, nodes []*models.DVENode) *models.DVENode {
	if len(nodes) == 0 {
		return nil
	}

	// If task has geographic preferences
	if preferredLocation, ok := task.Parameters["preferred_location"].(string); ok {
		for _, node := range nodes {
			if node.Location == preferredLocation {
				return node
			}
		}
	}

	// If task has coordinates, find closest node
	if taskLat, hasLat := task.Parameters["latitude"].(float64); hasLat {
		if taskLon, hasLon := task.Parameters["longitude"].(float64); hasLon {
			return lb.findClosestNode(taskLat, taskLon, nodes)
		}
	}

	// Fallback to reputation-based selection
	return lb.selectByReputation(nodes)
}

// findClosestNode finds the geographically closest node
func (lb *LoadBalancer) findClosestNode(taskLat, taskLon float64, nodes []*models.DVENode) *models.DVENode {
	var closestNode *models.DVENode
	minDistance := math.MaxFloat64

	for _, node := range nodes {
		if node.Latitude != 0 && node.Longitude != 0 {
			distance := lb.calculateDistance(taskLat, taskLon, node.Latitude, node.Longitude)
			if distance < minDistance {
				minDistance = distance
				closestNode = node
			}
		}
	}

	if closestNode != nil {
		return closestNode
	}

	// If no nodes have coordinates, fallback to first node
	return nodes[0]
}

// calculateDistance calculates the distance between two geographic points
func (lb *LoadBalancer) calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	// Haversine formula for calculating distance between two points on Earth
	const R = 6371 // Earth's radius in kilometers

	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	distance := R * c

	return distance
}

// selectHybrid uses a combination of factors to select the best node
func (lb *LoadBalancer) selectHybrid(task *models.ValidationTask, nodes []*models.DVENode) *models.DVENode {
	if len(nodes) == 0 {
		return nil
	}

	type nodeScore struct {
		node  *models.DVENode
		score float64
	}

	var scores []nodeScore
	for _, node := range nodes {
		score := lb.calculateHybridScore(task, node)
		scores = append(scores, nodeScore{
			node:  node,
			score: score,
		})
	}

	// Sort by hybrid score (descending)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	return scores[0].node
}

// calculateHybridScore calculates a hybrid score considering multiple factors
func (lb *LoadBalancer) calculateHybridScore(task *models.ValidationTask, node *models.DVENode) float64 {
	// Reputation score (0-1)
	reputationScore := float64(node.ReputationScore) / 100.0

	// Resource availability score (0-1)
	cpuScore := (100 - node.CPUUsage) / 100
	memoryScore := (100 - node.MemoryUsage) / 100
	resourceScore := (cpuScore + memoryScore) / 2

	// Stake weight (0-1, normalized)
	stakeScore := math.Min(float64(node.StakeAmount)/1000000, 1.0) // Normalize to 1M max

	// Network latency score (inverse of latency, 0-1)
	latencyScore := 1.0 / (1.0 + float64(node.NetworkLatency)/1000.0)

	// Task priority weight
	priorityWeight := float64(task.Priority) / 10.0

	// Combine scores with weights
	weights := map[string]float64{
		"reputation": 0.3,
		"resources":  0.25,
		"stake":      0.2,
		"latency":    0.15,
		"priority":   0.1,
	}

	hybridScore := (reputationScore * weights["reputation"]) +
		(resourceScore * weights["resources"]) +
		(stakeScore * weights["stake"]) +
		(latencyScore * weights["latency"]) +
		(priorityWeight * weights["priority"])

	return hybridScore
}

// SetAlgorithm sets the load balancing algorithm
func (lb *LoadBalancer) SetAlgorithm(algorithm string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.algorithm = algorithm
}

// GetAlgorithm returns the current load balancing algorithm
func (lb *LoadBalancer) GetAlgorithm() string {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	return lb.algorithm
}

// GetLoadBalancingStats returns statistics about load balancing
func (lb *LoadBalancer) GetLoadBalancingStats(nodes []*models.DVENode) map[string]interface{} {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	stats := map[string]interface{}{
		"algorithm":      lb.algorithm,
		"total_nodes":    len(nodes),
		"eligible_nodes": 0,
		"avg_cpu_usage":  0.0,
		"avg_memory_usage": 0.0,
		"avg_reputation": 0.0,
	}

	if len(nodes) == 0 {
		return stats
	}

	var totalCPU, totalMemory, totalReputation float64
	var eligibleCount int

	for _, node := range nodes {
		if node.Status == "online" {
			eligibleCount++
			totalCPU += node.CPUUsage
			totalMemory += node.MemoryUsage
			totalReputation += float64(node.ReputationScore)
		}
	}

	if eligibleCount > 0 {
		stats["eligible_nodes"] = eligibleCount
		stats["avg_cpu_usage"] = totalCPU / float64(eligibleCount)
		stats["avg_memory_usage"] = totalMemory / float64(eligibleCount)
		stats["avg_reputation"] = totalReputation / float64(eligibleCount)
	}

	return stats
}
