package icme

import (
	"context"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/guiperry/text-embedder/pkg/embed"
	"go.uber.org/zap"
)

var (
	searchSimilarityThreshold = func() float32 {
		if v := os.Getenv("KNIRV_SEARCH_CACHE_THRESHOLD"); v != "" {
			if f, err := strconv.ParseFloat(v, 32); err == nil {
				return float32(f)
			}
		}
		return 0.97
	}()
	searchCacheTTL = func() time.Duration {
		if v := os.Getenv("KNIRV_SEARCH_CACHE_TTL_SECONDS"); v != "" {
			if s, err := strconv.Atoi(v); err == nil {
				return time.Duration(s) * time.Second
			}
		}
		return 30 * time.Second
	}()
)

type cachedSearch struct {
	vec     []float32
	results []HybridResult
	expiry  time.Time
}

type HybridSearchEngine struct {
	faissManager  *FAISSIndexManager
	graph         *TemporalHypergraph
	embedProvider *EmbeddingProvider
	intentReg     *IntentRegistry
	logger        *zap.Logger
	cacheMu       sync.Mutex
	cache         []cachedSearch
}

func NewHybridSearchEngine(
	faissManager *FAISSIndexManager,
	graph *TemporalHypergraph,
	intentReg *IntentRegistry,
	logger *zap.Logger,
) *HybridSearchEngine {
	return &HybridSearchEngine{
		faissManager:  faissManager,
		graph:         graph,
		intentReg:     intentReg,
		logger:        logger,
	}
}

func (s *HybridSearchEngine) Search(ctx context.Context, query, agentID, dveID string, topK int) ([]HybridResult, error) {
	queryVec := embed.Embed(query)

	s.cacheMu.Lock()
	now := time.Now()
	for _, entry := range s.cache {
		if entry.expiry.After(now) &&
			float32(embed.CosineSimilarity(queryVec, entry.vec)) >= searchSimilarityThreshold {
			s.cacheMu.Unlock()
			return entry.results, nil
		}
	}
	s.cacheMu.Unlock()

	metas, scores, err := s.faissManager.Search(queryVec, topK*3)
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
		results = results[:topK]
	}

	s.cacheMu.Lock()
	s.cache = append(s.cache, cachedSearch{vec: queryVec, results: results, expiry: now.Add(searchCacheTTL)})
	s.cacheMu.Unlock()

	return results, nil
}
