package embeddings

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLlamaEmbeddingsUseOpenAICompatibleAPI(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("local TCP listeners unavailable: %v", err)
	}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		var request OpenAIEmbeddingsRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Model != "local-model" || len(request.Input) != 2 {
			t.Fatalf("unexpected request: %#v", request)
		}
		_ = json.NewEncoder(w).Encode(OpenAIEmbeddingsResponse{Data: []OpenAIEmbedding{{Index: 1, Embedding: []float32{2}}, {Index: 0, Embedding: []float32{1}}}})
	}))
	server.Listener = listener
	server.Start()
	defer server.Close()
	t.Setenv("EMBEDDING_BACKEND", "llama")
	t.Setenv("KNIRVLLAMA_URL", server.URL+"/v1/embeddings")
	t.Setenv("KNIRVLLAMA_EMBEDDING_MODEL", "local-model")
	embeddings, err := New().GetBatchEmbeddings([]string{"one", "two"})
	if err != nil {
		t.Fatal(err)
	}
	if len(embeddings) != 2 || embeddings[0][0] != 1 || embeddings[1][0] != 2 {
		t.Fatalf("unexpected embeddings: %#v", embeddings)
	}
}

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
