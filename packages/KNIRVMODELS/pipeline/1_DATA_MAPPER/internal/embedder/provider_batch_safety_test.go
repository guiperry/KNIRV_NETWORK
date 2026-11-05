package embedder

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestCloudflareFailureDiscardsWholeBatch(t *testing.T) {
	var cloudHits, localHits atomic.Int32
	cloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if cloudHits.Add(1) == 2 {
			http.Error(w, "unavailable", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "embedding": make([]float32, 768)})
	}))
	defer cloud.Close()
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		localHits.Add(1)
		_, _ = w.Write([]byte(`{"embedding":[1,2,3]}`))
	}))
	defer local.Close()
	provider := NewHybridEmbeddingProvider(cloud.URL, local.URL, "model", 10)
	vectors, id, err := provider.GetBatchEmbeddings([]string{"first", "second"})
	if err == nil || vectors != nil || id != "" || cloudHits.Load() != 2 || localHits.Load() != 0 {
		t.Fatalf("vectors=%v id=%q err=%v cloud=%d local=%d", vectors, id, err, cloudHits.Load(), localHits.Load())
	}
}
