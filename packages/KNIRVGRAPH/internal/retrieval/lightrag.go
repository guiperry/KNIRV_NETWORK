package retrieval

import (
	"KNIRVGRAPH/internal/types"
	"context"
)

type LightRAGRetriever struct {
	pipeline *RetrievalPipeline
}

func NewLightRAGRetriever(pipeline *RetrievalPipeline) *LightRAGRetriever {
	return &LightRAGRetriever{pipeline: pipeline}
}

func (r *LightRAGRetriever) Retrieve(ctx context.Context, query string, queryVec []float32, topK int) ([]types.VectorSearchResult, error) {
	results, err := r.pipeline.Search(query, queryVec, topK)
	if err != nil {
		return nil, err
	}
	for i := range results {
		if results[i].Metadata == nil {
			results[i].Metadata = map[string]interface{}{}
		}
		results[i].Metadata["strategy"] = "lightrag"
	}
	return results, nil
}
