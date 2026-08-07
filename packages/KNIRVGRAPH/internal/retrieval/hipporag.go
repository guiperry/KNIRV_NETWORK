package retrieval

import (
	"KNIRVGRAPH/internal/types"
	"context"
)

type HippoRAGRetriever struct {
	pipeline *RetrievalPipeline
}

func NewHippoRAGRetriever(pipeline *RetrievalPipeline) *HippoRAGRetriever {
	return &HippoRAGRetriever{pipeline: pipeline}
}

func (r *HippoRAGRetriever) Retrieve(ctx context.Context, query string, queryVec []float32, topK int) ([]types.VectorSearchResult, error) {
	results, err := r.pipeline.Search(query, queryVec, topK*2)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	var unique []types.VectorSearchResult
	for _, res := range results {
		if !seen[res.ID] {
			seen[res.ID] = true
			unique = append(unique, res)
		}
	}
	if len(unique) > topK {
		unique = unique[:topK]
	}
	for i := range unique {
		if unique[i].Metadata == nil {
			unique[i].Metadata = map[string]interface{}{}
		}
		unique[i].Metadata["strategy"] = "hipporag"
	}
	return unique, nil
}
