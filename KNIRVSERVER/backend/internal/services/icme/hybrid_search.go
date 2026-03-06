package icme

import (
	"context"
	"sort"

	"go.uber.org/zap"
)

type HybridSearchEngine struct {
	faissManager  *FAISSIndexManager
	graph         *TemporalHypergraph
	embedProvider *EmbeddingProvider
	intentReg     *IntentRegistry
	logger        *zap.Logger
}

func NewHybridSearchEngine(
	faissManager *FAISSIndexManager,
	graph *TemporalHypergraph,
	embedProvider *EmbeddingProvider,
	intentReg *IntentRegistry,
	logger *zap.Logger,
) *HybridSearchEngine {
	return &HybridSearchEngine{
		faissManager:  faissManager,
		graph:         graph,
		embedProvider: embedProvider,
		intentReg:     intentReg,
		logger:        logger,
	}
}

func (s *HybridSearchEngine) Search(ctx context.Context, query, agentID, dveID string, topK int) ([]HybridResult, error) {
	embedding, err := s.embedProvider.GetEmbedding(query)
	if err != nil {
		return nil, err
	}

	metas, scores, err := s.faissManager.Search(embedding, topK*3)
	if err != nil {
		return nil, err
	}

	obj := s.intentReg.GetObjectiveForAgent(agentID, dveID)
	results := make([]HybridResult, 0, len(metas))

	for i, meta := range metas {
		nodes := s.graph.Neighbors(meta.SignalID, 2, "")

		alignBoost := 0.0
		if obj != nil && meta.Objective == obj.Name {
			alignBoost = 0.15
		}

		vecSim := 1.0 / (1.0 + float64(scores[i]))
		combined := vecSim + float64(len(nodes))*0.01 + alignBoost

		results = append(results, HybridResult{
			SignalID:      meta.SignalID,
			AgentID:       meta.AgentID,
			DVEID:         meta.DVEID,
			Content:       meta.Summary,
			ObjectiveName: meta.Objective,
			VectorScore:   scores[i],
			GraphHops:     len(nodes),
			CombinedScore: combined,
			Nodes:         nodes,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].CombinedScore > results[j].CombinedScore
	})

	if len(results) > topK {
		return results[:topK], nil
	}
	return results, nil
}
