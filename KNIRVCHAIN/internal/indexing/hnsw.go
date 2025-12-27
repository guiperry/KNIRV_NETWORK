package indexing

import (
	"github.com/google/uuid"
)

type HNSWIndex struct {
	dimension int
	vectors   map[uuid.UUID][]float32
}

func NewHNSWIndex(dimension, m, efConstruction int) *HNSWIndex {
	return &HNSWIndex{
		dimension: dimension,
		vectors:   make(map[uuid.UUID][]float32),
	}
}

func (h *HNSWIndex) Add(id uuid.UUID, vector []float32) error {
	h.vectors[id] = vector
	return nil
}

func (h *HNSWIndex) Search(query []float32, k int) ([]uuid.UUID, error) {
	// Simple brute force search
	type result struct {
		id    uuid.UUID
		score float64
	}

	var results []result
	for id, vec := range h.vectors {
		score := cosineSimilarity(query, vec)
		results = append(results, result{id, score})
	}

	// Sort by score descending
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].score < results[j].score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	var ids []uuid.UUID
	for i, r := range results {
		if i >= k {
			break
		}
		ids = append(ids, r.id)
	}

	return ids, nil
}

func (h *HNSWIndex) Remove(id uuid.UUID) error {
	delete(h.vectors, id)
	return nil
}

func cosineSimilarity(v1, v2 []float32) float64 {
	if len(v1) != len(v2) {
		return 0.0
	}

	var dot, norm1, norm2 float64
	for i := range v1 {
		dot += float64(v1[i] * v2[i])
		norm1 += float64(v1[i] * v1[i])
		norm2 += float64(v2[i] * v2[i])
	}

	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}

	return dot / (sqrt(norm1) * sqrt(norm2))
}

func sqrt(x float64) float64 {
	if x < 0 {
		return 0
	}
	z := 1.0
	for i := 0; i < 10; i++ {
		z -= (z*z - x) / (2 * z)
	}
	return z
}