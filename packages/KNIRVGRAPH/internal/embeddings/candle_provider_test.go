package embeddings

import (
	"KNIRVGRAPH/internal/types"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type candleRoundTrip func(*http.Request) (*http.Response, error)

func (f candleRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestCandleProviderHTTP(t *testing.T) {
	p, err := NewCandleProvider(ProviderConfig{Type: types.EmbeddingProviderCandle, Endpoint: "http://candle", Dimension: 2})
	if err != nil {
		t.Fatal(err)
	}
	p.client = &http.Client{Transport: candleRoundTrip(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"embeddings":[[1,0],[0,1]]}`)), Header: make(http.Header)}, nil
	})}
	vecs, err := p.Embed(context.Background(), []string{"a", "b"})
	if err != nil || len(vecs) != 2 {
		t.Fatalf("vecs=%v err=%v", vecs, err)
	}
}
