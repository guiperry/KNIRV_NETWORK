package app

import (
	"strings"

	"data-mapper/internal/embedder"
)

// effectiveEmbeddingBackend enforces the data-class boundary even when a
// caller constructs Config directly instead of going through ParseFlags.
func (c *Config) effectiveEmbeddingBackend() string {
	backend := strings.ToLower(strings.TrimSpace(c.EmbeddingBackend))
	if strings.EqualFold(strings.TrimSpace(c.EmbeddingDataClass), "private") && backend == "cloudflare" && !c.AllowCloudflareForPrivate {
		return "ollama"
	}
	return backend
}

func newConfiguredEmbeddingProvider(c *Config, maxDaily int) embedder.EmbeddingProvider {
	ollamaURL := strings.TrimSuffix(c.OllamaHost, "/") + "/api/embeddings"
	return embedder.NewEmbeddingProvider(c.effectiveEmbeddingBackend(), c.CloudflareEndpoint, ollamaURL, c.OllamaModel, maxDaily)
}
