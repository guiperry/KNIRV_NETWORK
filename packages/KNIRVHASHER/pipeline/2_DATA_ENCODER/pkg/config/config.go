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

// GetEmbeddingBackend returns the embedding backend to use
func GetEmbeddingBackend() string {
	backend := os.Getenv("EMBEDDING_BACKEND")
	if backend == "" {
		return "deterministic" // default to deterministic
	}
	return backend
}
