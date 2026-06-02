package embeddings

import "context"

type Provider interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	Dimension() int
}

type EmbeddingConfig struct {
	Provider string `yaml:"embedding_provider"`
	Model    string `yaml:"embedding_model"`
}
