package retrieval

import (
	"KNIRVGRAPH/internal/types"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	_ "unsafe"
)

var (
	nonAlphaNum = regexp.MustCompile(`[^a-z0-9\s]`)
	multiSpace  = regexp.MustCompile(`\s+`)
)

type BM25Index struct {
	docs      map[string][]string
	df        map[string]int
	totalLen  int
	docCount  int
	avgDocLen float64
	k1        float64
	b         float64
}

func NewBM25Index() *BM25Index {
	return &BM25Index{
		docs: make(map[string][]string),
		df:   make(map[string]int),
		k1:   1.5,
		b:    0.75,
	}
}

func (b *BM25Index) Add(id, text string) {
	tokens := tokenize(text)
	b.docs[id] = tokens
	b.totalLen += len(tokens)
	b.docCount++
	for _, t := range tokens {
		b.df[t]++
	}
	if b.docCount > 0 {
		b.avgDocLen = float64(b.totalLen) / float64(b.docCount)
	}
}

func (b *BM25Index) Remove(id string) {
	tokens, ok := b.docs[id]
	if !ok {
		return
	}
	b.totalLen -= len(tokens)
	b.docCount--
	for _, t := range tokens {
		b.df[t]--
		if b.df[t] <= 0 {
			delete(b.df, t)
		}
	}
	delete(b.docs, id)
	if b.docCount > 0 {
		b.avgDocLen = float64(b.totalLen) / float64(b.docCount)
	}
}

func (b *BM25Index) Search(query string, topK int) ([]types.VectorSearchResult, error) {
	if topK <= 0 {
		topK = 10
	}
	queryTokens := tokenize(query)
	if len(queryTokens) == 0 {
		return nil, nil
	}
	type scored struct {
		id    string
		score float64
	}
	scores := make([]scored, 0, len(b.docs))
	for docID, docTokens := range b.docs {
		score := 0.0
		docLen := float64(len(docTokens))
		for _, qt := range queryTokens {
			tf := 0
			for _, dt := range docTokens {
				if dt == qt {
					tf++
				}
			}
			if tf == 0 {
				continue
			}
			idf := math.Log((float64(b.docCount)-float64(b.df[qt])+0.5)/(float64(b.df[qt])+0.5) + 1.0)
			if idf < 0 {
				idf = 0
			}
			numerator := float64(tf) * (b.k1 + 1)
			denominator := float64(tf) + b.k1*(1-b.b+b.b*(docLen/b.avgDocLen))
			score += idf * numerator / denominator
		}
		if score > 0 {
			scores = append(scores, scored{id: docID, score: score})
		}
	}
	sort.Slice(scores, func(i, j int) bool { return scores[i].score > scores[j].score })
	if len(scores) > topK {
		scores = scores[:topK]
	}
	result := make([]types.VectorSearchResult, len(scores))
	for i, s := range scores {
		result[i] = types.VectorSearchResult{
			ID:    s.id,
			Score: s.score,
			Metadata: map[string]interface{}{
				"strategy": "bm25",
				"text":     strings.Join(b.docs[s.id], " "),
			},
		}
	}
	return result, nil
}

func tokenize(text string) []string {
	text = strings.ToLower(text)
	text = nonAlphaNum.ReplaceAllString(text, " ")
	text = multiSpace.ReplaceAllString(text, " ")
	return strings.Fields(text)
}

type ChunkStore interface {
	GetChunk(id string) (*types.Chunk, error)
	GetChunksByDoc(documentID string) ([]types.Chunk, error)
	GetAllChunks() ([]types.Chunk, error)
}

type InMemoryChunkStore struct {
	chunks map[string]*types.Chunk
	byDoc  map[string][]string
}

func NewInMemoryChunkStore() *InMemoryChunkStore {
	return &InMemoryChunkStore{
		chunks: make(map[string]*types.Chunk),
		byDoc:  make(map[string][]string),
	}
}

func (s *InMemoryChunkStore) Put(c *types.Chunk) {
	s.chunks[c.ID] = c
	s.byDoc[c.DocumentID] = append(s.byDoc[c.DocumentID], c.ID)
}

func (s *InMemoryChunkStore) Delete(id string) {
	c, ok := s.chunks[id]
	if !ok {
		return
	}
	delete(s.chunks, id)
	ids := s.byDoc[c.DocumentID]
	out := ids[:0]
	for _, existing := range ids {
		if existing != id {
			out = append(out, existing)
		}
	}
	if len(out) == 0 {
		delete(s.byDoc, c.DocumentID)
	} else {
		s.byDoc[c.DocumentID] = out
	}
}

func (s *InMemoryChunkStore) GetChunk(id string) (*types.Chunk, error) {
	c, ok := s.chunks[id]
	if !ok {
		return nil, fmt.Errorf("chunk not found: %s", id)
	}
	return c, nil
}

func (s *InMemoryChunkStore) GetChunksByDoc(documentID string) ([]types.Chunk, error) {
	ids, ok := s.byDoc[documentID]
	if !ok {
		return nil, nil
	}
	out := make([]types.Chunk, len(ids))
	for i, id := range ids {
		out[i] = *s.chunks[id]
	}
	return out, nil
}

func (s *InMemoryChunkStore) GetAllChunks() ([]types.Chunk, error) {
	out := make([]types.Chunk, 0, len(s.chunks))
	for _, c := range s.chunks {
		out = append(out, *c)
	}
	return out, nil
}

func BuildBM25FromChunks(chunks []types.Chunk) *BM25Index {
	idx := NewBM25Index()
	for _, c := range chunks {
		idx.Add(c.ID, c.Text)
	}
	return idx
}

func MarshalSearchResults(results []types.VectorSearchResult) ([]byte, error) {
	return json.Marshal(results)
}
