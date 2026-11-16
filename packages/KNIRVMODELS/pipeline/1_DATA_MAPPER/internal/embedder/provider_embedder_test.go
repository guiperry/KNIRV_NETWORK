package embedder

// Plan 3.1 (PendingImplementations.md, "Embedder safety (E1)") tests: a Cloudflare failure
// mid-ingest never yields a mixed-space vector, and a changed vector space is rejected before
// it reaches a batch. Everything runs against httptest fakes on loopback: no GPU, no network,
// no native libraries.
//
// The plan's third item ("private class never reaches a fake Cloudflare") is enforced by
// Config.effectiveEmbeddingBackend in internal/app/embedding_provider.go. That package links
// libspacy_wrapper.so, so its test (internal/app/embedding_provider_test.go) cannot run on
// hosts without that library and is not duplicated here.

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// fakeCloudflare returns an httptest server that answers each request with the handler's
// (status, body) decision. The counter is the 1-based request number.
func fakeCloudflare(t *testing.T, decide func(n int32) (int, []byte), hits *atomic.Int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		status, body := decide(hits.Add(1))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func cloudflareOK(t *testing.T) []byte {
	t.Helper()
	vec := make([]float32, 768)
	for i := range vec {
		vec[i] = 0.5
	}
	b, err := json.Marshal(map[string]any{"success": true, "embedding": vec})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestCloudflareMidIngestFailureWritesNoVectors: the first text embeds, the second hits a
// Cloudflare 500. The batch must return no vectors and no embedder id, must not make a third
// request, and must not fall back to the local Ollama endpoint.
func TestCloudflareMidIngestFailureWritesNoVectors(t *testing.T) {
	var cloudHits, localHits atomic.Int32
	ok := cloudflareOK(t)
	cloud := fakeCloudflare(t, func(n int32) (int, []byte) {
		if n == 2 {
			return http.StatusInternalServerError, []byte("unavailable")
		}
		return http.StatusOK, ok
	}, &cloudHits)
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		localHits.Add(1)
		_, _ = w.Write([]byte(`{"embedding":[1,2,3]}`))
	}))
	defer local.Close()

	p := NewHybridEmbeddingProvider(cloud.URL, local.URL, "nomic-embed-text", 100)
	vectors, id, err := p.GetBatchEmbeddings([]string{"first", "second", "third"})
	if err == nil {
		t.Fatal("mid-ingest Cloudflare failure must be an error")
	}
	if vectors != nil || id != "" {
		t.Fatalf("partial batch written: vectors=%d id=%q", len(vectors), id)
	}
	if cloudHits.Load() != 2 {
		t.Fatalf("cloud requests = %d, want 2 (batch must stop at the failure)", cloudHits.Load())
	}
	if localHits.Load() != 0 {
		t.Fatalf("local endpoint called %d times: mixed-space fallback", localHits.Load())
	}
}

// TestCloudflareQuotaMidBatchWritesNoVectors: the daily cap is reached after the first text.
// The batch must fail with ErrQuotaExhausted and must not switch to the local endpoint.
func TestCloudflareQuotaMidBatchWritesNoVectors(t *testing.T) {
	var cloudHits, localHits atomic.Int32
	ok := cloudflareOK(t)
	cloud := fakeCloudflare(t, func(int32) (int, []byte) { return http.StatusOK, ok }, &cloudHits)
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		localHits.Add(1)
		_, _ = w.Write([]byte(`{"embedding":[1,2,3]}`))
	}))
	defer local.Close()

	p := NewHybridEmbeddingProvider(cloud.URL, local.URL, "nomic-embed-text", 1)
	vectors, id, err := p.GetBatchEmbeddings([]string{"first", "second"})
	if !errors.Is(err, ErrQuotaExhausted) {
		t.Fatalf("err = %v, want ErrQuotaExhausted", err)
	}
	if vectors != nil || id != "" {
		t.Fatalf("partial batch written: vectors=%d id=%q", len(vectors), id)
	}
	if cloudHits.Load() != 1 || localHits.Load() != 0 {
		t.Fatalf("cloud=%d local=%d, want 1 and 0", cloudHits.Load(), localHits.Load())
	}
}

// TestCloudflareHealthyBatchIsOneSpace: a fully successful Cloudflare batch returns one id,
// equal to the provider's EmbedderID, for every vector, and each vector has the declared width.
func TestCloudflareHealthyBatchIsOneSpace(t *testing.T) {
	var cloudHits atomic.Int32
	ok := cloudflareOK(t)
	cloud := fakeCloudflare(t, func(int32) (int, []byte) { return http.StatusOK, ok }, &cloudHits)

	p := NewHybridEmbeddingProvider(cloud.URL, "http://127.0.0.1:1", "nomic-embed-text", 100)
	vectors, id, err := p.GetBatchEmbeddings([]string{"a", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}
	want := "cloudflare:" + CloudflareModelName + "@" + CloudflareModelName + ":768:plain"
	if id != want || id != p.EmbedderID() {
		t.Fatalf("batch id = %q, want %q", id, want)
	}
	if len(vectors) != 3 {
		t.Fatalf("vectors = %d, want 3", len(vectors))
	}
	for i, v := range vectors {
		if len(v) != 768 {
			t.Fatalf("vector %d has %d dims, want 768", i, len(v))
		}
	}
}

// TestOllamaDimensionChangeMidBatchWritesNoVectors: the first vector fixes the run's vector
// space at 3 dims; the second arrives with 2. The batch must be rejected with no vectors and
// no id, and the third text must never be requested.
func TestOllamaDimensionChangeMidBatchWritesNoVectors(t *testing.T) {
	var hits atomic.Int32
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if hits.Add(1) == 1 {
			_, _ = w.Write([]byte(`{"embedding":[1,2,3]}`))
			return
		}
		_, _ = w.Write([]byte(`{"embedding":[1,2]}`))
	}))
	defer ollama.Close()

	p := NewEmbeddingProvider("ollama", "", ollama.URL, "nomic-embed-text", 10)
	vectors, id, err := p.GetBatchEmbeddings([]string{"first", "second", "third"})
	if err == nil {
		t.Fatal("dimension change inside a batch must be an error")
	}
	if vectors != nil || id != "" {
		t.Fatalf("mixed-space batch written: vectors=%d id=%q", len(vectors), id)
	}
	if hits.Load() != 2 {
		t.Fatalf("ollama requests = %d, want 2 (batch must stop at the mismatch)", hits.Load())
	}
}
