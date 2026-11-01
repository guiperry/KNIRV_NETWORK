package replica

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPSourceReadsPublicHead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/oracle/v3/replication/head" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		_, _ = w.Write([]byte(`{"head":{"seq":1},"signed":{}}`))
	}))
	defer server.Close()
	source, err := NewHTTPSource(server.URL, "oracle-test", nil, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	head, err := source.Head(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if head.Head.Seq != 1 {
		t.Fatalf("head sequence = %d, want 1", head.Head.Seq)
	}
}
