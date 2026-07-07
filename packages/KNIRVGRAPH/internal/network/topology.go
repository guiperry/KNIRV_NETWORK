package network

import (
	"math"
	"math/rand"
	"sort"
	"time"

	"KNIRVGRAPH/internal/drq"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// NetworkTopology manages scale-free graph structure
type NetworkTopology struct {
	nodes           map[string]*TopologyNode
	adjacencyList   map[string][]string
	degreeDistrib   map[int]int              // degree → count
	scalingExponent float64                  // γ parameter
	minDegree       int                      // m₀ = 3
}

type TopologyNode struct {
	NodeID          string
	Degree          int
	PageRank        float64
	ClusterCoeff    float64                  // Local clustering
	BetweennessCent float64                  // Centrality measure
	ErrorClusters   []string                 // Hosted clusters
}

// AttachNewNode adds node using preferential attachment
func (nt *NetworkTopology) AttachNewNode(
	newNodeID string,
	edgeCount int,
) error {
	newNode := &TopologyNode{
		NodeID: newNodeID,
		Degree: 0,
	}
	
	// Calculate attachment probabilities
	totalDegree := 0
	for _, node := range nt.nodes {
		totalDegree += node.Degree
	}
	
	// Select targets via preferential attachment
	targets := make([]string, 0, edgeCount)
	for i := 0; i < edgeCount; i++ {
		target := nt.selectByPreference(totalDegree)
		targets = append(targets, target)
		
		// Update degrees
		nt.nodes[target].Degree++
		totalDegree++
	}
	
	newNode.Degree = edgeCount
	nt.nodes[newNodeID] = newNode
	nt.adjacencyList[newNodeID] = targets
	
	return nil
}

// selectByPreference chooses node proportional to degree
func (nt *NetworkTopology) selectByPreference(
	totalDegree int,
) string {
	randVal := randomInt(0, totalDegree)
	cumulative := 0
	
	for nodeID, node := range nt.nodes {
		cumulative += node.Degree
		if randVal < cumulative {
			return nodeID
		}
	}
	
	return ""  // Should never reach
}

// randomInt is a helper function to generate a random integer within a range
func randomInt(min, max int) int {
	return min + rand.Intn(max-min+1)
}

// IdentifyHubs finds high-degree nodes for cluster hosting
func (nt *NetworkTopology) IdentifyHubs(percentile float64) []string {
	degrees := make([]int, 0, len(nt.nodes))
	for _, node := range nt.nodes {
		degrees = append(degrees, node.Degree)
	}
	
	sort.Ints(degrees)
	// Handle empty slice or percentile out of bounds
	if len(degrees) == 0 || percentile < 0 || percentile > 1 {
		return []string{}
	}
	thresholdIndex := int(float64(len(degrees)-1) * percentile)
	threshold := degrees[thresholdIndex]
	
	hubs := make([]string, 0)
	for nodeID, node := range nt.nodes {
		if node.Degree >= threshold {
			hubs = append(hubs, nodeID)
		}
	}
	
	return hubs
}

// AssignClusterToHub uses PageRank for optimal placement
func (nt *NetworkTopology) AssignClusterToHub(
	clusterID string,
	clusterState drq.ErrorClusterState, // ErrorClusterState from drq package
) string {
	hubs := nt.IdentifyHubs(0.90)  // Top 10%
	
	var bestHub string
	var bestScore float64 = -1.0 // Initialize with a value that ensures any valid score is greater
	
	for _, hubID := range hubs {
		hub := nt.nodes[hubID]
		
		// Composite score: PageRank × (1 - load)
		// Ensure load calculation handles division by zero or large number of clusters
		load := float64(len(hub.ErrorClusters)) / math.Max(1.0, float64(nt.maxDegree())) // Use Max to prevent div by zero
		score := hub.PageRank * (1.0 - load)
		
		if score > bestScore {
			bestScore = score
			bestHub = hubID
		}
	}
	
	if bestHub != "" { // Only append if a hub is found
		nt.nodes[bestHub].ErrorClusters = append(
			nt.nodes[bestHub].ErrorClusters, 
			clusterID,
		)
	}
	
	return bestHub
}

// maxDegree is a helper to find the maximum degree in the network
func (nt *NetworkTopology) maxDegree() int {
	maxD := 0
	for _, node := range nt.nodes {
		if node.Degree > maxD {
			maxD = node.Degree
		}
	}
	return maxD
}

// CalculateSmallWorldMetrics computes clustering & path length
func (nt *NetworkTopology) CalculateSmallWorldMetrics() (float64, float64) {
	if len(nt.nodes) == 0 {
		return 0.0, 0.0
	}

	// Average clustering coefficient
	totalCC := 0.0
	for _, node := range nt.nodes {
		node.ClusterCoeff = nt.localClusteringCoeff(node.NodeID)
		totalCC += node.ClusterCoeff
	}
	avgCC := totalCC / float64(len(nt.nodes))

	// Average shortest path length
	totalPathLen := 0.0
	pathCount := 0

	for sourceID := range nt.nodes {
		pathLengths := nt.bfsPathLengths(sourceID)
		for _, length := range pathLengths {
			totalPathLen += float64(length)
			pathCount++
		}
	}
	avgPathLen := 0.0
	if pathCount > 0 { // Avoid division by zero
		avgPathLen = totalPathLen / float64(pathCount)
	}

	return avgCC, avgPathLen
}

// localClusteringCoeff computes the local clustering coefficient for a node
func (nt *NetworkTopology) localClusteringCoeff(nodeID string) float64 {
	neighbors := nt.adjacencyList[nodeID]
	k := len(neighbors)
	if k < 2 {
		return 0.0 // Node with less than 2 neighbors has 0 clustering coefficient
	}

	numTriangles := 0
	for i := 0; i < k; i++ {
		for j := i + 1; j < k; j++ {
			if nt.hasEdge(neighbors[i], neighbors[j]) {
				numTriangles++
			}
		}
	}
	
	// k(k-1)/2 is the maximum possible edges between neighbors
	return float64(numTriangles) / (float64(k) * float64(k-1) / 2.0)
}

// bfsPathLengths computes shortest path lengths from a source node using BFS
func (nt *NetworkTopology) bfsPathLengths(sourceID string) map[string]int {
	distances := make(map[string]int)
	for nodeID := range nt.nodes {
		distances[nodeID] = -1 // -1 indicates unreachable
	}
	distances[sourceID] = 0

	queue := []string{sourceID}
	head := 0

	for head < len(queue) {
		u := queue[head]
		head++

		for _, v := range nt.adjacencyList[u] {
			if distances[v] == -1 {
				distances[v] = distances[u] + 1
				queue = append(queue, v)
			}
		}
	}
	// Remove self-loop path length (0) from results for average calculation
	delete(distances, sourceID)
	return distances
}

// hasEdge is a helper to check if an edge exists between two nodes
func (nt *NetworkTopology) hasEdge(node1, node2 string) bool {
	for _, neighbor := range nt.adjacencyList[node1] {
		if neighbor == node2 {
			return true
		}
	}
	return false
}

// GetClusterLoad is a stub for getting the load of a cluster domain
func (nt *NetworkTopology) GetClusterLoad(domain string) float64 {
	// TODO: Implement actual cluster load logic based on domain
	_ = domain
	return 0.5 // Dummy load
}

// ComputePageRank calculates node importance
func (nt *NetworkTopology) ComputePageRank(iterations int) {
	dampingFactor := 0.85
	n := float64(len(nt.nodes))
	
	// Initialize PageRank
	for _, node := range nt.nodes {
		node.PageRank = 1.0 / n
	}
	
	// Power iteration
	for iter := 0; iter < iterations; iter++ {
		newRanks := make(map[string]float64)
		
		for nodeID := range nt.nodes {
			rank := (1.0 - dampingFactor) / n
			
			// Sum contributions from incoming edges
			for neighborID := range nt.nodes { // Iterate through all nodes to find incoming edges
				if nt.hasEdge(neighborID, nodeID) {
					neighbor := nt.nodes[neighborID]
					outDegree := float64(neighbor.Degree)
					if outDegree > 0 { // Avoid division by zero
						rank += dampingFactor * neighbor.PageRank / outDegree
					}
				}
			}
			
			newRanks[nodeID] = rank
		}
		
		// Update ranks
		for nodeID, rank := range newRanks {
			nt.nodes[nodeID].PageRank = rank
		}
	}
}

