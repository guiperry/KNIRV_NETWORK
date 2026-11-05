package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestPrivateEmbeddingRoutingForDirectConfig(t *testing.T) {
	for _, allow := range []bool{false, true} {
		t.Run(map[bool]string{false: "local by default", true: "explicit cloud opt-in"}[allow], func(t *testing.T) {
			var cloudHits, localHits atomic.Int32
			cloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				cloudHits.Add(1)
				_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "embedding": make([]float32, 768)})
			}))
			defer cloud.Close()
			local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				localHits.Add(1)
				_, _ = w.Write([]byte(`{"embedding":[1,2,3]}`))
			}))
			defer local.Close()
			cfg := &Config{EmbeddingBackend: " CLOUDFLARE ", EmbeddingDataClass: " PRIVATE ", AllowCloudflareForPrivate: allow, CloudflareEndpoint: cloud.URL, OllamaHost: local.URL, OllamaModel: "nomic-embed-text"}
			provider := newConfiguredEmbeddingProvider(cfg, 10)
			_, id, err := provider.GetEmbedding("private text")
			if err != nil {
				t.Fatal(err)
			}
			if allow {
				if cloudHits.Load() != 1 || localHits.Load() != 0 || !strings.HasPrefix(id, "cloudflare:") {
					t.Fatalf("cloud=%d local=%d id=%q", cloudHits.Load(), localHits.Load(), id)
				}
			} else if cloudHits.Load() != 0 || localHits.Load() != 1 || !strings.HasPrefix(id, "ollama:") {
				t.Fatalf("cloud=%d local=%d id=%q", cloudHits.Load(), localHits.Load(), id)
			}
		})
	}
}
