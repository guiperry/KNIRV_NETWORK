package retrieval

import (
	"KNIRVGRAPH/internal/storage"
	"KNIRVGRAPH/internal/types"
	"KNIRVGRAPH/internal/vector"
	"fmt"
	"sort"
	"sync"
)

type RetrievalPipeline struct {
	vectorIndex  *vector.VectorIndex
	bm25Index    *BM25Index
	chunkStore   ChunkStore
	chunks       []types.Chunk
	hybridWeight float64
	mu           sync.RWMutex
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

func NewPersistentRetrievalPipeline(hybridWeight float64, dimension int, metric vector.Metric, store storage.Storage) (*RetrievalPipeline, error) {
	p := NewRetrievalPipeline(hybridWeight)
	idx, err := vector.NewPersistentVectorIndex(dimension, vector.Options{Metric: metric}, store)
	if err != nil {
		return nil, err
	}
	p.vectorIndex = idx
	return p, nil
}

func (p *RetrievalPipeline) IndexChunks(chunks []types.Chunk) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range chunks {
		p.chunkStore.(*InMemoryChunkStore).Put(&c)
		p.bm25Index.Add(c.ID, c.Text)
		if len(c.Embedding) > 0 {
			_ = p.vectorIndex.Add(c.ID, c.Embedding)
		}
	}
	p.chunks = append(p.chunks, chunks...)
}

func (p *RetrievalPipeline) IndexChunksWithError(chunks []types.Chunk) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range chunks {
		p.chunkStore.(*InMemoryChunkStore).Put(&c)
		p.bm25Index.Add(c.ID, c.Text)
		if len(c.Embedding) > 0 {
			if err := p.vectorIndex.Add(c.ID, c.Embedding); err != nil {
				return err
			}
		}
	}
	p.chunks = append(p.chunks, chunks...)
	return nil
}

func (p *RetrievalPipeline) DeleteChunks(ids []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	remove := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		remove[id] = struct{}{}
		p.vectorIndex.Delete(id)
		p.bm25Index.Remove(id)
		p.chunkStore.(*InMemoryChunkStore).Delete(id)
	}
	out := p.chunks[:0]
	for _, c := range p.chunks {
		if _, ok := remove[c.ID]; !ok {
			out = append(out, c)
		}
	}
	p.chunks = out
	return nil
}

func (p *RetrievalPipeline) Optimize() error  { return p.vectorIndex.Optimize() }
func (p *RetrievalPipeline) VectorCount() int { return p.vectorIndex.Len() }

func (p *RetrievalPipeline) Search(query string, queryVec []float32, topK int) ([]types.VectorSearchResult, error) {
	return p.SearchFiltered(query, queryVec, topK, nil)
}

func (p *RetrievalPipeline) SearchFiltered(query string, queryVec []float32, topK int, filters map[string]interface{}) ([]types.VectorSearchResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
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
	results := p.fuseResults(vectorResults, bm25Results, topK*2)
	filtered := results[:0]
	for _, result := range results {
		chunk, err := p.chunkStore.GetChunk(result.ID)
		if err != nil {
			continue
		}
		if result.Metadata == nil {
			result.Metadata = map[string]interface{}{}
		}
		for k, v := range chunk.Metadata {
			result.Metadata[k] = v
		}
		result.Metadata["document_id"] = chunk.DocumentID
		result.Metadata["text"] = chunk.Text
		if matchesFilters(chunk, filters) {
			filtered = append(filtered, result)
			if len(filtered) == topK {
				break
			}
		}
	}
	return filtered, nil
}

func matchesFilters(chunk *types.Chunk, filters map[string]interface{}) bool {
	for key, want := range filters {
		var got interface{}
		switch key {
		case "kb_id", "document_id":
			got = chunk.DocumentID
		default:
			got = chunk.Metadata[key]
		}
		if fmt.Sprint(got) != fmt.Sprint(want) {
			return false
		}
	}
	return true
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
