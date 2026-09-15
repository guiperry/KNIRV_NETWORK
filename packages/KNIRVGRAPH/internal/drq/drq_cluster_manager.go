package drq

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// DRQClusterManager handles dynamic error clustering via DRQ.
type DRQClusterManager struct {
	clusters         map[string]*ErrorCluster
	embeddingModel   *EmbeddingModel // 768-dim BERT-based
	similarityThresh float64         // Cosine similarity > 0.85
	maxClusterSize   int             // 100 errors max
	drqSync          *DRQSyncProtocol
}

// DefaultSimilarityThreshold mirrors app.ClusteringConfig.SimilarityThreshold.
const DefaultSimilarityThreshold = 0.85

// DefaultMaxClusterSize mirrors app.ClusteringConfig.MaxClusterSize.
const DefaultMaxClusterSize = 100

// NewDRQClusterManager builds a clustering manager.
//
// embeddingModel is required: clustering is defined by embedding similarity, and
// a nil model would make every error look equally (dis)similar, silently
// collapsing the clustering into an arbitrary partition. drqSync may be nil, in
// which case assignment falls back to the similarity constraint alone.
func NewDRQClusterManager(embeddingModel *EmbeddingModel, similarityThresh float64, maxClusterSize int) (*DRQClusterManager, error) {
	if embeddingModel == nil {
		return nil, fmt.Errorf("an embedding model is required to cluster errors by similarity")
	}
	if similarityThresh <= 0 || similarityThresh > 1 {
		similarityThresh = DefaultSimilarityThreshold
	}
	if maxClusterSize <= 0 {
		maxClusterSize = DefaultMaxClusterSize
	}
	return &DRQClusterManager{
		clusters:         make(map[string]*ErrorCluster),
		embeddingModel:   embeddingModel,
		similarityThresh: similarityThresh,
		maxClusterSize:   maxClusterSize,
	}, nil
}

// SetSyncProtocol attaches the distributed Q-value protocol. Optional: without
// it, cluster assignment is driven purely by embedding similarity.
func (cm *DRQClusterManager) SetSyncProtocol(sync *DRQSyncProtocol) {
	cm.drqSync = sync
}

// SimilarityThreshold returns the cosine threshold in force.
func (cm *DRQClusterManager) SimilarityThreshold() float64 { return cm.similarityThresh }

// Clusters returns a snapshot of all clusters, keyed by cluster id.
func (cm *DRQClusterManager) Clusters() map[string]*ErrorCluster {
	out := make(map[string]*ErrorCluster, len(cm.clusters))
	for id, cluster := range cm.clusters {
		out[id] = cluster
	}
	return out
}

// ClusterError assigns an error to the optimal cluster, creating one when no
// existing cluster is similar enough.
func (cm *DRQClusterManager) ClusterError(errorNode *ErrorNode) (string, error) {
	if errorNode == nil {
		return "", fmt.Errorf("error node is required")
	}
	if cm.embeddingModel == nil {
		return "", fmt.Errorf("no embedding model configured; cannot cluster error %s", errorNode.Id)
	}
	if cm.clusters == nil {
		cm.clusters = make(map[string]*ErrorCluster)
	}

	embedding := cm.embeddingModel.Encode(errorNode.FailureContext)
	if len(embedding) == 0 {
		return "", fmt.Errorf("embedding model returned no vector for error %s", errorNode.Id)
	}

	// Evaluate every existing cluster assignment.
	var bestCluster string
	bestQValue := math.Inf(-1)
	for clusterID, cluster := range cm.clusters {
		if cluster == nil {
			continue
		}
		qValue := 0.0
		if cm.drqSync != nil {
			state := ErrorClusterState{
				ClusterID:       clusterID,
				ClusterCentroid: cluster.Centroid,
				ClusterDensity:  float64(len(cluster.Errors)),
				ComplexityScore: cluster.AvgComplexity,
			}
			qValue = cm.drqSync.GetQValue(state, DRQAction{
				Type:          ASSIGN_NEW_AGENT,
				TargetCluster: clusterID,
			})
		}

		// The similarity constraint is a hard gate, not a soft preference:
		// semantics must dominate the learned Q-value, or the RL policy can
		// route an error into a cluster it does not resemble.
		if len(cluster.Centroid) != len(embedding) {
			// A centroid from a different embedding dimension cannot be
			// compared; skipping silently would let a stale cluster absorb
			// unrelated errors forever.
			return "", fmt.Errorf("cluster %s has a %d-dim centroid but the model produced %d dims",
				clusterID, len(cluster.Centroid), len(embedding))
		}
		similarity := cosineSimilarity(embedding, cluster.Centroid)
		if similarity < cm.similarityThresh {
			continue
		}
		if len(cluster.Errors) >= cm.maxClusterSize {
			// Full clusters must not absorb more members; a new one is created.
			continue
		}
		if qValue > bestQValue {
			bestQValue = qValue
			bestCluster = clusterID
		}
	}

	if bestCluster == "" {
		return cm.createNewCluster(errorNode, embedding), nil
	}
	cm.addToCluster(bestCluster, errorNode, embedding)
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

	// Keep the member error's own embedding in sync with the centroid: the
	// convergence check in ClusterManager.isReadyForTraining measures how
	// tightly the members cluster around the centroid, and would otherwise
	// never see an embedding to measure.
	errorNode.Embedding = embedding

	cm.clusters[clusterID] = newCluster
	errorNode.ClusterId = clusterID
	recordContributor(newCluster, errorNode)

	return clusterID
}

// addToCluster adds an error to an existing cluster and updates its properties.
func (cm *DRQClusterManager) addToCluster(clusterID string, errorNode *ErrorNode, embedding []float64) {
	cluster, ok := cm.clusters[clusterID]
	if !ok || cluster == nil {
		return // Should not happen if called from ClusterError
	}
	if len(cluster.Centroid) != len(embedding) {
		// Refuse rather than index out of range or silently corrupt the
		// centroid: the caller (ClusterError) rejects dimension mismatches
		// before reaching here, so this is a belt-and-braces guard.
		return
	}
	if cluster.AgentCounts == nil {
		cluster.AgentCounts = make(map[string]int)
	}

	// Rolling centroid: the mean of the previous members and the new one.
	totalErrors := float64(len(cluster.Errors))
	for i := range cluster.Centroid {
		cluster.Centroid[i] = (cluster.Centroid[i]*totalErrors + embedding[i]) / (totalErrors + 1)
	}
	cluster.AvgComplexity = (cluster.AvgComplexity*totalErrors + float64(errorNode.Complexity)) / (totalErrors + 1)

	cluster.Errors = append(cluster.Errors, errorNode)
	errorNode.Embedding = embedding
	errorNode.ClusterId = clusterID
	recordContributor(cluster, errorNode)
}

// recordContributor credits the error's resolver in the cluster's agent counts,
// which is what ownership and bounty distribution are computed from.
func recordContributor(cluster *ErrorCluster, errorNode *ErrorNode) {
	agent := strings.TrimSpace(errorNode.ResolvedBy)
	if agent == "" {
		return
	}
	if cluster.AgentCounts == nil {
		cluster.AgentCounts = make(map[string]int)
	}
	cluster.AgentCounts[agent]++
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
