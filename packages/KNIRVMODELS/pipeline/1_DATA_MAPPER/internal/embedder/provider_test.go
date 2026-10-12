package embedder

import (
	"math"
	"testing"
)

func TestDeterministicProvider_ImplementsInterface(t *testing.T) {
	provider := NewDeterministicProvider()

	var _ EmbeddingProvider = provider
}

func TestDeterministicProvider_GetEmbedding(t *testing.T) {
	provider := NewDeterministicProvider()

	embedding, err := provider.GetEmbedding("hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(embedding) == 0 {
		t.Fatal("expected non-empty embedding")
	}
}

func TestDeterministicProvider_OutputDimensions(t *testing.T) {
	provider := NewDeterministicProvider()

	embedding, err := provider.GetEmbedding("test text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(embedding) != 768 {
		t.Errorf("expected 768 dims, got %d", len(embedding))
	}
}

func TestDeterministicProvider_Deterministic(t *testing.T) {
	provider := NewDeterministicProvider()

	text := "the quick brown fox jumps over the lazy dog"

	v1, err := provider.GetEmbedding(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	v2, err := provider.GetEmbedding(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i := range v1 {
		if v1[i] != v2[i] {
			t.Fatalf("dimension %d differs: %v vs %v", i, v1[i], v2[i])
		}
	}
}

func TestDeterministicProvider_UnitNorm(t *testing.T) {
	provider := NewDeterministicProvider()

	texts := []string{
		"machine learning is fascinating",
		"a",
		"the quick brown fox",
		"   spaces   and\nnewlines\t",
	}

	for _, text := range texts {
		embedding, err := provider.GetEmbedding(text)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", text, err)
		}

		var sumSq float64
		for _, x := range embedding {
			sumSq += float64(x) * float64(x)
		}
		norm := math.Sqrt(sumSq)

		if math.Abs(norm-1.0) > 1e-5 {
			t.Errorf("text %q: expected unit norm, got %f", text, norm)
		}
	}
}

func TestDeterministicProvider_EmptyInput(t *testing.T) {
	provider := NewDeterministicProvider()

	_, err := provider.GetEmbedding("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestDeterministicProvider_WhitespaceInput(t *testing.T) {
	provider := NewDeterministicProvider()

	_, err := provider.GetEmbedding("   ")
	if err == nil {
		t.Error("expected error for whitespace-only input")
	}
}

func TestDeterministicProvider_GetBatchEmbeddings(t *testing.T) {
	provider := NewDeterministicProvider()

	texts := []string{
		"hello world",
		"machine learning",
		"artificial intelligence",
	}

	embeddings, err := provider.GetBatchEmbeddings(texts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(embeddings) != len(texts) {
		t.Errorf("expected %d embeddings, got %d", len(texts), len(embeddings))
	}

	for i, emb := range embeddings {
		if len(emb) != 768 {
			t.Errorf("embedding %d: expected 768 dims, got %d", i, len(emb))
		}
	}
}

func TestDeterministicProvider_BatchDeterministic(t *testing.T) {
	provider := NewDeterministicProvider()

	texts := []string{"test one", "test two"}

	batch1, err := provider.GetBatchEmbeddings(texts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	batch2, err := provider.GetBatchEmbeddings(texts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i := range batch1 {
		for j := range batch1[i] {
			if batch1[i][j] != batch2[i][j] {
				t.Fatalf("batch embedding %d dimension %d differs", i, j)
			}
		}
	}
}

func TestDeterministicProvider_GetProviderStats(t *testing.T) {
	provider := NewDeterministicProvider()

	stats := provider.GetProviderStats()

	if stats["backend"] != "deterministic" {
		t.Errorf("expected backend 'deterministic', got %v", stats["backend"])
	}

	if stats["dimensions"].(int) != 768 {
		t.Errorf("expected dimensions 768, got %v", stats["dimensions"])
	}

	if stats["reproducible"] != true {
		t.Error("expected reproducible=true")
	}

	if stats["requires_internet"] != false {
		t.Error("expected requires_internet=false")
	}
}

func TestNewEmbeddingProvider_Deterministic(t *testing.T) {
	provider := NewEmbeddingProvider("deterministic", "", "", "", 0)

	if _, ok := provider.(*DeterministicProvider); !ok {
		t.Error("expected DeterministicProvider")
	}
}

func TestNewEmbeddingProvider_Cloudflare(t *testing.T) {
	provider := NewEmbeddingProvider("cloudflare", "http://test.com", "http://ollama.com", "model", 100)

	if _, ok := provider.(*HybridEmbeddingProvider); !ok {
		t.Error("expected HybridEmbeddingProvider")
	}
}

func TestNewEmbeddingProvider_Ollama(t *testing.T) {
	provider := NewEmbeddingProvider("ollama", "http://test.com", "http://ollama.com", "model", 100)

	hybrid, ok := provider.(*HybridEmbeddingProvider)
	if !ok {
		t.Fatal("expected HybridEmbeddingProvider")
	}

	if hybrid.UseCloudflare {
		t.Error("expected UseCloudflare=false for ollama backend")
	}
}

func TestNewEmbeddingProvider_Unknown(t *testing.T) {
	provider := NewEmbeddingProvider("unknown-backend", "", "", "", 0)

	if _, ok := provider.(*DeterministicProvider); !ok {
		t.Error("expected DeterministicProvider as default")
	}
}

func TestRequestTracker_CanMakeRequest(t *testing.T) {
	tracker := NewRequestTracker(10)

	if !tracker.CanMakeRequest() {
		t.Error("expected can make request")
	}

	tracker.SetRequests(10)

	if tracker.CanMakeRequest() {
		t.Error("expected cannot make request when at limit")
	}
}

func TestRequestTracker_IncrementRequest(t *testing.T) {
	tracker := NewRequestTracker(10)

	tracker.IncrementRequest()
	tracker.IncrementRequest()
	tracker.IncrementRequest()

	used, _, _ := tracker.GetStats()

	if used != 3 {
		t.Errorf("expected 3 requests, got %d", used)
	}
}

func TestRequestTracker_DailyReset(t *testing.T) {
	tracker := NewRequestTrackerWithCount(10, 5)

	if !tracker.CanMakeRequest() {
		t.Error("expected can make request with 5 used")
	}

	tracker.SetRequests(10)

	if tracker.CanMakeRequest() {
		t.Error("expected cannot make request when at limit")
	}
}
