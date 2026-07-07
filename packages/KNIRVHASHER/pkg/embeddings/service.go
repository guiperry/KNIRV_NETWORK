package embeddings

import "math"

type DeterministicService struct {
	BatchSize int
}

const DefaultBatchSize = 32

func NewDeterministicService() *DeterministicService {
	return &DeterministicService{BatchSize: DefaultBatchSize}
}

func (s *DeterministicService) GetEmbedding(text string) []float32 {
	embedding := make([]float32, 768)
	if len(text) == 0 {
		return embedding
	}
	for i, r := range text {
		idx := int(r) % 768
		embedding[idx] += float32(int(r)) * (float32(i%7) + 1.0)
	}
	var norm float32
	for _, v := range embedding {
		norm += v * v
	}
	if norm > 0 {
		scale := float32(math.Sqrt(float64(norm)))
		for i := range embedding {
			embedding[i] /= scale
		}
	}
	return embedding
}

func (s *DeterministicService) GetBatchEmbeddings(texts []string) [][]float32 {
	if len(texts) == 0 {
		return nil
	}
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embeddings[i] = s.GetEmbedding(text)
	}
	return embeddings
}

func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0
	}
	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0.0
	}
	return dot / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}
