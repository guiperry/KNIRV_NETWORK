package graphrag

import (
	"context"
	"log/slog"
	"os"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient(nil)
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	if c.indexes == nil {
		t.Error("expected indexes map to be initialized")
	}
	if c.logger == nil {
		t.Error("expected default logger when nil passed")
	}
}

func TestNewClientWithLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	c := NewClient(logger)
	if c.logger != logger {
		t.Error("expected provided logger to be used")
	}
}

func TestClientQueryNil(t *testing.T) {
	c := NewClient(nil)
	_, err := c.Query(context.Background(), "test-kb", nil)
	if err == nil {
		t.Fatal("expected error for nil query")
	}
}

func TestClientQueryMissingIndex(t *testing.T) {
	c := NewClient(nil)
	q := &GraphQuery{Query: "test", Limit: 10}
	_, err := c.Query(context.Background(), "missing-kb", q)
	if err == nil || err.Error() != "index not found for knowledge base: missing-kb" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientQueryBuildingStatus(t *testing.T) {
	c := NewClient(nil)
	c.mu.Lock()
	c.indexes["test-kb"] = &Index{KBID: "test-kb", Status: "building"}
	c.mu.Unlock()

	q := &GraphQuery{Query: "test", Limit: 10}
	_, err := c.Query(context.Background(), "test-kb", q)
	if err == nil {
		t.Fatal("expected error for building status")
	}
}

func TestClientBuildIndex(t *testing.T) {
	c := NewClient(nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.BuildIndex(ctx, "test-kb", "full")
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestClientGetIndexStatusMissing(t *testing.T) {
	c := NewClient(nil)
	status, err := c.GetIndexStatus(context.Background(), "missing-kb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Status != "not_built" {
		t.Errorf("expected not_built, got %s", status.Status)
	}
	if status.KBId != "missing-kb" {
		t.Errorf("expected kb_id missing-kb, got %s", status.KBId)
	}
}

func TestClientGetIndexStatusReady(t *testing.T) {
	c := NewClient(nil)
	c.mu.Lock()
	c.indexes["ready-kb"] = &Index{
		KBID:        "ready-kb",
		Status:      "ready",
		NodesCount:  100,
		EdgesCount:  200,
		ChunksCount: 500,
	}
	c.mu.Unlock()

	status, err := c.GetIndexStatus(context.Background(), "ready-kb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Status != "ready" {
		t.Errorf("expected ready, got %s", status.Status)
	}
	if status.NodesCount != 100 {
		t.Errorf("expected 100 nodes, got %d", status.NodesCount)
	}
}

func TestClientGetIndexStatusError(t *testing.T) {
	c := NewClient(nil)
	c.mu.Lock()
	c.indexes["error-kb"] = &Index{
		KBID:   "error-kb",
		Status: "error",
		Err:    os.ErrNotExist,
	}
	c.mu.Unlock()

	status, err := c.GetIndexStatus(context.Background(), "error-kb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Status != "error" {
		t.Errorf("expected error, got %s", status.Status)
	}
	if status.ErrorMessage == "" {
		t.Error("expected non-empty error message")
	}
}

func TestClientIndexDocumentEmptyDocID(t *testing.T) {
	c := NewClient(nil)
	err := c.IndexDocument(context.Background(), "", []byte("content"))
	if err == nil {
		t.Fatal("expected error for empty docID")
	}
}

func TestClientIndexDocumentEmptyContent(t *testing.T) {
	c := NewClient(nil)
	err := c.IndexDocument(context.Background(), "test-doc", nil)
	if err == nil {
		t.Fatal("expected error for nil content")
	}
}

func TestClientEmbedTextsEmpty(t *testing.T) {
	c := NewClient(nil)
	_, err := c.EmbedTexts(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for empty texts")
	}
}

func TestClientClose(t *testing.T) {
	c := NewClient(nil)
	c.mu.Lock()
	c.indexes["test"] = &Index{KBID: "test", Status: "ready"}
	c.mu.Unlock()

	_ = c.Close()
	if len(c.indexes) != 0 {
		t.Error("expected empty indexes map after close")
	}
}
