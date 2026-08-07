package embeddings

import (
	"context"
)

type stubProvider struct {
	dimension int
}

func NewStubProvider(config ProviderConfig) (*stubProvider, error) {
	if config.Dimension <= 0 {
		config.Dimension = 384
	}
	return &stubProvider{dimension: config.Dimension}, nil
}

func (p *stubProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = make([]float32, p.dimension)
		for j := range result[i] {
			result[i][j] = float32(i+j) / float32(p.dimension)
		}
	}
	return result, nil
}

func (p *stubProvider) Dimension() int {
	return p.dimension
}

func (p *stubProvider) Health(ctx context.Context) error {
	return nil
}

func (p *stubProvider) Close() error {
	return nil
}
