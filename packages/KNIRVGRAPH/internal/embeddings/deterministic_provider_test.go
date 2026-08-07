package embeddings

import (
	"context"
	"testing"
)

func TestDeterministicProvider(t *testing.T) {
	provider, err := NewDeterministicProvider(ProviderConfig{Dimension: 128})
	if err != nil {
		t.Fatalf("NewDeterministicProvider failed: %v", err)
	}
	if provider.Dimension() != 128 {
		t.Errorf("Expected dimension 128, got %d", provider.Dimension())
	}
	vecs, err := provider.Embed(context.Background(), []string{"hello world"})
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}
	if len(vecs) != 1 {
		t.Fatalf("Expected 1 vector, got %d", len(vecs))
	}
	if len(vecs[0]) != 128 {
		t.Fatalf("Expected vector dimension 128, got %d", len(vecs[0]))
	}
	if provider.Health(context.Background()) != nil {
		t.Error("Expected Health to return nil")
	}
	if provider.Close() != nil {
		t.Error("Expected Close to return nil")
	}
}

func TestStubProvider(t *testing.T) {
	provider, err := NewStubProvider(ProviderConfig{Dimension: 64})
	if err != nil {
		t.Fatalf("NewStubProvider failed: %v", err)
	}
	vecs, err := provider.Embed(context.Background(), []string{"a", "b"})
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}
	if len(vecs) != 2 {
		t.Fatalf("Expected 2 vectors, got %d", len(vecs))
	}
	for i, v := range vecs {
		if len(v) != 64 {
			t.Errorf("Expected vector %d dimension 64, got %d", i, len(v))
		}
	}
}

func TestDefaultProviderConfig(t *testing.T) {
	cfg := DefaultProviderConfig("ollama")
	if cfg.Type != "ollama" {
		t.Errorf("Expected type ollama, got %s", cfg.Type)
	}
	if cfg.Dimension != 768 {
		t.Errorf("Expected dimension 768, got %d", cfg.Dimension)
	}
}
