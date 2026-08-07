package vector

import (
	"fmt"
	"math"
	"sync"
)

type VectorIndex struct {
	dimension int
	nodes     map[string][]float32
	mu        sync.RWMutex
}

func NewVectorIndex(dimension int) *VectorIndex {
	return &VectorIndex{
		dimension: dimension,
		nodes:     make(map[string][]float32),
	}
}

func (v *VectorIndex) Add(id string, vector []float32) error {
	if len(vector) != v.dimension {
		return fmt.Errorf("dimension mismatch: expected %d, got %d", v.dimension, len(vector))
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	vec := make([]float32, len(vector))
	copy(vec, vector)
	v.nodes[id] = vec
	return nil
}

func (v *VectorIndex) Delete(id string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	delete(v.nodes, id)
}

func (v *VectorIndex) Search(query []float32, topK int) ([]string, []float64, error) {
	if len(query) != v.dimension {
		return nil, nil, fmt.Errorf("dimension mismatch: expected %d, got %d", v.dimension, len(query))
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	if topK <= 0 || topK > len(v.nodes) {
		topK = len(v.nodes)
	}
	type scored struct {
		id    string
		score float64
	}
	scores := make([]scored, 0, len(v.nodes))
	for id, vec := range v.nodes {
		score := cosineSimilarityFloat32(query, vec)
		scores = append(scores, scored{id: id, score: score})
	}
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}
	if len(scores) > topK {
		scores = scores[:topK]
	}
	ids := make([]string, len(scores))
	sims := make([]float64, len(scores))
	for i, s := range scores {
		ids[i] = s.id
		sims[i] = s.score
	}
	return ids, sims, nil
}

func (v *VectorIndex) Len() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.nodes)
}

func cosineSimilarityFloat32(a, b []float32) float64 {
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
