package embeddings

import (
	"context"
	"hash/fnv"
	"math"
)

type deterministicProvider struct {
	dimension int
}

func NewDeterministicProvider(config ProviderConfig) (*deterministicProvider, error) {
	if config.Dimension <= 0 {
		config.Dimension = 384
	}
	return &deterministicProvider{dimension: config.Dimension}, nil
}

func (p *deterministicProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i, text := range texts {
		result[i] = p.hashEmbedding(text)
	}
	return result, nil
}

func (p *deterministicProvider) Dimension() int {
	return p.dimension
}

func (p *deterministicProvider) Health(ctx context.Context) error {
	return nil
}

func (p *deterministicProvider) Close() error {
	return nil
}

func (p *deterministicProvider) hashEmbedding(text string) []float32 {
	vec := make([]float32, p.dimension)
	h := fnv.New64a()
	h.Write([]byte(text))
	sum := h.Sum64()
	for i := 0; i < p.dimension; i++ {
		val := float32((sum >> (i % 64)) & 0xFF)
		vec[i] = (val/127.0 - 1.0)
	}
	return l2Normalize(vec)
}

func l2Normalize(vec []float32) []float32 {
	var mag float32
	for _, v := range vec {
		mag += v * v
	}
	if mag == 0 {
		return vec
	}
	mag = float32(math.Sqrt(float64(mag)))
	out := make([]float32, len(vec))
	for i, v := range vec {
		out[i] = v / mag
	}
	return out
}
