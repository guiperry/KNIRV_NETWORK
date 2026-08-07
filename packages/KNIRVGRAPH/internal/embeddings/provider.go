package embeddings

import (
	"KNIRVGRAPH/internal/types"
	"context"
)

type Provider interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	Dimension() int
	Health(ctx context.Context) error
	Close() error
}

type ProviderConfig struct {
	Type           types.EmbeddingProviderType
	Endpoint       string
	Model          string
	Dimension      int
	BatchSize      int
	TimeoutSeconds int
	APIKey         string
}

func DefaultProviderConfig(t types.EmbeddingProviderType) ProviderConfig {
	switch t {
	case types.EmbeddingProviderOllama:
		return ProviderConfig{Type: t, Endpoint: "http://localhost:11434", Model: "nomic-embed-text", Dimension: 768, BatchSize: 32, TimeoutSeconds: 30}
	case types.EmbeddingProviderTextEmbedder:
		return ProviderConfig{Type: t, Endpoint: "http://localhost:8080", Model: "text-embedder", Dimension: 384, BatchSize: 64, TimeoutSeconds: 15}
	default:
		return ProviderConfig{Type: types.EmbeddingProviderDeterministic, Dimension: 384, BatchSize: 32, TimeoutSeconds: 10}
	}
}

func NewProvider(config ProviderConfig) (Provider, error) {
	switch config.Type {
	case types.EmbeddingProviderOllama:
		return NewOllamaProvider(config)
	case types.EmbeddingProviderTextEmbedder:
		return NewTextEmbedderProvider(config)
	case types.EmbeddingProviderStub:
		return NewStubProvider(config)
	default:
		return NewDeterministicProvider(config)
	}
}
