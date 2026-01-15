package drq

import (
	"fmt"
	"math"
	"time"
)

// DRQClusterManager handles dynamic error clustering via DRQ
type DRQClusterManager struct {
	clusters         map[string]*ErrorCluster
	embeddingModel   *EmbeddingModel           // 768-dim BERT-based
	similarityThresh float64                    // Cosine similarity > 0.85
	maxClusterSize   int                        // 100 errors max
	drqSync          *DRQSyncProtocol
}

// ClusterError assigns error to optimal cluster via DRQ policy
func (cm *DRQClusterManager) ClusterError(
	errorNode *ErrorNode,
) (string, error) {
	// Generate error embedding (stub)
	embedding := cm.embeddingModel.Encode(errorNode.FailureContext)

	// Evaluate all possible cluster assignments
	var bestCluster string
	var bestQValue float64 = math.Inf(-1)

	for clusterID, cluster := range cm.clusters {
		// Compute state representation
		state := ErrorClusterState{
			ClusterID:         clusterID,
			ClusterCentroid:   cluster.Centroid,
			ClusterDensity:    float64(len(cluster.Errors)),
			ComplexityScore:   cluster.AvgComplexity,
		}

		// Compute action (assign to this cluster)
		action := DRQAction{
			Type:          ASSIGN_NEW_AGENT,
			TargetCluster: clusterID,
		}

		// Retrieve Q-value from distributed table (stub)
		qValue := cm.drqSync.GetQValue(state, action)

		// Consider similarity constraint (stub)
		similarity := cosineSimilarity(embedding, cluster.Centroid)
		if similarity < cm.similarityThresh {
			qValue -= 100.0  // Heavy penalty
		}

		if qValue > bestQValue {
			bestQValue = qValue
			bestCluster = clusterID
		}
	}

	// Create new cluster if no good match
	if bestQValue < 0 {
		bestCluster = cm.createNewCluster(errorNode, embedding)
	} else {
		cm.addToCluster(bestCluster, errorNode, embedding)
	}

	return bestCluster, nil
}

// createNewCluster creates a new error cluster.
func (cm *DRQClusterManager) createNewCluster(errorNode *ErrorNode, embedding []float64) string {
	clusterID := fmt.Sprintf("cluster_%d", time.Now().UnixNano())
	newCluster := &ErrorCluster{
		ClusterID:     clusterID,
		Errors:        []*ErrorNode{errorNode},
		Centroid:      embedding,
		AgentCounts:   make(map[string]int),
		Solutions:     make(map[string][]*Solution),
		Status:        CLUSTER_ACTIVE,
		CreatedAt:     time.Now(),
		AvgComplexity: float64(errorNode.Complexity),
	}

	cm.clusters[clusterID] = newCluster
	errorNode.ClusterID = clusterID

	return clusterID
}

// addToCluster adds an error to an existing cluster and updates its properties.
func (cm *DRQClusterManager) addToCluster(clusterID string, errorNode *ErrorNode, embedding []float64) {
	cluster, ok := cm.clusters[clusterID]
	if !ok {
		return // Should not happen if called from ClusterError
	}

	// Update Centroid
	totalErrors := float64(len(cluster.Errors))
	for i := range cluster.Centroid {
		cluster.Centroid[i] = (cluster.Centroid[i]*totalErrors + embedding[i]) / (totalErrors + 1)
	}

	// Update AvgComplexity
	cluster.AvgComplexity = (cluster.AvgComplexity*totalErrors + float64(errorNode.Complexity)) / (totalErrors + 1)

	// Add error to cluster
	cluster.Errors = append(cluster.Errors, errorNode)
	errorNode.ClusterID = clusterID
}

// cosineSimilarity calculates the cosine similarity between two vectors.
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0
	}

	var dotProduct, magA, magB float64
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		magA += a[i] * a[i]
		magB += b[i] * b[i]
	}

	magA = math.Sqrt(magA)
	magB = math.Sqrt(magB)

	if magA == 0 || magB == 0 {
		return 0.0
	}

	return dotProduct / (magA * magB)
}

// GetActiveClusters retrieves all clusters with an "active" status.
func (cm *DRQClusterManager) GetActiveClusters() map[string]*ErrorCluster {
	activeClusters := make(map[string]*ErrorCluster)
	for id, cluster := range cm.clusters {
		if cluster.Status == CLUSTER_ACTIVE {
			activeClusters[id] = cluster
		}
	}
	return activeClusters
}

