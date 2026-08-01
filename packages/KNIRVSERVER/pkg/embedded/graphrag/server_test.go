package graphrag

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"
	"time"
)

// TestServerEmbedRoundTrip exercises the actual Unix-socket wire path
// end-to-end: Client dials the socket StartServer binds, and the request
// reaches the real CGo/Rust graphrag_embed_texts export. EmbedTexts is
// stateless in ffi.rs (it doesn't touch the process-global INSTANCE), so
// this is safe to run without first calling Init() — unlike Query/
// IndexDocument, which do require an initialized engine.
func TestServerEmbedRoundTrip(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "graphrag-embed-test.sock")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, err := StartServer(ctx, socketPath)
	if err != nil {
		t.Fatalf("StartServer failed: %v", err)
	}
	defer srv.Close()

	c := NewClient(nil, socketPath)
	embeddings, err := c.EmbedTexts(context.Background(), []string{"hello world"})
	if err != nil {
		t.Fatalf("EmbedTexts over socket failed: %v", err)
	}
	if len(embeddings) != 1 {
		t.Fatalf("expected 1 embedding, got %d", len(embeddings))
	}
	if len(embeddings[0]) == 0 {
		t.Error("expected non-empty embedding vector")
	}
}

// TestServerHealthRoundTripUninitialized confirms the socket bridge itself
// is reachable and correctly reports engine health over the wire, even when
// (as in this test binary) the engine was never Init'd.
func TestServerHealthRoundTripUninitialized(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "graphrag-health-test.sock")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, err := StartServer(ctx, socketPath)
	if err != nil {
		t.Fatalf("StartServer failed: %v", err)
	}
	defer srv.Close()

	client := unixHTTPClient(socketPath, 3*time.Second)
	resp, err := client.Get("http://unix/health")
	if err != nil {
		t.Fatalf("health request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503 for an uninitialized engine, got %d", resp.StatusCode)
	}
}

// TestServerQueryUninitializedReturnsError confirms a query against an
// uninitialized engine fails over the socket with a clear error rather than
// hanging or the request itself erroring out — this is the exact failure
// mode backend_server hit before Init() was wired into KNIRVSERVER's boot
// sequence (see startGraphRAG in main.go).
func TestServerQueryUninitializedReturnsError(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "graphrag-query-test.sock")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, err := StartServer(ctx, socketPath)
	if err != nil {
		t.Fatalf("StartServer failed: %v", err)
	}
	defer srv.Close()

	c := NewClient(nil, socketPath)
	c.mu.Lock()
	c.indexes["default"] = &Index{KBID: "default", Status: "ready"}
	c.mu.Unlock()

	_, err = c.Query(context.Background(), "default", &GraphQuery{Query: "test", Limit: 5})
	if err == nil {
		t.Fatal("expected an error querying an uninitialized engine")
	}
}
