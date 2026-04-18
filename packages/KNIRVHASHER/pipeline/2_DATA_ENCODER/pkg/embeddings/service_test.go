package embeddings

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNew_WithOllamaBackend(t *testing.T) {
	t.Setenv("EMBEDDING_BACKEND", "ollama")
	t.Setenv("CLOUDFLARE_EMBEDDINGS_URL", "")

	svc := New()

	if svc == nil {
		t.Fatal("expected service to not be nil")
	}

	if svc.GetBatchSize() != DefaultBatchSize {
		t.Errorf("expected batch size %d, got %d", DefaultBatchSize, svc.GetBatchSize())
	}
}

func TestNewWithBatchSize(t *testing.T) {
	t.Setenv("EMBEDDING_BACKEND", "deterministic")

	customBatchSize := 16
	svc := NewWithBatchSize(customBatchSize)

	if svc.GetBatchSize() != customBatchSize {
		t.Errorf("expected batch size %d, got %d", customBatchSize, svc.GetBatchSize())
	}
}

func TestGetBatchSize(t *testing.T) {
	t.Setenv("EMBEDDING_BACKEND", "deterministic")

	svc := New()
	if svc.GetBatchSize() != DefaultBatchSize {
		t.Errorf("GetBatchSize() = %d, want %d", svc.GetBatchSize(), DefaultBatchSize)
	}
}

func TestSetTimeout(t *testing.T) {
	t.Setenv("EMBEDDING_BACKEND", "deterministic")

	svc := New()
	newTimeout := 45 * time.Second
	svc.SetTimeout(newTimeout)
}

func TestGetBatchEmbeddingsEmpty(t *testing.T) {
	t.Setenv("EMBEDDING_BACKEND", "deterministic")

	svc := New()
	embeddings, err := svc.GetBatchEmbeddings([]string{})

	if err != nil {
		t.Errorf("unexpected error for empty input: %v", err)
	}

	if embeddings != nil {
		t.Errorf("expected nil for empty input, got %v", embeddings)
	}
}

func TestEmbeddingRequestMarshal(t *testing.T) {
	req := CloudflareWorkersRequest{
		Texts: []string{"test text", "another test"},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var unmarshaled CloudflareWorkersRequest
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(unmarshaled.Texts) != 2 {
		t.Errorf("expected 2 texts, got %d", len(unmarshaled.Texts))
	}

	if unmarshaled.Texts[0] != "test text" {
		t.Errorf("expected first text %s, got %s", "test text", unmarshaled.Texts[0])
	}
}

func TestNew_DefaultToDeterministic(t *testing.T) {
	t.Setenv("EMBEDDING_BACKEND", "")

	svc := New()

	switch svc.(type) {
	case *DeterministicService:
	default:
		t.Error("expected DeterministicService as default")
	}
}

func BenchmarkNew(b *testing.B) {
	for i := 0; i < b.N; i++ {
		New()
	}
}

func BenchmarkNewWithBatchSize(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewWithBatchSize(16)
	}
}
