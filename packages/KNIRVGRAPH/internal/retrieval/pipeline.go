package retrieval

import (
	"KNIRVGRAPH/internal/types"
	"KNIRVGRAPH/internal/vector"
	"sort"
)

type RetrievalPipeline struct {
	vectorIndex  *vector.VectorIndex
	bm25Index    *BM25Index
	chunkStore   ChunkStore
	chunks       []types.Chunk
	hybridWeight float64
}

func NewRetrievalPipeline(hybridWeight float64) *RetrievalPipeline {
	if hybridWeight <= 0 {
		hybridWeight = 0.5
	}
	if hybridWeight > 1 {
		hybridWeight = 1
	}
	return &RetrievalPipeline{
		vectorIndex:  vector.NewVectorIndex(384),
		bm25Index:    NewBM25Index(),
		chunkStore:   NewInMemoryChunkStore(),
		hybridWeight: hybridWeight,
	}
}

func (p *RetrievalPipeline) IndexChunks(chunks []types.Chunk) {
	for _, c := range chunks {
		p.chunkStore.(*InMemoryChunkStore).Put(&c)
		p.bm25Index.Add(c.ID, c.Text)
		if len(c.Embedding) > 0 {
			p.vectorIndex.Add(c.ID, c.Embedding)
		}
	}
	p.chunks = append(p.chunks, chunks...)
}

func (p *RetrievalPipeline) Search(query string, queryVec []float32, topK int) ([]types.VectorSearchResult, error) {
	if topK <= 0 {
		topK = 10
	}
	var vectorResults []types.VectorSearchResult
	if len(queryVec) > 0 {
		ids, scores, err := p.vectorIndex.Search(queryVec, topK*2)
		if err == nil {
			vectorResults = make([]types.VectorSearchResult, len(ids))
			for i, id := range ids {
				vectorResults[i] = types.VectorSearchResult{
					ID:    id,
					Score: scores[i],
					Metadata: map[string]interface{}{
						"strategy": "vector",
					},
				}
			}
		}
	}
	bm25Results, _ := p.bm25Index.Search(query, topK*2)
	return p.fuseResults(vectorResults, bm25Results, topK), nil
}

func (p *RetrievalPipeline) fuseResults(a, b []types.VectorSearchResult, topK int) []types.VectorSearchResult {
	scores := make(map[string]float64)
	metas := make(map[string]map[string]interface{})
	for _, r := range a {
		scores[r.ID] += r.Score * p.hybridWeight
		metas[r.ID] = r.Metadata
	}
	for _, r := range b {
		scores[r.ID] += r.Score * (1 - p.hybridWeight)
		if metas[r.ID] == nil {
			metas[r.ID] = r.Metadata
		}
	}
	type item struct {
		id    string
		score float64
	}
	items := make([]item, 0, len(scores))
	for id, score := range scores {
		items = append(items, item{id: id, score: score})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].score > items[j].score })
	if len(items) > topK {
		items = items[:topK]
	}
	out := make([]types.VectorSearchResult, len(items))
	for i, it := range items {
		out[i] = types.VectorSearchResult{
			ID:       it.id,
			Score:    it.score,
			Metadata: metas[it.id],
		}
	}
	return out
}

func NormalizeScores(results []types.VectorSearchResult) []types.VectorSearchResult {
	if len(results) == 0 {
		return results
	}
	maxScore := results[0].Score
	if maxScore == 0 {
		return results
	}
	out := make([]types.VectorSearchResult, len(results))
	for i, r := range results {
		out[i] = r
		out[i].Score = r.Score / maxScore
	}
	return out
}
