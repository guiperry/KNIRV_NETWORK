package icme

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

type TemporalHypergraph struct {
	mu         sync.RWMutex
	nodes      map[string]*HyperNode
	edges      map[string][]*HyperEdge
	textIndex  map[string]string
	windowSize time.Duration
	maxNodes   int
	logger     *zap.Logger
}

func NewTemporalHypergraph(windowSize time.Duration, maxNodes int, logger *zap.Logger) *TemporalHypergraph {
	return &TemporalHypergraph{
		nodes:      make(map[string]*HyperNode),
		edges:      make(map[string][]*HyperEdge),
		textIndex:  make(map[string]string),
		windowSize: windowSize,
		maxNodes:   maxNodes,
		logger:     logger,
	}
}

func (hg *TemporalHypergraph) InsertSignal(sig *IntentionalSignal) {
	hg.mu.Lock()
	defer hg.mu.Unlock()

	nodeIDs := make(map[string]string)
	for _, ent := range sig.Entities {
		nodeID := hg.upsertNode(ent, sig)
		nodeIDs[ent.ID] = nodeID
	}

	for _, rel := range sig.Relations {
		fromNodeID, ok1 := nodeIDs[rel.FromEntityID]
		toNodeID, ok2 := nodeIDs[rel.ToEntityID]
		if !ok1 || !ok2 {
			continue
		}
		edge := &HyperEdge{
			ID:             sig.ID + ":" + rel.FromEntityID + ":" + rel.ToEntityID,
			FromNodeID:     fromNodeID,
			ToNodeID:       toNodeID,
			RelationType:   rel.RelationType,
			Timestamp:      sig.Timestamp,
			SignalID:       sig.ID,
			AgentID:        sig.AgentID,
			DVEID:          sig.DVEID,
			ObjectiveName:  sig.ObjectiveName,
			AlignmentScore: sig.AlignmentScore,
			Metadata: map[string]interface{}{
				"confidence": rel.Confidence,
				"source":     string(sig.Source),
			},
		}
		hg.edges[fromNodeID] = append(hg.edges[fromNodeID], edge)
	}

	hg.pruneOldEdges()
}

func (hg *TemporalHypergraph) upsertNode(ent ExtractedEntity, sig *IntentionalSignal) string {
	nodeID := ent.Text + ":" + ent.Label
	if existing, ok := hg.nodes[nodeID]; ok {
		existing.LastSeen = sig.Timestamp
		existing.SignalIDs = append(existing.SignalIDs, sig.ID)
		return nodeID
	}

	hg.nodes[nodeID] = &HyperNode{
		ID:         nodeID,
		Type:       ent.Label,
		Text:       ent.Text,
		Attributes: map[string]interface{}{},
		FirstSeen:  sig.Timestamp,
		LastSeen:   sig.Timestamp,
		SignalIDs:  []string{sig.ID},
	}
	hg.textIndex[nodeID] = ent.Text
	return nodeID
}

func (hg *TemporalHypergraph) pruneOldEdges() {
	if hg.windowSize == 0 {
		return
	}
	cutoff := time.Now().Add(-hg.windowSize)
	for nodeID, edgeList := range hg.edges {
		var filtered []*HyperEdge
		for _, edge := range edgeList {
			if edge.Timestamp.After(cutoff) {
				filtered = append(filtered, edge)
			}
		}
		if len(filtered) == 0 {
			delete(hg.edges, nodeID)
		} else {
			hg.edges[nodeID] = filtered
		}
	}
}

func (hg *TemporalHypergraph) Neighbors(nodeID string, maxHops int, relType string) []*HyperNode {
	hg.mu.RLock()
	defer hg.mu.RUnlock()

	visited := make(map[string]bool)
	var result []*HyperNode
	hg.dfs(nodeID, maxHops, relType, visited, &result)
	return result
}

func (hg *TemporalHypergraph) dfs(nodeID string, maxHops int, relType string, visited map[string]bool, result *[]*HyperNode) {
	if maxHops < 0 || visited[nodeID] {
		return
	}
	visited[nodeID] = true

	if node, ok := hg.nodes[nodeID]; ok {
		*result = append(*result, node)
	}

	edges := hg.edges[nodeID]
	for _, edge := range edges {
		if relType == "" || edge.RelationType == relType {
			hg.dfs(edge.ToNodeID, maxHops-1, relType, visited, result)
		}
	}
}

func (hg *TemporalHypergraph) Snapshot() (nodes []*HyperNode, edges []*HyperEdge) {
	hg.mu.RLock()
	defer hg.mu.RUnlock()
	for _, n := range hg.nodes {
		nodes = append(nodes, n)
	}
	for _, edgeList := range hg.edges {
		edges = append(edges, edgeList...)
	}
	return
}

func (hg *TemporalHypergraph) GetNode(nodeID string) *HyperNode {
	hg.mu.RLock()
	defer hg.mu.RUnlock()
	return hg.nodes[nodeID]
}
