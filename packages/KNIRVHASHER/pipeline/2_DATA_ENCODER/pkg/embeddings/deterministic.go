package embeddings

import (
	"fmt"
	"strings"

	"github.com/guiperry/text-embedder/embed"
)

type DeterministicService struct {
	batchSize int
}

func NewDeterministicService() *DeterministicService {
	return &DeterministicService{
		batchSize: DefaultBatchSize,
	}
}

func (s *DeterministicService) GetEmbedding(text string) ([]float32, error) {
	embedding := embed.Embed(text)
	return embedding, nil
}

func (s *DeterministicService) GetBatchEmbeddings(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	allEmbeddings := make([][]float32, 0, len(texts))
	for _, text := range texts {
		embedding := embed.Embed(text)
		allEmbeddings = append(allEmbeddings, embedding)
	}
	return allEmbeddings, nil
}

func (s *DeterministicService) GetBatchSize() int {
	return s.batchSize
}

func (s *DeterministicService) ValidateEndpoint() error {
	return nil
}

func (s *DeterministicService) SetTimeout(timeout interface{}) {
}

type EmbeddingProvider interface {
	GetEmbedding(text string) ([]float32, error)
	GetBatchEmbeddings(texts []string) ([][]float32, error)
	GetProviderStats() map[string]interface{}
}

type DeterministicEmbedder struct{}

func NewDeterministicEmbedder() *DeterministicEmbedder {
	return &DeterministicEmbedder{}
}

func (e *DeterministicEmbedder) GetEmbedding(text string) ([]float32, error) {
	cleanText := strings.TrimSpace(text)
	if len(cleanText) == 0 {
		return nil, fmt.Errorf("empty text provided")
	}
	return embed.Embed(cleanText), nil
}

func (e *DeterministicEmbedder) GetBatchEmbeddings(texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		cleanText := strings.TrimSpace(text)
		if len(cleanText) == 0 {
			continue
		}
		embeddings[i] = embed.Embed(cleanText)
	}
	return embeddings, nil
}

func (e *DeterministicEmbedder) GetProviderStats() map[string]interface{} {
	return map[string]interface{}{
		"backend":           "deterministic",
		"model":             embed.ModelID,
		"dimensions":        embed.Dims,
		"reproducible":      true,
		"requires_internet": false,
	}
}
