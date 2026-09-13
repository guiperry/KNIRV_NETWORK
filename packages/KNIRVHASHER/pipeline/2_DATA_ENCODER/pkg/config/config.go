package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// LoadEnv loads environment variables from .env file
func LoadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}
}

// GetCloudflareEndpoint returns the Cloudflare embeddings endpoint from environment variables
func GetCloudflareEndpoint() string {
	return os.Getenv("CLOUDFLARE_EMBEDDINGS_URL")
}

// GetLlamaEndpoint returns the OpenAI-compatible knirvllama embeddings API.
// It intentionally defaults to the local launcher endpoint; callers opt into
// this backend with EMBEDDING_BACKEND=llama.
func GetLlamaEndpoint() string {
	endpoint := os.Getenv("KNIRVLLAMA_URL")
	if endpoint == "" {
		return "http://127.0.0.1:8080/v1/embeddings"
	}
	return endpoint
}

func GetLlamaEmbeddingModel() string {
	if model := os.Getenv("KNIRVLLAMA_EMBEDDING_MODEL"); model != "" {
		return model
	}
	return "knirv-embed"
}

// GetEmbeddingBackend returns the embedding backend to use
func GetEmbeddingBackend() string {
	backend := os.Getenv("EMBEDDING_BACKEND")
	if backend == "" {
		return "deterministic" // default to deterministic
	}
	return backend
}
