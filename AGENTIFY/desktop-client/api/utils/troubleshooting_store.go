package utils
// Package utils provides utility functions for the API


import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TroubleshootingVectorStore represents the vector store for troubleshooting data
type TroubleshootingVectorStore struct {
	Chunks     []TroubleshootingChunk  `json:"chunks"`
	Embeddings [][]float64             `json:"embeddings"`
	Metadata   []TroubleshootingMeta   `json:"metadata"`
}

// TroubleshootingChunk represents a chunk of troubleshooting information
type TroubleshootingChunk struct {
	Category string   `json:"category"`
	Issue    string   `json:"issue"`
	Symptoms []string `json:"symptoms"`
	Content  string   `json:"content"`
	RawHTML  string   `json:"raw_html"`
}

// TroubleshootingMeta represents metadata for a troubleshooting chunk
type TroubleshootingMeta struct {
	Category string   `json:"category"`
	Issue    string   `json:"issue"`
	Symptoms []string `json:"symptoms"`
}

// SearchResult represents a search result from the vector store
type SearchResult struct {
	Chunk    TroubleshootingChunk
	Metadata TroubleshootingMeta
	Score    float64
}

// TroubleshootingStore manages access to the troubleshooting vector store
type TroubleshootingStore struct {
	store *TroubleshootingVectorStore
}

// NewTroubleshootingStore creates a new troubleshooting store
func NewTroubleshootingStore(filePath string) (*TroubleshootingStore, error) {
	// If filePath is not absolute, make it relative to the current directory
	if !filepath.IsAbs(filePath) {
		dir, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		filePath = filepath.Join(dir, filePath)
	}

	// Read the JSON file
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read vector store file: %w", err)
	}

	// Parse the JSON
	var store TroubleshootingVectorStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("failed to parse vector store: %w", err)
	}

	// Validate the store
	if len(store.Chunks) != len(store.Embeddings) || len(store.Chunks) != len(store.Metadata) {
		return nil, fmt.Errorf("invalid vector store: chunks, embeddings, and metadata must have the same length")
	}

	return &TroubleshootingStore{
		store: &store,
	}, nil
}

// Search searches the vector store for relevant troubleshooting information
func (ts *TroubleshootingStore) Search(query []float64, topK int) ([]SearchResult, error) {
	if ts.store == nil {
		return nil, fmt.Errorf("vector store not initialized")
	}

	// Calculate cosine similarity between query and all embeddings
	similarities := make([]float64, len(ts.store.Embeddings))
	for i, embedding := range ts.store.Embeddings {
		similarities[i] = cosineSimilarity(query, embedding)
	}

	// Create search results with scores
	results := make([]SearchResult, len(ts.store.Chunks))
	for i := range ts.store.Chunks {
		results[i] = SearchResult{
			Chunk:    ts.store.Chunks[i],
			Metadata: ts.store.Metadata[i],
			Score:    similarities[i],
		}
	}

	// Sort by similarity score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Return top K results
	if topK > 0 && topK < len(results) {
		results = results[:topK]
	}

	return results, nil
}

// SearchByErrorType searches for troubleshooting information based on error type and symptoms
func (ts *TroubleshootingStore) SearchByErrorType(errorType string, symptoms []string, topK int) ([]SearchResult, error) {
	if ts.store == nil {
		return nil, fmt.Errorf("vector store not initialized")
	}

	// Simple keyword matching for now (can be improved with embeddings)
	var results []SearchResult

	for i, chunk := range ts.store.Chunks {
		score := 0.0

		// Match category
		if containsIgnoreCase(chunk.Category, errorType) {
			score += 0.5
		}

		// Match issue
		if containsIgnoreCase(chunk.Issue, errorType) {
			score += 0.7
		}

		// Match symptoms
		for _, symptom := range symptoms {
			for _, chunkSymptom := range chunk.Symptoms {
				if containsIgnoreCase(chunkSymptom, symptom) {
					score += 0.3
				}
			}
		}

		// Add to results if there's any match
		if score > 0 {
			results = append(results, SearchResult{
				Chunk:    chunk,
				Metadata: ts.store.Metadata[i],
				Score:    score,
			})
		}
	}

	// Sort by score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Return top K results
	if topK > 0 && topK < len(results) {
		results = results[:topK]
	}

	return results, nil
}

// GetAllCategories returns all troubleshooting categories
func (ts *TroubleshootingStore) GetAllCategories() []string {
	if ts.store == nil {
		return nil
	}

	// Use a map to deduplicate categories
	categories := make(map[string]bool)
	for _, chunk := range ts.store.Chunks {
		categories[chunk.Category] = true
	}

	// Convert map keys to slice
	result := make([]string, 0, len(categories))
	for category := range categories {
		result = append(result, category)
	}

	sort.Strings(result)
	return result
}

// GetChunksByCategory returns all chunks for a specific category
func (ts *TroubleshootingStore) GetChunksByCategory(category string) []TroubleshootingChunk {
	if ts.store == nil {
		return nil
	}

	var result []TroubleshootingChunk
	for _, chunk := range ts.store.Chunks {
		if chunk.Category == category {
			result = append(result, chunk)
		}
	}

	return result
}

// Helper function to calculate cosine similarity between two vectors
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	normA = math.Sqrt(normA)
	normB = math.Sqrt(normB)

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (normA * normB)
}

// Helper function to check if a string contains another string (case insensitive)
func containsIgnoreCase(s, substr string) bool {
	s, substr = strings.ToLower(s), strings.ToLower(substr)
	return strings.Contains(s, substr)
}