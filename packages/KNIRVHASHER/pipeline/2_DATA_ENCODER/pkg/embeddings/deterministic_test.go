package embeddings

import (
	"math"
	"testing"
)

func TestDeterministicService_ImplementsInterface(t *testing.T) {
	svc := NewDeterministicService()

	var _ EmbeddingService = svc
}

func TestDeterministicService_GetEmbedding(t *testing.T) {
	svc := NewDeterministicService()

	embedding, err := svc.GetEmbedding("hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(embedding) == 0 {
		t.Fatal("expected non-empty embedding")
	}

	if len(embedding) != 768 {
		t.Errorf("expected 768 dims, got %d", len(embedding))
	}
}

func TestDeterministicService_Deterministic(t *testing.T) {
	svc := NewDeterministicService()

	text := "the quick brown fox jumps over the lazy dog"

	v1, err := svc.GetEmbedding(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	v2, err := svc.GetEmbedding(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i := range v1 {
		if v1[i] != v2[i] {
			t.Fatalf("dimension %d differs: %v vs %v", i, v1[i], v2[i])
		}
	}
}

func TestDeterministicService_UnitNorm(t *testing.T) {
	svc := NewDeterministicService()

	texts := []string{
		"machine learning is fascinating",
		"a",
		"the quick brown fox",
		"   spaces   and\nnewlines\t",
	}

	for _, text := range texts {
		embedding, err := svc.GetEmbedding(text)
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

func TestDeterministicService_GetBatchEmbeddings(t *testing.T) {
	svc := NewDeterministicService()

	texts := []string{
		"hello world",
		"machine learning",
		"artificial intelligence",
	}

	embeddings, err := svc.GetBatchEmbeddings(texts)
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

func TestDeterministicService_BatchEmpty(t *testing.T) {
	svc := NewDeterministicService()

	embeddings, err := svc.GetBatchEmbeddings([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if embeddings != nil {
		t.Errorf("expected nil for empty input, got %v", embeddings)
	}
}

func TestDeterministicService_BatchDeterministic(t *testing.T) {
	svc := NewDeterministicService()

	texts := []string{"test one", "test two"}

	batch1, err := svc.GetBatchEmbeddings(texts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	batch2, err := svc.GetBatchEmbeddings(texts)
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

func TestDeterministicService_GetBatchSize(t *testing.T) {
	svc := NewDeterministicService()

	if svc.GetBatchSize() != DefaultBatchSize {
		t.Errorf("expected batch size %d, got %d", DefaultBatchSize, svc.GetBatchSize())
	}
}

func TestDeterministicService_GetBatchSizeCustom(t *testing.T) {
	svc := NewDeterministicService()
	svc.batchSize = 16

	if svc.GetBatchSize() != 16 {
		t.Errorf("expected batch size 16, got %d", svc.GetBatchSize())
	}
}

func TestDeterministicService_ValidateEndpoint(t *testing.T) {
	svc := NewDeterministicService()

	err := svc.ValidateEndpoint()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDeterministicService_SetTimeout(t *testing.T) {
	svc := NewDeterministicService()

	svc.SetTimeout(30)
	svc.SetTimeout(30 * 1e9)
}

func TestDeterministicEmbedder_ImplementsProviderInterface(t *testing.T) {
	emb := NewDeterministicEmbedder()

	var _ EmbeddingProvider = emb
}

func TestDeterministicEmbedder_GetEmbedding(t *testing.T) {
	emb := NewDeterministicEmbedder()

	embedding, err := emb.GetEmbedding("hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(embedding) != 768 {
		t.Errorf("expected 768 dims, got %d", len(embedding))
	}
}

func TestDeterministicEmbedder_Deterministic(t *testing.T) {
	emb := NewDeterministicEmbedder()

	text := "test deterministic"

	v1, _ := emb.GetEmbedding(text)
	v2, _ := emb.GetEmbedding(text)

	for i := range v1 {
		if v1[i] != v2[i] {
			t.Fatalf("dimension %d differs", i)
		}
	}
}

func TestDeterministicEmbedder_EmptyInput(t *testing.T) {
	emb := NewDeterministicEmbedder()

	_, err := emb.GetEmbedding("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestDeterministicEmbedder_GetBatchEmbeddings(t *testing.T) {
	emb := NewDeterministicEmbedder()

	texts := []string{"a", "b", "c"}

	embeddings, err := emb.GetBatchEmbeddings(texts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(embeddings) != len(texts) {
		t.Errorf("expected %d embeddings, got %d", len(texts), len(embeddings))
	}
}

func TestDeterministicEmbedder_GetProviderStats(t *testing.T) {
	emb := NewDeterministicEmbedder()

	stats := emb.GetProviderStats()

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

func TestNew_WithDeterministicBackend(t *testing.T) {
	t.Setenv("EMBEDDING_BACKEND", "deterministic")

	svc := New()

	if _, ok := svc.(*DeterministicService); !ok {
		t.Error("expected DeterministicService")
	}
}

func TestNewWithBatchSize_WithDeterministicBackend(t *testing.T) {
	t.Setenv("EMBEDDING_BACKEND", "deterministic")

	svc := NewWithBatchSize(64)

	if svc.GetBatchSize() != 64 {
		t.Errorf("expected batch size 64, got %d", svc.GetBatchSize())
	}
}
